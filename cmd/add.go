package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/mit-pdos/perennial-cli/opam"
	"github.com/spf13/cobra"
)

// parseGitURL parses a git URL with optional commit hash
// Returns: baseURL (without commit), commit hash (or empty), error
func parseGitURL(url string) (string, string, error) {
	// Check for commit hash in URL (format: url#commit)
	if idx := strings.IndexByte(url, '#'); idx >= 0 {
		return url[:idx], url[idx+1:], nil
	}
	return url, "", nil
}

func doAdd(cmd *cobra.Command, args []string) error {
	opamFileName, _ := cmd.Flags().GetString("file")
	packageFlag, _ := cmd.Flags().GetString("package")
	urlArg := args[0]

	// Parse the URL to extract base URL and optional commit
	baseURL, commit, err := parseGitURL(urlArg)
	if err != nil {
		return err
	}

	// Ensure URL has git+ prefix
	if !strings.HasPrefix(baseURL, "git+") {
		baseURL = "git+" + baseURL
	}

	// Get commit hash (either from URL or fetch latest)
	if commit == "" {
		fmt.Printf("fetching latest commit...\n")
		commit, err = opam.GetLatestCommit(baseURL)
		if err != nil {
			return fmt.Errorf("failed to get latest commit: %w", err)
		}
	}

	// Determine package name
	var packageName string
	if packageFlag != "" {
		packageName = packageFlag
	} else {
		fmt.Printf("finding opam package in repository...\n")
		packageName, err = opam.FindOpamPackage(baseURL, commit)
		if err != nil {
			return fmt.Errorf("failed to find opam package: %w", err)
		}
	}

	// Read the opam file
	contents, err := os.ReadFile(opamFileName)
	if err != nil {
		return err
	}

	opamFile, err := opam.Parse(bytes.NewReader(contents))
	if err != nil {
		return err
	}

	// Add dependency to depends block
	opamFile.AddDependency(packageName)

	// Add pin-depends entry
	dep := opam.PinDepend{
		Package: packageName,
		URL:     baseURL,
		Commit:  commit,
	}
	opamFile.AddPinDepend(dep)

	// Update indirect dependencies
	err = opamFile.UpdateIndirectDependencies()
	if err != nil {
		return fmt.Errorf("failed to update indirect dependencies: %w", err)
	}

	// Write the updated opam file
	newContents := opamFile.String()
	if err := os.WriteFile(opamFileName, []byte(newContents), 0644); err != nil {
		return err
	}

	fmt.Printf("added %s (pinned to %s)\n", packageName, commit)
	return nil
}

// addCmd represents the opam add command
var addCmd = &cobra.Command{
	Use:   "add <url> [-p <package>]",
	Short: "add a dependency",
	Long: `Add a dependency and pin it.

If the URL has a commit hash, it will be pinned to that commit; otherwise, it
will be pinned to the latest commit of the default branch.

The package is the base name of the opam file. If not provided, perennial-cli
will look for a unique opam file in the repo and fail if multiple are found.

If the dependency already exists, it will be updated.

`,
	Args: cobra.ExactArgs(1),
	RunE: doAdd,
}

func init() {
	opamCmd.AddCommand(addCmd)
	addCmd.Flags().StringP("package", "p", "", "opam package name")
}
