package depgraph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRocqdep(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	rocqdepFile := filepath.Join(tmpDir, "test.rocqdeps.d")

	// Write test data to the file
	testData := `src/proof/example.vo src/proof/example.glob: src/proof/example.v src/base.vo /usr/lib/rocqworker
src/proof/example.vos: src/proof/example.v src/base.vos /usr/lib/rocqworker
src/base.vo: src/base.v /usr/lib/rocqworker
`

	err := os.WriteFile(rocqdepFile, []byte(testData), 0644)
	require.NoError(t, err)

	// Parse the file
	g, err := ParseRocqdep(rocqdepFile)
	require.NoError(t, err)

	// Verify the graph only contains .vo and .v files (filtered)
	for _, dep := range g.allDeps() {
		assert.True(t,
			strings.HasSuffix(dep.Target, ".vo") || strings.HasSuffix(dep.Target, ".v"),
			"target should be .vo or .v: %s", dep.Target)
		assert.True(t,
			strings.HasSuffix(dep.Source, ".vo") || strings.HasSuffix(dep.Source, ".v"),
			"source should be .vo or .v: %s", dep.Source)
	}

	// Check specific dependencies
	assert.Contains(t, g.allDeps(), Dep{
		Target: "src/proof/example.vo",
		Source: "src/proof/example.v",
	})
	assert.Contains(t, g.allDeps(), Dep{
		Target: "src/proof/example.vo",
		Source: "src/base.vo",
	})
	assert.Contains(t, g.allDeps(), Dep{
		Target: "src/base.vo",
		Source: "src/base.v",
	})

	// Should not contain dependencies on rocqworker (filtered out)
	for _, dep := range g.allDeps() {
		assert.NotContains(t, dep.Source, "rocqworker")
	}
}

func TestParseRocqdepNonexistentFile(t *testing.T) {
	_, err := ParseRocqdep("/nonexistent/file.d")
	assert.Error(t, err)
}

func TestRocqDeps(t *testing.T) {
	// Create a test graph
	// A.vo depends on A.v and B.vo
	// B.vo depends on B.v and C.vo
	// C.vo depends on C.v
	testData := `A.vo: A.v B.vo
B.vo: B.v C.vo
C.vo: C.v
`

	g, err := Parse(strings.NewReader(testData))
	require.NoError(t, err)
	filterRocq(g)

	// Test with .vo file
	sources := RocqDeps(g, []string{"A.vo"})
	assert.Equal(t, []string{"A.v", "B.v", "C.v"}, sources)

	// Test with .v file (should convert to .vo)
	sources = RocqDeps(g, []string{"A.v"})
	assert.Equal(t, []string{"A.v", "B.v", "C.v"}, sources)

	// Test with direct dependency
	sources = RocqDeps(g, []string{"B.vo"})
	assert.Equal(t, []string{"B.v", "C.v"}, sources)

	// Test with leaf node
	sources = RocqDeps(g, []string{"C.vo"})
	assert.Equal(t, []string{"C.v"}, sources)
}

func TestRocqDepsWithMixedExtensions(t *testing.T) {
	testData := `A.vo: A.v B.vo
B.vo: B.v
`

	g, err := Parse(strings.NewReader(testData))
	require.NoError(t, err)
	filterRocq(g)

	// Test with mix of .v and .vo arguments
	sources := RocqDeps(g, []string{"A.v", "B.vo"})
	assert.ElementsMatch(t, []string{"A.v", "B.v"}, sources)
}

func TestRocqDepsNoDependencies(t *testing.T) {
	testData := `A.vo: A.v
`

	g, err := Parse(strings.NewReader(testData))
	require.NoError(t, err)
	filterRocq(g)

	// Test with file that has no dependencies (just itself)
	sources := RocqDeps(g, []string{"A.vo"})
	assert.Equal(t, []string{"A.v"}, sources)
}

func TestRocqDepsDiamond(t *testing.T) {
	// Test diamond dependency pattern:
	//     A
	//    / \
	//   B   C
	//    \ /
	//     D
	testData := `A.vo: A.v B.vo C.vo
B.vo: B.v D.vo
C.vo: C.v D.vo
D.vo: D.v
`

	g, err := Parse(strings.NewReader(testData))
	require.NoError(t, err)
	filterRocq(g)

	sources := RocqDeps(g, []string{"A.vo"})
	// D should only appear once despite being reachable via both B and C
	assert.ElementsMatch(t, []string{"A.v", "B.v", "C.v", "D.v"}, sources)
}
