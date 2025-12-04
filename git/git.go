package git

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"slices"
	"strings"
)

// GetLatestCommit returns the latest commit hash from a git URL.
//
// Trims the commit hash to 10 characters.
func GetLatestCommit(gitURL string) (string, error) {
	// Strip git+ prefix if present
	url := strings.TrimPrefix(gitURL, "git+")

	// Use git ls-remote to get the latest commit on the default branch
	cmd := exec.Command("git", "ls-remote", url, "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run git ls-remote: %w", err)
	}

	// Output format: "commit_hash\tHEAD"
	parts := strings.Fields(string(output))
	if len(parts) < 1 {
		return "", fmt.Errorf("unexpected git ls-remote output: %s", output)
	}

	commit := parts[0]
	if len(commit) > 10 {
		commit = commit[:10]
	}

	return commit, nil
}

// ListFiles returns a list of files at the root of a git repository at a specific commit.
// Uses the GitHub/GitLab API to list directory contents.
func ListFiles(gitURL, commit string) ([]string, error) {
	url := strings.TrimPrefix(gitURL, "git+")
	url = strings.TrimSuffix(url, ".git")

	var apiURL string
	if strings.Contains(url, "github.com") {
		// GitHub API: https://api.github.com/repos/user/repo/contents?ref=commit
		url = strings.Replace(url, "https://github.com/", "", 1)
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/contents?ref=%s", url, commit)
	} else if strings.Contains(url, "gitlab") {
		// GitLab API: https://gitlab.com/api/v4/projects/user%2Frepo/repository/tree?ref=commit
		// Extract the path after the domain
		parts := strings.SplitN(url, "/", 4)
		if len(parts) < 4 {
			return nil, fmt.Errorf("invalid GitLab URL format: %s", url)
		}
		domain := parts[0] + "//" + parts[2]
		projectPath := strings.ReplaceAll(parts[3], "/", "%2F")
		apiURL = fmt.Sprintf("%s/api/v4/projects/%s/repository/tree?ref=%s", domain, projectPath, commit)
	} else {
		return nil, fmt.Errorf("unsupported git hosting service: %s", url)
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository listing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch repository listing: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read repository listing: %w", err)
	}

	// Look for files in the response
	// This is a simple string search rather than full JSON parsing
	content := string(body)
	var files []string

	// Extract file names (works for both GitHub and GitLab JSON responses)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Look for patterns like "name": "filename" or "path":"filename"
		if idx := strings.Index(line, "\"name\""); idx >= 0 {
			line = line[idx:]
		} else if idx := strings.Index(line, "\"path\""); idx >= 0 {
			line = line[idx:]
		} else {
			continue
		}

		// Extract the quoted filename
		start := strings.Index(line, "\"") + 1
		if start < 1 {
			continue
		}
		line = line[start:]
		start = strings.Index(line, "\"") + 1
		if start < 1 {
			continue
		}
		line = line[start:]
		end := strings.Index(line, "\"")
		if end > 0 {
			filename := line[:end]
			// Only include files at the root (no subdirectories)
			if !strings.Contains(filename, "/") && filename != "" {
				if !slices.Contains(files, filename) {
					files = append(files, filename)
				}
			}
		}
	}

	return files, nil
}

// GetRawFileURL constructs a URL to fetch a raw file from a git repository at a specific commit.
// Works with GitHub and GitLab repositories.
func GetRawFileURL(gitURL, commit, filename string) (string, error) {
	url := strings.TrimPrefix(gitURL, "git+")
	url = strings.TrimSuffix(url, ".git")

	var rawURL string
	if strings.Contains(url, "github.com") {
		// GitHub: https://github.com/user/repo -> https://raw.githubusercontent.com/user/repo/commit/filename
		url = strings.Replace(url, "github.com", "raw.githubusercontent.com", 1)
		rawURL = fmt.Sprintf("%s/%s/%s", url, commit, filename)
	} else if strings.Contains(url, "gitlab") {
		// GitLab: https://gitlab.com/user/repo -> https://gitlab.com/user/repo/-/raw/commit/filename
		rawURL = fmt.Sprintf("%s/-/raw/%s/%s", url, commit, filename)
	} else {
		return "", fmt.Errorf("unsupported git hosting service: %s", url)
	}

	return rawURL, nil
}

