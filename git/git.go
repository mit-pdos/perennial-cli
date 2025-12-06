package git

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// GetLatestCommit returns the latest commit hash from a git URL.
//
// Trims the commit hash to 10 characters.
func GetLatestCommit(gitURL string) (string, error) {
	if strings.HasPrefix(gitURL, "https://gitlab") {
		// avoid a redirect warning
		if !strings.HasSuffix(gitURL, ".git") {
			gitURL = gitURL + ".git"
		}
	}
	cmd := exec.Command("git", "ls-remote", gitURL, "HEAD")
	cmd.Stderr = os.Stderr
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
