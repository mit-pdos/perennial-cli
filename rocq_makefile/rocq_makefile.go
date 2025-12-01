package rocq_makefile

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

// GetMakefileVars extracts variable values from a Makefile.
//
// It does this by running make (using a temporary Makefile to provide a rule to
// just print values).
func GetMakefileVars(makefilePath string, vars []string) map[string]string {
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

// getRocqVarsForProjFile gets the COQLIBS and COQLIBINSTALL variables that rocq
// makefile generates for a given _RocqProject file.
func getRocqVarsForProjFile(projFile string) map[string]string {
	// 1. Run rocq makefile -f projFile -o <tmp Makefile.rocq>
	tmpPath := ".tmp.Makefile.rocq"
	defer os.Remove(tmpPath)
	defer os.Remove(tmpPath + ".conf")
	defer os.Remove("." + tmpPath + ".d")
	cmd := exec.Command("rocq", "makefile", "-f", projFile, "-o", tmpPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(fmt.Sprintf("failed to run rocq makefile: %v", err))
	}

	// 2. Get COQLIB and COQLIBINSTALL using GetMakefileVars
	return GetMakefileVars(tmpPath, []string{"COQLIBS", "COQLIBINSTALL"})
}

// GetRocqVars extracts the COQLIBS and COQLIBINSTALL variables that rocq
// makefile generates.
//
// It uses _RocqProject (falling back to _CoqProject) for the COQLIBS
// configuration.
func GetRocqVars() (map[string]string, error) {
	projFile := "_RocqProject"
	if _, err := os.Stat(projFile); os.IsNotExist(err) {
		// Fall back to _CoqProject
		projFile = "_CoqProject"
		if _, err := os.Stat(projFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("neither _RocqProject nor _CoqProject file found")
		}
	}
	return getRocqVarsForProjFile(projFile), nil
}

// DestinationOf determines the installation path for a compiled file. Returns
// the directory for the file `target`.
//
// It uses "rocq makefile -destination-of" to identify where the target file
// (typically a .vo file) should be installed, the same as the rocq makefile
// `install` rule.
func DestinationOf(makeVars map[string]string, target string) string {
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
	installRoot := makeVars["COQLIBINSTALL"]
	return path.Join(installRoot, strings.TrimSpace(string(output)))
}
