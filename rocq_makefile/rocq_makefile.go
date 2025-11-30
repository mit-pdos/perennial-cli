package rocq_makefile

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

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

func GetMakefileVarsForProjFile(projFile string) map[string]string {
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

func GetRocqVars() (map[string]string, error) {
	projFile := "_RocqProject"
	if _, err := os.Stat(projFile); os.IsNotExist(err) {
		// Fall back to _CoqProject
		projFile = "_CoqProject"
		if _, err := os.Stat(projFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("neither _RocqProject nor _CoqProject file found")
		}
	}
	return GetMakefileVarsForProjFile(projFile), nil
}

// DestinationOf uses rocq makefile to identify where target (a vo file, for example) should be installed.
//
// Expects makeVars to have COQLIBS and COQLIBINSTALL, which are computed once
// to avoid redundant calls to rocq makefile.
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
	return path.Join(installRoot, strings.TrimSpace(string(output)), path.Base(target))
}
