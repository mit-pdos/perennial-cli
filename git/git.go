// git has functions for interacting with git remotes
package git

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// GetLatestCommit returns the latest commit hash from a git URL.
//
// Returns the full 40-character commit hash.
// Uses the GitHub/GitLab API to avoid requiring git credentials.
func GetLatestCommit(gitURL string) (string, error) {
	url := strings.TrimPrefix(gitURL, "git+")
	url = strings.TrimSuffix(url, ".git")

	if strings.Contains(url, "github.com") {
		return getLatestCommitGitHub(url)
	} else if strings.Contains(url, "gitlab") {
		return getLatestCommitGitLab(url)
	}
	return "", fmt.Errorf("unsupported git hosting service: %s", url)
}

func getLatestCommitGitHub(url string) (string, error) {
	repo := strings.Replace(url, "https://github.com/", "", 1)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/commits/HEAD", repo)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest commit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest commit: status %d", resp.StatusCode)
	}

	var result struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	return result.SHA, nil
}

func getLatestCommitGitLab(url string) (string, error) {
	parts := strings.SplitN(url, "/", 4)
	if len(parts) < 4 {
		return "", fmt.Errorf("invalid GitLab URL format: %s", url)
	}
	domain := parts[0] + "//" + parts[2]
	projectPath := strings.ReplaceAll(parts[3], "/", "%2F")
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits/HEAD", domain, projectPath)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest commit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest commit: status %d", resp.StatusCode)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse GitLab API response: %w", err)
	}

	return result.ID, nil
}

// ResolveCommit resolves an abbreviated commit hash to a full hash.
// If the commit is already a full hash (40 characters), it returns it unchanged.
// Uses the GitHub/GitLab API to resolve the hash.
func ResolveCommit(gitURL, commit string) (string, error) {
	// If already a full hash, return as-is
	if len(commit) == 40 {
		return commit, nil
	}

	url := strings.TrimPrefix(gitURL, "git+")
	url = strings.TrimSuffix(url, ".git")

	if strings.Contains(url, "github.com") {
		return resolveCommitGitHub(url, commit)
	} else if strings.Contains(url, "gitlab") {
		return resolveCommitGitLab(url, commit)
	}
	return "", fmt.Errorf("unsupported git hosting service: %s", url)
}

func resolveCommitGitHub(url, commit string) (string, error) {
	// GitHub API: https://api.github.com/repos/user/repo/commits/sha
	url = strings.Replace(url, "https://github.com/", "", 1)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", url, commit)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch commit info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch commit info: status %d", resp.StatusCode)
	}

	// Parse GitHub API response
	var result struct {
		SHA string `json:"sha"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	return result.SHA, nil
}

func resolveCommitGitLab(url, commit string) (string, error) {
	// GitLab API: https://gitlab.com/api/v4/projects/user%2Frepo/repository/commits/sha
	parts := strings.SplitN(url, "/", 4)
	if len(parts) < 4 {
		return "", fmt.Errorf("invalid GitLab URL format: %s", url)
	}
	domain := parts[0] + "//" + parts[2]
	projectPath := strings.ReplaceAll(parts[3], "/", "%2F")
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/commits/%s", domain, projectPath, commit)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch commit info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch commit info: status %d", resp.StatusCode)
	}

	// Parse GitLab API response
	var result struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse GitLab API response: %w", err)
	}

	return result.ID, nil
}

// ListFiles returns a list of files at the root of a git repository at a specific commit.
// Uses the GitHub/GitLab API to list directory contents.
func ListFiles(gitURL, commit string) ([]string, error) {
	url := strings.TrimPrefix(gitURL, "git+")
	url = strings.TrimSuffix(url, ".git")

	if strings.Contains(url, "github.com") {
		return listFilesGitHub(url, commit)
	} else if strings.Contains(url, "gitlab") {
		return listFilesGitLab(url, commit)
	}
	return nil, fmt.Errorf("unsupported git hosting service: %s", url)
}

func listFilesGitHub(url, commit string) ([]string, error) {
	// GitHub API: https://api.github.com/repos/user/repo/contents?ref=commit
	url = strings.Replace(url, "https://github.com/", "", 1)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/contents?ref=%s", url, commit)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository listing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch repository listing: status %d", resp.StatusCode)
	}

	// Parse GitHub API response (array of objects with "name", "type", etc.)
	var entries []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Path string `json:"path"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	var files []string
	for _, entry := range entries {
		// Only include files (not directories) at the root
		if entry.Type == "file" && !strings.Contains(entry.Path, "/") {
			files = append(files, entry.Name)
		}
	}

	return files, nil
}

func listFilesGitLab(url, commit string) ([]string, error) {
	// GitLab API: https://gitlab.com/api/v4/projects/user%2Frepo/repository/tree?ref=commit
	parts := strings.SplitN(url, "/", 4)
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid GitLab URL format: %s", url)
	}
	domain := parts[0] + "//" + parts[2]
	projectPath := strings.ReplaceAll(parts[3], "/", "%2F")
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/repository/tree?ref=%s", domain, projectPath, commit)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository listing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch repository listing: status %d", resp.StatusCode)
	}

	// Parse GitLab API response (array of objects with "name", "type", "path")
	var entries []struct {
		Name string `json:"name"`
		Type string `json:"type"`
		Path string `json:"path"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to parse GitLab API response: %w", err)
	}

	var files []string
	for _, entry := range entries {
		// Only include files (blobs) at the root
		if entry.Type == "blob" && !strings.Contains(entry.Path, "/") {
			files = append(files, entry.Name)
		}
	}

	return files, nil
}

// CompareURL returns a URL to view the diff between two commits.
// Supports GitHub and GitLab repositories.
func CompareURL(url, fromCommit, toCommit string) string {
	if strings.Contains(url, "github.com") {
		return fmt.Sprintf("%s/compare/%s...%s", url, fromCommit, toCommit)
	} else if strings.Contains(url, "gitlab") {
		return fmt.Sprintf("%s/-/compare/%s...%s", url, fromCommit, toCommit)
	}
	return ""
}

// GetFile fetches a file from a git repository at a specific commit.
// Works with GitHub and GitLab repositories.
func GetFile(gitURL, commit, path string) ([]byte, error) {
	url := strings.TrimPrefix(gitURL, "git+")
	url = strings.TrimSuffix(url, ".git")

	var rawURL string
	if strings.Contains(url, "github.com") {
		// GitHub: https://github.com/user/repo -> https://raw.githubusercontent.com/user/repo/commit/path
		url = strings.Replace(url, "github.com", "raw.githubusercontent.com", 1)
		rawURL = fmt.Sprintf("%s/%s/%s", url, commit, path)
	} else if strings.Contains(url, "gitlab") {
		// GitLab: https://gitlab.com/user/repo -> https://gitlab.com/user/repo/-/raw/commit/path
		rawURL = fmt.Sprintf("%s/-/raw/%s/%s", url, commit, path)
	} else {
		return nil, fmt.Errorf("unsupported git hosting service: %s", url)
	}

	resp, err := http.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch file: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}
