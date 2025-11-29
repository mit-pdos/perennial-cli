package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mit-pdos/perennial-cli/depgraph"
	"github.com/spf13/cobra"
)

func getMakefileVars(makefilePath string, vars []string) map[string]string {
	// Create a temporary Makefile with just the print-% rule
	tmpFile, err := os.CreateTemp("", "makefile-*.mk")
	if err != nil {
		panic(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Write just the print rule
	if _, err := tmpFile.WriteString("print-%: ; @echo $($*)\n"); err != nil {
		panic(err)
	}
	tmpFile.Close()

	// Run make for each variable, passing both makefiles with -f flags
	result := make(map[string]string)
	for _, varName := range vars {
		target := "print-" + varName
		cmd := exec.Command("make", "-f", makefilePath, "-f", tmpFile.Name(), target)
		output, err := cmd.Output()
		if err != nil {
			panic(fmt.Sprintf("failed to get variable %s: %v", varName, err))
		}
		result[varName] = strings.TrimSpace(string(output))
	}
	return result
}

func getRocqMakefileVars(projFile string) map[string]string {
	// 1. Run rocq makefile -f projFile -o <tmp Makefile.rocq>
	tmpFile, err := os.CreateTemp("", "Makefile.rocq-*")
	if err != nil {
		panic(err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	cmd := exec.Command("rocq", "makefile", "-f", projFile, "-o", tmpPath)
	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("failed to run rocq makefile: %v", err))
	}

	// 2. Get COQLIB and COQLIBINSTALL using getMakefileVars
	return getMakefileVars(tmpPath, []string{"COQLIBS", "COQLIBINSTALL"})
}

func destinationOf(makeVars map[string]string, target string) string {
	// Build command arguments: rocq makefile <COQLIBS args> -destination-of <target>
	args := []string{"makefile"}

	// Split COQLIBS using shell splitting rules
	coqlibs := strings.Fields(makeVars["COQLIBS"])
	args = append(args, coqlibs...)
	args = append(args, "-destination-of", target)

	cmd := exec.Command("rocq", args...)
	output, err := cmd.Output()
	if err != nil {
		panic(fmt.Sprintf("failed to get destination of %s: %v", target, err))
	}
	return strings.TrimSpace(string(output))
}

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

func getDirVFiles(dir string) ([]string, error) {
	var sources []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".v") {
			sources = append(sources, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %v", dir, err)
	}

	return sources, nil
}

func gatherVFiles(paths []string) ([]string, error) {
	var sources []string

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("error accessing %s: %v", path, err)
		}

		if info.IsDir() {
			// Walk directory and find all .v files
			dirSources, err := getDirVFiles(path)
			if err != nil {
				return nil, fmt.Errorf("error walking directory %s: %v", path, err)
			}
			sources = append(sources, dirSources...)

		} else if strings.HasSuffix(path, ".v") {
			sources = append(sources, path)
		} else if strings.HasSuffix(path, ".vo") {
			sources = append(sources, setExtension(path, ".v"))
		} else {
			fmt.Fprintf(os.Stderr, "Skipping non-.v file: %s\n", path)
		}
	}

	return sources, nil
}

func installSources(quietMode bool, makeVars map[string]string, sources []string) error {
	for _, source := range sources {
		// NOTE: not installing glob files
		voFile := setExtension(source, ".vo")

		// Install the file
		dest := destinationOf(makeVars, voFile)
		if err := installFile(voFile, dest); err != nil {
			return err
		}

		if !quietMode {
			fmt.Printf("INSTALL %s to %s\n", voFile, dest)
		}
	}
	return nil
}

// installCmd represents the install command
var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install build outputs to COQLIB",
	Long: `Install .vo files, typically to an opam switch.

Takes a list of either .v files or directories (which are searched recursively
for all *.v files). Assumes all input files are compiled. Will automatically
install any dependencies required by the input .v files, using .rocqdeps.d.

Emulates the functionality of "make install" when using rocq makefile.
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		quietMode, _ := cmd.Flags().GetBool("quiet")
		installDeps, _ := cmd.Flags().GetBool("install-deps")
		if len(args) == 0 {
			// If no args, walk current directory
			args = []string{"."}
		}

		// Gather list of .v files
		sources, err := gatherVFiles(args)
		if err != nil {
			return err
		}

		if installDeps {
			// Parse dependency graph from .rocqdeps.d
			deps, err := depgraph.ParseRocqdep(".rocqdeps.d")
			if err != nil {
				return fmt.Errorf("failed to parse .rocqdeps.d: %v", err)
			}

			// Get all dependencies of the sources
			sources = depgraph.RocqDeps(deps, sources)
		}

		if len(sources) == 0 {
			fmt.Println("No .v files found to install")
			return nil
		}

		// Get makefile vars from _RocqProject or _CoqProject
		projFile := "_RocqProject"
		if _, err := os.Stat(projFile); os.IsNotExist(err) {
			// Fall back to _CoqProject
			projFile = "_CoqProject"
			if _, err := os.Stat(projFile); os.IsNotExist(err) {
				return fmt.Errorf("neither _RocqProject nor _CoqProject file found")
			}
		}

		makeVars := getRocqMakefileVars(projFile)

		// Install sources
		if err := installSources(quietMode, makeVars, sources); err != nil {
			return fmt.Errorf("error installing sources: %v", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)

	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "quiet mode (don't print list of installed files)")
	rootCmd.PersistentFlags().Bool("install-deps", true, "install dependencies of supplied files")

}
