package depgraph

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	input := `src/proof/github_com/tchajed/perennial_example_proof.vo src/proof/github_com/tchajed/perennial_example_proof.glob src/proof/github_com/tchajed/perennial_example_proof.v.beautified src/proof/github_com/tchajed/perennial_example_proof.required_vo: src/proof/github_com/tchajed/perennial_example_proof.v src/generatedproof/github_com/tchajed/perennial_example_proof.vo /Users/tchajed/.opam/ocaml-5/lib/rocq-runtime/rocqworker
src/proof/github_com/tchajed/perennial_example_proof.vos src/proof/github_com/tchajed/perennial_example_proof.vok src/proof/github_com/tchajed/perennial_example_proof.required_vos: src/proof/github_com/tchajed/perennial_example_proof.v src/generatedproof/github_com/tchajed/perennial_example_proof.vos /Users/tchajed/.opam/ocaml-5/lib/rocq-runtime/rocqworker
src/code/github_com/tchajed/perennial_example_proof.vo src/code/github_com/tchajed/perennial_example_proof.glob src/code/github_com/tchajed/perennial_example_proof.v.beautified src/code/github_com/tchajed/perennial_example_proof.required_vo: src/code/github_com/tchajed/perennial_example_proof.v /Users/tchajed/.opam/ocaml-5/lib/rocq-runtime/rocqworker
src/code/github_com/tchajed/perennial_example_proof.vos src/code/github_com/tchajed/perennial_example_proof.vok src/code/github_com/tchajed/perennial_example_proof.required_vos: src/code/github_com/tchajed/perennial_example_proof.v /Users/tchajed/.opam/ocaml-5/lib/rocq-runtime/rocqworker
src/generatedproof/github_com/tchajed/perennial_example_proof.vo src/generatedproof/github_com/tchajed/perennial_example_proof.glob src/generatedproof/github_com/tchajed/perennial_example_proof.v.beautified src/generatedproof/github_com/tchajed/perennial_example_proof.required_vo: src/generatedproof/github_com/tchajed/perennial_example_proof.v src/code/github_com/tchajed/perennial_example_proof.vo /Users/tchajed/.opam/ocaml-5/lib/rocq-runtime/rocqworker
src/generatedproof/github_com/tchajed/perennial_example_proof.vos src/generatedproof/github_com/tchajed/perennial_example_proof.vok src/generatedproof/github_com/tchajed/perennial_example_proof.required_vos: src/generatedproof/github_com/tchajed/perennial_example_proof.v src/code/github_com/tchajed/perennial_example_proof.vos /Users/tchajed/.opam/ocaml-5/lib/rocq-runtime/rocqworker
`

	g, err := Parse(strings.NewReader(input))
	assert.NoError(t, err)

	assert.Len(t, g.allDeps(), 56)

	g.FilterNodes(func(name string) bool {
		return strings.HasSuffix(name, ".vo") || strings.HasSuffix(name, ".v")
	})

	assert.Len(t, g.allDeps(), 5)

	// Check some specific dependencies
	assert.Contains(t, g.allDeps(), Dep{
		Target: "src/proof/github_com/tchajed/perennial_example_proof.vo",
		Source: "src/proof/github_com/tchajed/perennial_example_proof.v",
	})
	assert.Contains(t, g.allDeps(), Dep{
		Target: "src/generatedproof/github_com/tchajed/perennial_example_proof.vo",
		Source: "src/code/github_com/tchajed/perennial_example_proof.vo",
	})
}

func TestParseEmptyAndComments(t *testing.T) {
	input := `# This is a comment
target1: dep1

# Another comment
target2: dep2
`

	g, err := Parse(strings.NewReader(input))
	assert.NoError(t, err)

	assert.Len(t, g.allDeps(), 2)
	assert.Contains(t, g.allDeps(), Dep{Target: "target1", Source: "dep1"})
	assert.Contains(t, g.allDeps(), Dep{Target: "target2", Source: "dep2"})
}
