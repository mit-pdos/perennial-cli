package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func indent(indentation string, t string) string {
	// indent every line by indentation
	t = strings.TrimSpace(t)
	lines := strings.Split(t, "\n")
	for i, line := range lines {
		lines[i] = indentation + strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "perennial-cli [command]",
	Short:        "CLI to manage perennial verification projects",
	Long:         `perennial-cli manages verification projects based on Perennial.`,
	SilenceUsage: true,
	Example: indent("  ", `
go run github.com/mit-pdos/perennial-cli@latest init <proj_url>

perennial-cli opam update
perennial-cli goose
`),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main().
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
