package store

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestNewID(t *testing.T) {
	id := NewID()
	if len(id) != 8 {
		t.Fatalf("expected 8-char ID, got %d chars: %q", len(id), id)
	}
	if !regexp.MustCompile(`^[0-9a-f]{8}$`).MatchString(id) {
		t.Fatalf("ID is not lowercase hex: %q", id)
	}

	id2 := NewID()
	if id == id2 {
		t.Fatalf("two consecutive IDs are identical: %q", id)
	}
}

func TestSaveAndLoadJSON(t *testing.T) {
	dir := t.TempDir()

	type sample struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	path := filepath.Join(dir, "sub", "data.json")
	want := sample{Name: "test", Count: 42}
	if err := SaveJSON(path, &want); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	if !Exists(path) {
		t.Fatal("file should exist after SaveJSON")
	}

	var got sample
	if err := LoadJSON(path, &got); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if got != want {
		t.Fatalf("roundtrip mismatch: got %+v, want %+v", got, want)
	}
}

func TestLoadJSON_Missing(t *testing.T) {
	err := LoadJSON(filepath.Join(t.TempDir(), "nope.json"), &struct{}{})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("expected os.ErrNotExist-wrapped error, got: %v", err)
	}
}

func TestEnsureDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "a", "b", "c")
	if err := EnsureDir(dir); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected directory")
	}
}

func TestListDir(t *testing.T) {
	dir := t.TempDir()
	// Create two files
	os.WriteFile(filepath.Join(dir, "a.json"), []byte("{}"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.json"), []byte("{}"), 0o644)

	names, err := ListDir(dir)
	if err != nil {
		t.Fatalf("ListDir: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(names))
	}
}

func TestListDir_Missing(t *testing.T) {
	names, err := ListDir(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got: %v", err)
	}
	if names != nil {
		t.Fatalf("expected nil slice for missing dir, got: %v", names)
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")

	if Exists(path) {
		t.Fatal("file should not exist yet")
	}

	os.WriteFile(path, []byte("hi"), 0o644)
	if !Exists(path) {
		t.Fatal("file should exist after write")
	}
}

func TestDataDir_Default(t *testing.T) {
	t.Setenv("BIRA_HOME", "")
	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	if filepath.Base(dir) != ".bira" {
		t.Fatalf("expected path ending in .bira, got %q", dir)
	}
}

func TestDataDir_EnvVar(t *testing.T) {
	want := t.TempDir()
	t.Setenv("BIRA_HOME", want)
	got, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	if got != want {
		t.Fatalf("DataDir = %q, want %q", got, want)
	}
}
