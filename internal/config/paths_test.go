package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() error: %v", err)
	}

	expected := filepath.Join(tmp, dirName)
	if dir != expected {
		t.Errorf("DataDir() = %q, want %q", dir, expected)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat data dir: %v", err)
	}
	if !info.IsDir() {
		t.Error("DataDir() did not create a directory")
	}
}

func TestDBPath(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	dbPath, err := DBPath()
	if err != nil {
		t.Fatalf("DBPath() error: %v", err)
	}

	expected := filepath.Join(tmp, dirName, dbFileName)
	if dbPath != expected {
		t.Errorf("DBPath() = %q, want %q", dbPath, expected)
	}
}
