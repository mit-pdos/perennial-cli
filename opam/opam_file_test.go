package opam

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const exampleOpam = `opam-version: "2.0"
license: "MIT"
maintainer: "Tej Chajed"
authors: "Tej Chajed"
homepage: "https://github.com/tchajed/perennial-example-proof"
bug-reports: "https://github.com/tchajed/perennial-example-proof"
dev-repo: "git+https://github.com/tchajed/perennial-example-proof.git"
version: "dev"
synopsis: "A test of perennial as a dependency"

depends: [
  "perennial"
  "coq-record-update" { (>= "0.3.6") }
]

pin-depends: [
  ["perennial.dev"           "git+https://github.com/mit-pdos/perennial#577140b0594fbdea1a2b3c4d5e6f7a8b9c0d1e2f"]

  ## begin indirect
  ["rocq-stdpp.dev"          "git+https://gitlab.mpi-sws.org/iris/stdpp#187909f0c1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6"]
  ["rocq-iris.dev"           "git+https://gitlab.mpi-sws.org/iris/iris#fde0f86992a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5"]
  ["iris-named-props.dev"    "git+https://github.com/tchajed/iris-named-props#c388714a93b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5"]
  ## end
]

build: [make "-j%{jobs}%"]
install: ["./etc/install.sh"]
`

// parseString is a helper function to parse an opam file from a string
func parseString(t *testing.T, content string) *OpamFile {
	r := strings.NewReader(content)
	f, err := Parse(r)
	require.NoError(t, err)
	return f
}

func TestParse(t *testing.T) {
	f := parseString(t, exampleOpam)

	// Check depends region
	assert.False(t, f.depends.empty(), "depends region not found")
	assert.Equal(t, 10, f.depends.startLine)
	assert.Equal(t, 14, f.depends.endLine)

	// Check pin-depends region
	assert.False(t, f.pinDepends.empty(), "pin-depends region not found")
	assert.Equal(t, 15, f.pinDepends.startLine)
	assert.Equal(t, 24, f.pinDepends.endLine)

	// Check indirect region
	assert.False(t, f.indirectPinDepends.empty(), "indirect pin-depends region not found")
	assert.Equal(t, 18, f.indirectPinDepends.startLine)
	assert.Equal(t, 23, f.indirectPinDepends.endLine)
}

func TestParse_AddMissingBlocks_Empty(t *testing.T) {
	// Test parsing a minimal opam file with no depends or pin-depends
	minimalOpam := `opam-version: "2.0"
version: "dev"
`
	f := parseString(t, minimalOpam)

	// Both depends and pin-depends should have been added
	assert.False(t, f.depends.empty(), "depends block should be added")
	assert.False(t, f.pinDepends.empty(), "pin-depends block should be added")

	// Verify the structure
	output := f.String()
	assert.Contains(t, output, "depends: [")
	assert.Contains(t, output, "pin-depends: [")
}

func TestParse_AddMissingBlocks_NoPinDepends(t *testing.T) {
	// Test parsing an opam file with depends but no pin-depends
	opamWithDepends := `opam-version: "2.0"
version: "dev"

depends: [
  "coq"
]
`
	f := parseString(t, opamWithDepends)

	// depends should exist, pin-depends should have been added
	assert.False(t, f.depends.empty(), "depends block should exist")
	assert.False(t, f.pinDepends.empty(), "pin-depends block should be added")

	// Verify the structure
	output := f.String()
	assert.Contains(t, output, "depends: [")
	assert.Contains(t, output, "pin-depends: [")
}

func TestParse_OneLineDepends(t *testing.T) {
	opamWithOneLinePinDepends := `opam-version: "2.0"
version: "dev"

depends: []
pin-depends: [
]
`
	r := strings.NewReader(opamWithOneLinePinDepends)
	_, err := Parse(r)
	// TODO: should fix this by separating depends into two lines
	require.Error(t, err, "depends: [] is not currently supported")
}

func TestParse_OneLinePinDepends(t *testing.T) {
	opamWithOneLinePinDepends := `opam-version: "2.0"
version: "dev"

depends: [
  "coq"
]
pin-depends: []
`
	r := strings.NewReader(opamWithOneLinePinDepends)
	_, err := Parse(r)
	// TODO: should fix this by separating pin-depends into two lines
	require.Error(t, err, "pin-depends: [] is not currently supported")
}

func TestListPinDepends(t *testing.T) {
	f := parseString(t, exampleOpam)

	deps := f.GetPinDepends()
	// Should only return direct dependencies (excluding indirect section)
	require.Len(t, deps, 1)

	// Check the direct dependency
	assert.Equal(t, "perennial", deps[0].Package)
	assert.Equal(t, "git+https://github.com/mit-pdos/perennial", deps[0].URL)
	assert.Equal(t, "577140b0594fbdea1a2b3c4d5e6f7a8b9c0d1e2f", deps[0].Commit)
}

func TestGetIndirect(t *testing.T) {
	f := parseString(t, exampleOpam)

	indirect := f.GetIndirect()
	require.Len(t, indirect, 3)

	// Check that we only got indirect dependencies
	assert.Equal(t, "rocq-stdpp", indirect[0].Package)
	assert.Equal(t, "rocq-iris", indirect[1].Package)
	assert.Equal(t, "iris-named-props", indirect[2].Package)
}

func TestAddPinDepend_Update(t *testing.T) {
	f := parseString(t, exampleOpam)

	// Update existing dependency
	f.AddPinDepend(PinDepend{
		Package: "perennial",
		URL:     "git+https://github.com/mit-pdos/perennial",
		Commit:  "newcommit1",
	})

	deps := f.GetPinDepends()
	found := false
	for _, dep := range deps {
		if dep.Package == "perennial" {
			found = true
			assert.Equal(t, "newcommit1", dep.Commit)
		}
	}
	assert.True(t, found, "perennial not found after update")
}

func TestAddPinDepend_Add(t *testing.T) {
	f := parseString(t, exampleOpam)

	// Add new dependency
	f.AddPinDepend(PinDepend{
		Package: "new-package",
		URL:     "git+https://example.com/package",
		Commit:  "abc123",
	})

	deps := f.GetPinDepends()
	found := false
	for _, dep := range deps {
		if dep.Package == "new-package" {
			found = true
			assert.Equal(t, "git+https://example.com/package", dep.URL)
			assert.Equal(t, "abc123", dep.Commit)
		}
	}
	assert.True(t, found, "new-package not found after adding")
}

func TestSetIndirect(t *testing.T) {
	f := parseString(t, exampleOpam)

	// Replace indirect dependencies
	newIndirect := []PinDepend{
		{Package: "pkg1", URL: "git+https://example.com/pkg1", Commit: "commit1"},
		{Package: "pkg2", URL: "git+https://example.com/pkg2", Commit: "commit2"},
	}
	f.SetIndirect(newIndirect)

	indirect := f.GetIndirect()
	require.Len(t, indirect, 2)

	assert.Equal(t, "pkg1", indirect[0].Package)
	assert.Equal(t, "pkg2", indirect[1].Package)
}

func TestString(t *testing.T) {
	f := parseString(t, exampleOpam)

	output := f.String()
	assert.Equal(t, exampleOpam, output)
}

func TestParsePinDependLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want *PinDepend
	}{
		{
			name: "standard line with full hash",
			line: `  ["perennial.dev"           "git+https://github.com/mit-pdos/perennial#577140b0594fbdea1a2b3c4d5e6f7a8b9c0d1e2f"]`,
			want: &PinDepend{
				Package: "perennial",
				URL:     "git+https://github.com/mit-pdos/perennial",
				Commit:  "577140b0594fbdea1a2b3c4d5e6f7a8b9c0d1e2f",
			},
		},
		{
			name: "short hash",
			line: `  ["pkg.dev"   "git+https://example.com/repo#abc123def"]`,
			want: &PinDepend{
				Package: "pkg",
				URL:     "git+https://example.com/repo",
				Commit:  "abc123def",
			},
		},
		{
			name: "no commit hash",
			line: `  ["pkg.dev" "git+https://example.com/repo"]`,
			want: &PinDepend{
				Package: "pkg",
				URL:     "git+https://example.com/repo",
				Commit:  "",
			},
		},
		{
			name: "commented out",
			line: `#  ["pkg.dev" "git+https://example.com/repo#abc123"]`,
			want: nil,
		},
		{
			name: "comment line",
			line: "  ## begin indirect",
			want: nil,
		},
		{
			name: "empty line",
			line: "",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePinDependLine(tt.line)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.want.Package, got.Package)
			assert.Equal(t, tt.want.URL, got.URL)
			assert.Equal(t, tt.want.Commit, got.Commit)
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		errMsg  string
	}{
		{
			name: "unclosed depends block",
			content: `depends: [
  "perennial"
`,
			errMsg: "unclosed depends block",
		},
		{
			name: "unclosed pin-depends block",
			content: `pin-depends: [
  ["pkg.dev" "git+https://example.com"]
`,
			errMsg: "unclosed pin-depends block",
		},
		{
			name: "unclosed indirect region",
			content: `pin-depends: [
  ## begin indirect
  ["pkg.dev" "git+https://example.com"]
]`,
			errMsg: "unclosed indirect region",
		},
		{
			name: "end without begin",
			content: `pin-depends: [
  ["pkg.dev" "git+https://example.com"]
  ## end
]`,
			errMsg: "## end marker without ## begin indirect",
		},
		{
			name: "nested begin indirect",
			content: `pin-depends: [
  ## begin indirect
  ## begin indirect
  ["pkg.dev" "git+https://example.com"]
  ## end
]`,
			errMsg: "nested ## begin indirect markers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.content)
			_, err := Parse(r)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestFormatPinDependLine(t *testing.T) {
	tests := []struct {
		name string
		dep  PinDepend
		want string
	}{
		{
			name: "with full commit",
			dep: PinDepend{
				Package: "perennial",
				URL:     "git+https://github.com/mit-pdos/perennial",
				Commit:  "577140b0594fbdea1a2b3c4d5e6f7a8b9c0d1e2f",
			},
			want: `  ["perennial.dev"             "git+https://github.com/mit-pdos/perennial#577140b0594fbdea1a2b3c4d5e6f7a8b9c0d1e2f"]`,
		},
		{
			name: "without commit",
			dep: PinDepend{
				Package: "pkg",
				URL:     "git+https://example.com/repo",
				Commit:  "",
			},
			want: `  ["pkg.dev"                   "git+https://example.com/repo"]`,
		},
		{
			name: "long package name",
			dep: PinDepend{
				Package: "very-long-package-name",
				URL:     "git+https://example.com/repo",
				Commit:  "abc123def456",
			},
			want: `  ["very-long-package-name.dev" "git+https://example.com/repo#abc123def456"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dep.String())
		})
	}
}

func TestGetDependencies(t *testing.T) {
	f := parseString(t, exampleOpam)

	deps := f.GetDependencies()
	require.Len(t, deps, 2)

	assert.Equal(t, "perennial", deps[0])
	assert.Equal(t, "coq-record-update", deps[1])
}

func TestGetDependencies_Empty(t *testing.T) {
	// Test with minimal opam file with empty depends block
	minimalOpam := `opam-version: "2.0"
version: "dev"
`
	f := parseString(t, minimalOpam)

	deps := f.GetDependencies()
	assert.Empty(t, deps)
}

func TestAddDependency(t *testing.T) {
	f := parseString(t, exampleOpam)

	// Add a new dependency
	f.AddDependency("new-package")

	deps := f.GetDependencies()
	require.Len(t, deps, 3)

	// New package should be first (added after the opening bracket)
	assert.Equal(t, "new-package", deps[0])
	assert.Equal(t, "perennial", deps[1])
	assert.Equal(t, "coq-record-update", deps[2])
}

func TestAddDependency_Duplicate(t *testing.T) {
	f := parseString(t, exampleOpam)

	// Try to add an existing dependency
	f.AddDependency("perennial")

	deps := f.GetDependencies()
	// Should still have only 2 dependencies
	require.Len(t, deps, 2)

	assert.Equal(t, "perennial", deps[0])
	assert.Equal(t, "coq-record-update", deps[1])
}

func TestAddDependency_Multiple(t *testing.T) {
	f := parseString(t, exampleOpam)

	// Add multiple new dependencies
	f.AddDependency("package-a")
	f.AddDependency("package-b")
	f.AddDependency("package-c")

	deps := f.GetDependencies()
	require.Len(t, deps, 5)

	// New packages are added in reverse order (each inserted after the opening bracket)
	assert.Equal(t, "package-c", deps[0])
	assert.Equal(t, "package-b", deps[1])
	assert.Equal(t, "package-a", deps[2])
	assert.Equal(t, "perennial", deps[3])
	assert.Equal(t, "coq-record-update", deps[4])
}

func TestSetIndirect_EmptyWhenNoIndirects(t *testing.T) {
	// Test with an opam file that has no indirect section
	opamWithoutIndirect := `opam-version: "2.0"
version: "dev"

depends: [
  "coq"
]

pin-depends: [
  ["perennial.dev"           "git+https://github.com/mit-pdos/perennial#577140b0594fbdea1a2b3c4d5e6f7a8b9c0d1e2f"]
]
`
	f := parseString(t, opamWithoutIndirect)

	// Call SetIndirect with an empty list
	f.SetIndirect([]PinDepend{})

	// Verify the file doesn't contain indirect marker
	output := f.String()
	assert.NotContains(t, output, "## begin indirect")
}
