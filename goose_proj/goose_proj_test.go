package gooseproj

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	input := `
go_path = "./go"
packages = ["github.com/example/pkg"]
rocq = "src/program_proof"
`
	r := strings.NewReader(input)
	cfg, err := Parse(r)
	require.NoError(t, err)
	assert.Equal(t, "./go", cfg.GoPath)
	assert.Equal(t, []string{"github.com/example/pkg"}, cfg.PkgPatterns)
	assert.Equal(t, "src/program_proof", cfg.RocqRoot)
}

func TestParseWithDefaults(t *testing.T) {
	// Minimal config with only go-path set
	input := `
go_path = "."
`
	r := strings.NewReader(input)
	cfg, err := Parse(r)
	require.NoError(t, err)

	// Verify defaults were applied
	assert.Equal(t, ".", cfg.GoPath)
	assert.Equal(t, "src", cfg.RocqRoot)
	assert.Equal(t, []string{"./..."}, cfg.PkgPatterns)
}

func TestParseRejectsUnknownFields(t *testing.T) {
	input := `
go_path = "."
unknown_field = "value"
`
	r := strings.NewReader(input)
	_, err := Parse(r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "strict mode")
}
