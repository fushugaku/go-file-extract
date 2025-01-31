package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// Constants for default values
const DefaultDelimiter = "======"

// Config represents the application's configuration.
type Config struct {
	Folders             map[string]FolderConfig `json:"folders"`
	FileTypeExecutables map[string]string       `json:"file_type_executables"` // Map of file extensions to executables
}

// FolderConfig represents saved configurations for a folder.
type FolderConfig struct {
	SavedName map[string][]string `json:"saved_name"`
}

// App encapsulates the application's state and dependencies.
type App struct {
	Config     Config
	ConfigPath string
}

// NewApp initializes a new App instance.
func NewApp(configPath string) (*App, error) {
	app := &App{
		Config: Config{
			Folders:             make(map[string]FolderConfig),
			FileTypeExecutables: make(map[string]string),
		},
		ConfigPath: configPath,
	}
	// Load the configuration file if it exists
	if err := app.loadConfig(); err != nil {
		return nil, err
	}
	return app, nil
}

// loadConfig loads the configuration from the specified path.
func (app *App) loadConfig() error {
	data, err := os.ReadFile(app.ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No config file exists yet
		}
		return fmt.Errorf("failed to read config file: %v", err)
	}
	if err := json.Unmarshal(data, &app.Config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}
	return nil
}

// saveConfig saves the current configuration to the specified path.
func (app *App) saveConfig() error {
	data, err := json.MarshalIndent(app.Config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(app.ConfigPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}
	if err := os.WriteFile(app.ConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}
	return nil
}

// getSavedConfig retrieves the saved configuration for the given folder and name.
func (app *App) getSavedConfig(currentDir, name string) ([]string, error) {
	folderConfig, exists := app.Config.Folders[currentDir]
	if !exists || folderConfig.SavedName == nil || len(folderConfig.SavedName[name]) == 0 {
		return nil, fmt.Errorf("no saved arguments found for name '%s' in folder '%s'", name, currentDir)
	}
	return folderConfig.SavedName[name], nil
}

// saveCurrentConfig saves the current arguments under the specified name for the given folder.
func (app *App) saveCurrentConfig(currentDir, name string, args []string) error {
	if app.Config.Folders == nil {
		app.Config.Folders = make(map[string]FolderConfig)
	}
	folderConfig := app.Config.Folders[currentDir]
	if folderConfig.SavedName == nil {
		folderConfig.SavedName = make(map[string][]string)
	}
	// Filter out -name and its value before saving
	filteredArgs := filterOutFlag(args, "-name")
	folderConfig.SavedName[name] = filteredArgs
	app.Config.Folders[currentDir] = folderConfig
	return app.saveConfig()
}

// filterOutFlag removes the specified flag and its value from the arguments list.
func filterOutFlag(args []string, flag string) []string {
	var filteredArgs []string
	for i := 0; i < len(args); i++ {
		if args[i] == flag {
			// Skip the flag and its value
			i++
			continue
		}
		filteredArgs = append(filteredArgs, args[i])
	}
	return filteredArgs
}

// parseArguments parses command-line arguments into structured data.
func parseArguments(args []string) (files []string, ignorePattern string, ignoreGitIgnore bool, delimiter string, wrapCode bool, saveName, byName, execCommand string, fileExecs map[string]string, err error) {
	fileExecs = make(map[string]string)
	delimiter = DefaultDelimiter // Set default delimiter
	wrapCode = true              // Default to true

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-ignore-pattern":
			if i+1 >= len(args) {
				return nil, "", false, "", false, "", "", "", nil, errors.New("missing value for -ignore-pattern")
			}
			ignorePattern = args[i+1]
			i++
		case "-ignore-gitignore":
			ignoreGitIgnore = true
		case "-delimiter":
			if i+1 >= len(args) {
				return nil, "", false, "", false, "", "", "", nil, errors.New("missing value for -delimiter")
			}
			delimiter = args[i+1]
			i++
		case "-wrap-code":
			if i+1 >= len(args) {
				return nil, "", false, "", false, "", "", "", nil, errors.New("missing value for -wrap-code")
			}
			wrapCodeStr := args[i+1]
			if wrapCodeStr == "false" {
				wrapCode = false
			}
			i++
		case "-name":
			if i+1 >= len(args) {
				return nil, "", false, "", false, "", "", "", nil, errors.New("missing value for -name")
			}
			saveName = args[i+1]
			i++
		case "-by-name":
			if i+1 >= len(args) {
				return nil, "", false, "", false, "", "", "", nil, errors.New("missing value for -by-name")
			}
			byName = args[i+1]
			i++
		case "-files":
			if i+1 >= len(args) {
				return nil, "", false, "", false, "", "", "", nil, errors.New("missing value for -files")
			}
			for i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				files = append(files, args[i+1])
				i++
			}
		case "-exec":
			if i+1 >= len(args) {
				return nil, "", false, "", false, "", "", "", nil, errors.New("missing value for -exec")
			}
			execCommand = args[i+1]
			i++
		case "-file-exec":
			if i+1 >= len(args) {
				return nil, "", false, "", false, "", "", "", nil, errors.New("missing value for -file-exec")
			}
			pairs := strings.Fields(args[i+1]) // Split by spaces to handle multiple pairs
			for _, pair := range pairs {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) != 2 {
					return nil, "", false, "", false, "", "", "", nil, errors.New("invalid format for -file-exec. Expected '.ext=executable'")
				}
				fileExecs[parts[0]] = parts[1]
			}
			i++
		default:
			return nil, "", false, "", false, "", "", "", nil, fmt.Errorf("unknown argument: %s", args[i])
		}
	}
	return files, ignorePattern, ignoreGitIgnore, delimiter, wrapCode, saveName, byName, execCommand, fileExecs, nil
}

// getData processes files, runs executables, and generates output.
func getData(files []string, ignorePattern string, ignoreGitIgnore bool, delimiter string, wrapCode bool, execCommand string, fileExecs map[string]string, fileTypeExecutables map[string]string) (string, error) {
	var output strings.Builder

	// Compile regex for ignore pattern
	var ignoreRegex *regexp.Regexp
	if ignorePattern != "" {
		var err error
		ignoreRegex, err = regexp.Compile(ignorePattern)
		if err != nil {
			return "", fmt.Errorf("invalid regex pattern: %v", err)
		}
	}

	// Load .gitignore rules if needed
	var gitIgnoreMatcher gitignore.Matcher
	if !ignoreGitIgnore {
		_, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
		if err == nil {
			patterns, err := gitignore.ReadPatterns(osfs.New("."), []string{})
			if err != nil {
				log.Printf("Error reading .gitignore patterns: %v", err)
			} else {
				gitIgnoreMatcher = gitignore.NewMatcher(patterns)
			}
		}
	}

	// Merge FileTypeExecutables from config and command-line overrides
	finalFileTypeExecutables := make(map[string]string)
	for ext, cmd := range fileTypeExecutables {
		finalFileTypeExecutables[ext] = cmd
	}
	for ext, cmd := range fileExecs {
		finalFileTypeExecutables[ext] = cmd
	}

	// Map of file extensions to programming languages
	languageMap := map[string]string{
		".go":   "go",
		".js":   "javascript",
		".ts":   "typescript",
		".fish": "fish",
		".py":   "python",
		".java": "java",
		".cpp":  "cpp",
		".c":    "c",
		".html": "html",
		".css":  "css",
		".sh":   "bash",
		".md":   "markdown",
		".json": "json",
		".yaml": "yaml",
		".yml":  "yaml",
		".rs":   "rust",
		".php":  "php",
		".rb":   "ruby",
	}

	// Process each file
	for _, filePath := range files {
		// Check if file should be ignored by regex
		if ignoreRegex != nil && ignoreRegex.MatchString(filePath) {
			continue
		}

		// Check if file should be ignored by .gitignore
		if !ignoreGitIgnore && gitIgnoreMatcher != nil {
			relPath, err := filepath.Rel(".", filePath)
			if err != nil {
				log.Printf("Error getting relative path for %s: %v", filePath, err)
				continue
			}
			if gitIgnoreMatcher.Match([]string{relPath}, false) {
				continue
			}
		}

		// Detect file extension
		ext := filepath.Ext(filePath)

		// Determine the executable command for this file type
		executable := ""
		if execCommand != "" {
			// Use the command-line override if provided
			executable = execCommand
		} else if cmd, exists := finalFileTypeExecutables[ext]; exists {
			// Use the executable from the merged map
			executable = cmd
		}

		// Run the executable if one is specified
		var executableOutput string
		if executable != "" {
			// Split the executable and its arguments
			parts := strings.Fields(executable)
			if len(parts) == 0 {
				return "", fmt.Errorf("invalid executable command: %s", executable)
			}
			cmd := exec.Command(parts[0], append(parts[1:], filePath)...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("failed to run executable '%s' with file '%s': %v\nOutput: %s", executable, filePath, err, string(out))
			}
			executableOutput = string(out)
		}

		// Read file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("Error reading file %s: %v", filePath, err)
			continue
		}

		// Detect language based on file extension
		language := languageMap[ext]
		if language == "" {
			language = "plaintext" // Default to plaintext if no match found
		}

		// Append output to buffer
		output.WriteString(filePath + "\n")
		if wrapCode {
			output.WriteString(fmt.Sprintf("```%s\n", language))
		}
		output.WriteString(string(content) + "\n")
		if wrapCode {
			output.WriteString("```\n")
		}

		// Add executable output before the delimiter
		if executableOutput != "" {
			output.WriteString(executableOutput + "\n")
		}
		output.WriteString(delimiter + "\n")
	}
	return output.String(), nil
}

func main() {
	// Initialize the application
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}
	configPath := filepath.Join(homeDir, ".config", "your_app_name", "config.json")
	app, err := NewApp(configPath)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Parse initial command-line arguments
	args := os.Args[1:]
	var ignorePattern string
	ignoreGitIgnore := false
	delimiter := DefaultDelimiter // Default delimiter
	wrapCode := true              // Default to true
	var saveName, execCommand string
	var fileExecs map[string]string
	var files []string

	// Handle interactive selection if no arguments are provided
	if len(args) == 0 {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current directory: %v", err)
		}

		// Load all saved names for the current folder
		folderConfig, exists := app.Config.Folders[currentDir]
		if !exists || len(folderConfig.SavedName) == 0 {
			log.Fatalf("No saved configurations found for folder '%s'", currentDir)
		}

		// List saved names
		var savedNames []string
		for name := range folderConfig.SavedName {
			savedNames = append(savedNames, name)
		}

		// Prompt user to select a saved name
		fmt.Println("Select a saved configuration:")
		for i, name := range savedNames {
			fmt.Printf("%d. %s\n", i+1, name)
		}
		fmt.Print("Enter the number of the configuration to load: ")

		var choice int
		if _, err := fmt.Scanln(&choice); err != nil || choice < 1 || choice > len(savedNames) {
			log.Fatalf("Invalid choice")
		}

		// Load the selected saved configuration
		selectedName := savedNames[choice-1]
		savedArgs, err := app.getSavedConfig(currentDir, selectedName)
		if err != nil {
			log.Fatalf("Failed to load saved configuration: %v", err)
		}

		// Reparse arguments from saved configuration
		os.Args = append([]string{os.Args[0]}, savedArgs...)
		args = os.Args[1:]
	}

	// Parse arguments
	files, ignorePattern, ignoreGitIgnore, delimiter, wrapCode, saveName, _, execCommand, fileExecs, err = parseArguments(args)
	if err != nil {
		log.Fatalf("Failed to parse arguments: %v", err)
	}

	// Save configuration if -name is provided
	if saveName != "" {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current directory: %v", err)
		}
		if err := app.saveCurrentConfig(currentDir, saveName, args); err != nil {
			log.Fatalf("Failed to save configuration: %v", err)
		}
		fmt.Printf("Arguments saved for name '%s' in folder '%s'\n", saveName, currentDir)
		return
	}

	// Ensure files are provided
	if len(files) == 0 {
		log.Fatalf("No files specified. Please provide at least one file.")
	}

	// Generate output
	output, err := getData(files, ignorePattern, ignoreGitIgnore, delimiter, wrapCode, execCommand, fileExecs, app.Config.FileTypeExecutables)
	if err != nil {
		log.Fatalf("Failed to process files: %v", err)
	}

	// Copy output to clipboard
	if err := clipboard.WriteAll(output); err != nil {
		log.Fatalf("Failed to copy output to clipboard: %v", err)
	}
	fmt.Println("Output has been copied to the clipboard.")
}
