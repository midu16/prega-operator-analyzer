package pkg

import (
	"fmt"
	"strings"
	"time"
)

// ReleaseNoteFormat defines the structure for consistent release notes
type ReleaseNoteFormat struct {
	Header           string
	RepositoryInfo   RepositoryInfo
	AnalysisPeriod   string
	AnalysisStart    time.Time
	AnalysisEnd      time.Time
	LatestCommit     CommitInfo
	WeeklySummary    WeeklySummary
	Contributors     []Contributor
	Commits          []CommitDetail
	Footer           string
}

// RepositoryInfo contains basic repository information
type RepositoryInfo struct {
	URL         string
	Name        string
	Description string
}

// CommitInfo contains latest commit information
type CommitInfo struct {
	Hash    string
	Message string
	Author  string
	Date    time.Time
}

// WeeklySummary contains weekly activity statistics
type WeeklySummary struct {
	TotalCommits     int
	TotalLinesChanged int
	ActiveContributors int
	AnalysisStart    time.Time
	AnalysisEnd      time.Time
}

// Contributor represents a contributor with their activity
type Contributor struct {
	Name        string
	CommitCount int
	Rank        int
}

// CommitDetail represents a detailed commit entry
type CommitDetail struct {
	Hash    string
	Message string
	Author  string
	Date    time.Time
}

// ReleaseNoteFormatter handles consistent formatting of release notes
type ReleaseNoteFormatter struct {
	MaxContributors int
	MaxCommits      int
}

// NewReleaseNoteFormatter creates a new formatter with default settings
func NewReleaseNoteFormatter() *ReleaseNoteFormatter {
	return &ReleaseNoteFormatter{
		MaxContributors: 5,
		MaxCommits:      50, // Limit to prevent extremely long outputs
	}
}

// FormatReleaseNote creates a consistently formatted release note
func (rnf *ReleaseNoteFormatter) FormatReleaseNote(format ReleaseNoteFormat) string {
	var output strings.Builder
	
	// Header
	output.WriteString(format.Header)
	output.WriteString("\n")
	
	// Repository Information
	output.WriteString(fmt.Sprintf("Repository: %s\n", format.RepositoryInfo.URL))
	if format.RepositoryInfo.Name != "" {
		output.WriteString(fmt.Sprintf("Name: %s\n", format.RepositoryInfo.Name))
	}
	if format.RepositoryInfo.Description != "" {
		output.WriteString(fmt.Sprintf("Description: %s\n", format.RepositoryInfo.Description))
	}
	output.WriteString(strings.Repeat("-", 80))
	output.WriteString("\n")
	
	// Analysis Period
	output.WriteString(fmt.Sprintf("Analysis Period: %s\n", format.AnalysisPeriod))
	output.WriteString(fmt.Sprintf("Analysis Start: %s\n", format.AnalysisStart.Format("2006-01-02 15:04:05")))
	output.WriteString(fmt.Sprintf("Analysis End: %s\n\n", format.AnalysisEnd.Format("2006-01-02 15:04:05")))
	
	// Latest Commit Information
	output.WriteString("=== LATEST COMMIT INFORMATION ===\n")
	output.WriteString(fmt.Sprintf("Hash: %s\n", format.LatestCommit.Hash))
	output.WriteString(fmt.Sprintf("Message: %s\n", format.LatestCommit.Message))
	output.WriteString(fmt.Sprintf("Author: %s\n", format.LatestCommit.Author))
	output.WriteString(fmt.Sprintf("Date: %s\n\n", format.LatestCommit.Date.Format("2006-01-02 15:04:05")))
	
	// Weekly Activity Summary
	output.WriteString("=== WEEKLY ACTIVITY SUMMARY ===\n")
	output.WriteString(fmt.Sprintf("Total Commits: %d\n", format.WeeklySummary.TotalCommits))
	output.WriteString(fmt.Sprintf("Total Lines Changed: %d\n", format.WeeklySummary.TotalLinesChanged))
	output.WriteString(fmt.Sprintf("Active Contributors: %d\n\n", format.WeeklySummary.ActiveContributors))
	
	// Top Contributors
	if len(format.Contributors) > 0 {
		output.WriteString("=== TOP CONTRIBUTORS (LAST WEEK) ===\n")
		for _, contributor := range format.Contributors {
			output.WriteString(fmt.Sprintf("%d. %s (%d commits)\n", 
				contributor.Rank, contributor.Name, contributor.CommitCount))
		}
		output.WriteString("\n")
	}
	
	// Recent Commits
	if len(format.Commits) > 0 {
		output.WriteString("=== COMMITS FROM LAST WEEK ===\n")
		commitCount := len(format.Commits)
		if commitCount > rnf.MaxCommits {
			output.WriteString(fmt.Sprintf("(Showing first %d of %d commits)\n", rnf.MaxCommits, commitCount))
			commitCount = rnf.MaxCommits
		}
		
		for i := 0; i < commitCount; i++ {
			commit := format.Commits[i]
			output.WriteString(fmt.Sprintf("- %s (%s) by %s on %s\n",
				strings.TrimSpace(commit.Message),
				commit.Hash,
				commit.Author,
				commit.Date.Format("2006-01-02 15:04:05")))
		}
	} else {
		output.WriteString("=== NO COMMITS IN LAST WEEK ===\n")
		output.WriteString("No commits found in the main branch during the last 7 days.\n")
	}
	
	// Footer
	if format.Footer != "" {
		output.WriteString("\n")
		output.WriteString(format.Footer)
	}
	
	output.WriteString("\n\n")
	return output.String()
}

// CreateStandardFormat creates a standard release note format structure
func (rnf *ReleaseNoteFormatter) CreateStandardFormat(
	repoURL string,
	analysisStart time.Time,
	analysisEnd time.Time,
	latestCommit CommitInfo,
	weeklySummary WeeklySummary,
	contributors []Contributor,
	commits []CommitDetail,
) ReleaseNoteFormat {
	
	// Limit contributors to max
	if len(contributors) > rnf.MaxContributors {
		contributors = contributors[:rnf.MaxContributors]
	}
	
	// Limit commits to max
	if len(commits) > rnf.MaxCommits {
		commits = commits[:rnf.MaxCommits]
	}
	
	// Calculate analysis period
	period := fmt.Sprintf("Last 7 days (since %s)", analysisStart.Format("2006-01-02 15:04:05"))
	
	return ReleaseNoteFormat{
		Header: fmt.Sprintf("Release Notes Generated on: %s", time.Now().Format("2006-01-02 15:04:05")),
		RepositoryInfo: RepositoryInfo{
			URL: repoURL,
		},
		AnalysisPeriod: period,
		AnalysisStart:  analysisStart,
		AnalysisEnd:    analysisEnd,
		LatestCommit:   latestCommit,
		WeeklySummary:  weeklySummary,
		Contributors:   contributors,
		Commits:        commits,
		Footer:         "Generated by Prega Operator Analyzer",
	}
}

// FormatErrorSection formats error information consistently
func (rnf *ReleaseNoteFormatter) FormatErrorSection(repoURL string, err error) string {
	var output strings.Builder
	
	output.WriteString(fmt.Sprintf("Repository: %s\n", repoURL))
	output.WriteString(strings.Repeat("-", 80))
	output.WriteString("\n")
	output.WriteString("=== ERROR PROCESSING REPOSITORY ===\n")
	output.WriteString(fmt.Sprintf("Error: %v\n", err))
	output.WriteString(fmt.Sprintf("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	output.WriteString("This repository could not be processed successfully.\n")
	output.WriteString("Please check the repository URL and network connectivity.\n\n")
	
	return output.String()
}