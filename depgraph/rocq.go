package depgraph

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	orderedmap "github.com/pb33f/ordered-map/v2"
)

func filterRocq(deps *Graph) {
	deps.FilterNodes(func(name string) bool {
		return strings.HasSuffix(name, ".vo") || strings.HasSuffix(name, ".v")
	})
}

func setExtension(path string, ext string) string {
	oldExt := filepath.Ext(path)
	return strings.TrimSuffix(path, oldExt) + ext
}

func ParseRocqdep(rocqdepFileName string) (*Graph, error) {
	f, err := os.Open(rocqdepFileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	deps, err := Parse(f)
	if err != nil {
		return nil, err
	}
	filterRocq(deps)
	return deps, nil
}

// Get the dependencies of files in args.
//
// Args can be a list of .v or .vo files: this function always uses the .vo
// files for dependencies
func RocqDeps(deps *Graph, args []string) []string {
	var targets []string
	for _, arg := range args {
		target := arg
		if strings.HasSuffix(arg, ".v") {
			target = strings.TrimSuffix(arg, ".v") + ".vo"
		}
		targets = append(targets, target)
	}
	chains := deps.Deps(targets)
	seen := orderedmap.New[string, struct{}]()
	for _, depChain := range chains {
		for _, source := range depChain.Sources() {
			if !strings.HasSuffix(source, ".v") {
				continue
			}
			seen.Set(source, struct{}{})
		}
	}
	return slices.Collect(seen.KeysFromOldest())
}

// Get the reverse dependencies of files in args (the files that depend on any
// of args)
//
// Args can be a list of .v or .vo files: this function searches for reverse
// dependencies starting from both the .v and .vo files for each argument
func RocqTargets(deps *Graph, args []string) []string {
	var sources []string
	for _, arg := range args {
		// Add both .v and .vo versions to seed the search
		sources = append(sources, setExtension(arg, ".v"))
		sources = append(sources, setExtension(arg, ".vo"))
	}

	allTargets := deps.Targets(sources)

	seen := orderedmap.New[string, struct{}]()
	for _, target := range allTargets {
		// Convert to .v files for the final result
		seen.Set(setExtension(target, ".v"), struct{}{})
	}
	return slices.Collect(seen.KeysFromOldest())
}
