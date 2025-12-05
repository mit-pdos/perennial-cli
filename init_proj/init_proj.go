package init_proj

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/mit-pdos/perennial-cli/git"
	"github.com/mit-pdos/perennial-cli/opam"
)

//go:embed init_template/*
var initTemplateFS embed.FS

// ProjectData holds the template data for project initialization
type ProjectData struct {
	Url         string
	Author      string
	Synopsis    string
	ProjectName string
}

func updatePerennialPin(opamPath string) error {
	contents, err := os.ReadFile(opamPath)
	if err != nil {
		panic("could not read back opam file")
	}
	f, err := opam.Parse(bytes.NewReader(contents))
	if err != nil {
		panic(fmt.Errorf("template opam does not parse: %w", err))
	}
	perennialUrl := "https://github.com/mit-pdos/perennial"
	commit, err := git.GetLatestCommit(perennialUrl)
	if err != nil {
		return fmt.Errorf("failed to get latest commit for perennial: %w", err)
	}
	f.AddPinDepend(opam.PinDepend{
		Package: "perennial",
		URL:     perennialUrl,
		Commit:  commit,
	})
	_, err = f.UpdateIndirectDependencies()
	if err != nil {
		return fmt.Errorf("failed to update indirect dependencies: %w", err)
	}
	if err := os.WriteFile(opamPath, []byte(f.String()), 0644); err != nil {
		panic("could not write back opam file")
	}
	fmt.Printf("added perennial dependency\n")

	return nil
}

func createGoMod(dir string, url string) error {
	// Check if go.mod exists, if not run go mod init
	goModPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		modName := strings.TrimPrefix(url, "https://")
		fmt.Printf("go mod init %s\n", modName)
		goModCmd := exec.Command("go", "mod", "init", modName)
		goModCmd.Dir = dir
		// go mod init outputs info messages on stderr; suppress those but print
		// if the command fails
		if output, err := goModCmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "%s", output)
			return fmt.Errorf("go mod init failed: %w", err)
		}
	}

	// fmt.Println("go get -tool github.com/mit-pdos/perennial-cli@latest")
	goGetCmd := exec.Command("go", "get", "-tool", "github.com/mit-pdos/perennial-cli@latest")
	goGetCmd.Dir = dir
	goGetCmd.Stdout = nil
	goGetCmd.Stderr = os.Stderr
	if err := goGetCmd.Run(); err != nil {
		return fmt.Errorf("go get failed: %w", err)
	}
	return nil
}

// New creates a new perennial project in the specified directory
func New(url, projectName, dir string) error {
	// Normalize URL
	if !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	url = strings.TrimSuffix(url, ".git")

	// Check if any files to be generated already exist
	filesToCheck := []string{
		projectName + ".opam",
		"Makefile",
		"_RocqProject",
		"goose.toml",
		".gitignore",
	}

	for _, file := range filesToCheck {
		filePath := filepath.Join(dir, file)
		if _, err := os.Stat(filePath); err == nil {
			return fmt.Errorf("file %s already exists, refusing to overwrite", file)
		}
	}

	if err := createGoMod(dir, url); err != nil {
		return err
	}

	// Create src directory
	srcDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		return fmt.Errorf("failed to create src directory: %w", err)
	}

	// Prepare template data
	data := ProjectData{
		Url:         url,
		Author:      "AUTHOR",   // placeholder
		Synopsis:    "SYNOPSIS", // placeholder
		ProjectName: projectName,
	}

	// Read and process template files
	opamFileName := projectName + ".opam"
	templateFiles := []struct {
		templatePath string
		outputPath   string
	}{
		{
			templatePath: "init_template/example.opam.tmpl",
			outputPath:   opamFileName,
		},
		{
			templatePath: "init_template/Makefile",
			outputPath:   "Makefile",
		},
		{
			templatePath: "init_template/_RocqProject",
			outputPath:   "_RocqProject",
		},
		{
			templatePath: "init_template/goose.toml",
			outputPath:   "goose.toml",
		},
		{
			templatePath: "init_template/gitignore",
			outputPath:   ".gitignore",
		},
	}

	for _, fileInfo := range templateFiles {
		content, err := initTemplateFS.ReadFile(fileInfo.templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", fileInfo.templatePath, err)
		}

		var output string
		if path.Ext(fileInfo.templatePath) == ".tmpl" {
			// Use text/template for processing
			tmpl := template.Must(template.New(fileInfo.outputPath).Parse(string(content)))
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				return fmt.Errorf("failed to execute template %s: %w", fileInfo.templatePath, err)
			}
			output = buf.String()
		} else {
			// Use content as-is
			output = string(content)
		}

		// Write output file
		fullOutputPath := filepath.Join(dir, fileInfo.outputPath)
		if err := os.WriteFile(fullOutputPath, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", fileInfo.outputPath, err)
		}
		fmt.Printf("created %s\n", fileInfo.outputPath)
	}

	if err := updatePerennialPin(filepath.Join(dir, opamFileName)); err != nil {
		return err
	}

	return nil
}
