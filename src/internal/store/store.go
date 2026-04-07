package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// DataDir returns the resolved path to the bira data directory.
// If BIRA_HOME is set, it is used directly; otherwise ~/.bira is returned.
func DataDir() (string, error) {
	if h := os.Getenv("BIRA_HOME"); h != "" {
		return h, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".bira"), nil
}

// ProjectDir returns the path to a specific project's data directory.
func ProjectDir(projectID string) (string, error) {
	dataDir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "projects", projectID), nil
}

// EnsureDir creates a directory (and parents) if it doesn't exist.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

// SaveJSON writes v as indented JSON to the given file path.
// Creates parent directories as needed.
func SaveJSON(path string, v any) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

// LoadJSON reads a JSON file into v.
func LoadJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

// ListDir returns the names of entries in a directory.
// Returns an empty slice if the directory does not exist.
func ListDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

// DeleteFile removes a file. Returns os.ErrNotExist if it doesn't exist.
func DeleteFile(path string) error {
	return os.Remove(path)
}

// DeleteDir removes a directory and all its contents.
func DeleteDir(path string) error {
	return os.RemoveAll(path)
}

// Exists returns true if the path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
