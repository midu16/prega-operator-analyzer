package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"prega-operator-analyzer/pkg"

	"github.com/sirupsen/logrus"
)

func main() {
	// Command line flags
	var (
		pregaIndex   = flag.String("prega-index", "quay.io/prega/prega-operator-index:v4.21", "Prega operator index image to analyze")
		outputFile   = flag.String("output", "", "Output file for release notes (default: auto-generated timestamp)")
		workDir      = flag.String("work-dir", "", "Temporary directory for cloning repositories")
		verbose      = flag.Bool("verbose", false, "Enable verbose logging")
		cursorAgent  = flag.Bool("cursor-agent", false, "Use cursor-agent vibe-tools for enhanced release notes")
		help         = flag.Bool("help", false, "Show help message")
		indexFile    = flag.String("index-file", "", "Path to index.json file")
		serverMode   = flag.Bool("server", false, "Run in web server mode")
		serverPort   = flag.Int("port", 8080, "Port for web server (default: 8080)")
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

	// Check for environment variable overrides
	if os.Getenv("SERVER_MODE") == "true" {
		*serverMode = true
	}
	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			*serverPort = port
		}
	}

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

	// Handle server mode
	if *serverMode {
		runServerMode(*serverPort, *workDir, outputDir, *pregaIndex, logger)
		return
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

// runServerMode starts the web server for interactive analysis
func runServerMode(port int, workDir, outputDir, pregaIndex string, logger *logrus.Logger) {
	logger.Info("Starting Prega Operator Analyzer in Web Server Mode")
	logger.Infof("Port: %d", port)
	logger.Infof("Work Directory: %s", workDir)
	logger.Infof("Output Directory: %s", outputDir)
	logger.Infof("Prega Index: %s", pregaIndex)

	// Create the server
	server := pkg.NewServer(port, workDir, outputDir, pregaIndex, logger)

	// Try to load repositories from existing index or generate new one
	indexJSONPath := filepath.Join(workDir, "prega-operator-index", "index.json")
	
	if _, err := os.Stat(indexJSONPath); os.IsNotExist(err) {
		logger.Info("Index JSON file not found, will generate on first refresh")
		logger.Info("Click 'Refresh Repositories' in the web UI to load operators")
	} else {
		logger.Infof("Loading repositories from: %s", indexJSONPath)
		repositories, err := pkg.ParseOperatorIndex(indexJSONPath)
		if err != nil {
			logger.Warnf("Failed to parse existing index: %v", err)
		} else {
			uniqueRepos := pkg.RemoveDuplicates(repositories)
			server.SetRepositories(uniqueRepos)
			logger.Infof("Loaded %d unique repositories", len(uniqueRepos))
		}
	}

	// Start the server
	logger.Infof("Web interface available at: http://localhost:%d", port)
	if err := server.Start(); err != nil {
		logger.Fatalf("Server failed: %v", err)
	}
}

// showHelp displays the help message
func showHelp() {
	fmt.Println("Prega Operator Analyzer")
	fmt.Println("======================")
	fmt.Println()
	fmt.Println("Analyzes Prega operator index files, extracts repository URLs, and generates release notes.")
	fmt.Println("Supports both CLI mode for batch processing and Web Server mode for interactive analysis.")
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
	fmt.Println("  SERVER_MODE   - Set to 'true' to run in web server mode")
	fmt.Println("  SERVER_PORT   - Port for web server (default: 8080)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # CLI Mode: Use default Prega index")
	fmt.Println("  prega-operator-analyzer")
	fmt.Println()
	fmt.Println("  # CLI Mode: Use custom Prega index")
	fmt.Println("  prega-operator-analyzer --prega-index=quay.io/prega/prega-operator-index:v4.19.0")
	fmt.Println()
	fmt.Println("  # CLI Mode: Specify output file")
	fmt.Println("  prega-operator-analyzer --output=my-release-notes.txt")
	fmt.Println()
	fmt.Println("  # CLI Mode: Enable verbose logging")
	fmt.Println("  prega-operator-analyzer --verbose")
	fmt.Println()
	fmt.Println("  # CLI Mode: Use cursor-agent vibe-tools")
	fmt.Println("  prega-operator-analyzer --cursor-agent")
	fmt.Println()
	fmt.Println("  # Web Server Mode: Start interactive web interface")
	fmt.Println("  prega-operator-analyzer --server")
	fmt.Println()
	fmt.Println("  # Web Server Mode: Custom port")
	fmt.Println("  prega-operator-analyzer --server --port=3000")
	fmt.Println()
	fmt.Println("Docker Usage:")
	fmt.Println("  # CLI Mode: Run with volume mounts")
	fmt.Println("  podman run -v $(pwd)/output:/app/output:Z,rw \\")
	fmt.Println("    -e PREGA_INDEX=quay.io/prega/prega-operator-index:v4.21 \\")
	fmt.Println("    quay.io/midu/prega-operator-analyzer:latest")
	fmt.Println()
	fmt.Println("  # Web Server Mode: Run interactive interface")
	fmt.Println("  podman run -p 8080:8080 \\")
	fmt.Println("    -e SERVER_MODE=true \\")
	fmt.Println("    -e PREGA_INDEX=quay.io/prega/prega-operator-index:v4.21 \\")
	fmt.Println("    quay.io/midu/prega-operator-analyzer:latest")
	fmt.Println()
	fmt.Println("Web Interface Features:")
	fmt.Println("  - Drag and drop operators to analyze")
	fmt.Println("  - Dynamic time period selection (1-90 days)")
	fmt.Println("  - Branch selection including release-* branches")
	fmt.Println("  - Real-time release notes generation")
	fmt.Println("  - Rich HTML and plain text views")
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
