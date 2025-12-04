package cmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/mit-pdos/perennial-cli/git"
	"github.com/mit-pdos/perennial-cli/opam"
	"github.com/spf13/cobra"
)

type completedUpdate struct {
	Package  string
	From, To string
}

func doUpdate(cmd *cobra.Command, args []string) error {
	packageFlag, _ := cmd.Flags().GetString("package")
	opamFileName, _ := cmd.Flags().GetString("file")
	contents, err := os.ReadFile(opamFileName)
	if err != nil {
		return err
	}
	opamFile, err := opam.Parse(bytes.NewReader(contents))
	var updates []completedUpdate
	for _, dep := range opamFile.GetPinDepends() {
		if packageFlag != "" && packageFlag != dep.Package {
			continue
		}
		hash, err := git.GetLatestCommit(dep.URL)
		if err != nil {
			return err
		}
		hash = opam.AbbreviateHash(hash)
		if hash != dep.Commit {
			oldCommit := dep.Commit
			dep.Commit = hash
			opamFile.AddPinDepend(dep)
			updates = append(updates, completedUpdate{
				Package: dep.Package,
				From:    oldCommit,
				To:      hash,
			})
		}
	}
	err = opamFile.UpdateIndirectDependencies()
	if err != nil {
		return err
	}
	newContents := opamFile.String()
	if newContents == string(contents) {
		// nothing to do, don't write the file
		return nil
	}
	if err := os.WriteFile(opamFileName, []byte(newContents), 0644); err != nil {
		return err
	}
	if len(updates) > 0 {
		fmt.Printf("upgraded %d packages:\n", len(updates))
		for _, update := range updates {
			fmt.Printf("  %s: %s -> %s\n", update.Package, update.From, update.To)
		}
	} else {
		fmt.Printf("updated indirect dependencies\n")
	}
	return nil
}

// updateCmd represents the opam update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update pinned dependencies",
	Long:  `Update dependencies in pin-depends to the latest commit hash.`,
	RunE:  doUpdate,
}

func init() {
	opamCmd.AddCommand(updateCmd)

	// Here you will define your flags and configuration settings.

	updateCmd.PersistentFlags().StringP("package", "p", "", "Update only a specific package")
}
