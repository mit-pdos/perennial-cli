package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sync"

	gooseproj "github.com/mit-pdos/perennial-cli/goose_proj"
	"github.com/spf13/cobra"
)

func runGooseCmd(localPath string, cmdName string, args []string) error {
	if localPath != "" {
		// Compile local goose binary to a temporary file
		tmpFile, err := os.CreateTemp("", fmt.Sprintf("goose-%s-*", cmdName))
		if err != nil {
			return fmt.Errorf("error creating temp file: %w", err)
		}
		tmpPath := tmpFile.Name()
		tmpFile.Close()
		defer os.Remove(tmpPath)

		buildCmd := exec.Command("go", "build", "-o", tmpPath, fmt.Sprintf("./cmd/%s", cmdName))
		buildCmd.Stderr = os.Stderr
		buildCmd.Dir = localPath
		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("error building local goose: %w", err)
		}

		cmd := exec.Command(tmpPath, args...)
		cmd.Stderr = os.Stderr
		return cmd.Run()
	} else {
		goArgs := append([]string{"tool", cmdName}, args...)
		cmd := exec.Command("go", goArgs...)
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}

// gooseCmd represents the goose command
var gooseCmd = &cobra.Command{
	Use:   "goose",
	Short: "Translate code with goose",
	Long:  `Run goose to translate a project configured with goose.toml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		localPath, _ := cmd.Flags().GetString("local")
		configContents, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("could not read config file: %w", err)
		}
		config, err := gooseproj.Parse(bytes.NewReader(configContents))
		if err != nil {
			return fmt.Errorf("error parsing config: %w", err)
		}
		configDir := path.Dir(configPath)
		var wg sync.WaitGroup
		var gooseErr, proofgenErr error
		wg.Add(2)
		go func() {
			gooseErr = runGooseCmd(localPath, "goose",
				append([]string{
					"-out", path.Join(config.RocqRoot, "code"),
					"-dir", configDir,
				}, config.PkgPatterns...))
			wg.Done()
		}()
		go func() {
			proofgenErr = runGooseCmd(localPath, "proofgen",
				append([]string{
					"-out", path.Join(config.RocqRoot, "generatedproof"),
					// directory with .v.toml files
					"-configdir", path.Join(config.RocqRoot, "code"),
					"-dir", configDir,
				}, config.PkgPatterns...))
			wg.Done()
		}()
		wg.Wait()
		if gooseErr != nil || proofgenErr != nil {
			return fmt.Errorf("error running goose")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(gooseCmd)

	gooseCmd.PersistentFlags().String("config", "goose.toml", "Path to the goose configuration file")
	gooseCmd.PersistentFlags().String("local", "", "Path to local goose repo to compile and run")
}
