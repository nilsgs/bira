package config

import (
	"fmt"
	"os"
	"path/filepath"

	"bira/internal/store"
)

const FileName = ".bira"

// Config is the per-repo bira configuration written at the repo root.
type Config struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
}

// Save writes the config file to the given directory.
func Save(dir string, cfg *Config) error {
	return store.SaveJSON(filepath.Join(dir, FileName), cfg)
}

// Load walks up from startDir looking for a .bira config file.
// Returns the config and the directory where it was found.
func Load(startDir string) (*Config, string, error) {
	dir := startDir
	for {
		path := filepath.Join(dir, FileName)
		if store.Exists(path) {
			var cfg Config
			if err := store.LoadJSON(path, &cfg); err != nil {
				return nil, "", fmt.Errorf("read %s: %w", path, err)
			}
			return &cfg, dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, "", fmt.Errorf("no %s config found (run 'bira init' first)", FileName)
}

// ExistsIn returns true if a .bira file exists in the given directory.
func ExistsIn(dir string) bool {
	return store.Exists(filepath.Join(dir, FileName))
}

// ResolveProjectID returns the project ID from the --project flag or from the
// .bira config file found by walking up from the current directory.
func ResolveProjectID(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	cfg, _, err := Load(cwd)
	if err != nil {
		return "", err
	}
	return cfg.ProjectID, nil
}
