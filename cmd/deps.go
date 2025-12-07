package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mit-pdos/perennial-cli/depgraph"
	"github.com/spf13/cobra"
)

func setExtension(path string, ext string) string {
	oldExt := filepath.Ext(path)
	return strings.TrimSuffix(path, oldExt) + ext
}

func getDirVFiles(dir string) ([]string, error) {
	var sources []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".v") {
			sources = append(sources, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %v", dir, err)
	}

	return sources, nil
}

func gatherVFiles(paths []string) ([]string, error) {
	var sources []string

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("error accessing %s: %v", path, err)
		}

		if info.IsDir() {
			// Walk directory and find all .v files
			dirSources, err := getDirVFiles(path)
			if err != nil {
				return nil, fmt.Errorf("error walking directory %s: %v", path, err)
			}
			sources = append(sources, dirSources...)

		} else if strings.HasSuffix(path, ".v") {
			sources = append(sources, path)
		} else if strings.HasSuffix(path, ".vo") {
			sources = append(sources, setExtension(path, ".v"))
		} else {
			fmt.Fprintf(os.Stderr, "Skipping non-.v file: %s\n", path)
		}
	}

	return sources, nil
}

// depsCmd represents the deps command
var depsCmd = &cobra.Command{
	Use: "deps",
	Example: indent("  ", `
		perennial-cli deps $(find src -name "*.v")
		perennial-cli deps new/proof/proof_prelude.v
		perennial-cli deps -r new/proof/proof_prelude.v
		perennial-cli deps --exclude-source $(find new -name "*.v")
`),
	Short: "List and analyze .rocqdeps.d dependencies",
	Long: `List and analyze .rocqdeps.d dependencies.

Parse .rocqdeps.d and report dependencies.
`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		rocqdepName, _ := cmd.Flags().GetString("file")
		if rocqdepName == "" {
			if _, err := os.Stat(".rocqdeps.d"); err != nil {
				return err
			}
			cmd.Flags().Set("file", ".rocqdeps.d")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		rocqdepFileName, _ := cmd.Flags().GetString("file")
		printVo, _ := cmd.Flags().GetBool("vo")
		reverse, _ := cmd.Flags().GetBool("reverse")
		excludeSource, _ := cmd.Flags().GetBool("exclude-source")

		// Gather .v files from arguments (handles directories)
		sources, err := gatherVFiles(args)
		if err != nil {
			return err
		}
		sourceSet := make(map[string]bool)
		for _, source := range sources {
			sourceSet[source] = true
		}

		deps, err := depgraph.ParseRocqdep(rocqdepFileName)
		if err != nil {
			return err
		}

		var depSources []string
		if reverse {
			// reverse dependencies (targets)
			depSources = depgraph.RocqTargets(deps, sources)
		} else {
			// normal dep behavior
			depSources = depgraph.RocqDeps(deps, sources)
		}
		for _, source := range depSources {
			if excludeSource && sourceSet[source] {
				continue
			}
			if printVo {
				fmt.Println(setExtension(source, ".vo"))
			} else {
				fmt.Println(source)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(depsCmd)

	depsCmd.PersistentFlags().StringP("file", "f", "", "Path to .rocqdeps.d file")
	depsCmd.PersistentFlags().Bool("vo", false, "Print .vo dependencies rather than .v sources")
	depsCmd.PersistentFlags().BoolP("reverse", "r", false, "Get reverse dependencies (files that depend on provided sources)")
	depsCmd.PersistentFlags().Bool("exclude-source", false, "Exclude source files from output")
}
