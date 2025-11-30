package gooseproj

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	input := `
goose-version = "v0.1.0"
go-path = "./go"
pkg-patterns = ["github.com/example/pkg"]
rocq = "src/program_proof"
`
	r := strings.NewReader(input)
	cfg, err := Parse(r)
	require.NoError(t, err)
	assert.Equal(t, "v0.1.0", cfg.GooseVersion)
	assert.Equal(t, "./go", cfg.GoPath)
	assert.Equal(t, []string{"github.com/example/pkg"}, cfg.PkgPatterns)
	assert.Equal(t, "src/program_proof", cfg.RocqRoot)
}

func TestNormalizeDefaults(t *testing.T) {
	cfg := &GooseConfig{
		GoPath: ".", // Set GoPath to avoid walkdir
	}
	err := cfg.normalize()
	require.NoError(t, err)
	assert.Equal(t, "latest", cfg.GooseVersion)
	assert.Equal(t, "src", cfg.RocqRoot)
	assert.Equal(t, []string{"./..."}, cfg.PkgPatterns)
}
