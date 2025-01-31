package main

import (
	"bufio"
	"encoding/json"
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

type Config struct {
	Folders             map[string]FolderConfig `json:"folders"`
	FileTypeExecutables map[string]string       `json:"file_type_executables"` // Map of file extensions to executables
}

type FolderConfig struct {
	SavedName map[string][]string `json:"saved_name"`
}

var config Config
var configPath string

func init() {
	// Determine the config file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}
	configPath = filepath.Join(homeDir, ".config", "your_app_name", "config.json")

	// Load the config file
	loadConfig()
}

func loadConfig() {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			config = Config{
				Folders:             make(map[string]FolderConfig),
				FileTypeExecutables: make(map[string]string),
			}
			return
		}
		log.Fatalf("Failed to read config file: %v", err)
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}
}

func saveConfig() {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		log.Fatalf("Failed to write config file: %v", err)
	}
}

// Map of file extensions to programming languages
var languageMap = map[string]string{
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

func main() {
	// Parse command-line arguments
	files := []string{}
	var ignorePattern string
	ignoreGitIgnore := false
	delimiter := "======"
	wrapCode := true // Default to true
	var saveName, byName, execCommand string
	fileExecs := make(map[string]string) // Map for command-line file type executables
	var executableOutput string

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-ignore-pattern":
			if i+1 >= len(args) {
				log.Fatalf("Missing value for -ignore-pattern")
			}
			ignorePattern = args[i+1]
			i++
		case "-ignore-gitignore":
			ignoreGitIgnore = true
		case "-delimiter":
			if i+1 >= len(args) {
				log.Fatalf("Missing value for -delimiter")
			}
			delimiter = args[i+1]
			i++
		case "-wrap-code":
			if i+1 >= len(args) {
				log.Fatalf("Missing value for -wrap-code")
			}
			wrapCodeStr := args[i+1]
			if wrapCodeStr == "false" {
				wrapCode = false
			}
			i++
		case "-name":
			if i+1 >= len(args) {
				log.Fatalf("Missing value for -name")
			}
			saveName = args[i+1]
			i++
		case "-by-name":
			if i+1 >= len(args) {
				log.Fatalf("Missing value for -by-name")
			}
			byName = args[i+1]
			i++
		case "-files":
			if i+1 >= len(args) {
				log.Fatalf("Missing value for -files")
			}
			// Collect all subsequent arguments as file paths until the next flag
			for i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				files = append(files, args[i+1])
				i++
			}
		case "-exec":
			if i+1 >= len(args) {
				log.Fatalf("Missing value for -exec")
			}
			execCommand = args[i+1]
			i++
		case "-file-exec":
			if i+1 >= len(args) {
				log.Fatalf("Missing value for -file-exec")
			}
			pairs := strings.Fields(args[i+1]) // Split by spaces to handle multiple pairs
			for _, pair := range pairs {
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) != 2 {
					log.Fatalf("Invalid format for -file-exec. Expected '.ext=executable'")
				}
				fileExecs[parts[0]] = parts[1]
			}
			i++
		default:
			log.Fatalf("Unknown argument: %s", args[i])
		}
	}

	// Handle -name option (save arguments)
	if saveName != "" {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current directory: %v", err)
		}

		// Ensure the Folders map is initialized
		if config.Folders == nil {
			config.Folders = make(map[string]FolderConfig)
		}

		// Retrieve or initialize the FolderConfig for the current directory
		folderConfig := config.Folders[currentDir]
		if folderConfig.SavedName == nil {
			folderConfig.SavedName = make(map[string][]string)
		}

		// Update the SavedName map
		folderConfig.SavedName[saveName] = os.Args[1:]

		// Reassign the updated FolderConfig back to the map
		config.Folders[currentDir] = folderConfig

		// Save the updated configuration
		saveConfig()
		fmt.Printf("Arguments saved for name '%s' in folder '%s'\n", saveName, currentDir)
		return
	}

	// Handle -by-name option or auto-select arguments
	if byName != "" {
		// Use arguments from the specified name
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current directory: %v", err)
		}
		folderConfig, exists := config.Folders[currentDir]
		if !exists || folderConfig.SavedName == nil || len(folderConfig.SavedName[byName]) == 0 {
			log.Fatalf("No saved arguments found for name '%s' in folder '%s'", byName, currentDir)
		}
		os.Args = append([]string{os.Args[0]}, folderConfig.SavedName[byName]...)
	} else if len(files) == 0 {
		// Auto-select arguments if no files are provided
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current directory: %v", err)
		}
		folderConfig, exists := config.Folders[currentDir]
		if !exists || folderConfig.SavedName == nil || len(folderConfig.SavedName) == 0 {
			log.Fatalf("Usage: %s -files <file1> <file2> ... [-ignore-pattern <regex>] [-ignore-gitignore] [-delimiter <string>] [-wrap-code <true|false>] [-name <name>] [-by-name <name>] [-exec <command>] [-file-exec .ext=executable]\n\nNo saved arguments found in folder '%s'. Please save arguments first using the -name option.", os.Args[0], currentDir)
		}

		savedNames := []string{}
		for name := range folderConfig.SavedName {
			savedNames = append(savedNames, name)
		}

		if len(savedNames) == 1 {
			// Use the only saved name
			os.Args = append([]string{os.Args[0]}, folderConfig.SavedName[savedNames[0]]...)
		} else {
			// Interactive selection
			fmt.Println("Multiple saved names found. Please select one:")
			for i, name := range savedNames {
				fmt.Printf("%d: %s\n", i+1, name)
			}
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter the number of the name to use: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			index := 0
			fmt.Sscanf(input, "%d", &index)
			if index < 1 || index > len(savedNames) {
				log.Fatalf("Invalid selection")
			}
			selectedName := savedNames[index-1]
			os.Args = append([]string{os.Args[0]}, folderConfig.SavedName[selectedName]...)
		}
	}

	// Compile regex for ignore pattern
	var ignoreRegex *regexp.Regexp
	if ignorePattern != "" {
		var err error
		ignoreRegex, err = regexp.Compile(ignorePattern)
		if err != nil {
			log.Fatalf("Invalid regex pattern: %v", err)
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
	for ext, cmd := range config.FileTypeExecutables {
		finalFileTypeExecutables[ext] = cmd
	}
	for ext, cmd := range fileExecs {
		finalFileTypeExecutables[ext] = cmd
	}

	// Buffer to store all output
	var output strings.Builder

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
		if executable != "" {
			// Split the executable and its arguments
			parts := strings.Fields(executable)
			if len(parts) == 0 {
				log.Fatalf("Invalid executable command: %s", executable)
			}
			cmd := exec.Command(parts[0], append(parts[1:], filePath)...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Fatalf("Failed to run executable '%s' with file '%s': %v\nOutput: %s", executable, filePath, err, string(out))
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

	// Copy output to clipboard
	err := clipboard.WriteAll(output.String())
	if err != nil {
		log.Fatalf("Failed to copy output to clipboard: %v", err)
	}
	fmt.Println("Output has been copied to the clipboard.")
}
