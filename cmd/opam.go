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

// opamCmd represents the opam command
var opamCmd = &cobra.Command{
	Use:   "opam [command]",
	Short: "Manage opam files",
	Long: `Manage opam files.

Helps update dependencies and maintain indirect pin-depends.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		opamFile, _ := cmd.Flags().GetString("file")
		if opamFile == "" {
			var ok bool
			opamFile, ok := findUniqueOpamFile()
			if !ok {
				return fmt.Errorf("no opam file provided and no unique file found")
			}
			// Set the flag value so Run can use it
			cmd.Flags().Set("file", opamFile)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(opamCmd)
	opamCmd.PersistentFlags().StringP("file", "f", "", "Opam file (if not provided, look in current directory)")
}
