package backup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupRealDir(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "skills")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	backed, err := Backup(dir)
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}

	// original is gone
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Errorf("original dir should be gone after backup")
	}
	// backup exists with correct prefix
	if !strings.HasPrefix(filepath.Base(backed), "skills_backup_") {
		t.Errorf("backup name %q should start with 'skills_backup_'", filepath.Base(backed))
	}
	if _, err := os.Lstat(backed); err != nil {
		t.Errorf("backup dir should exist: %v", err)
	}
}

func TestBackupAbsentDir(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "skills")
	// dir deliberately not created

	backed, err := Backup(dir)
	if err != nil {
		t.Fatalf("Backup on absent dir returned error: %v", err)
	}
	if backed != "" {
		t.Errorf("Backup of absent dir should return empty path, got %q", backed)
	}
}

func TestBackupIdempotent(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "skills")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	first, err := Backup(dir)
	if err != nil {
		t.Fatalf("first Backup: %v", err)
	}

	// dir is now gone; calling Backup again should be a no-op
	second, err := Backup(dir)
	if err != nil {
		t.Fatalf("second Backup: %v", err)
	}
	if second != "" {
		t.Errorf("second Backup should return empty (no-op), got %q", second)
	}

	// first backup still intact
	if _, err := os.Lstat(first); err != nil {
		t.Errorf("first backup should still exist: %v", err)
	}
}
