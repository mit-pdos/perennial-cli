package init_proj_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mit-pdos/perennial-cli/init_proj"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeProject(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirTemp("", "perennial-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	url := "https://github.com/example/test-project"
	projectName := "test-project"

	// Initialize the project
	err = init_proj.New(url, projectName, tmpDir)
	require.NoError(t, err)

	// Verify that all expected files were created
	expectedFiles := []string{
		"test-project.opam",
		"Makefile",
		"_RocqProject",
		"goose.toml",
		".gitignore",
		"go.mod",
		"go.sum",
	}

	for _, file := range expectedFiles {
		filePath := filepath.Join(tmpDir, file)
		_, err := os.Stat(filePath)
		assert.NoError(t, err, "file %s should exist", file)
	}

	// Verify src directory was created
	srcDir := filepath.Join(tmpDir, "src")
	info, err := os.Stat(srcDir)
	assert.NoError(t, err, "src directory should exist")
	assert.True(t, info.IsDir(), "src should be a directory")

	// Verify opam file content
	opamPath := filepath.Join(tmpDir, "test-project.opam")
	opamContent, err := os.ReadFile(opamPath)
	require.NoError(t, err)
	opamStr := string(opamContent)

	assert.Contains(t, opamStr, "homepage: \"https://github.com/example/test-project\"")
	assert.Contains(t, opamStr, "bug-reports: \"https://github.com/example/test-project/issues\"")
	assert.Contains(t, opamStr, "dev-repo: \"git+https://github.com/example/test-project.git\"")
	assert.Contains(t, opamStr, "maintainer: \"AUTHOR\"")
	assert.Contains(t, opamStr, "synopsis: \"SYNOPSIS\"")

	// Verify go.mod was created with correct module name
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	require.NoError(t, err)
	assert.Contains(t, string(goModContent), "module github.com/example/test-project")
}

func TestInitializeProject_URLNormalization(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "perennial-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test with URL without https:// prefix
	url := "github.com/example/test-project"
	projectName := "test-project"

	err = init_proj.New(url, projectName, tmpDir)
	require.NoError(t, err)

	// Verify opam file has normalized URL
	opamPath := filepath.Join(tmpDir, "test-project.opam")
	opamContent, err := os.ReadFile(opamPath)
	require.NoError(t, err)

	assert.Contains(t, string(opamContent), "homepage: \"https://github.com/example/test-project\"")
}

func TestInitializeProject_RefusesOverwrite(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "perennial-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a file that would conflict
	conflictFile := filepath.Join(tmpDir, "test-project.opam")
	err = os.WriteFile(conflictFile, []byte("existing content"), 0644)
	require.NoError(t, err)

	url := "https://github.com/example/test-project"
	projectName := "test-project"

	// Should fail because file already exists
	err = init_proj.New(url, projectName, tmpDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestInitializeProject_WithExistingGoMod(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "perennial-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create an existing go.mod with different module name
	goModPath := filepath.Join(tmpDir, "go.mod")
	existingGoMod := "module github.com/different/module\n\ngo 1.21\n"
	err = os.WriteFile(goModPath, []byte(existingGoMod), 0644)
	require.NoError(t, err)

	url := "https://github.com/example/test-project"
	projectName := "test-project"

	err = init_proj.New(url, projectName, tmpDir)
	require.NoError(t, err)

	// Verify go.mod was not overwritten
	goModContent, err := os.ReadFile(goModPath)
	require.NoError(t, err)
	// Should still have the original module name since we don't overwrite
	assert.Contains(t, string(goModContent), "github.com/different/module")
}

func TestInitializeProject_ProjectNameExtraction(t *testing.T) {
	tests := []struct {
		url          string
		projectName  string
		expectedName string
	}{
		{
			url:          "https://github.com/user/my-project",
			projectName:  "my-project",
			expectedName: "my-project",
		},
		{
			url:          "https://github.com/org/another-proj.git",
			projectName:  "another-proj",
			expectedName: "another-proj",
		},
	}

	for _, tt := range tests {
		t.Run(tt.projectName, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "perennial-init-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			err = init_proj.New(tt.url, tt.projectName, tmpDir)
			require.NoError(t, err)

			// Verify opam file has correct name
			opamFile := filepath.Join(tmpDir, tt.expectedName+".opam")
			_, err = os.Stat(opamFile)
			assert.NoError(t, err, "opam file should exist")
		})
	}
}

func TestInitializeProject_TemplateSubstitutions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "perennial-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	url := "https://github.com/testorg/myproject"
	projectName := "myproject"

	err = init_proj.New(url, projectName, tmpDir)
	require.NoError(t, err)

	// Read the opam file and check all substitutions
	opamPath := filepath.Join(tmpDir, "myproject.opam")
	content, err := os.ReadFile(opamPath)
	require.NoError(t, err)
	opamStr := string(content)

	// Ensure no unsubstituted placeholders remain
	assert.NotContains(t, opamStr, "{{Url}}")
	assert.NotContains(t, opamStr, "{{ProjectName}}")

	// Verify specific substitutions
	assert.Contains(t, opamStr, url)
	assert.Contains(t, opamStr, "AUTHOR") // Placeholder values
	assert.Contains(t, opamStr, "SYNOPSIS")
}

func TestInitializeProject_GitIgnoreCreated(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "perennial-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	url := "https://github.com/example/test-project"
	projectName := "test-project"

	err = init_proj.New(url, projectName, tmpDir)
	require.NoError(t, err)

	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	content, err := os.ReadFile(gitignorePath)
	require.NoError(t, err)

	gitignoreStr := string(content)
	// Verify some expected patterns
	assert.Contains(t, gitignoreStr, "*.vo")
	assert.Contains(t, gitignoreStr, "*.vos")
	assert.Contains(t, gitignoreStr, ".rocqdeps.d")
	assert.Contains(t, gitignoreStr, ".goose-output")
}

func TestProjectNameExtraction(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://github.com/user/project", "project"},
		{"https://github.com/user/project.git", "project"},
		{"github.com/user/my-proj", "my-proj"},
		{"github.com/user/my-proj.git", "my-proj"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			// Extract project name using filepath.Base and TrimSuffix
			projectName := filepath.Base(tt.url)
			projectName = strings.TrimSuffix(projectName, ".git")
			assert.Equal(t, tt.expected, projectName)
		})
	}
}
