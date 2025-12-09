package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLatestCommit(t *testing.T) {
	// Test with a real repository (this is a live test)
	commit, err := GetLatestCommit("https://github.com/mit-pdos/perennial")
	require.NoError(t, err)

	// Commit should be 40 characters (full hash)
	assert.Len(t, commit, 40)

	// Commit should be a valid hex string
	for _, c := range commit {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"commit hash should only contain hex characters")
	}
}

func TestListFiles(t *testing.T) {
	// Test with perennial repository (this is a live test)
	// List files at the root
	files, err := ListFiles("https://github.com/mit-pdos/perennial", "eb8dbfceb7a15fddf623bf526a70a694918987b2")
	require.NoError(t, err)

	// Should have some files
	assert.Greater(t, len(files), 0, "repository should have files at root")

	// Should have perennial.opam file
	assert.Contains(t, files, "perennial.opam", "should find perennial.opam file")

	// Files should not contain subdirectories (no slashes)
	for _, file := range files {
		assert.NotContains(t, file, "/", "file should not contain slashes (no subdirectories)")
		assert.NotEmpty(t, file, "file name should not be empty")
	}
}
