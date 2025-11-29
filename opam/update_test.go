package opam

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestCommit(t *testing.T) {
	// Test with a real repository (this is a live test)
	commit, err := GetLatestCommit("git+https://github.com/mit-pdos/perennial")
	require.NoError(t, err)

	// Commit should be exactly 15 characters (normalized)
	assert.Len(t, commit, 15)

	// Commit should be a valid hex string
	for _, c := range commit {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"commit hash should only contain hex characters")
	}
}

func TestGetDependencies_KnownPackage(t *testing.T) {
	// Test with a package known to not have pin-depends (shouldn't trigger an HTTP request)
	deps, err := GetDependencies("coq-record-update", "git+https://github.com/tchajed/coq-record-update")
	require.NoError(t, err)
	assert.Nil(t, deps)
}

func TestGetDependencies(t *testing.T) {
	// Test with perennial-example-proof repository (this is a live test)
	deps, err := GetDependencies("example-proof", "git+https://github.com/tchajed/perennial-example-proof")
	require.NoError(t, err)

	// The function should return only direct pin-depends (excluding indirect dependencies)
	assert.NotNil(t, deps)

	// Verify that we got some dependencies
	assert.Greater(t, len(deps), 0, "perennial-example-proof should have at least one direct pin-depend")

	// Check that all returned dependencies have required fields
	for _, dep := range deps {
		assert.NotEmpty(t, dep.Package, "package name should not be empty")
		assert.NotEmpty(t, dep.URL, "URL should not be empty")
		assert.NotEmpty(t, dep.Commit, "commit should not be empty")

		// Commit should be exactly 15 characters (normalized)
		assert.Len(t, dep.Commit, 15, "commit hash should be normalized to 15 characters")
	}
}

func TestPackagesWithoutPinDepends(t *testing.T) {
	knownPackages := []string{
		"coq-record-update",
		"rocq-stdpp",
		"rocq-iris",
		"iris-named-props",
	}

	for _, pkg := range knownPackages {
		assert.True(t, packagesWithoutPinDepends[pkg],
			"package %s should be in packagesWithoutPinDepends", pkg)
	}
}
