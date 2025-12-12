package pkg

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sirupsen/logrus"
)

// Server represents the web server for the analyzer
type Server struct {
	Port           int
	WorkDir        string
	OutputDir      string
	Repositories   []string
	PregaIndex     string
	Logger         *logrus.Logger
	mu             sync.Mutex
	cachedData     *CachedData
	lastCacheTime  time.Time
	cacheDuration  time.Duration
}

// CachedData holds cached repository and branch information
type CachedData struct {
	Repositories []RepositoryData `json:"repositories"`
	LastUpdated  time.Time        `json:"lastUpdated"`
}

// RepositoryData holds repository information with branches
type RepositoryData struct {
	URL         string   `json:"url"`
	Name        string   `json:"name"`
	Branches    []string `json:"branches"`
	Description string   `json:"description,omitempty"`
}

// ReleaseNotesRequest represents a request for release notes
type ReleaseNotesRequest struct {
	Repository string `json:"repository"`
	Branch     string `json:"branch"`
	Days       int    `json:"days"`
}

// ReleaseNotesResponse represents the response with release notes
type ReleaseNotesResponse struct {
	Success      bool   `json:"success"`
	HTML         string `json:"html"`
	Text         string `json:"text"`
	Repository   string `json:"repository"`
	Branch       string `json:"branch"`
	Days         int    `json:"days"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// NewServer creates a new web server
func NewServer(port int, workDir, outputDir, pregaIndex string, logger *logrus.Logger) *Server {
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}
	return &Server{
		Port:          port,
		WorkDir:       workDir,
		OutputDir:     outputDir,
		PregaIndex:    pregaIndex,
		Logger:        logger,
		cacheDuration: 5 * time.Minute,
	}
}

// Start starts the web server
func (s *Server) Start() error {
	// Create directories
	os.MkdirAll(s.WorkDir, 0755)
	os.MkdirAll(s.OutputDir, 0755)

	// Set up routes
	mux := http.NewServeMux()
	
	// Static files and main page
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/repositories", s.handleRepositories)
	mux.HandleFunc("/api/branches", s.handleBranches)
	mux.HandleFunc("/api/release-notes", s.handleReleaseNotes)
	mux.HandleFunc("/api/refresh", s.handleRefresh)

	s.Logger.Infof("Starting web server on port %d", s.Port)
	s.Logger.Infof("Access the web interface at: http://localhost:%d", s.Port)
	
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), mux)
}

// SetRepositories sets the list of repositories
func (s *Server) SetRepositories(repos []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Repositories = repos
}

// handleIndex serves the main HTML page
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("index").Parse(indexHTML))
	tmpl.Execute(w, nil)
}

// handleRepositories returns the list of repositories
func (s *Server) handleRepositories(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	s.mu.Lock()
	repos := s.Repositories
	s.mu.Unlock()

	var repoData []RepositoryData
	for _, repo := range repos {
		name := extractRepoNameFromURL(repo)
		repoData = append(repoData, RepositoryData{
			URL:  repo,
			Name: name,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"repositories": repoData,
	})
}

// handleBranches returns the branches for a repository
func (s *Server) handleBranches(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	repoURL := r.URL.Query().Get("repository")
	if repoURL == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "repository parameter is required",
		})
		return
	}

	branches, err := s.fetchBranches(repoURL)
	if err != nil {
		s.Logger.Errorf("Failed to fetch branches for %s: %v", repoURL, err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"branches": branches,
	})
}

// handleReleaseNotes generates release notes for a repository
func (s *Server) handleReleaseNotes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(ReleaseNotesResponse{
			Success:      false,
			ErrorMessage: "POST method required",
		})
		return
	}

	var req ReleaseNotesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(ReleaseNotesResponse{
			Success:      false,
			ErrorMessage: "Invalid request body: " + err.Error(),
		})
		return
	}

	// Validate request
	if req.Repository == "" {
		json.NewEncoder(w).Encode(ReleaseNotesResponse{
			Success:      false,
			ErrorMessage: "repository is required",
		})
		return
	}
	if req.Branch == "" {
		req.Branch = "main"
	}
	if req.Days <= 0 {
		req.Days = 7
	}
	if req.Days > 365 {
		req.Days = 365 // Cap at 1 year
	}

	// Generate release notes
	htmlNotes, textNotes, err := s.generateReleaseNotesForBranch(req.Repository, req.Branch, req.Days)
	if err != nil {
		json.NewEncoder(w).Encode(ReleaseNotesResponse{
			Success:      false,
			Repository:   req.Repository,
			Branch:       req.Branch,
			Days:         req.Days,
			ErrorMessage: err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(ReleaseNotesResponse{
		Success:    true,
		HTML:       htmlNotes,
		Text:       textNotes,
		Repository: req.Repository,
		Branch:     req.Branch,
		Days:       req.Days,
	})
}

// RefreshRequest represents a request to refresh repositories
type RefreshRequest struct {
	IndexImage string `json:"indexImage"`
}

// handleRefresh refreshes the repository list from the Prega index
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "POST method required",
		})
		return
	}

	// Parse request body for custom index image
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If decoding fails, use default index
		req.IndexImage = s.PregaIndex
	}

	// Use custom index if provided, otherwise use server default
	indexImage := req.IndexImage
	if indexImage == "" {
		indexImage = s.PregaIndex
	}

	s.Logger.Infof("Refreshing repositories from index: %s", indexImage)

	// Update the server's PregaIndex
	s.mu.Lock()
	s.PregaIndex = indexImage
	s.mu.Unlock()

	// Re-generate index and reload repositories
	indexPath := filepath.Join(s.WorkDir, "prega-operator-index", "index.json")
	
	// Generate index with the specified image
	if err := s.generateIndexJSON(indexPath); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to generate index: " + err.Error(),
		})
		return
	}

	// Parse repositories
	repos, err := ParseOperatorIndex(indexPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to parse index: " + err.Error(),
		})
		return
	}

	uniqueRepos := RemoveDuplicates(repos)
	s.SetRepositories(uniqueRepos)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"count":       len(uniqueRepos),
		"indexImage":  indexImage,
		"message":     fmt.Sprintf("Successfully refreshed %d repositories from %s", len(uniqueRepos), indexImage),
	})
}

// fetchBranches fetches all branches from a repository
func (s *Server) fetchBranches(repoURL string) ([]string, error) {
	repoName := extractRepoNameFromURL(repoURL)
	repoPath := filepath.Join(s.WorkDir, "branch-check", repoName)
	
	// Remove existing and clone fresh
	os.RemoveAll(repoPath)
	os.MkdirAll(filepath.Dir(repoPath), 0755)

	_, err := git.PlainClone(repoPath, false, &git.CloneOptions{
		URL:          repoURL,
		NoCheckout:   true,
		SingleBranch: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}
	defer os.RemoveAll(repoPath)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	refs, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	var branches []string
	branchSet := make(map[string]bool)

	refs.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().String()
		
		// Filter for remote branches
		if strings.HasPrefix(name, "refs/remotes/origin/") {
			branchName := strings.TrimPrefix(name, "refs/remotes/origin/")
			if branchName != "HEAD" {
				branchSet[branchName] = true
			}
		}
		return nil
	})

	for branch := range branchSet {
		branches = append(branches, branch)
	}

	// Sort branches: main/master first, then release-* branches, then others
	sort.Slice(branches, func(i, j int) bool {
		bi, bj := branches[i], branches[j]
		
		// Prioritize main/master
		if bi == "main" || bi == "master" {
			return true
		}
		if bj == "main" || bj == "master" {
			return false
		}
		
		// Then release branches
		isReleaseI := strings.HasPrefix(bi, "release-")
		isReleaseJ := strings.HasPrefix(bj, "release-")
		
		if isReleaseI && !isReleaseJ {
			return true
		}
		if !isReleaseI && isReleaseJ {
			return false
		}
		
		// For release branches, sort by version (descending)
		if isReleaseI && isReleaseJ {
			return bi > bj
		}
		
		return bi < bj
	})

	return branches, nil
}

// generateReleaseNotesForBranch generates release notes for a specific branch and period
func (s *Server) generateReleaseNotesForBranch(repoURL, branch string, days int) (string, string, error) {
	repoName := extractRepoNameFromURL(repoURL)
	repoPath := filepath.Join(s.WorkDir, "analysis", repoName)
	
	// Remove existing and clone fresh
	os.RemoveAll(repoPath)
	os.MkdirAll(filepath.Dir(repoPath), 0755)

	s.Logger.Infof("Cloning %s (branch: %s) for analysis...", repoURL, branch)

	_, err := git.PlainClone(repoPath, false, &git.CloneOptions{
		URL:           repoURL,
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
	})
	if err != nil {
		// Try with origin/branch reference
		_, err = git.PlainClone(repoPath, false, &git.CloneOptions{
			URL:           repoURL,
			ReferenceName: plumbing.NewRemoteReferenceName("origin", branch),
			SingleBranch:  true,
		})
		if err != nil {
			return "", "", fmt.Errorf("failed to clone branch %s: %w", branch, err)
		}
	}
	defer os.RemoveAll(repoPath)

	// Open repo and analyze
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to open repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get latest commit
	latestCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", "", fmt.Errorf("failed to get latest commit: %w", err)
	}

	// Calculate date range
	now := time.Now()
	since := now.AddDate(0, 0, -days)
	
	s.Logger.Infof("Analyzing commits from the last %d days (since %s)", days, since.Format("2006-01-02"))

	// Get commits from the specified period
	commitIter, err := repo.Log(&git.LogOptions{
		From:  head.Hash(),
		Since: &since,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to get commit log: %w", err)
	}

	var commitDetails []CommitDetail
	authorStats := make(map[string]int)
	var totalChanges int

	commitIter.ForEach(func(c *object.Commit) error {
		// Safe stats calculation with panic recovery
		func() {
			defer func() {
				if r := recover(); r != nil {
					s.Logger.Debugf("Recovered from panic calculating stats: %v", r)
				}
			}()
			
			stats, err := c.Stats()
			if err == nil {
				for _, stat := range stats {
					totalChanges += stat.Addition + stat.Deletion
				}
			}
		}()

		authorStats[c.Author.Name]++
		
		commitDetails = append(commitDetails, CommitDetail{
			Hash:    c.Hash.String()[:8],
			Message: strings.Split(strings.TrimSpace(c.Message), "\n")[0], // First line only
			Author:  c.Author.Name,
			Date:    c.Author.When,
		})
		
		return nil
	})

	// Create contributors list sorted by commit count
	type authorCommit struct {
		author string
		count  int
	}
	var sortedAuthors []authorCommit
	for author, count := range authorStats {
		sortedAuthors = append(sortedAuthors, authorCommit{author, count})
	}
	sort.Slice(sortedAuthors, func(i, j int) bool {
		return sortedAuthors[i].count > sortedAuthors[j].count
	})

	var contributors []Contributor
	for i, a := range sortedAuthors {
		contributors = append(contributors, Contributor{
			Name:        a.author,
			CommitCount: a.count,
			Rank:        i + 1,
		})
	}

	// Generate HTML output
	htmlOutput := s.generateHTMLReleaseNotes(
		repoURL,
		branch,
		days,
		since,
		now,
		CommitInfo{
			Hash:    latestCommit.Hash.String()[:8],
			Message: strings.Split(strings.TrimSpace(latestCommit.Message), "\n")[0],
			Author:  latestCommit.Author.Name,
			Date:    latestCommit.Author.When,
		},
		WeeklySummary{
			TotalCommits:       len(commitDetails),
			TotalLinesChanged:  totalChanges,
			ActiveContributors: len(authorStats),
			AnalysisStart:      since,
			AnalysisEnd:        now,
		},
		contributors,
		commitDetails,
	)

	// Generate text output
	formatter := NewReleaseNoteFormatter()
	format := formatter.CreateStandardFormatWithDays(
		repoURL,
		days,
		since,
		now,
		CommitInfo{
			Hash:    latestCommit.Hash.String()[:8],
			Message: latestCommit.Message,
			Author:  latestCommit.Author.Name,
			Date:    latestCommit.Author.When,
		},
		WeeklySummary{
			TotalCommits:       len(commitDetails),
			TotalLinesChanged:  totalChanges,
			ActiveContributors: len(authorStats),
			AnalysisStart:      since,
			AnalysisEnd:        now,
		},
		contributors,
		commitDetails,
	)
	textOutput := formatter.FormatReleaseNote(format)

	return htmlOutput, textOutput, nil
}

// generateHTMLReleaseNotes generates HTML formatted release notes
func (s *Server) generateHTMLReleaseNotes(
	repoURL, branch string,
	days int,
	analysisStart, analysisEnd time.Time,
	latestCommit CommitInfo,
	summary WeeklySummary,
	contributors []Contributor,
	commits []CommitDetail,
) string {
	var html strings.Builder
	
	// Build commit URL base
	commitURLBase := strings.TrimSuffix(repoURL, ".git")
	latestCommitURL := fmt.Sprintf("%s/commit/%s", commitURLBase, latestCommit.Hash)
	
	html.WriteString(fmt.Sprintf(`<div class="release-notes-content">
		<div class="notes-header">
			<h3>%s</h3>
			<div class="notes-meta">
				<span class="branch-tag">📌 %s</span>
				<span class="period-tag">📅 Last %d days</span>
				<span class="date-range">%s → %s</span>
			</div>
		</div>
		
		<div class="latest-commit">
			<h4>🔥 Latest Commit</h4>
			<a href="%s" target="_blank" class="commit-box-link">
				<div class="commit-box highlight">
					<div class="commit-box-header">
						<code class="commit-hash">%s</code>
						<span class="view-commit-btn">View on GitHub →</span>
					</div>
					<span class="commit-message">%s</span>
					<span class="commit-author">👤 %s</span>
					<span class="commit-date">📅 %s</span>
				</div>
			</a>
		</div>
		
		<div class="activity-summary">
			<h4>📊 Activity Summary</h4>
			<div class="stats-grid">
				<div class="stat-card">
					<span class="stat-value">%d</span>
					<span class="stat-label">Commits</span>
				</div>
				<div class="stat-card">
					<span class="stat-value">%d</span>
					<span class="stat-label">Lines Changed</span>
				</div>
				<div class="stat-card">
					<span class="stat-value">%d</span>
					<span class="stat-label">Contributors</span>
				</div>
			</div>
		</div>`,
		extractRepoNameFromURL(repoURL),
		branch,
		days,
		analysisStart.Format("Jan 02, 2006"),
		analysisEnd.Format("Jan 02, 2006"),
		latestCommitURL,
		latestCommit.Hash,
		template.HTMLEscapeString(latestCommit.Message),
		template.HTMLEscapeString(latestCommit.Author),
		latestCommit.Date.Format("Jan 02, 2006 15:04"),
		summary.TotalCommits,
		summary.TotalLinesChanged,
		summary.ActiveContributors,
	))

	// Contributors section
	if len(contributors) > 0 {
		html.WriteString(`<div class="contributors-section">
			<h4>👥 Top Contributors</h4>
			<div class="contributors-list">`)
		
		maxContributors := 5
		if len(contributors) < maxContributors {
			maxContributors = len(contributors)
		}
		
		for i := 0; i < maxContributors; i++ {
			c := contributors[i]
			html.WriteString(fmt.Sprintf(`
				<div class="contributor">
					<span class="rank">#%d</span>
					<span class="name">%s</span>
					<span class="commits">%d commits</span>
				</div>`,
				c.Rank,
				template.HTMLEscapeString(c.Name),
				c.CommitCount,
			))
		}
		html.WriteString(`</div></div>`)
	}

	// Commits section
	html.WriteString(`<div class="commits-section">
		<h4>📝 Recent Commits</h4>
		<div class="commits-list">`)
	
	maxCommits := 50
	if len(commits) < maxCommits {
		maxCommits = len(commits)
	}
	
	if maxCommits == 0 {
		html.WriteString(`<div class="no-commits">No commits found in this period</div>`)
	} else {
		if len(commits) > maxCommits {
			html.WriteString(fmt.Sprintf(`<div class="commits-note">Showing %d of %d commits</div>`, maxCommits, len(commits)))
		}
		
		// Build commit URL base (remove .git suffix if present)
		commitURLBase := strings.TrimSuffix(repoURL, ".git")
		
		for i := 0; i < maxCommits; i++ {
			c := commits[i]
			commitURL := fmt.Sprintf("%s/commit/%s", commitURLBase, c.Hash)
			html.WriteString(fmt.Sprintf(`
				<a href="%s" target="_blank" class="commit-item-link">
					<div class="commit-item">
						<div class="commit-header">
							<code class="commit-hash">%s</code>
							<span class="commit-link-icon">🔗</span>
						</div>
						<span class="commit-message">%s</span>
						<div class="commit-meta">
							<span class="author">👤 %s</span>
							<span class="date">📅 %s</span>
						</div>
					</div>
				</a>`,
				commitURL,
				c.Hash,
				template.HTMLEscapeString(c.Message),
				template.HTMLEscapeString(c.Author),
				c.Date.Format("Jan 02, 15:04"),
			))
		}
	}
	
	html.WriteString(`</div></div></div>`)
	
	return html.String()
}

// generateIndexJSON generates the index JSON file using opm render
func (s *Server) generateIndexJSON(outputPath string) error {
	dir := filepath.Dir(outputPath)
	os.MkdirAll(dir, 0755)

	opmPath, err := exec.LookPath("opm")
	if err != nil {
		return fmt.Errorf("opm command not found: %w", err)
	}
	s.Logger.Debugf("Found opm at: %s", opmPath)

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	cmd := exec.Command("opm", "render", s.PregaIndex, "--output=json")
	cmd.Stdout = outputFile
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute opm render: %w", err)
	}

	return nil
}

// extractRepoNameFromURL extracts repository name from URL
func extractRepoNameFromURL(repoURL string) string {
	repoURL = strings.TrimSuffix(repoURL, ".git")
	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown-repo"
}

// The main HTML template for the web interface
const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Prega Operator Analyzer</title>
    <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600&family=Outfit:wght@300;400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-primary: #0a0a0f;
            --bg-secondary: #12121a;
            --bg-tertiary: #1a1a24;
            --bg-card: #16161f;
            --accent-primary: #ff6b35;
            --accent-secondary: #f7c859;
            --accent-tertiary: #00d4aa;
            --accent-blue: #5b8def;
            --text-primary: #f5f5f7;
            --text-secondary: #a0a0b0;
            --text-muted: #6b6b7b;
            --border-color: #2a2a3a;
            --success: #00d4aa;
            --warning: #f7c859;
            --error: #ff5555;
            --gradient-accent: linear-gradient(135deg, #ff6b35 0%, #f7c859 100%);
            --gradient-bg: radial-gradient(ellipse at top, #1a1a2e 0%, #0a0a0f 50%);
            --shadow-glow: 0 0 40px rgba(255, 107, 53, 0.15);
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Outfit', -apple-system, BlinkMacSystemFont, sans-serif;
            background: var(--gradient-bg);
            color: var(--text-primary);
            min-height: 100vh;
            line-height: 1.6;
        }

        .app-container {
            display: grid;
            grid-template-columns: 380px 1fr;
            min-height: 100vh;
        }

        /* Sidebar */
        .sidebar {
            background: var(--bg-secondary);
            border-right: 1px solid var(--border-color);
            display: flex;
            flex-direction: column;
            height: 100vh;
            position: sticky;
            top: 0;
        }

        .sidebar-header {
            padding: 24px;
            border-bottom: 1px solid var(--border-color);
            background: linear-gradient(180deg, var(--bg-tertiary) 0%, var(--bg-secondary) 100%);
        }

        .logo {
            display: flex;
            align-items: center;
            gap: 12px;
            margin-bottom: 16px;
        }

        .logo-icon {
            width: 40px;
            height: 40px;
            background: var(--gradient-accent);
            border-radius: 10px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 20px;
        }

        .logo-text {
            font-size: 20px;
            font-weight: 700;
            background: var(--gradient-accent);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .version-badge {
            font-size: 11px;
            color: var(--text-muted);
            font-family: 'JetBrains Mono', monospace;
        }

        /* Controls */
        .controls {
            padding: 20px 24px;
            border-bottom: 1px solid var(--border-color);
        }

        .control-group {
            margin-bottom: 20px;
        }

        .control-group:last-child {
            margin-bottom: 0;
        }

        .control-label {
            display: block;
            font-size: 12px;
            font-weight: 600;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 8px;
        }

        .period-slider-container {
            display: flex;
            align-items: center;
            gap: 16px;
        }

        .period-slider {
            flex: 1;
            -webkit-appearance: none;
            height: 6px;
            background: var(--bg-tertiary);
            border-radius: 3px;
            outline: none;
        }

        .period-slider::-webkit-slider-thumb {
            -webkit-appearance: none;
            width: 20px;
            height: 20px;
            background: var(--accent-primary);
            border-radius: 50%;
            cursor: pointer;
            box-shadow: 0 0 10px rgba(255, 107, 53, 0.5);
            transition: transform 0.2s;
        }

        .period-slider::-webkit-slider-thumb:hover {
            transform: scale(1.2);
        }

        .period-value {
            font-family: 'JetBrains Mono', monospace;
            font-size: 16px;
            font-weight: 600;
            color: var(--accent-primary);
            min-width: 70px;
            text-align: right;
        }

        .index-input-container {
            display: flex;
            flex-direction: column;
            gap: 6px;
        }

        .index-prefix {
            font-family: 'JetBrains Mono', monospace;
            font-size: 11px;
            color: var(--text-muted);
        }

        .text-input {
            width: 100%;
            padding: 10px 14px;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: 8px;
            color: var(--text-primary);
            font-family: 'JetBrains Mono', monospace;
            font-size: 14px;
            outline: none;
            transition: border-color 0.2s, box-shadow 0.2s;
        }

        .text-input:focus {
            border-color: var(--accent-primary);
            box-shadow: 0 0 0 2px rgba(255, 107, 53, 0.2);
        }

        .text-input::placeholder {
            color: var(--text-muted);
        }

        .btn {
            padding: 12px 20px;
            border: none;
            border-radius: 8px;
            font-family: 'Outfit', sans-serif;
            font-size: 14px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
        }

        .btn-primary {
            background: var(--gradient-accent);
            color: var(--bg-primary);
            width: 100%;
        }

        .btn-primary:hover {
            box-shadow: var(--shadow-glow);
            transform: translateY(-2px);
        }

        .btn-secondary {
            background: var(--bg-tertiary);
            color: var(--text-primary);
            border: 1px solid var(--border-color);
        }

        .btn-secondary:hover {
            border-color: var(--accent-primary);
            color: var(--accent-primary);
        }

        .btn:disabled {
            opacity: 0.5;
            cursor: not-allowed;
            transform: none !important;
        }

        /* Repository List */
        .repo-section {
            flex: 1;
            overflow-y: auto;
            padding: 16px;
        }

        .section-title {
            font-size: 12px;
            font-weight: 600;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            padding: 8px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .repo-count {
            background: var(--bg-tertiary);
            padding: 2px 8px;
            border-radius: 10px;
            font-family: 'JetBrains Mono', monospace;
            font-size: 11px;
            color: var(--accent-primary);
        }

        .repo-list {
            list-style: none;
        }

        .repo-item {
            padding: 14px 16px;
            margin-bottom: 6px;
            background: var(--bg-card);
            border: 1px solid transparent;
            border-radius: 10px;
            cursor: pointer;
            transition: all 0.2s;
            position: relative;
        }

        .repo-item:hover {
            border-color: var(--border-color);
            background: var(--bg-tertiary);
        }

        .repo-item.selected {
            border-color: var(--accent-primary);
            background: rgba(255, 107, 53, 0.1);
        }

        .repo-item.dragging {
            opacity: 0.5;
            transform: scale(0.98);
        }

        .repo-name {
            font-weight: 500;
            font-size: 14px;
            margin-bottom: 4px;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .repo-url {
            font-size: 11px;
            color: var(--text-muted);
            font-family: 'JetBrains Mono', monospace;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .drag-handle {
            color: var(--text-muted);
            cursor: grab;
        }

        /* Main Content */
        .main-content {
            padding: 32px;
            overflow-y: auto;
        }

        .content-header {
            margin-bottom: 32px;
        }

        .content-title {
            font-size: 32px;
            font-weight: 700;
            margin-bottom: 8px;
        }

        .content-subtitle {
            font-size: 16px;
            color: var(--text-secondary);
        }

        /* Drop Zone */
        .drop-zone {
            border: 2px dashed var(--border-color);
            border-radius: 16px;
            padding: 60px 40px;
            text-align: center;
            margin-bottom: 32px;
            transition: all 0.3s;
            background: var(--bg-secondary);
        }

        .drop-zone.drag-over {
            border-color: var(--accent-primary);
            background: rgba(255, 107, 53, 0.05);
            box-shadow: var(--shadow-glow);
        }

        .drop-zone-icon {
            font-size: 48px;
            margin-bottom: 16px;
            opacity: 0.6;
        }

        .drop-zone-text {
            font-size: 18px;
            font-weight: 500;
            margin-bottom: 8px;
        }

        .drop-zone-hint {
            font-size: 14px;
            color: var(--text-muted);
        }

        /* Selected Operators */
        .selected-section {
            margin-bottom: 32px;
        }

        .selected-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 16px;
        }

        .selected-title {
            font-size: 18px;
            font-weight: 600;
        }

        .clear-btn {
            font-size: 13px;
            color: var(--text-muted);
            background: none;
            border: none;
            cursor: pointer;
            padding: 4px 8px;
        }

        .clear-btn:hover {
            color: var(--error);
        }

        .selected-operators {
            display: flex;
            flex-wrap: wrap;
            gap: 10px;
        }

        .selected-chip {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 10px 16px;
            background: var(--bg-tertiary);
            border: 1px solid var(--border-color);
            border-radius: 30px;
            font-size: 14px;
            font-weight: 500;
            cursor: pointer;
            transition: all 0.2s;
        }

        .selected-chip.active {
            border-color: var(--accent-primary);
            background: rgba(255, 107, 53, 0.1);
        }

        .selected-chip:hover {
            border-color: var(--accent-primary);
        }

        .chip-remove {
            color: var(--text-muted);
            font-size: 16px;
            line-height: 1;
            transition: color 0.2s;
        }

        .chip-remove:hover {
            color: var(--error);
        }

        /* Branch Selector - Dropdown Style */
        .branch-selector {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            padding: 16px 20px;
            margin-bottom: 24px;
        }

        .branch-selector-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .branch-selector-title {
            font-size: 14px;
            font-weight: 600;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .branch-selector-title::before {
            content: '🌿';
        }

        .branch-loading {
            font-size: 13px;
            color: var(--text-muted);
        }

        .branch-dropdown-container {
            position: relative;
            flex: 1;
            max-width: 400px;
            margin-left: 16px;
        }

        .branch-dropdown {
            width: 100%;
            padding: 12px 40px 12px 16px;
            background: var(--bg-tertiary);
            border: 2px solid var(--border-color);
            border-radius: 10px;
            color: var(--text-primary);
            font-family: 'JetBrains Mono', monospace;
            font-size: 14px;
            cursor: pointer;
            appearance: none;
            -webkit-appearance: none;
            -moz-appearance: none;
            transition: all 0.2s;
        }

        .branch-dropdown:hover {
            border-color: var(--accent-blue);
        }

        .branch-dropdown:focus {
            outline: none;
            border-color: var(--accent-primary);
            box-shadow: 0 0 0 3px rgba(255, 107, 53, 0.2);
        }

        .branch-dropdown option {
            background: var(--bg-secondary);
            color: var(--text-primary);
            padding: 12px;
        }

        .branch-dropdown option:checked {
            background: var(--accent-primary);
            color: var(--bg-primary);
        }

        .branch-dropdown-arrow {
            position: absolute;
            right: 14px;
            top: 50%;
            transform: translateY(-50%);
            pointer-events: none;
            color: var(--text-muted);
            font-size: 12px;
        }

        .branch-type-indicator {
            display: inline-block;
            padding: 2px 8px;
            border-radius: 4px;
            font-size: 11px;
            font-weight: 600;
            margin-left: 8px;
            text-transform: uppercase;
        }

        .branch-type-indicator.main {
            background: rgba(0, 212, 170, 0.2);
            color: var(--accent-tertiary);
        }

        .branch-type-indicator.release {
            background: rgba(247, 200, 89, 0.2);
            color: var(--accent-secondary);
        }

        .branch-type-indicator.other {
            background: rgba(91, 141, 239, 0.2);
            color: var(--accent-blue);
        }

        /* Release Notes */
        .release-notes-container {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            border-radius: 16px;
            overflow: hidden;
        }

        .release-notes-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 20px 24px;
            border-bottom: 1px solid var(--border-color);
            background: var(--bg-tertiary);
        }

        .release-notes-title {
            font-size: 16px;
            font-weight: 600;
        }

        .view-toggle {
            display: flex;
            gap: 4px;
            background: var(--bg-secondary);
            padding: 4px;
            border-radius: 8px;
        }

        .toggle-btn {
            padding: 8px 16px;
            background: transparent;
            border: none;
            border-radius: 6px;
            font-size: 13px;
            font-weight: 500;
            color: var(--text-muted);
            cursor: pointer;
            transition: all 0.2s;
        }

        .toggle-btn.active {
            background: var(--accent-primary);
            color: var(--bg-primary);
        }

        .release-notes-body {
            padding: 24px;
            max-height: 70vh;
            overflow-y: auto;
        }

        .release-notes-body pre {
            font-family: 'JetBrains Mono', monospace;
            font-size: 13px;
            line-height: 1.6;
            white-space: pre-wrap;
            word-break: break-word;
            color: var(--text-secondary);
        }

        /* Release Notes HTML Content Styles */
        .release-notes-content {
            color: var(--text-primary);
        }

        .notes-header {
            margin-bottom: 24px;
            padding-bottom: 16px;
            border-bottom: 1px solid var(--border-color);
        }

        .notes-header h3 {
            font-size: 24px;
            font-weight: 700;
            margin-bottom: 12px;
        }

        .notes-meta {
            display: flex;
            flex-wrap: wrap;
            gap: 12px;
            font-size: 13px;
        }

        .notes-meta span {
            padding: 4px 12px;
            background: var(--bg-tertiary);
            border-radius: 16px;
        }

        .branch-tag {
            color: var(--accent-blue);
        }

        .period-tag {
            color: var(--accent-secondary);
        }

        .date-range {
            color: var(--text-muted);
        }

        .latest-commit, .activity-summary, .contributors-section, .commits-section {
            margin-bottom: 24px;
        }

        .latest-commit h4, .activity-summary h4, .contributors-section h4, .commits-section h4 {
            font-size: 16px;
            font-weight: 600;
            margin-bottom: 16px;
            color: var(--text-secondary);
        }

        .commit-box-link {
            text-decoration: none;
            color: inherit;
            display: block;
        }

        .commit-box {
            padding: 16px;
            background: var(--bg-tertiary);
            border-radius: 10px;
            border-left: 3px solid var(--accent-primary);
            transition: all 0.2s;
        }

        .commit-box-link:hover .commit-box {
            background: rgba(247, 200, 89, 0.1);
            transform: translateX(4px);
        }

        .commit-box.highlight {
            border-left-color: var(--accent-secondary);
        }

        .commit-box-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 8px;
        }

        .view-commit-btn {
            font-size: 12px;
            color: var(--accent-blue);
            opacity: 0;
            transition: opacity 0.2s;
        }

        .commit-box-link:hover .view-commit-btn {
            opacity: 1;
        }

        .commit-hash {
            font-family: 'JetBrains Mono', monospace;
            font-size: 12px;
            padding: 3px 8px;
            background: var(--bg-secondary);
            border-radius: 4px;
            color: var(--accent-blue);
            margin-right: 10px;
        }

        .commit-message {
            font-weight: 500;
        }

        .commit-author, .commit-date {
            display: block;
            font-size: 13px;
            color: var(--text-muted);
            margin-top: 8px;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 16px;
        }

        .stat-card {
            background: var(--bg-tertiary);
            padding: 20px;
            border-radius: 12px;
            text-align: center;
        }

        .stat-value {
            display: block;
            font-size: 32px;
            font-weight: 700;
            color: var(--accent-primary);
            font-family: 'JetBrains Mono', monospace;
        }

        .stat-label {
            font-size: 13px;
            color: var(--text-muted);
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .contributors-list {
            display: grid;
            gap: 8px;
        }

        .contributor {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px 16px;
            background: var(--bg-tertiary);
            border-radius: 8px;
        }

        .contributor .rank {
            font-family: 'JetBrains Mono', monospace;
            font-size: 12px;
            color: var(--accent-secondary);
            min-width: 30px;
        }

        .contributor .name {
            flex: 1;
            font-weight: 500;
        }

        .contributor .commits {
            font-size: 13px;
            color: var(--text-muted);
        }

        .commits-list {
            display: grid;
            gap: 8px;
        }

        .commits-note {
            font-size: 13px;
            color: var(--text-muted);
            margin-bottom: 12px;
            font-style: italic;
        }

        .commit-item-link {
            text-decoration: none;
            color: inherit;
            display: block;
        }

        .commit-item {
            padding: 14px 16px;
            background: var(--bg-tertiary);
            border-radius: 8px;
            display: grid;
            gap: 8px;
            border: 1px solid transparent;
            transition: all 0.2s;
        }

        .commit-item-link:hover .commit-item {
            border-color: var(--accent-blue);
            background: rgba(91, 141, 239, 0.1);
            transform: translateX(4px);
        }

        .commit-item-link:hover .commit-link-icon {
            opacity: 1;
        }

        .commit-header {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .commit-link-icon {
            font-size: 12px;
            opacity: 0;
            transition: opacity 0.2s;
        }

        .commit-item .commit-message {
            font-weight: 400;
            line-height: 1.4;
        }

        .commit-meta {
            display: flex;
            gap: 16px;
            font-size: 12px;
            color: var(--text-muted);
        }

        .no-commits {
            padding: 40px;
            text-align: center;
            color: var(--text-muted);
            font-style: italic;
        }

        /* Loading State */
        .loading-overlay {
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(10, 10, 15, 0.9);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 1000;
            opacity: 0;
            pointer-events: none;
            transition: opacity 0.3s;
        }

        .loading-overlay.active {
            opacity: 1;
            pointer-events: all;
        }

        .loading-spinner {
            text-align: center;
        }

        .spinner {
            width: 50px;
            height: 50px;
            border: 3px solid var(--border-color);
            border-top-color: var(--accent-primary);
            border-radius: 50%;
            animation: spin 1s linear infinite;
            margin: 0 auto 20px;
        }

        @keyframes spin {
            to { transform: rotate(360deg); }
        }

        .loading-text {
            font-size: 16px;
            color: var(--text-secondary);
        }

        /* Empty State */
        .empty-state {
            text-align: center;
            padding: 60px 40px;
            color: var(--text-muted);
        }

        .empty-icon {
            font-size: 64px;
            opacity: 0.4;
            margin-bottom: 20px;
        }

        .empty-title {
            font-size: 20px;
            font-weight: 600;
            color: var(--text-secondary);
            margin-bottom: 8px;
        }

        /* Responsive */
        @media (max-width: 1024px) {
            .app-container {
                grid-template-columns: 1fr;
            }

            .sidebar {
                position: relative;
                height: auto;
                max-height: 50vh;
            }

            .stats-grid {
                grid-template-columns: 1fr;
            }
        }

        /* Scrollbar */
        ::-webkit-scrollbar {
            width: 8px;
        }

        ::-webkit-scrollbar-track {
            background: var(--bg-secondary);
        }

        ::-webkit-scrollbar-thumb {
            background: var(--border-color);
            border-radius: 4px;
        }

        ::-webkit-scrollbar-thumb:hover {
            background: var(--text-muted);
        }
    </style>
</head>
<body>
    <div class="app-container">
        <!-- Sidebar -->
        <aside class="sidebar">
            <div class="sidebar-header">
                <div class="logo">
                    <div class="logo-icon">🔍</div>
                    <div>
                        <div class="logo-text">Prega Analyzer</div>
                        <div class="version-badge">Release Notes Generator</div>
                    </div>
                </div>
            </div>

            <div class="controls">
                <div class="control-group">
                    <label class="control-label">Prega Index Tag</label>
                    <div class="index-input-container">
                        <input type="text" class="text-input" id="indexTagInput" value="v4.21" placeholder="e.g., v4.21">
                        <span class="index-prefix">quay.io/prega/prega-operator-index:</span>
                    </div>
                </div>

                <div class="control-group">
                    <label class="control-label">Analysis Period</label>
                    <div class="period-slider-container">
                        <input type="range" class="period-slider" id="periodSlider" min="1" max="90" value="7">
                        <span class="period-value" id="periodValue">7 days</span>
                    </div>
                </div>

                <div class="control-group">
                    <button class="btn btn-primary" id="generateBtn" disabled>
                        <span>🚀</span> Generate Release Notes
                    </button>
                </div>

                <div class="control-group">
                    <button class="btn btn-secondary" id="refreshBtn">
                        <span>🔄</span> Refresh Repositories
                    </button>
                </div>
            </div>

            <div class="repo-section">
                <div class="section-title">
                    <span>Operators</span>
                    <span class="repo-count" id="repoCount">0</span>
                </div>
                <ul class="repo-list" id="repoList">
                    <!-- Repositories will be loaded here -->
                </ul>
            </div>
        </aside>

        <!-- Main Content -->
        <main class="main-content">
            <div class="content-header">
                <h1 class="content-title">Release Notes</h1>
                <p class="content-subtitle">Drag operators from the sidebar or click to select, then choose a branch</p>
            </div>

            <!-- Drop Zone -->
            <div class="drop-zone" id="dropZone">
                <div class="drop-zone-icon">📦</div>
                <div class="drop-zone-text">Drop operators here</div>
                <div class="drop-zone-hint">or click on an operator in the sidebar</div>
            </div>

            <!-- Selected Operators -->
            <div class="selected-section" id="selectedSection" style="display: none;">
                <div class="selected-header">
                    <span class="selected-title">Selected Operators</span>
                    <button class="clear-btn" id="clearAllBtn">Clear all</button>
                </div>
                <div class="selected-operators" id="selectedOperators"></div>
            </div>

            <!-- Branch Selector - Dropdown -->
            <div class="branch-selector" id="branchSelector" style="display: none;">
                <div class="branch-selector-header">
                    <span class="branch-selector-title">Select Branch</span>
                    <div class="branch-dropdown-container">
                        <select class="branch-dropdown" id="branchDropdown">
                            <option value="">-- Select a branch --</option>
                        </select>
                        <span class="branch-dropdown-arrow">▼</span>
                    </div>
                    <span class="branch-loading" id="branchLoading"></span>
                </div>
            </div>

            <!-- Release Notes -->
            <div class="release-notes-container" id="releaseNotesContainer" style="display: none;">
                <div class="release-notes-header">
                    <span class="release-notes-title">📋 Release Notes</span>
                    <div class="view-toggle">
                        <button class="toggle-btn active" data-view="html">Rich View</button>
                        <button class="toggle-btn" data-view="text">Plain Text</button>
                    </div>
                </div>
                <div class="release-notes-body" id="releaseNotesBody">
                    <!-- Release notes content -->
                </div>
            </div>

            <!-- Empty State -->
            <div class="empty-state" id="emptyState">
                <div class="empty-icon">📝</div>
                <div class="empty-title">No release notes yet</div>
                <p>Select an operator and branch to generate release notes</p>
            </div>
        </main>
    </div>

    <!-- Loading Overlay -->
    <div class="loading-overlay" id="loadingOverlay">
        <div class="loading-spinner">
            <div class="spinner"></div>
            <div class="loading-text" id="loadingText">Loading...</div>
        </div>
    </div>

    <script>
        // State
        let repositories = [];
        let selectedOps = [];
        let activeOperator = null;
        let selectedBranch = null;
        let currentReleaseNotes = { html: '', text: '' };
        let currentView = 'html';

        // DOM Elements
        const indexTagInput = document.getElementById('indexTagInput');
        const periodSlider = document.getElementById('periodSlider');
        const periodValue = document.getElementById('periodValue');
        const generateBtn = document.getElementById('generateBtn');
        const refreshBtn = document.getElementById('refreshBtn');
        const repoList = document.getElementById('repoList');
        const repoCount = document.getElementById('repoCount');
        const dropZone = document.getElementById('dropZone');
        const selectedSection = document.getElementById('selectedSection');
        const selectedOperatorsEl = document.getElementById('selectedOperators');
        const branchSelector = document.getElementById('branchSelector');
        const branchDropdown = document.getElementById('branchDropdown');
        const branchLoading = document.getElementById('branchLoading');
        const releaseNotesContainer = document.getElementById('releaseNotesContainer');
        const releaseNotesBody = document.getElementById('releaseNotesBody');
        const emptyState = document.getElementById('emptyState');
        const loadingOverlay = document.getElementById('loadingOverlay');
        const loadingText = document.getElementById('loadingText');
        const clearAllBtn = document.getElementById('clearAllBtn');

        // Initialize
        document.addEventListener('DOMContentLoaded', () => {
            loadRepositories();
            setupEventListeners();
        });

        function setupEventListeners() {
            // Period slider
            periodSlider.addEventListener('input', () => {
                periodValue.textContent = periodSlider.value + ' days';
            });

            // Generate button
            generateBtn.addEventListener('click', generateReleaseNotes);

            // Refresh button
            refreshBtn.addEventListener('click', refreshRepositories);

            // Clear all button
            clearAllBtn.addEventListener('click', clearAllSelected);

            // Drop zone
            dropZone.addEventListener('dragover', (e) => {
                e.preventDefault();
                dropZone.classList.add('drag-over');
            });

            dropZone.addEventListener('dragleave', () => {
                dropZone.classList.remove('drag-over');
            });

            dropZone.addEventListener('drop', (e) => {
                e.preventDefault();
                dropZone.classList.remove('drag-over');
                const repoData = e.dataTransfer.getData('application/json');
                if (repoData) {
                    const repo = JSON.parse(repoData);
                    addSelectedOperator(repo);
                }
            });

            // View toggle
            document.querySelectorAll('.toggle-btn').forEach(btn => {
                btn.addEventListener('click', () => {
                    document.querySelectorAll('.toggle-btn').forEach(b => b.classList.remove('active'));
                    btn.classList.add('active');
                    currentView = btn.dataset.view;
                    updateReleaseNotesView();
                });
            });
        }

        async function loadRepositories() {
            showLoading('Loading repositories...');
            try {
                const response = await fetch('/api/repositories');
                const data = await response.json();
                if (data.success) {
                    repositories = data.repositories || [];
                    renderRepositoryList();
                } else {
                    console.error('Failed to load repositories:', data.error);
                }
            } catch (error) {
                console.error('Error loading repositories:', error);
            }
            hideLoading();
        }

        async function refreshRepositories() {
            const indexTag = indexTagInput.value.trim() || 'v4.21';
            const fullIndex = 'quay.io/prega/prega-operator-index:' + indexTag;
            showLoading('Refreshing from ' + fullIndex + '...');
            try {
                const response = await fetch('/api/refresh', { 
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ indexImage: fullIndex })
                });
                const data = await response.json();
                if (data.success) {
                    await loadRepositories();
                    alert('Successfully refreshed ' + data.count + ' repositories from ' + fullIndex);
                } else {
                    alert('Failed to refresh: ' + data.error);
                }
            } catch (error) {
                console.error('Error refreshing:', error);
                alert('Error refreshing repositories');
            }
            hideLoading();
        }

        function renderRepositoryList() {
            repoCount.textContent = repositories.length;
            repoList.innerHTML = '';
            
            repositories.forEach(repo => {
                const li = document.createElement('li');
                li.className = 'repo-item';
                li.draggable = true;
                li.innerHTML = ` + "`" + `
                    <div class="repo-name">
                        <span class="drag-handle">⋮⋮</span>
                        ${escapeHtml(repo.name)}
                    </div>
                    <div class="repo-url">${escapeHtml(repo.url)}</div>
                ` + "`" + `;

                // Click to select
                li.addEventListener('click', () => addSelectedOperator(repo));

                // Drag start
                li.addEventListener('dragstart', (e) => {
                    e.dataTransfer.setData('application/json', JSON.stringify(repo));
                    li.classList.add('dragging');
                });

                li.addEventListener('dragend', () => {
                    li.classList.remove('dragging');
                });

                repoList.appendChild(li);
            });
        }

        function addSelectedOperator(repo) {
            // Check if already selected
            if (selectedOps.find(r => r.url === repo.url)) {
                setActiveOperator(repo);
                return;
            }

            selectedOps.push(repo);
            setActiveOperator(repo);
            updateSelectedOperatorsUI();
        }

        function setActiveOperator(repo) {
            activeOperator = repo;
            selectedBranch = null;
            updateSelectedOperatorsUI();
            loadBranches(repo);
        }

        function removeSelectedOperator(repo) {
            selectedOps = selectedOps.filter(r => r.url !== repo.url);
            if (activeOperator && activeOperator.url === repo.url) {
                activeOperator = selectedOps.length > 0 ? selectedOps[0] : null;
                if (activeOperator) {
                    loadBranches(activeOperator);
                } else {
                    branchSelector.style.display = 'none';
                }
            }
            updateSelectedOperatorsUI();
        }

        function clearAllSelected() {
            selectedOps = [];
            activeOperator = null;
            selectedBranch = null;
            updateSelectedOperatorsUI();
            branchSelector.style.display = 'none';
            releaseNotesContainer.style.display = 'none';
            emptyState.style.display = 'block';
        }

        function updateSelectedOperatorsUI() {
            if (selectedOps.length === 0) {
                selectedSection.style.display = 'none';
                dropZone.style.display = 'block';
                generateBtn.disabled = true;
                return;
            }

            selectedSection.style.display = 'block';
            dropZone.style.display = 'none';
            
            selectedOperatorsEl.innerHTML = '';
            selectedOps.forEach(repo => {
                const chip = document.createElement('div');
                chip.className = 'selected-chip' + (activeOperator && activeOperator.url === repo.url ? ' active' : '');
                chip.innerHTML = ` + "`" + `
                    <span>${escapeHtml(repo.name)}</span>
                    <span class="chip-remove">&times;</span>
                ` + "`" + `;
                
                chip.querySelector('.chip-remove').addEventListener('click', (e) => {
                    e.stopPropagation();
                    removeSelectedOperator(repo);
                });
                
                chip.addEventListener('click', () => setActiveOperator(repo));
                
                selectedOperatorsEl.appendChild(chip);
            });

            generateBtn.disabled = !selectedBranch;
        }

        async function loadBranches(repo) {
            branchSelector.style.display = 'block';
            branchLoading.textContent = 'Loading...';
            branchDropdown.innerHTML = '<option value="">Loading branches...</option>';
            branchDropdown.disabled = true;

            try {
                const response = await fetch('/api/branches?repository=' + encodeURIComponent(repo.url));
                const data = await response.json();
                
                if (data.success) {
                    branchLoading.textContent = '';
                    branchDropdown.disabled = false;
                    renderBranches(data.branches || []);
                } else {
                    branchLoading.textContent = 'Error: ' + data.error;
                    branchDropdown.innerHTML = '<option value="">Error loading branches</option>';
                }
            } catch (error) {
                branchLoading.textContent = 'Error loading branches';
                branchDropdown.innerHTML = '<option value="">Error loading branches</option>';
                console.error('Error loading branches:', error);
            }
        }

        function renderBranches(branches) {
            // Clear dropdown and add placeholder
            branchDropdown.innerHTML = '<option value="">-- Select a branch --</option>';
            
            // Group branches by type
            const mainBranches = branches.filter(b => b === 'main' || b === 'master');
            const releaseBranches = branches.filter(b => b.startsWith('release-')).sort((a, b) => b.localeCompare(a));
            const otherBranches = branches.filter(b => b !== 'main' && b !== 'master' && !b.startsWith('release-'));
            
            // Add main/master first
            if (mainBranches.length > 0) {
                const optgroup = document.createElement('optgroup');
                optgroup.label = '🏠 Main Branch';
                mainBranches.forEach(branch => {
                    const option = document.createElement('option');
                    option.value = branch;
                    option.textContent = branch;
                    optgroup.appendChild(option);
                });
                branchDropdown.appendChild(optgroup);
            }
            
            // Add release branches
            if (releaseBranches.length > 0) {
                const optgroup = document.createElement('optgroup');
                optgroup.label = '📦 Release Branches';
                releaseBranches.forEach(branch => {
                    const option = document.createElement('option');
                    option.value = branch;
                    option.textContent = branch;
                    optgroup.appendChild(option);
                });
                branchDropdown.appendChild(optgroup);
            }
            
            // Add other branches
            if (otherBranches.length > 0) {
                const optgroup = document.createElement('optgroup');
                optgroup.label = '🔀 Other Branches';
                otherBranches.slice(0, 20).forEach(branch => { // Limit to 20 to keep dropdown manageable
                    const option = document.createElement('option');
                    option.value = branch;
                    option.textContent = branch.length > 50 ? branch.substring(0, 47) + '...' : branch;
                    option.title = branch; // Full name on hover
                    optgroup.appendChild(option);
                });
                if (otherBranches.length > 20) {
                    const option = document.createElement('option');
                    option.disabled = true;
                    option.textContent = '... and ' + (otherBranches.length - 20) + ' more';
                    optgroup.appendChild(option);
                }
                branchDropdown.appendChild(optgroup);
            }

            // Auto-select main/master if available
            const mainBranch = branches.find(b => b === 'main' || b === 'master');
            if (mainBranch) {
                branchDropdown.value = mainBranch;
                selectedBranch = mainBranch;
                generateBtn.disabled = false;
            }
        }
        
        // Add event listener for dropdown change
        branchDropdown.addEventListener('change', (e) => {
            selectedBranch = e.target.value;
            generateBtn.disabled = !selectedBranch;
        });

        async function generateReleaseNotes() {
            if (!activeOperator || !selectedBranch) return;

            showLoading('Generating release notes for ' + activeOperator.name + '...');
            
            try {
                const response = await fetch('/api/release-notes', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        repository: activeOperator.url,
                        branch: selectedBranch,
                        days: parseInt(periodSlider.value)
                    })
                });

                const data = await response.json();
                
                if (data.success) {
                    currentReleaseNotes = { html: data.html, text: data.text };
                    releaseNotesContainer.style.display = 'block';
                    emptyState.style.display = 'none';
                    updateReleaseNotesView();
                } else {
                    alert('Error: ' + data.errorMessage);
                }
            } catch (error) {
                console.error('Error generating release notes:', error);
                alert('Failed to generate release notes');
            }
            
            hideLoading();
        }

        function updateReleaseNotesView() {
            if (currentView === 'html') {
                releaseNotesBody.innerHTML = currentReleaseNotes.html;
            } else {
                releaseNotesBody.innerHTML = '<pre>' + escapeHtml(currentReleaseNotes.text) + '</pre>';
            }
        }

        function showLoading(text) {
            loadingText.textContent = text;
            loadingOverlay.classList.add('active');
        }

        function hideLoading() {
            loadingOverlay.classList.remove('active');
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
    </script>
</body>
</html>
`

