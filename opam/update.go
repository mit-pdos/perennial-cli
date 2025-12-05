package opam

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/mit-pdos/perennial-cli/git"
)

// packagesWithoutPinDepends is a list of packages known to not have pin-depends
// (so they can be skipped in checking for indirect dependencies)
var packagesWithoutPinDepends = map[string]bool{
	"coq-record-update": true,
	"rocq-stdpp":        true,
	"rocq-iris":         true,
	"iris-named-props":  true,
}

// fetchOpamFile fetches an opam file from a URL at a specific commit.
// The URL should be a git repository URL (with or without git+ prefix).
func fetchOpamFile(gitURL, packageName, commit string) ([]byte, error) {
	path := packageName + ".opam"
	data, err := git.GetFile(gitURL, commit, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch opam file: %w", err)
	}
	return data, nil
}

// FindOpamPackage tries to find the unique opam package in a repository at a specific commit.
// Returns the package name (without .opam extension) if a unique opam file is found.
func FindOpamPackage(gitURL, commit string) (string, error) {
	files, err := git.ListFiles(gitURL, commit)
	if err != nil {
		return "", err
	}

	// Look for .opam files
	var opamFiles []string
	for _, filename := range files {
		if strings.HasSuffix(filename, ".opam") {
			packageName := strings.TrimSuffix(filename, ".opam")
			opamFiles = append(opamFiles, packageName)
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

// UpdateIndirectDependencies updates the indirect dependencies of an opam file.
//
// It returns true if the indirect dependencies were updated, false otherwise.
func (f *OpamFile) UpdateIndirectDependencies() (bool, error) {
	seen := make(map[string]bool)
	oldIndirects := f.GetIndirect()
	indirects := []PinDepend{}
	for _, dep := range f.GetPinDepends() {
		newIndirects, err := dep.FetchDependencies()
		if err != nil {
			return false, err
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
	return !slices.Equal(oldIndirects, indirects), nil
}
