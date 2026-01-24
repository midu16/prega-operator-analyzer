package pkg

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// DependencyManager handles downloading and managing external dependencies
type DependencyManager struct {
	BinDir string
	Logger *logrus.Logger
}

// NewDependencyManager creates a new dependency manager
func NewDependencyManager(binDir string, logger *logrus.Logger) *DependencyManager {
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}
	return &DependencyManager{
		BinDir: binDir,
		Logger:  logger,
	}
}

// FindOrDownloadTool finds a tool in PATH or downloads it to .bin/
// Returns the path to the executable
func (dm *DependencyManager) FindOrDownloadTool(toolName string) (string, error) {
	// First, check if tool is in PATH
	if path, err := exec.LookPath(toolName); err == nil {
		dm.Logger.Debugf("Found %s in PATH: %s", toolName, path)
		return toolName, nil // Return just the name so exec.Command uses PATH
	}

	dm.Logger.Infof("%s not found in PATH, checking .bin/ directory", toolName)

	// Check if tool exists in .bin/
	binPath := filepath.Join(dm.BinDir, toolName)
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	if _, err := os.Stat(binPath); err == nil {
		dm.Logger.Infof("Found %s in .bin/: %s", toolName, binPath)
		return binPath, nil
	}

	// Tool not found, try to download it
	dm.Logger.Infof("%s not found, attempting to download...", toolName)
	return dm.downloadTool(toolName, binPath)
}

// downloadTool downloads a tool to the bin directory
func (dm *DependencyManager) downloadTool(toolName, binPath string) (string, error) {
	// Create .bin directory if it doesn't exist
	if err := os.MkdirAll(dm.BinDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Download based on tool name
	switch toolName {
	case "opm":
		return dm.downloadOPM(binPath)
	case "vibe-tools":
		return dm.downloadVibeTools(binPath)
	case "cursor-agent":
		return "", fmt.Errorf("cursor-agent cannot be auto-downloaded, please install it manually")
	default:
		return "", fmt.Errorf("auto-download not supported for %s", toolName)
	}
}

// downloadOPM downloads the OPM tool
func (dm *DependencyManager) downloadOPM(binPath string) (string, error) {
	goos := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go arch to OPM arch names
	archMap := map[string]string{
		"amd64": "x86_64",
		"arm64": "aarch64",
	}
	opmArch, ok := archMap[arch]
	if !ok {
		return "", fmt.Errorf("unsupported architecture: %s", arch)
	}

	// OPM version - using a recent stable version
	version := "4.17.21"
	
	// Determine OS-specific file name
	var osName, fileExt string
	switch goos {
	case "linux":
		osName = "linux"
		fileExt = "tar.gz"
	case "darwin":
		osName = "mac"
		fileExt = "tar.gz"
	case "windows":
		osName = "windows"
		fileExt = "zip"
	default:
		return "", fmt.Errorf("unsupported OS: %s", goos)
	}

	// Construct download URL
	// OPM is available from OpenShift mirror
	url := fmt.Sprintf("https://mirror.openshift.com/pub/openshift-v4/%s/clients/ocp/%s/opm-%s-%s.%s",
		opmArch, version, osName, version, fileExt)

	dm.Logger.Infof("Downloading OPM from: %s", url)

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download OPM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download OPM: HTTP %d", resp.StatusCode)
	}

	// Create temporary file
	tmpFile := binPath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	// Copy download to temp file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(tmpFile)
		return "", fmt.Errorf("failed to save download: %w", err)
	}
	out.Close()

	// Extract based on file type
	if fileExt == "tar.gz" {
		if err := dm.extractTarGz(tmpFile, dm.BinDir); err != nil {
			os.Remove(tmpFile)
			return "", fmt.Errorf("failed to extract OPM: %w", err)
		}
	} else {
		return "", fmt.Errorf("zip extraction not yet implemented for Windows")
	}

	// Remove temp file
	os.Remove(tmpFile)

	// Find the extracted opm binary
	var opmBinaryName string
	switch goos {
	case "linux":
		opmBinaryName = "opm-rhel8"
	case "darwin":
		opmBinaryName = "opm-darwin"
	}

	extractedPath := filepath.Join(dm.BinDir, opmBinaryName)
	if _, err := os.Stat(extractedPath); err != nil {
		// Try alternative names
		altNames := []string{"opm", "opm-linux", "opm-mac"}
		found := false
		for _, altName := range altNames {
			altPath := filepath.Join(dm.BinDir, altName)
			if _, err := os.Stat(altPath); err == nil {
				extractedPath = altPath
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("could not find extracted OPM binary")
		}
	}

	// Rename to standard name
	if err := os.Rename(extractedPath, binPath); err != nil {
		// If rename fails, try copying
		if err := dm.copyFile(extractedPath, binPath); err != nil {
			return "", fmt.Errorf("failed to move OPM binary: %w", err)
		}
		os.Remove(extractedPath)
	}

	// Make executable
	if err := os.Chmod(binPath, 0755); err != nil {
		return "", fmt.Errorf("failed to make OPM executable: %w", err)
	}

	dm.Logger.Infof("Successfully downloaded OPM to: %s", binPath)
	return binPath, nil
}

// downloadVibeTools downloads vibe-tools (placeholder - implementation depends on availability)
func (dm *DependencyManager) downloadVibeTools(binPath string) (string, error) {
	return "", fmt.Errorf("vibe-tools auto-download not yet implemented")
}

// extractTarGz extracts a tar.gz file to the destination directory
func (dm *DependencyManager) extractTarGz(src, dst string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Skip if not a regular file
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Extract to destination
		target := filepath.Join(dst, filepath.Base(header.Name))

		// Create parent directories
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		// Create the file
		outFile, err := os.Create(target)
		if err != nil {
			return err
		}

		// Copy file contents
		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()

		// Make executable if it's a binary
		if strings.Contains(header.Name, "opm") {
			os.Chmod(target, 0755)
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func (dm *DependencyManager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// GetToolPath is a convenience function that finds or downloads a tool
func GetToolPath(toolName string, logger *logrus.Logger) (string, error) {
	binDir := ".bin"
	dm := NewDependencyManager(binDir, logger)
	return dm.FindOrDownloadTool(toolName)
}
