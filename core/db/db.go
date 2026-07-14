package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

// GetDB returns a ready-to-use database handle
func GetDB() (*sql.DB, error) {
	dbPath, err := dbPathFromExecutable()
	if err != nil {
		return nil, err
	}

	//Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	// Open DB (created automatically if missing)
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	//Apply schema (runs safely every time)
	if err := applySchema(database); err != nil {
		return nil, err
	}

	return database, nil
}

// Resolve DB path relative to the executable
func dbPathFromExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}

	exeDir := filepath.Dir(exe)
	return filepath.Join(exeDir, ".aqualog", "aqualog.db"), nil
}

func Path() (string, error) {
	return dbPathFromExecutable()
}

func applySchema(db *sql.DB) error {
	schemaBytes, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	_, err = db.Exec(string(schemaBytes))
	if err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}

	return nil
}
