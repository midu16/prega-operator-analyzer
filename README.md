# Prega Operator Analyzer

A Golang tool that analyzes Prega operator index JSON files, extracts unique repository URLs, and generates release notes using vibe-tools.

## Features

- **Command-line interface** with configurable flags for Prega index, output file, and verbosity
- **Automatic index generation** - Can generate operator index JSON from Prega index images
- **Weekly commit analysis** - Focuses on commits from the last 7 days on main branch
- **Enhanced release notes** - Provides detailed weekly activity summaries including:
  - Total commits and lines changed in the last week
  - Top contributors with commit counts
  - Detailed commit information with authors and dates
- **Smart fallback** - Uses cursor-agent vibe-tools, regular vibe-tools, or enhanced git analysis based on availability
- **Duplicate removal** - Automatically removes duplicate repository URLs
- **Comprehensive output** - Saves all release notes to a timestamped text file

## Prerequisites

1. **Go 1.21 or later**
2. **Git** (for cloning repositories)
3. **vibe-tools** (optional, for enhanced release notes generation)
4. **cursor-agent** (optional, for AI-enhanced release notes when using `--cursor-agent` flag)

## Installation

1. Clone or download this project
2. Navigate to the project directory
3. Install dependencies:
   ```bash
   go mod tidy
   ```

## Usage

### Basic Usage

```bash
# Run with default settings
go run cmd/main.go

# Or build and run the binary
go build -o bin/prega-operator-analyzer cmd/main.go
./bin/prega-operator-analyzer
```

### Command Line Options

```bash
# Show help
./bin/prega-operator-analyzer --help

# Use a different Prega index
./bin/prega-operator-analyzer --prega-index=quay.io/prega/prega-operator-index:v4.19.0

# Specify custom output file
./bin/prega-operator-analyzer --output=my-release-notes.txt

# Enable verbose logging
./bin/prega-operator-analyzer --verbose

# Use cursor-agent vibe-tools
./bin/prega-operator-analyzer --cursor-agent

# Use custom work directory
./bin/prega-operator-analyzer --work-dir=/tmp/my-repos
```

### Available Flags

- `--prega-index`: Prega operator index image to analyze (default: `quay.io/prega/prega-operator-index:v4.20.0-ec.6`)
- `--output`: Output file for release notes (default: auto-generated timestamp)
- `--work-dir`: Temporary directory for cloning repositories (default: `temp-repos`)
- `--verbose`: Enable verbose logging
- `--cursor-agent`: Use cursor-agent vibe-tools for enhanced release notes
- `--help`: Show help message

### How It Works

The tool will:
1. **Auto-generate index JSON** if not present (using the specified Prega index)
2. **Parse the operator index** to extract repository URLs
3. **Remove duplicates** and display unique repositories
4. **Clone each repository** and analyze the main branch
5. **Generate weekly release notes** focusing on commits from the last 7 days
6. **Save comprehensive output** to a timestamped file

### Manual Index Generation (Optional)

If you prefer to generate the index JSON manually:

```bash
mkdir -p prega-operator-index
opm render quay.io/prega/prega-operator-index:v4.20.0-ec.6 --output=json >> prega-operator-index/index.json
```

## Project Structure

```
prega-operator-analyzer/
├── cmd/
│   └── main.go              # Main application entry point
├── pkg/
│   ├── parser.go            # JSON parsing and repository extraction
│   └── vibe_tools.go        # Vibe-tools integration and release notes generation
├── go.mod                   # Go module dependencies
├── go.sum                   # Go module checksums
└── README.md               # This file
```

## Configuration

You can modify the following variables in `cmd/main.go`:

- `indexJSONPath`: Path to the operator index JSON file
- `outputFile`: Output file name for release notes
- `workDir`: Temporary directory for cloning repositories

## Dependencies

- `github.com/go-git/go-git/v5`: Git operations
- `github.com/sirupsen/logrus`: Logging

## Output

The tool generates a comprehensive text file containing:

1. **Header** with generation timestamp
2. **For each repository**:
   - Repository URL and analysis period
   - Latest commit information
   - **Weekly Activity Summary**:
     - Total commits in the last week
     - Total lines changed
     - Number of active contributors
   - **Top Contributors** (last week) with commit counts
   - **Detailed commit list** from the last 7 days with:
     - Commit messages
     - Author names
     - Commit hashes
     - Timestamps

### Sample Output Structure

```
Release Notes Generated on: 2024-01-15 14:30:25
================================================================================

Repository: https://github.com/example/operator
------------------------------------------------
Repository: https://github.com/example/operator
Analysis Period: Last 7 days (since 2024-01-08 14:30:25)
Latest Commit: a1b2c3d4
Latest Commit Message: Fix security vulnerability in authentication
Latest Commit Author: John Doe
Latest Commit Date: 2024-01-15 10:30:00

=== WEEKLY ACTIVITY SUMMARY ===
Total Commits in Last Week: 15
Total Lines Changed: 1,234
Active Contributors: 3

Top Contributors (Last Week):
  1. John Doe (8 commits)
  2. Jane Smith (5 commits)
  3. Bob Wilson (2 commits)

=== COMMITS FROM LAST WEEK ===
- Fix security vulnerability in authentication (a1b2c3d4) by John Doe on 2024-01-15 10:30:00
- Add unit tests for new feature (b2c3d4e5) by Jane Smith on 2024-01-14 16:45:00
- Update documentation (c3d4e5f6) by Bob Wilson on 2024-01-13 09:15:00
...
```

## Error Handling

- If vibe-tools is not available, the tool falls back to generating basic release notes
- Failed repository processing is logged and included in the output
- Temporary directories are cleaned up after processing

## Example Output

```
Release Notes Generated on: 2024-01-15 14:30:25
================================================================================

Repository: https://github.com/ComplianceAsCode/compliance-operator
--------------------------------------------------------------------------------
[Release notes content from vibe-tools or basic repository info]

Repository: https://github.com/quay/container-security-operator
--------------------------------------------------------------------------------
[Release notes content from vibe-tools or basic repository info]

...
```

## Troubleshooting

1. **"Index JSON file not found"**: Make sure you've run the `opm render` command first
2. **"vibe-tools not found"**: Install vibe-tools or the tool will use basic release notes
3. **Git clone failures**: Check network connectivity and repository accessibility
4. **Permission errors**: Ensure write permissions for the output directory