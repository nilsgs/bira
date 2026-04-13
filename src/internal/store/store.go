package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// SavePlan writes markdown content to a .plan.md sidecar file.
// If content is empty, the sidecar is deleted (if it exists).
func SavePlan(jsonPath string, content string) error {
	planPath := strings.TrimSuffix(jsonPath, ".json") + ".plan.md"
	if content == "" {
		if err := os.Remove(planPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove plan: %w", err)
		}
		return nil
	}
	if err := EnsureDir(filepath.Dir(planPath)); err != nil {
		return err
	}
	return os.WriteFile(planPath, []byte(content), 0o644)
}

// LoadPlan reads the .plan.md sidecar for a .json file.
// Returns "" if the sidecar does not exist.
func LoadPlan(jsonPath string) (string, error) {
	planPath := strings.TrimSuffix(jsonPath, ".json") + ".plan.md"
	data, err := os.ReadFile(planPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read plan: %w", err)
	}
	return string(data), nil
}

// ResolveProjectBySlug finds a project ID by normalised name slug.
func ResolveProjectBySlug(slug string) (string, error) {
	dataDir, err := DataDir()
	if err != nil {
		return "", err
	}
	projectsDir := filepath.Join(dataDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		metaPath := filepath.Join(projectsDir, e.Name(), "meta.json")
		var meta struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := LoadJSON(metaPath, &meta); err != nil {
			continue
		}
		if normaliseSlug(meta.Name) == slug {
			return meta.ID, nil
		}
	}
	return "", nil
}

func normaliseSlug(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

// LoadProjectMeta loads the project's meta.json.
func LoadProjectMeta(projectID string) (*ProjectMeta, error) {
	dir, err := ProjectDir(projectID)
	if err != nil {
		return nil, err
	}
	var m ProjectMeta
	if err := LoadJSON(filepath.Join(dir, "meta.json"), &m); err != nil {
		return nil, fmt.Errorf("load project meta: %w", err)
	}
	return &m, nil
}

type ProjectMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
