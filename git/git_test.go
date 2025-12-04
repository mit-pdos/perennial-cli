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

	// Commit should be 10 characters (our truncation)
	assert.Len(t, commit, 10)

	// Commit should be a valid hex string
	for _, c := range commit {
		assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
			"commit hash should only contain hex characters")
	}
}
