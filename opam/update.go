package opam

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os/exec"
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
// Trims the commit hash to the first 15 characters.
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
	// Normalize to first 15 characters
	if len(commit) > 15 {
		commit = commit[:15]
	}

	return commit, nil
}

// fetchOpamFile fetches an opam file from a URL.
// The URL should be a git repository URL (with or without git+ prefix).
// It constructs the raw file URL based on the repository host.
func fetchOpamFile(gitURL, packageName string) ([]byte, error) {
	// Strip git+ prefix if present
	url := strings.TrimPrefix(gitURL, "git+")

	// Convert git URL to raw file URL
	var rawURL string
	if strings.Contains(url, "github.com") {
		// GitHub: https://github.com/user/repo -> https://raw.githubusercontent.com/user/repo/master/package.opam
		url = strings.TrimSuffix(url, ".git")
		url = strings.Replace(url, "github.com", "raw.githubusercontent.com", 1)
		rawURL = fmt.Sprintf("%s/master/%s.opam", url, packageName)
	} else if strings.Contains(url, "gitlab") {
		// GitLab: https://gitlab.com/user/repo -> https://gitlab.com/user/repo/-/raw/master/package.opam
		url = strings.TrimSuffix(url, ".git")
		rawURL = fmt.Sprintf("%s/-/raw/master/%s.opam", url, packageName)
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

// GetDependencies fetches the (transitive) dependencies of a package.
// It fetches the package's opam file from the given git URL and returns
// its pin-depends.
func GetDependencies(packageName, gitURL string) ([]PinDepend, error) {
	// Check if this package is known to not have pin-depends
	if packagesWithoutPinDepends[packageName] {
		return nil, nil
	}

	// Fetch the opam file
	data, err := fetchOpamFile(gitURL, packageName)
	if err != nil {
		return nil, err
	}

	// Parse the opam file
	opamFile, err := Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse opam file: %w", err)
	}

	deps := append(opamFile.ListPinDepends(), opamFile.GetIndirect()...)
	return deps, nil
}
