package cmd

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/mit-pdos/perennial-cli/depgraph"
	"github.com/mit-pdos/perennial-cli/rocq_makefile"
	orderedmap "github.com/pb33f/ordered-map/v2"
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
	// Create request and response channels
	numWorkers := runtime.NumCPU()
	requests := make(chan string, numWorkers)
	responses := make(chan []fileToInstall, numWorkers)

	// Start worker pool
	for range numWorkers {
		go func() {
			for vFile := range requests {
				// NOTE: not installing glob files
				voFile := setExtension(vFile, ".vo")
				destDir := rocq_makefile.DestinationOf(makeVars, voFile)

				result := []fileToInstall{
					{src: voFile, dest: path.Join(destDir, path.Base(voFile))},
					{src: vFile, dest: path.Join(destDir, path.Base(vFile))},
				}
				responses <- result
			}
		}()
	}

	// Send all requests
	go func() {
		for _, vFile := range sources {
			requests <- vFile
		}
		close(requests)
	}()

	// Collect all responses
	var files []fileToInstall
	for range len(sources) {
		files = append(files, <-responses...)
	}

	// Sort by destination
	slices.SortFunc(files, func(a, b fileToInstall) int {
		return strings.Compare(a.dest, b.dest)
	})

	return files
}

func installAll(quietMode bool, filesToInstall []fileToInstall) error {
	for _, f := range filesToInstall {
		if err := installFile(f.src, f.dest); err != nil {
			return err
		}

		if !quietMode {
			fmt.Printf("INSTALL %s\n", f.src)
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

func getInstallFiles(cmd *cobra.Command, args []string) ([]fileToInstall, map[string]string, error) {
	rocqdepName, _ := cmd.Flags().GetString("file")
	installDeps, _ := cmd.Flags().GetBool("install-deps")
	if len(args) == 0 {
		// If no args, walk current directory
		args = []string{"."}
	}

	// Gather list of .v files
	sources, err := gatherVFiles(args)
	if err != nil {
		return nil, nil, err
	}

	if installDeps {
		sourceList := orderedmap.New[string, struct{}]()
		for _, source := range sources {
			sourceList.Set(source, struct{}{})
		}

		// Parse dependency graph from .rocqdeps.d
		deps, err := depgraph.ParseRocqdep(rocqdepName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse deps %s: %w", rocqdepName, err)
		}

		// Add all dependencies not already in sources
		sourceDeps := depgraph.RocqDeps(deps, sources)
		for _, f := range sourceDeps {
			if _, present := sourceList.Get(f); !present {
				sources = append(sources, f)
			}
		}
	}

	if len(sources) == 0 {
		return nil, nil, fmt.Errorf("no sources to install")
	}

	// Get makefile vars from _RocqProject or _CoqProject
	makeVars, err := rocq_makefile.GetRocqVars()
	if err != nil {
		return nil, nil, err
	}

	// Install sources
	return getFilesToInstall(makeVars, sources), makeVars, nil
}

// installCmd represents the install command
var installCmd = &cobra.Command{
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
		filesToInstall, makeVars, err := getInstallFiles(cmd, args)
		if err != nil {
			return err
		}
		if err := installAll(quietMode, filesToInstall); err != nil {
			return fmt.Errorf("error installing sources: %v", err)
		}
		if !quietMode {
			fmt.Printf("installed to %s\n", path.Clean(makeVars["COQLIBINSTALL"]))
		}

		return nil
	},
}

// uninstallCmd represents the uninstall command
var uninstallCmd = &cobra.Command{
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
		filesToInstall, _, err := getInstallFiles(cmd, args)
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
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)

	installCmd.PersistentFlags().StringP("file", "f", ".rocqdeps.d", "Path to .rocqdeps.d file")
	installCmd.PersistentFlags().BoolP("quiet", "q", false, "quiet mode (don't print list of installed files)")
	installCmd.PersistentFlags().Bool("install-deps", true, "install dependencies of supplied files")

	uninstallCmd.PersistentFlags().StringP("file", "f", ".rocqdeps.d", "Path to .rocqdeps.d file")
	uninstallCmd.PersistentFlags().BoolP("quiet", "q", false, "quiet mode (don't print list of uninstalled files)")
	uninstallCmd.PersistentFlags().Bool("install-deps", true, "also uninstall dependencies")
}
