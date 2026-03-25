package config

import (
	"os"
	"path/filepath"
	"testing"
)

func testHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("BIRA_HOME", home)
	return home
}

func TestSaveAndLoad(t *testing.T) {
	testHome(t)
	dir := t.TempDir()

	cfg := &Config{ProjectID: "abc123", ProjectName: "test-proj"}
	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, foundDir, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if foundDir != dir {
		t.Errorf("foundDir = %q, want %q", foundDir, dir)
	}
	if got.ProjectID != "abc123" || got.ProjectName != "test-proj" {
		t.Errorf("got %+v", got)
	}
}

func TestLoad_WalksUp(t *testing.T) {
	testHome(t)
	root := t.TempDir()

	cfg := &Config{ProjectID: "walk123", ProjectName: "walkproj"}
	if err := Save(root, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	child := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	got, foundDir, err := Load(child)
	if err != nil {
		t.Fatalf("Load from child: %v", err)
	}
	if foundDir != root {
		t.Errorf("foundDir = %q, want %q", foundDir, root)
	}
	if got.ProjectID != "walk123" {
		t.Errorf("ProjectID = %q, want walk123", got.ProjectID)
	}
}

func TestLoad_NotFound(t *testing.T) {
	testHome(t)
	dir := t.TempDir()

	_, _, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestExistsIn(t *testing.T) {
	testHome(t)
	dir := t.TempDir()

	if ExistsIn(dir) {
		t.Fatal("ExistsIn should be false before Save")
	}

	if err := Save(dir, &Config{ProjectID: "x"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if !ExistsIn(dir) {
		t.Fatal("ExistsIn should be true after Save")
	}
}

func TestResolveProjectID_FromFlag(t *testing.T) {
	testHome(t)
	id, err := ResolveProjectID("flag-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "flag-id" {
		t.Errorf("got %q, want flag-id", id)
	}
}

func TestResolveProjectID_FromConfig(t *testing.T) {
	testHome(t)
	dir := t.TempDir()

	if err := Save(dir, &Config{ProjectID: "cfg-id", ProjectName: "p"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Change to the directory with a config file
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	id, err := ResolveProjectID("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "cfg-id" {
		t.Errorf("got %q, want cfg-id", id)
	}
}
