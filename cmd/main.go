package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"prega-operator-analyzer/pkg"

	"github.com/sirupsen/logrus"
)

func main() {
	// Command line flags
	var (
		pregaIndex   = flag.String("prega-index", "quay.io/prega/prega-operator-index:v4.21-20251025T205504", "Prega operator index image to analyze")
		outputFile   = flag.String("output", "", "Output file for release notes (default: auto-generated timestamp)")
		workDir      = flag.String("work-dir", "", "Temporary directory for cloning repositories")
		verbose      = flag.Bool("verbose", false, "Enable verbose logging")
		cursorAgent  = flag.Bool("cursor-agent", false, "Use cursor-agent vibe-tools for enhanced release notes")
		help         = flag.Bool("help", false, "Show help message")
		indexFile    = flag.String("index-file", "", "Path to index.json file")
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

	// Configuration with environment variable support
	indexJSONPath := getEnvOrDefault("INDEX_FILE", "prega-operator-index/index.json")
	if *indexFile != "" {
		indexJSONPath = *indexFile
	}

	defaultWorkDir := getEnvOrDefault("WORK_DIR", "temp-repos")
	if *workDir == "" {
		*workDir = defaultWorkDir
	}

	outputDir := getEnvOrDefault("OUTPUT_DIR", ".")
	if *outputFile == "" {
		timestamp := time.Now().Format("2006-01-02-15-04-05")
		*outputFile = filepath.Join(outputDir, fmt.Sprintf("release-notes-%s.txt", timestamp))
	}

	logger.Infof("Configuration:")
	logger.Infof("  Index file: %s", indexJSONPath)
	logger.Infof("  Work directory: %s", *workDir)
	logger.Infof("  Output file: %s", *outputFile)
	logger.Infof("  Prega index: %s", *pregaIndex)

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

	// Ensure output directory exists
	outputDirPath := filepath.Dir(*outputFile)
	if err := os.MkdirAll(outputDirPath, 0755); err != nil {
		logger.Fatalf("Failed to create output directory: %v", err)
	}

	// Initialize VibeToolsManager with cursor-agent flag
	vibeManager := pkg.NewVibeToolsManager(*workDir, *outputFile, *cursorAgent)

	// Process repositories and generate release notes
	logger.Info("Starting release notes generation...")
	if err := vibeManager.ProcessRepositories(uniqueRepositories); err != nil {
		logger.Fatalf("Failed to process repositories: %v", err)
	}

	// Clean up work directory

	// Clean up prega-operator-index directory if it was created
	if _, err := os.Stat("prega-operator-index"); err == nil {
		if err := os.RemoveAll("prega-operator-index"); err != nil {
			logger.Warnf("Failed to clean up prega-operator-index directory: %v", err)
		} else {
			logger.Debug("Successfully cleaned up prega-operator-index directory")
		}
	}
	if err := os.RemoveAll(*workDir); err != nil {
		logger.Warnf("Failed to clean up work directory: %v", err)
	}

	logger.Infof("Release notes generated successfully: %s", *outputFile)
	fmt.Printf("\nRelease notes saved to: %s\n", *outputFile)
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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
	fmt.Println("Environment Variables:")
	fmt.Println("  INDEX_FILE    - Path to index.json file (default: prega-operator-index/index.json)")
	fmt.Println("  WORK_DIR      - Temporary directory for cloning repositories (default: temp-repos)")
	fmt.Println("  OUTPUT_DIR    - Directory for output files (default: current directory)")
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
	fmt.Println()
	fmt.Println("  # Use custom index file")
	fmt.Println("  prega-operator-analyzer --index-file=/path/to/index.json")
	fmt.Println()
	fmt.Println("Docker Usage:")
	fmt.Println("  # Run with volume mounts")
	fmt.Println("  docker run -v /host/data:/app/data -v /host/output:/app/output \\")
	fmt.Println("    quay.io/midu/prega-operator-analyzer:latest")
	fmt.Println()
	fmt.Println("  # Run with custom index file")
	fmt.Println("  docker run -v /host/index.json:/app/data/index.json \\")
	fmt.Println("    quay.io/midu/prega-operator-analyzer:latest")
}

// generateIndexJSON generates the index JSON file using opm render
func generateIndexJSON(pregaIndex, outputPath string, logger *logrus.Logger) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Check if opm is available
	opmPath, err := exec.LookPath("opm")
	if err != nil {
		return fmt.Errorf("opm command not found in PATH. Please ensure opm is installed and available: %w", err)
	}
	logger.Debugf("Found opm at: %s", opmPath)

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	// Execute opm render command
	cmd := exec.Command("opm", "render", pregaIndex, "--output=json")
	cmd.Stdout = outputFile
	cmd.Stderr = os.Stderr

	logger.Debugf("Executing command: opm render %s --output=json > %s", pregaIndex, outputPath)
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute opm render command: %w", err)
	}

	logger.Debugf("Successfully generated index JSON at: %s", outputPath)
	return nil
}
