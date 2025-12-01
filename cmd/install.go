package cmd

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/mit-pdos/perennial-cli/depgraph"
	"github.com/mit-pdos/perennial-cli/rocq_makefile"
	"github.com/spf13/cobra"
)

// Install src to dest, creating destination directory if needed.
func installFile(src string, dest string) error {
	// Check if source file exists
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", src)
	}

	// Create destination directory if needed
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", destDir, err)
	}

	// Copy source file to destination
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open %s: %v", src, err)
	}

	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		srcFile.Close()
		return fmt.Errorf("failed to create %s: %v", dest, err)
	}

	_, err = io.Copy(destFile, srcFile)
	srcFile.Close()
	destFile.Close()
	if err != nil {
		return fmt.Errorf("failed to copy %s to %s: %v", src, dest, err)
	}

	return nil
}

type fileToInstall struct {
	src  string
	dest string
}

func getFilesToInstall(makeVars map[string]string, sources []string) []fileToInstall {
	var files []fileToInstall
	// NOTE: if rocq makefile is slow, this function can be run in parallel for
	// all source files
	for _, vFile := range sources {
		// NOTE: not installing glob files
		voFile := setExtension(vFile, ".vo")
		destDir := rocq_makefile.DestinationOf(makeVars, voFile)
		files = append(files, fileToInstall{src: voFile, dest: path.Join(destDir, path.Base(voFile))})
		files = append(files, fileToInstall{src: vFile, dest: path.Join(destDir, path.Base(vFile))})
	}
	return files
}

func installAll(quietMode bool, filesToInstall []fileToInstall) error {
	for _, f := range filesToInstall {
		if err := installFile(f.src, f.dest); err != nil {
			return err
		}

		if !quietMode {
			fmt.Printf("INSTALL %s to %s\n", f.src, f.dest)
		}
	}
	return nil
}

func uninstallAll(quietMode bool, filesToInstall []fileToInstall) error {
	for _, f := range filesToInstall {
		// Delete the destination file, ignoring if it doesn't exist
		if err := os.Remove(f.dest); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %s: %v", f.dest, err)
		}

		if !quietMode {
			fmt.Printf("RM %s\n", f.dest)
		}
	}
	return nil
}

func getInstallFiles(cmd *cobra.Command, args []string) ([]fileToInstall, error) {
	rocqdepName, _ := cmd.Flags().GetString("file")
	installDeps, _ := cmd.Flags().GetBool("install-deps")
	if len(args) == 0 {
		// If no args, walk current directory
		args = []string{"."}
	}

	// Gather list of .v files
	sources, err := gatherVFiles(args)
	if err != nil {
		return nil, err
	}

	if installDeps {
		// Parse dependency graph from .rocqdeps.d
		deps, err := depgraph.ParseRocqdep(rocqdepName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse deps %s: %w", rocqdepName, err)
		}

		// Get all dependencies of the sources
		sources = depgraph.RocqDeps(deps, sources)
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no sources to install")
	}

	// Get makefile vars from _RocqProject or _CoqProject
	makeVars, err := rocq_makefile.GetRocqVars()
	if err != nil {
		return nil, err
	}

	// Install sources
	return getFilesToInstall(makeVars, sources), nil
}

// opamInstallCmd represents the opam install command
var opamInstallCmd = &cobra.Command{
	Use:   "install <FILES>",
	Short: "Install build outputs to Rocq user-contrib",
	Long: `Install .vo files, typically to an opam switch.

Takes a list of either .v files or directories (which are searched recursively
for all *.v files). Assumes all input files are compiled. Will automatically
install any dependencies required by the input .v files, using .rocqdeps.d.

Emulates the functionality of "make install" when using rocq makefile.
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		quietMode, _ := cmd.Flags().GetBool("quiet")
		filesToInstall, err := getInstallFiles(cmd, args)
		if err != nil {
			return err
		}
		if err := installAll(quietMode, filesToInstall); err != nil {
			return fmt.Errorf("error installing sources: %v", err)
		}

		return nil
	},
}

// opamUninstallCmd represents the opam uninstall command
var opamUninstallCmd = &cobra.Command{
	Use:   "uninstall <FILES>",
	Short: "Uninstall build outputs to Rocq user-contrib",
	Long: `Uninstall .vo files from the opam switch.

Takes a list of either .v files or directories (which are searched recursively
for all *.v files). Will automatically uninstall any dependencies required by
the input .v files, using .rocqdeps.d.

Emulates the functionality of "make uninstall" when using rocq makefile.
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		quietMode, _ := cmd.Flags().GetBool("quiet")
		filesToInstall, err := getInstallFiles(cmd, args)
		if err != nil {
			return err
		}
		if err := uninstallAll(quietMode, filesToInstall); err != nil {
			return fmt.Errorf("error uninstalling sources: %v", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(opamInstallCmd)
	rootCmd.AddCommand(opamUninstallCmd)

	opamInstallCmd.PersistentFlags().StringP("file", "f", ".rocqdeps.d", "Path to .rocqdeps.d file")
	opamInstallCmd.PersistentFlags().BoolP("quiet", "q", false, "quiet mode (don't print list of installed files)")
	opamInstallCmd.PersistentFlags().Bool("install-deps", true, "install dependencies of supplied files")

	opamUninstallCmd.PersistentFlags().StringP("file", "f", ".rocqdeps.d", "Path to .rocqdeps.d file")
	opamUninstallCmd.PersistentFlags().BoolP("quiet", "q", false, "quiet mode (don't print list of uninstalled files)")
	opamUninstallCmd.PersistentFlags().Bool("install-deps", true, "also uninstall dependencies")
}
