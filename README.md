
# Script Documentation

This script processes files, applies custom executables to them based on file types, and outputs the results to the clipboard. It supports configuration via both command-line arguments and a configuration file.

## Table of Contents

1. [Overview](#overview)
2. [Configuration File](#configuration-file)
3. [Command-Line Arguments](#command-line-arguments)
4. [Examples](#examples)
5. [Saved Settings Location](#saved-settings-location)

---

## Overview

The script is designed to:
- Process files specified by the user.
- Apply custom executables to files based on their extensions.
- Ignore files using regex patterns or `.gitignore` rules.
- Save and reuse configurations for different folders.
- Copy the processed output to the clipboard.

---

## Configuration File

The script uses a JSON configuration file to store persistent settings. The file is located at:

```
~/.config/your_app_name/config.json
```

### Structure of `config.json`

```json
{
  "folders": {
    "/path/to/folder": {
      "saved_name": {
        "my-config": [
          "-files",
          "file1.ts",
          "file2.go",
          "-file-exec",
          ".ts=check-ts-errors .go=gofmt"
        ]
      }
    }
  },
  "file_type_executables": {
    ".ts": "check-ts-errors --verbose",
    ".go": "gofmt -l"
  }
}
```

- **`folders`**: A map of folder paths to saved configurations.
  - Each folder can have multiple named configurations (`saved_name`).
  - Each named configuration stores a list of arguments that were passed to the script.
- **`file_type_executables`**: A map of file extensions to default executables.

---

## Command-Line Arguments

The script supports the following command-line arguments:

| Argument                  | Description                                                                                     | Example                                                                 |
|---------------------------|-------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------|
| `-files`                  | Specifies the files to process.                                                                | `-files file1.ts file2.go`                                              |
| `-ignore-pattern`         | Ignores files matching the provided regex pattern.                                             | `-ignore-pattern "*.tmp"`                                               |
| `-ignore-gitignore`       | Ignores `.gitignore` rules when processing files.                                              | `-ignore-gitignore`                                                     |
| `-delimiter`              | Sets the delimiter used between file outputs.                                                  | `-delimiter "======"`                                                   |
| `-wrap-code`              | Wraps file content in code blocks with syntax highlighting (default: `true`).                  | `-wrap-code false`                                                      |
| `-name`                   | Saves the current arguments under a name for future use.                                       | `-name my-config`                                                       |
| `-by-name`                | Reuses previously saved arguments by name.                                                    | `-by-name my-config`                                                    |
| `-exec`                   | Specifies a global executable to run on all files.                                             | `-exec check-ts-errors --verbose`                                       |
| `-file-exec`              | Specifies executables for specific file types. Multiple mappings can be provided in one flag. | `-file-exec .ts=check-ts-errors .go=gofmt`                              |

---

## Examples

### Example 1: Basic Usage

Process two files and copy their contents to the clipboard.

```bash
./script -files file1.ts file2.go
```

**Output in Clipboard**:
```
file1.ts
```typescript
<contents of file1.ts>
```
======
file2.go
```go
<contents of file2.go>
```
======


### Example 2: Ignore Files by Pattern

Ignore temporary files (`.tmp`) during processing.

```bash
./script -files file1.ts file2.tmp -ignore-pattern "*.tmp"
```

**Output in Clipboard**:

file1.ts
```typescript
<contents of file1.ts>
```
======


---

### Example 3: Use Executables for Specific File Types

Run `check-ts-errors` for `.ts` files and `gofmt` for `.go` files.

```bash
./script -files file1.ts file2.go -file-exec .ts=check-ts-errors .go=gofmt
```

**Output in Clipboard**:

file1.ts
```typescript
<contents of file1.ts>
```
Executable Output for file1.ts
======
file2.go
```go
<contents of file2.go>
```
Executable Output for file2.go
======
---

### Example 4: Save and Reuse Configurations

Save the current arguments under the name `my-config`.

```bash
./script -files file1.ts file2.go -file-exec .ts=check-ts-errors .go=gofmt -name my-config
```

Reuse the saved configuration later:

```bash
./script -by-name my-config
```

---

### Example 5: Disable Code Wrapping

Disable wrapping file content in code blocks.

```bash
./script -files file1.ts -wrap-code false
```

**Output in Clipboard**:
```
file1.ts
<contents of file1.ts>
======
```

---

## Saved Settings Location

Saved settings are stored in the configuration file located at:

```
~/.config/your_app_name/config.json
```

Each folder has its own section in the `folders` map. For example:

```json
{
  "folders": {
    "/path/to/folder": {
      "saved_name": {
        "my-config": [
          "-files",
          "file1.ts",
          "file2.go",
          "-file-exec",
          ".ts=check-ts-errors .go=gofmt"
        ]
      }
    }
  }
}
```

- **Folder Path**: The key in the `folders` map represents the absolute path of the folder.
- **Named Configurations**: Each folder can have multiple named configurations (`saved_name`), which store lists of arguments.

To view or edit saved settings, open the `config.json` file in a text editor.

---

## Notes

1. **Priority of Executables**:
   - Command-line overrides (`-file-exec`) take precedence over the `file_type_executables` map in the configuration file.
   - The `-exec` flag applies globally to all files.

2. **Error Handling**:
   - If an executable fails, the script logs detailed error messages, including the file path and output from the executable.
