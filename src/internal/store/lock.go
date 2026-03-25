package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// Lock represents a held advisory lock on a project directory.
type Lock struct {
	file *os.File
}

// AcquireLock takes an exclusive advisory lock on the project directory.
// Blocks until the lock is available.
func AcquireLock(projectDir string) (*Lock, error) {
	lockPath := filepath.Join(projectDir, ".lock")
	if err := EnsureDir(projectDir); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	if err := lockFile(f); err != nil {
		f.Close()
		return nil, fmt.Errorf("acquire lock: %w", err)
	}
	return &Lock{file: f}, nil
}

// Release releases the advisory lock.
func (l *Lock) Release() {
	if l.file != nil {
		unlockFile(l.file)
		l.file.Close()
		l.file = nil
	}
}
