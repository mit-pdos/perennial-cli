package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

func findUniqueOpamFile() (string, bool) {
	files, err := filepath.Glob("*.opam")
	if err != nil || len(files) != 1 {
		return "", false
	}
	return files[0], true
}

// updateCmd represents the opam update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update pinned dependencies",
	Long:  `Update dependencies in pin-depends to the latest commit hash.`,
	Run: func(cmd *cobra.Command, args []string) {
		packageFlag, _ := cmd.Flags().GetString("package")
		opamFile, _ := cmd.Flags().GetString("file")
		if opamFile == "" {
			var ok bool
			opamFile, ok = findUniqueOpamFile()
			if !ok {
				panic(fmt.Errorf("no opam file provided and no unique file found"))
			}
		}
		fmt.Printf("update called on package %s and file %s\n", packageFlag, opamFile)
	},
}

func init() {
	opamCmd.AddCommand(updateCmd)

	// Here you will define your flags and configuration settings.

	updateCmd.PersistentFlags().StringP("package", "p", "", "Update only a specific package")
	updateCmd.PersistentFlags().StringP("file", "f", "", "Opam file (if not provided, look in current directory)")
}
