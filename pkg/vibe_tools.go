package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sirupsen/logrus"
)

// VibeToolsManager handles vibe-tools operations
type VibeToolsManager struct {
	WorkDir      string
	OutputFile   string
	Logger       *logrus.Logger
	ErrorHandler *ErrorHandler
	Formatter    *ReleaseNoteFormatter
	UseCursorAgent bool
}

// NewVibeToolsManager creates a new VibeToolsManager
func NewVibeToolsManager(workDir, outputFile string, useCursorAgent bool) *VibeToolsManager {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	return &VibeToolsManager{
		WorkDir:        workDir,
		OutputFile:     outputFile,
		Logger:         logger,
		ErrorHandler:   NewErrorHandler(3, logger), // 3 retries by default
		Formatter:      NewReleaseNoteFormatter(),
		UseCursorAgent: useCursorAgent,
	}
}

// ProcessRepositories processes all repositories and generates release notes
func (vtm *VibeToolsManager) ProcessRepositories(repositories []string) error {
	// Create output file with error handling
	outputFile, err := os.Create(vtm.OutputFile)
	if err != nil {
		return WrapError(err, ErrorTypeFileSystem, "failed to create output file", map[string]interface{}{
			"output_file": vtm.OutputFile,
		})
	}
	defer func() {
		if closeErr := outputFile.Close(); closeErr != nil {
			vtm.Logger.Errorf("Failed to close output file: %v", closeErr)
		}
	}()

	// Write header
	header := fmt.Sprintf("Release Notes Generated on: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	header += "=" + strings.Repeat("=", len(header)-1) + "\n\n"
	if _, err := outputFile.WriteString(header); err != nil {
		return WrapError(err, ErrorTypeFileSystem, "failed to write header", map[string]interface{}{
			"output_file": vtm.OutputFile,
		})
	}

	successCount := 0
	errorCount := 0

	for i, repo := range repositories {
		vtm.Logger.Infof("Processing repository %d/%d: %s", i+1, len(repositories), repo)
		
		// Use retry mechanism for repository processing
		err := vtm.ErrorHandler.HandleWithRetry(func() error {
			releaseNotes, err := vtm.generateReleaseNotes(repo)
			if err != nil {
				return err
			}

			// Write repository section to output file
			if _, writeErr := outputFile.WriteString(releaseNotes); writeErr != nil {
				return WrapError(writeErr, ErrorTypeFileSystem, "failed to write release notes", map[string]interface{}{
					"repository": repo,
					"output_file": vtm.OutputFile,
				})
			}
			return nil
		}, fmt.Sprintf("process repository %s", repo))

		if err != nil {
			errorCount++
			vtm.Logger.Errorf("Failed to generate release notes for %s: %v", repo, err)
			
			// Write error section using formatter
			errorSection := vtm.Formatter.FormatErrorSection(repo, err)
			if _, writeErr := outputFile.WriteString(errorSection); writeErr != nil {
				vtm.Logger.Errorf("Failed to write error section: %v", writeErr)
			}
		} else {
			successCount++
		}
	}

	// Write summary
	summary := fmt.Sprintf("\n=== PROCESSING SUMMARY ===\n")
	summary += fmt.Sprintf("Total Repositories: %d\n", len(repositories))
	summary += fmt.Sprintf("Successfully Processed: %d\n", successCount)
	summary += fmt.Sprintf("Failed: %d\n", errorCount)
	summary += fmt.Sprintf("Success Rate: %.1f%%\n", float64(successCount)/float64(len(repositories))*100)
	summary += fmt.Sprintf("Generated on: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	
	if _, err := outputFile.WriteString(summary); err != nil {
		vtm.Logger.Errorf("Failed to write summary: %v", err)
	}

	vtm.Logger.Infof("Release notes saved to: %s (Success: %d, Failed: %d)", vtm.OutputFile, successCount, errorCount)
	return nil
}

// generateReleaseNotes generates release notes for a single repository
func (vtm *VibeToolsManager) generateReleaseNotes(repoURL string) (string, error) {
	// Clone repository to temporary directory
	repoName := vtm.extractRepoName(repoURL)
	repoPath := filepath.Join(vtm.WorkDir, repoName)
	
	// Remove existing directory if it exists
	if err := os.RemoveAll(repoPath); err != nil {
		vtm.Logger.Warnf("Failed to remove existing directory %s: %v", repoPath, err)
	}
	
	vtm.Logger.Infof("Cloning repository: %s", repoURL)
	_, err := git.PlainClone(repoPath, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})
	if err != nil {
		return "", WrapError(err, ErrorTypeGit, "failed to clone repository", map[string]interface{}{
			"repository": repoURL,
			"repo_path":  repoPath,
		})
	}

	// Check if we should use cursor-agent or regular vibe-tools
	if vtm.UseCursorAgent {
		if !vtm.isCursorAgentAvailable() {
			vtm.Logger.Info("cursor-agent not found, falling back to basic release notes")
			return vtm.generateBasicReleaseNotes(repoPath, repoURL)
		}
		return vtm.generateCursorAgentReleaseNotes(repoPath, repoURL)
	} else if vtm.isVibeToolsAvailable() {
		return vtm.generateVibeToolsReleaseNotes(repoPath, repoURL)
	} else {
		// No vibe-tools available, use basic release notes
		return vtm.generateBasicReleaseNotes(repoPath, repoURL)
	}
}

// isVibeToolsAvailable checks if vibe-tools is available in PATH
func (vtm *VibeToolsManager) isVibeToolsAvailable() bool {
	_, err := exec.LookPath("vibe-tools")
	return err == nil
}

// isCursorAgentAvailable checks if cursor-agent is available in PATH
func (vtm *VibeToolsManager) isCursorAgentAvailable() bool {
	_, err := exec.LookPath("cursor-agent")
	return err == nil
}

// generateCursorAgentReleaseNotes generates release notes using cursor-agent vibe-tools
func (vtm *VibeToolsManager) generateCursorAgentReleaseNotes(repoPath, repoURL string) (string, error) {
	vtm.Logger.Infof("Running cursor-agent vibe-tools on: %s", repoPath)
	
	// Calculate date range for last week
	now := time.Now()
	oneWeekAgo := now.AddDate(0, 0, -7)
	sinceDate := oneWeekAgo.Format("2006-01-02")
	
	// Try cursor-agent with date range first
	cmd := exec.Command("cursor-agent", "vibe-tools", "release-notes", "--repo", repoPath, "--branch", "main", "--since", sinceDate)
	cmd.Dir = repoPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try without date range if the --since flag is not supported
		vtm.Logger.Infof("cursor-agent with date range failed, trying without date filter: %v", err)
		cmd = exec.Command("cursor-agent", "vibe-tools", "release-notes", "--repo", repoPath, "--branch", "main")
		cmd.Dir = repoPath
		
		output, err = cmd.CombinedOutput()
		if err != nil {
			vtm.Logger.Infof("cursor-agent failed for %s, falling back to basic notes: %v", repoURL, err)
			return vtm.generateBasicReleaseNotes(repoPath, repoURL)
		}
	}

	// Clean up cloned repository
	if err := os.RemoveAll(repoPath); err != nil {
		vtm.Logger.Warnf("Failed to clean up repository directory %s: %v", repoPath, err)
	}
	
	return string(output), nil
}

// generateVibeToolsReleaseNotes generates release notes using regular vibe-tools
func (vtm *VibeToolsManager) generateVibeToolsReleaseNotes(repoPath, repoURL string) (string, error) {
	vtm.Logger.Infof("Running vibe-tools on: %s", repoPath)
	
	// Calculate date range for last week
	now := time.Now()
	oneWeekAgo := now.AddDate(0, 0, -7)
	sinceDate := oneWeekAgo.Format("2006-01-02")
	
	// Try vibe-tools with date range first
	cmd := exec.Command("vibe-tools", "release-notes", "--repo", repoPath, "--branch", "main", "--since", sinceDate)
	cmd.Dir = repoPath
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try without date range if the --since flag is not supported
		vtm.Logger.Infof("vibe-tools with date range failed, trying without date filter: %v", err)
		cmd = exec.Command("vibe-tools", "release-notes", "--repo", repoPath, "--branch", "main")
		cmd.Dir = repoPath
		
		output, err = cmd.CombinedOutput()
		if err != nil {
			vtm.Logger.Infof("vibe-tools failed for %s, falling back to basic notes: %v", repoURL, err)
			return vtm.generateBasicReleaseNotes(repoPath, repoURL)
		}
	}

	// Clean up cloned repository
	if err := os.RemoveAll(repoPath); err != nil {
		vtm.Logger.Warnf("Failed to clean up repository directory %s: %v", repoPath, err)
	}
	
	return string(output), nil
}

// generateBasicReleaseNotes generates basic release notes when vibe-tools is not available
func (vtm *VibeToolsManager) generateBasicReleaseNotes(repoPath, repoURL string) (string, error) {
	// Get basic repository information
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", WrapError(err, ErrorTypeGit, "failed to open repository", map[string]interface{}{
			"repo_path": repoPath,
		})
	}

	// Get main branch reference
	ref, err := repo.Reference("refs/heads/main", true)
	if err != nil {
		// Try master branch if main doesn't exist
		ref, err = repo.Reference("refs/heads/master", true)
		if err != nil {
			return "", WrapError(err, ErrorTypeGit, "failed to get main/master branch reference", map[string]interface{}{
				"repo_path": repoPath,
			})
		}
	}

	// Get commit information
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", WrapError(err, ErrorTypeGit, "failed to get commit object", map[string]interface{}{
			"repo_path": repoPath,
		})
	}

	// Calculate date range for last week
	now := time.Now()
	oneWeekAgo := now.AddDate(0, 0, -7)
	
	vtm.Logger.Infof("Analyzing commits from the last week (since %s)", oneWeekAgo.Format("2006-01-02 15:04:05"))

	// Get commits from the last week
	commitIter, err := repo.Log(&git.LogOptions{
		From: ref.Hash(),
		All:  false,
		Since: &oneWeekAgo,
	})
	if err != nil {
		return "", WrapError(err, ErrorTypeGit, "failed to get commit log", map[string]interface{}{
			"repo_path": repoPath,
		})
	}

	var commitDetails []CommitDetail
	var commitCount int
	var authorStats = make(map[string]int)
	var totalChanges int
	
	commitIter.ForEach(func(c *object.Commit) error {
		commitCount++
		
		// Count changes in this commit with panic recovery
		// Some commits with very large diffs can cause panics in the diff library
		func() {
			defer func() {
				if r := recover(); r != nil {
					vtm.Logger.Warnf("Failed to calculate stats for commit %s (panic recovered): %v", c.Hash.String()[:8], r)
				}
			}()
			
			stats, err := c.Stats()
			if err == nil {
				for _, stat := range stats {
					totalChanges += stat.Addition + stat.Deletion
				}
			} else {
				vtm.Logger.Debugf("Failed to get stats for commit %s: %v", c.Hash.String()[:8], err)
			}
		}()
		
		// Track author activity
		authorStats[c.Author.Name]++
		
		// Add commit detail
		commitDetails = append(commitDetails, CommitDetail{
			Hash:    c.Hash.String()[:8],
			Message: strings.TrimSpace(c.Message),
			Author:  c.Author.Name,
			Date:    c.Author.When,
		})
		
		return nil
	})

	// Clean up cloned repository
	if err := os.RemoveAll(repoPath); err != nil {
		vtm.Logger.Warnf("Failed to clean up repository directory %s: %v", repoPath, err)
	}

	// Create contributors list
	var contributors []Contributor
	type authorCommit struct {
		author string
		count  int
	}
	var sortedAuthors []authorCommit
	for author, count := range authorStats {
		sortedAuthors = append(sortedAuthors, authorCommit{author, count})
	}
	
	// Simple sort by count (descending)
	for i := 0; i < len(sortedAuthors); i++ {
		for j := i + 1; j < len(sortedAuthors); j++ {
			if sortedAuthors[i].count < sortedAuthors[j].count {
				sortedAuthors[i], sortedAuthors[j] = sortedAuthors[j], sortedAuthors[i]
			}
		}
	}
	
	// Convert to contributors
	for i, author := range sortedAuthors {
		contributors = append(contributors, Contributor{
			Name:        author.author,
			CommitCount: author.count,
			Rank:        i + 1,
		})
	}

	// Create standard format using formatter
	format := vtm.Formatter.CreateStandardFormat(
		repoURL,
		oneWeekAgo,
		now,
		CommitInfo{
			Hash:    commit.Hash.String()[:8],
			Message: commit.Message,
			Author:  commit.Author.Name,
			Date:    commit.Author.When,
		},
		WeeklySummary{
			TotalCommits:      commitCount,
			TotalLinesChanged: totalChanges,
			ActiveContributors: len(authorStats),
			AnalysisStart:     oneWeekAgo,
			AnalysisEnd:       now,
		},
		contributors,
		commitDetails,
	)

	return vtm.Formatter.FormatReleaseNote(format), nil
}

// extractRepoName extracts repository name from URL
func (vtm *VibeToolsManager) extractRepoName(repoURL string) string {
	// Remove .git suffix if present
	repoURL = strings.TrimSuffix(repoURL, ".git")
	
	// Extract name from URL
	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	
	return "unknown-repo"
}