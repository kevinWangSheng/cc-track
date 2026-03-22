package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const maxFieldBytes = 10240

type Store struct {
	db *sql.DB
}

// Open opens a SQLite database at the given path and runs migrations.
func Open(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("store: open db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: set busy_timeout: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	var version int
	// schema_version table might not exist yet
	row := s.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version")
	if err := row.Scan(&version); err != nil {
		// Table doesn't exist, run full schema
		version = 0
	}

	if version < 1 {
		if _, err := s.db.Exec(schemaSQL); err != nil {
			return fmt.Errorf("store: run schema: %w", err)
		}
		version = 1
	}

	if version < 2 {
		if _, err := s.db.Exec(migrationV2SQL); err != nil {
			return fmt.Errorf("store: run migration v2: %w", err)
		}
		version = 2
	}

	if _, err := s.db.Exec(
		"INSERT OR REPLACE INTO schema_version (version) VALUES (?)",
		currentSchemaVersion,
	); err != nil {
		return fmt.Errorf("store: set schema version: %w", err)
	}
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...[truncated]"
}
