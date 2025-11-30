package gooseproj

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// GooseConfig defines the format for the goose.toml file that defines a
// translation config.
type GooseConfig struct {
	GooseVersion string `toml:"goose-version"`
	// Path to directory with go.mod
	GoPath string `toml:"go-path"`
	// Packages to translate
	PkgPatterns []string `toml:"pkg-patterns"`
	// Root output directory for Rocq code
	RocqRoot string `toml:"rocq"`
}

func Parse(r io.Reader) (*GooseConfig, error) {
	cfg := &GooseConfig{}
	err := toml.NewDecoder(r).Decode(cfg)
	if err != nil {
		return nil, fmt.Errorf("error decoding config: %w", err)
	}
	return cfg, nil
}

func (c *GooseConfig) normalize() error {
	if c.GooseVersion == "" {
		c.GooseVersion = "latest"
	}
	if c.GoPath == "" {
		// Walk directory tree to find a unique go.mod file
		var goModPaths []string
		err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && d.Name() == "go.mod" {
				goModPaths = append(goModPaths, filepath.Dir(path))
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error walking directory: %w", err)
		}

		if len(goModPaths) == 0 {
			return fmt.Errorf("no go.mod file found")
		}
		if len(goModPaths) > 1 {
			return fmt.Errorf("multiple go.mod files found: %v", goModPaths)
		}
		c.GoPath = goModPaths[0]
	}

	if c.RocqRoot == "" {
		c.RocqRoot = "src"
	}

	if len(c.PkgPatterns) == 0 {
		c.PkgPatterns = []string{"./..."}
	}
	return nil
}
