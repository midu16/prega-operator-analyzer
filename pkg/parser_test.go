package pkg

import (
	"testing"
)

func TestParseOperatorIndex(t *testing.T) {
	tests := []struct {
		name           string
		indexFile      string
		expectedCount  int
		expectedRepos  []string
		expectError    bool
	}{
		{
			name:          "valid index file",
			indexFile:     "../testdata/sample_index.json",
			expectedCount: 2, // Should deduplicate the duplicate repository
			expectedRepos: []string{
				"https://github.com/ComplianceAsCode/compliance-operator",
				"https://github.com/quay/container-security-operator",
			},
			expectError: false,
		},
		{
			name:        "non-existent file",
			indexFile:   "../testdata/non_existent.json",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositories, err := ParseOperatorIndex(tt.indexFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(repositories) != tt.expectedCount {
				t.Errorf("Expected %d repositories, got %d", tt.expectedCount, len(repositories))
			}

			// Check if expected repositories are present
			for _, expectedRepo := range tt.expectedRepos {
				found := false
				for _, repo := range repositories {
					if repo == expectedRepo {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected repository %s not found in results", expectedRepo)
				}
			}
		})
	}
}

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"repo1", "repo2", "repo3"},
			expected: []string{"repo1", "repo2", "repo3"},
		},
		{
			name:     "with duplicates",
			input:    []string{"repo1", "repo2", "repo1", "repo3", "repo2"},
			expected: []string{"repo1", "repo2", "repo3"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single element",
			input:    []string{"repo1"},
			expected: []string{"repo1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveDuplicates(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d elements, got %d", len(tt.expected), len(result))
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected %s at position %d, got %s", expected, i, result[i])
				}
			}
		})
	}
}

func TestIsValidRepositoryURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "valid https URL",
			url:      "https://github.com/user/repo",
			expected: true,
		},
		{
			name:     "valid http URL",
			url:      "http://github.com/user/repo",
			expected: true,
		},
		{
			name:     "valid git SSH URL",
			url:      "git@github.com:user/repo.git",
			expected: true,
		},
		{
			name:     "invalid URL",
			url:      "ftp://github.com/user/repo",
			expected: false,
		},
		{
			name:     "empty URL",
			url:      "",
			expected: false,
		},
		{
			name:     "non-URL string",
			url:      "just a string",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidRepositoryURL(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v for URL %s, got %v", tt.expected, tt.url, result)
			}
		})
	}
}
