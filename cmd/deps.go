/*
Copyright © 2025 Tej Chajed <tchajed@gmail.com>
*/
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

// depsCmd represents the deps command
var depsCmd = &cobra.Command{
	Use:   "deps",
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
		deps, err := depgraph.ParseRocqdep(rocqdepFileName)
		if err != nil {
			return err
		}
		sources := depgraph.RocqDeps(deps, args)
		for _, source := range sources {
			fmt.Println(source)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(depsCmd)

	depsCmd.PersistentFlags().StringP("file", "f", "", "Path to .rocqdeps.d file")
}
