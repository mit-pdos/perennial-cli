package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mit-pdos/perennial-cli/init_proj"
	"github.com/spf13/cobra"
)

func doInit(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: perennial-cli init <git-url>")
	}
	url := args[0]

	// Use current directory
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Get project name from current directory name
	projectName := filepath.Base(dir)

	return init_proj.New(url, projectName, dir)
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init <git-url>",
	Short: "Initialize a new perennial project",
	Long: `Create a new perennial project with template files.

	Run in a new directory to add an initial project skeleton.
	`,
	Args: cobra.ExactArgs(1),
	RunE: doInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}
