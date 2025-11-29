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

func TestGetLatestCommit_WithoutGitPrefix(t *testing.T) {
	// Test with URL without git+ prefix
	commit, err := GetLatestCommit("https://github.com/mit-pdos/perennial")
	require.NoError(t, err)
	assert.Len(t, commit, 15)
}

func TestGetIndirectDependencies_KnownPackage(t *testing.T) {
	// Test with a package known to not have pin-depends
	deps, err := GetIndirectDependencies("coq-record-update", "git+https://github.com/tchajed/coq-record-update")
	require.NoError(t, err)
	assert.Nil(t, deps)
}

// Note: Live test for GetIndirectDependencies is not included because it requires
// a real repository with an opam file containing pin-depends. The perennial repository
// does not have an opam file in its source tree. To test this functionality:
// 1. Find a repository with an opam file that has pin-depends
// 2. Create a test using that repository
// 3. The function will fetch the opam file, parse it, and return direct pin-depends

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
