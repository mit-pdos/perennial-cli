package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// addCmd represents the opam add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "add a dependency",
	Long: `Add a dependency and pin it to the latest version.

If the dependency already exists, it will be updated.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("add called")
		return fmt.Errorf("not implemented")
	},
}

func init() {
	opamCmd.AddCommand(addCmd)
}
