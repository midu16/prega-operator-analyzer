package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"prega-operator-analyzer/pkg"

	"github.com/sirupsen/logrus"
)

func main() {
	// Command line flags
	var (
		pregaIndex   = flag.String("prega-index", "quay.io/prega/prega-operator-index:v4.20.0-ec.6", "Prega operator index image to analyze")
		outputFile   = flag.String("output", "", "Output file for release notes (default: auto-generated timestamp)")
		workDir      = flag.String("work-dir", "temp-repos", "Temporary directory for cloning repositories")
		verbose      = flag.Bool("verbose", false, "Enable verbose logging")
		cursorAgent  = flag.Bool("cursor-agent", false, "Use cursor-agent vibe-tools for enhanced release notes")
		help         = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Set up logging
	logger := logrus.New()
	if *verbose {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Configuration
	indexJSONPath := "prega-operator-index/index.json"
	if *outputFile == "" {
		*outputFile = fmt.Sprintf("release-notes-%s.txt", time.Now().Format("2006-01-02-15-04-05"))
	}

	// Check if index.json exists, if not, generate it
	if _, err := os.Stat(indexJSONPath); os.IsNotExist(err) {
		logger.Infof("Index JSON file not found: %s", indexJSONPath)
		logger.Info("Generating index JSON from Prega operator index...")
		
		if err := generateIndexJSON(*pregaIndex, indexJSONPath, logger); err != nil {
			logger.Fatalf("Failed to generate index JSON: %v", err)
		}
		logger.Info("Index JSON generated successfully")
	}

	logger.Info("Starting Prega Operator Analyzer")
	logger.Infof("Reading index from: %s", indexJSONPath)

	// Parse the operator index JSON
	repositories, err := pkg.ParseOperatorIndex(indexJSONPath)
	if err != nil {
		logger.Fatalf("Failed to parse operator index: %v", err)
	}

	logger.Infof("Found %d repository entries", len(repositories))

	// Remove duplicates
	uniqueRepositories := pkg.RemoveDuplicates(repositories)
	logger.Infof("Found %d unique repositories after deduplication", len(uniqueRepositories))

	// Display unique repositories
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("UNIQUE REPOSITORIES FOUND:")
	fmt.Println(strings.Repeat("=", 80))
	for i, repo := range uniqueRepositories {
		fmt.Printf("%3d. %s\n", i+1, repo)
	}
	fmt.Println(strings.Repeat("=", 80))

	// Create work directory
	if err := os.MkdirAll(*workDir, 0755); err != nil {
		logger.Fatalf("Failed to create work directory: %v", err)
	}

	// Initialize VibeToolsManager with cursor-agent flag
	vibeManager := pkg.NewVibeToolsManager(*workDir, *outputFile, *cursorAgent)

	// Process repositories and generate release notes
	logger.Info("Starting release notes generation...")
	if err := vibeManager.ProcessRepositories(uniqueRepositories); err != nil {
		logger.Fatalf("Failed to process repositories: %v", err)
	}

	// Clean up work directory
	if err := os.RemoveAll(*workDir); err != nil {
		logger.Warnf("Failed to clean up work directory: %v", err)
	}

	logger.Infof("Release notes generated successfully: %s", *outputFile)
	fmt.Printf("\nRelease notes saved to: %s\n", *outputFile)
}

// showHelp displays the help message
func showHelp() {
	fmt.Println("Prega Operator Analyzer")
	fmt.Println("======================")
	fmt.Println()
	fmt.Println("Analyzes Prega operator index files, extracts repository URLs, and generates release notes.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  prega-operator-analyzer [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Use default Prega index")
	fmt.Println("  prega-operator-analyzer")
	fmt.Println()
	fmt.Println("  # Use custom Prega index")
	fmt.Println("  prega-operator-analyzer --prega-index=quay.io/prega/prega-operator-index:v4.19.0")
	fmt.Println()
	fmt.Println("  # Specify output file")
	fmt.Println("  prega-operator-analyzer --output=my-release-notes.txt")
	fmt.Println()
	fmt.Println("  # Enable verbose logging")
	fmt.Println("  prega-operator-analyzer --verbose")
	fmt.Println()
	fmt.Println("  # Use cursor-agent vibe-tools")
	fmt.Println("  prega-operator-analyzer --cursor-agent")
}

// generateIndexJSON generates the index JSON file using opm render
func generateIndexJSON(pregaIndex, outputPath string, logger *logrus.Logger) error {
	// Create directory if it doesn't exist
	dir := "prega-operator-index"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Run opm render command
	cmd := fmt.Sprintf("opm render %s --output=json >> %s", pregaIndex, outputPath)
	logger.Debugf("Executing command: %s", cmd)
	
	// Note: In a production environment, you might want to use exec.Command
	// For now, we'll provide instructions to the user
	logger.Info("Please run the following command to generate the index JSON:")
	logger.Infof("  %s", cmd)
	
	return fmt.Errorf("index JSON file not found. Please generate it first using the command above")
}