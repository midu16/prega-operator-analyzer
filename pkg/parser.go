package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// OperatorIndex represents the structure of the operator index JSON
type OperatorIndex struct {
	Schema         string      `json:"schema"`
	Image          string      `json:"image"`
	RelatedImages  interface{} `json:"relatedImages"`
	Properties     interface{} `json:"properties"`
	Packages       []Package   `json:"packages"`
}

// Package represents a package in the operator index
type Package struct {
	Schema         string      `json:"schema"`
	Name           string      `json:"name"`
	DefaultChannel string      `json:"defaultChannel"`
	Description    string      `json:"description"`
	Icon           interface{} `json:"icon"`
	Channels       []Channel   `json:"channels"`
}

// Channel represents a channel in a package
type Channel struct {
	Name       string     `json:"name"`
	CurrentCSV string     `json:"currentCSV"`
	Entries    []Entry    `json:"entries"`
}

// Entry represents an entry in a channel
type Entry struct {
	Name     string                 `json:"name"`
	Replaces string                 `json:"replaces,omitempty"`
	Skips    []string               `json:"skips,omitempty"`
	SkipRange string                `json:"skipRange,omitempty"`
	Properties []Property           `json:"properties,omitempty"`
}

// Property represents a property in an entry
type Property struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// ParserRepositoryInfo represents repository information from parser
type ParserRepositoryInfo struct {
	URL         string `json:"repository"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// ParseOperatorIndex parses the operator index JSON file and extracts repository URLs
func ParseOperatorIndex(filePath string) ([]string, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, WrapError(err, ErrorTypeFileSystem, "index file does not exist", map[string]interface{}{
			"file_path": filePath,
		})
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, WrapError(err, ErrorTypeFileSystem, "failed to open index file", map[string]interface{}{
			"file_path": filePath,
		})
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Log but don't return error for close failures
			fmt.Printf("Warning: failed to close file %s: %v\n", filePath, closeErr)
		}
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, WrapError(err, ErrorTypeFileSystem, "failed to read index file", map[string]interface{}{
			"file_path": filePath,
		})
	}

	// Check if file is empty
	if len(content) == 0 {
		return nil, WrapError(nil, ErrorTypeValidation, "index file is empty", map[string]interface{}{
			"file_path": filePath,
		})
	}

	var index OperatorIndex
	if err := json.Unmarshal(content, &index); err != nil {
		return nil, WrapError(err, ErrorTypeParsing, "failed to parse JSON", map[string]interface{}{
			"file_path": filePath,
			"file_size": len(content),
		})
	}

	repositories := make(map[string]bool)
	
	// Extract repositories from packages
	for _, pkg := range index.Packages {
		for _, channel := range pkg.Channels {
			for _, entry := range channel.Entries {
				for _, prop := range entry.Properties {
					if prop.Type == "olm.package" || prop.Type == "olm.bundle" {
						if propMap, ok := prop.Value.(map[string]interface{}); ok {
							if repo, exists := propMap["repository"]; exists {
								if repoStr, ok := repo.(string); ok {
									// Validate repository URL
									if isValidRepositoryURL(repoStr) {
										repositories[repoStr] = true
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Also try to extract from raw JSON content as fallback
	rawRepositories := extractRepositoriesFromRawJSON(string(content))
	for _, repo := range rawRepositories {
		if isValidRepositoryURL(repo) {
			repositories[repo] = true
		}
	}

	// Convert map keys to slice
	var result []string
	for repo := range repositories {
		result = append(result, repo)
	}

	if len(result) == 0 {
		return nil, WrapError(nil, ErrorTypeValidation, "no valid repositories found in index", map[string]interface{}{
			"file_path": filePath,
		})
	}

	return result, nil
}

// isValidRepositoryURL validates if a string is a valid repository URL
func isValidRepositoryURL(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "git@")
}

// extractRepositoriesFromRawJSON extracts repository URLs from raw JSON content
func extractRepositoriesFromRawJSON(content string) []string {
	var repositories []string
	
	// Split content into lines and look for repository fields
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `"repository":`) {
			// Extract the repository URL from the line
			start := strings.Index(line, `"repository":`)
			if start != -1 {
				start += len(`"repository":`)
				// Find the opening quote
				start = strings.Index(line[start:], `"`)
				if start != -1 {
					start += len(`"repository":`) + start + 1
					// Find the closing quote
					end := strings.Index(line[start:], `"`)
					if end != -1 {
						repo := line[start : start+end]
						if repo != "" && strings.HasPrefix(repo, "http") {
							repositories = append(repositories, repo)
						}
					}
				}
			}
		}
	}
	
	return repositories
}

// RemoveDuplicates removes duplicate strings from a slice
func RemoveDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	
	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	
	return result
}