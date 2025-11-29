package cmd

import (
	"github.com/spf13/cobra"
)

// opamCmd represents the opam command
var opamCmd = &cobra.Command{
	Use:   "opam",
	Short: "Manage opam files",
	Long:  `Manage opam files, especially updating dependencies and maintaining indirect pin-depends.`,
}

func init() {
	rootCmd.AddCommand(opamCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// opamCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// opamCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
