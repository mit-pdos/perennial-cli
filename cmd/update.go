package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mit-pdos/perennial-cli/opam"
	"github.com/spf13/cobra"
)

func findUniqueOpamFile() (string, bool) {
	files, err := filepath.Glob("*.opam")
	if err != nil || len(files) != 1 {
		return "", false
	}
	return files[0], true
}

type completedUpdate struct {
	Package  string
	From, To string
}

// updateCmd represents the opam update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update pinned dependencies",
	Long:  `Update dependencies in pin-depends to the latest commit hash.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		opamFile, _ := cmd.Flags().GetString("file")
		if opamFile == "" {
			var ok bool
			opamFile, ok = findUniqueOpamFile()
			if !ok {
				return fmt.Errorf("no opam file provided and no unique file found")
			}
			// Set the flag value so Run can use it
			cmd.Flags().Set("file", opamFile)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		packageFlag, _ := cmd.Flags().GetString("package")
		opamFileName, _ := cmd.Flags().GetString("file")
		f, err := os.Open(opamFileName)
		if err != nil {
			return err
		}
		opamFile, err := opam.Parse(f)
		var updates []completedUpdate
		for _, dep := range opamFile.ListPinDepends() {
			if packageFlag != "" && packageFlag != dep.Package {
				continue
			}
			hash, err := opam.GetLatestCommit(dep.URL)
			if err != nil {
				return err
			}
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
		if len(updates) > 0 {
			err := os.WriteFile(opamFileName, []byte(opamFile.String()), 0644)
			if err != nil {
				return err
			}
			fmt.Printf("upgraded %d packages:\n", len(updates))
			for _, update := range updates {
				fmt.Printf("  %s: %s -> %s\n", update.Package, update.From, update.To)
			}
		}
		return nil
	},
}

func init() {
	opamCmd.AddCommand(updateCmd)

	// Here you will define your flags and configuration settings.

	updateCmd.PersistentFlags().StringP("package", "p", "", "Update only a specific package")
	updateCmd.PersistentFlags().StringP("file", "f", "", "Opam file (if not provided, look in current directory)")
}
