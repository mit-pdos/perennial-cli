package cmd

import (
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

// opamCmd represents the opam command
var opamCmd = &cobra.Command{
	Use:   "opam [command]",
	Short: "Manage opam files",
	Long: `Manage opam files.

Helps update dependencies and maintain indirect pin-depends.`,
}

func init() {
	rootCmd.AddCommand(opamCmd)
	opamCmd.PersistentFlags().StringP("file", "f", "", "Opam file (if not provided, look in current directory)")
}
