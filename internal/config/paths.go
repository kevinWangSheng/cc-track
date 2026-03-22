package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const dirName = ".cc-track"
const dbFileName = "data.db"

// DataDir returns the path to ~/.cc-track/, creating it if needed.
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("config: get home dir: %w", err)
	}
	dir := filepath.Join(home, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("config: create data dir: %w", err)
	}
	return dir, nil
}

// DBPath returns the path to ~/.cc-track/data.db.
func DBPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, dbFileName), nil
}
