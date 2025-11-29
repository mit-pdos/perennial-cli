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
]

pin-depends: [
  ["perennial.dev"           "git+https://github.com/mit-pdos/perennial#577140b0594fbde"]

  ## begin indirect
  ["coq-record-update.dev"   "git+https://github.com/tchajed/coq-record-update#7b2645210331c3e"]
  ["rocq-stdpp.dev"          "git+https://gitlab.mpi-sws.org/iris/stdpp#187909f0c15b7c8"]
  ["rocq-iris.dev"           "git+https://gitlab.mpi-sws.org/iris/iris#fde0f8699242184"]
  ["iris-named-props.dev"    "git+https://github.com/tchajed/iris-named-props#c388714a93b1c04"]
  ## end
]

build: [make "-j%{jobs}%"]
install: ["./etc/install.sh"]
`

func TestParse(t *testing.T) {
	r := strings.NewReader(exampleOpam)
	f, err := Parse(r)
	require.NoError(t, err)

	// Check depends region
	assert.False(t, f.depends.empty(), "depends region not found")
	assert.Equal(t, 10, f.depends.startLine)
	assert.Equal(t, 13, f.depends.endLine)

	// Check pin-depends region
	assert.False(t, f.pinDepends.empty(), "pin-depends region not found")
	assert.Equal(t, 14, f.pinDepends.startLine)
	assert.Equal(t, 24, f.pinDepends.endLine)

	// Check indirect region
	assert.False(t, f.indirectPinDepends.empty(), "indirect pin-depends region not found")
	assert.Equal(t, 17, f.indirectPinDepends.startLine)
	assert.Equal(t, 23, f.indirectPinDepends.endLine)
}

func TestListPinDepends(t *testing.T) {
	r := strings.NewReader(exampleOpam)
	f, err := Parse(r)
	require.NoError(t, err)

	deps := f.ListPinDepends()
	// Should only return direct dependencies (excluding indirect section)
	require.Len(t, deps, 1)

	// Check the direct dependency
	assert.Equal(t, "perennial", deps[0].Package)
	assert.Equal(t, "git+https://github.com/mit-pdos/perennial", deps[0].URL)
	assert.Equal(t, "577140b0594fbde", deps[0].Commit)
}

func TestGetIndirect(t *testing.T) {
	r := strings.NewReader(exampleOpam)
	f, err := Parse(r)
	require.NoError(t, err)

	indirect := f.GetIndirect()
	require.Len(t, indirect, 4)

	// Check that we only got indirect dependencies
	assert.Equal(t, "coq-record-update", indirect[0].Package)
	assert.Equal(t, "iris-named-props", indirect[3].Package)
}

func TestAddPinDepend_Update(t *testing.T) {
	r := strings.NewReader(exampleOpam)
	f, err := Parse(r)
	require.NoError(t, err)

	// Update existing dependency
	f.AddPinDepend(PinDepend{
		Package: "perennial",
		URL:     "git+https://github.com/mit-pdos/perennial",
		Commit:  "newcommit123",
	})

	deps := f.ListPinDepends()
	found := false
	for _, dep := range deps {
		if dep.Package == "perennial" {
			found = true
			assert.Equal(t, "newcommit123", dep.Commit)
		}
	}
	assert.True(t, found, "perennial not found after update")
}

func TestAddPinDepend_Add(t *testing.T) {
	r := strings.NewReader(exampleOpam)
	f, err := Parse(r)
	require.NoError(t, err)

	// Add new dependency
	f.AddPinDepend(PinDepend{
		Package: "new-package",
		URL:     "git+https://example.com/package",
		Commit:  "abc123",
	})

	deps := f.ListPinDepends()
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
	r := strings.NewReader(exampleOpam)
	f, err := Parse(r)
	require.NoError(t, err)

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
	r := strings.NewReader(exampleOpam)
	f, err := Parse(r)
	require.NoError(t, err)

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
			name: "standard line",
			line: `  ["perennial.dev"           "git+https://github.com/mit-pdos/perennial#577140b0594fbdea"]`,
			want: &PinDepend{
				Package: "perennial",
				URL:     "git+https://github.com/mit-pdos/perennial",
				Commit:  "577140b0594fbde",
			},
		},
		{
			name: "extra whitespace",
			line: `    [  "pkg.dev"   "git+https://example.com/repo#abc123"  ]`,
			want: &PinDepend{
				Package: "pkg",
				URL:     "git+https://example.com/repo",
				Commit:  "abc123",
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
			name: "with commit",
			dep: PinDepend{
				Package: "perennial",
				URL:     "git+https://github.com/mit-pdos/perennial",
				Commit:  "577140b0594fbdea",
			},
			want: `  ["perennial.dev"             "git+https://github.com/mit-pdos/perennial#577140b0594fbde"]`,
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
				Commit:  "abc",
			},
			want: `  ["very-long-package-name.dev" "git+https://example.com/repo#abc"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dep.String())
		})
	}
}
