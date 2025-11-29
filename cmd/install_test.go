package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallFile(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a source .vo file
	srcFile := filepath.Join(tmpDir, "test.vo")
	content := []byte("test content for vo file")
	err := os.WriteFile(srcFile, content, 0644)
	require.NoError(t, err)

	// Define destination path
	destDir := filepath.Join(tmpDir, "dest", "subdir")
	destFile := filepath.Join(destDir, "test.vo")

	// Install the file
	err = installFile(srcFile, destFile)
	require.NoError(t, err)

	// Verify destination file exists
	assert.FileExists(t, destFile)

	// Verify content is correct
	destContent, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, content, destContent)

	// Verify file permissions
	info, err := os.Stat(destFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())

	// Verify directory was created
	dirInfo, err := os.Stat(destDir)
	require.NoError(t, err)
	assert.True(t, dirInfo.IsDir())
}

func TestInstallFileNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	// Try to install a file that doesn't exist
	srcFile := filepath.Join(tmpDir, "nonexistent.vo")
	destFile := filepath.Join(tmpDir, "dest.vo")

	err := installFile(srcFile, destFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestInstallFileOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source file
	srcFile := filepath.Join(tmpDir, "test.vo")
	newContent := []byte("new content")
	err := os.WriteFile(srcFile, newContent, 0644)
	require.NoError(t, err)

	// Create existing destination file with different content
	destFile := filepath.Join(tmpDir, "dest.vo")
	oldContent := []byte("old content")
	err = os.WriteFile(destFile, oldContent, 0644)
	require.NoError(t, err)

	// Install should overwrite
	err = installFile(srcFile, destFile)
	require.NoError(t, err)

	// Verify content was overwritten
	destContent, err := os.ReadFile(destFile)
	require.NoError(t, err)
	assert.Equal(t, newContent, destContent)
}
