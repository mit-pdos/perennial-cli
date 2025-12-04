package opam

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"slices"
	"strings"
)

// packagesWithoutPinDepends is a list of packages known to not have pin-depends
// (so they can be skipped in checking for indirect dependencies)
var packagesWithoutPinDepends = map[string]bool{
	"coq-record-update": true,
	"rocq-stdpp":        true,
	"rocq-iris":         true,
	"iris-named-props":  true,
}

// GetLatestCommit returns the latest commit hash from a git URL.
//
// Trims the commit hash to HASH_ABBREV_LENGTH characters.
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

	commit := abbreviateHash(parts[0])

	return commit, nil
}

// fetchOpamFile fetches an opam file from a URL at a specific commit.
// The URL should be a git repository URL (with or without git+ prefix).
// It constructs the raw file URL based on the repository host.
func fetchOpamFile(gitURL, packageName, commit string) ([]byte, error) {
	// Strip git+ prefix if present
	url := strings.TrimPrefix(gitURL, "git+")

	// Convert git URL to raw file URL
	var rawURL string
	if strings.Contains(url, "github.com") {
		// GitHub: https://github.com/user/repo -> https://raw.githubusercontent.com/user/repo/commit/package.opam
		url = strings.TrimSuffix(url, ".git")
		url = strings.Replace(url, "github.com", "raw.githubusercontent.com", 1)
		rawURL = fmt.Sprintf("%s/%s/%s.opam", url, commit, packageName)
	} else if strings.Contains(url, "gitlab") {
		// GitLab: https://gitlab.com/user/repo -> https://gitlab.com/user/repo/-/raw/commit/package.opam
		url = strings.TrimSuffix(url, ".git")
		rawURL = fmt.Sprintf("%s/-/raw/%s/%s.opam", url, commit, packageName)
	} else {
		return nil, fmt.Errorf("unsupported git hosting service: %s", url)
	}

	resp, err := http.Get(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch opam file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch opam file: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read opam file: %w", err)
	}

	return data, nil
}

// FindOpamPackage tries to find the unique opam package in a repository at a specific commit.
// Returns the package name (without .opam extension) if a unique opam file is found.
// Uses the GitHub/GitLab API to list directory contents.
func FindOpamPackage(gitURL, commit string) (string, error) {
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
			return "", fmt.Errorf("invalid GitLab URL format: %s", url)
		}
		domain := parts[0] + "//" + parts[2]
		projectPath := strings.ReplaceAll(parts[3], "/", "%2F")
		apiURL = fmt.Sprintf("%s/api/v4/projects/%s/repository/tree?ref=%s", domain, projectPath, commit)
	} else {
		return "", fmt.Errorf("unsupported git hosting service: %s", url)
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch repository listing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch repository listing: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read repository listing: %w", err)
	}

	// Look for .opam files in the response
	// This is a simple string search rather than full JSON parsing
	content := string(body)
	var opamFiles []string

	// Extract file names (works for both GitHub and GitLab JSON responses)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.Contains(line, ".opam\"") {
			// Extract the filename from JSON
			// Look for patterns like "name": "package.opam" or "path":"package.opam"
			if idx := strings.Index(line, "\"name\""); idx >= 0 {
				line = line[idx:]
			} else if idx := strings.Index(line, "\"path\""); idx >= 0 {
				line = line[idx:]
			}

			// Extract the quoted filename
			start := strings.Index(line, "\"") + 1
			line = line[start:]
			start = strings.Index(line, "\"") + 1
			line = line[start:]
			end := strings.Index(line, "\"")
			if end > 0 {
				filename := line[:end]
				if strings.HasSuffix(filename, ".opam") && !strings.Contains(filename, "/") {
					packageName := strings.TrimSuffix(filename, ".opam")
					if !slices.Contains(opamFiles, packageName) {
						opamFiles = append(opamFiles, packageName)
					}
				}
			}
		}
	}

	if len(opamFiles) == 0 {
		return "", fmt.Errorf("no opam files found in repository")
	}
	if len(opamFiles) > 1 {
		return "", fmt.Errorf("multiple opam files found in repository: %v", opamFiles)
	}

	return opamFiles[0], nil
}

// FetchDependencies fetches the (transitive) dependencies of a package.
// It fetches the package's opam file at the specified git commit and returns
// its pin-depends.
func (dep *PinDepend) FetchDependencies() ([]PinDepend, error) {
	// Check if this package is known to not have pin-depends
	if packagesWithoutPinDepends[dep.Package] {
		return nil, nil
	}

	// Fetch the opam file at the specific commit
	data, err := fetchOpamFile(dep.URL, dep.Package, dep.Commit)
	if err != nil {
		return nil, err
	}

	// Parse the opam file
	opamFile, err := Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse opam file: %w", err)
	}

	deps := append(opamFile.GetPinDepends(), opamFile.GetIndirect()...)
	return deps, nil
}

func (f *OpamFile) UpdateIndirectDependencies() error {
	seen := make(map[string]bool)
	indirects := []PinDepend{}
	for _, dep := range f.GetPinDepends() {
		newIndirects, err := dep.FetchDependencies()
		if err != nil {
			return err
		}
		for _, newDep := range newIndirects {
			if !seen[newDep.Package] {
				indirects = append(indirects, newDep)
				seen[newDep.Package] = true
			}
		}
	}
	slices.SortFunc(indirects, func(a, b PinDepend) int {
		if a.Package < b.Package {
			return -1
		} else if a.Package > b.Package {
			return 1
		}
		return 0
	})
	f.SetIndirect(indirects)
	return nil
}
