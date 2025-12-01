package cmd

import (
	"github.com/spf13/cobra"
)

// opamCmd represents the opam command
var opamCmd = &cobra.Command{
	Use:   "opam [command]",
	Short: "Manage opam files",
	Long: `Manage opam files.

Helps update dependencies and maintain indirect pin-depends.`,
}

func init() {
	rootCmd.AddCommand(opamCmd)
}
