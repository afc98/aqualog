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
	if err := migrate(database); err != nil {
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

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return err
	}

	type columnMigration struct {
		table string
		name  string
		sql   string
	}
	columns := []columnMigration{
		{"loggers", "role", `ALTER TABLE loggers ADD COLUMN role TEXT DEFAULT 'water_level'`},
		{"logger_files", "measurement_kind", `ALTER TABLE logger_files ADD COLUMN measurement_kind TEXT`},
		{"logger_files", "measurement_unit", `ALTER TABLE logger_files ADD COLUMN measurement_unit TEXT`},
		{"logger_data", "imported_value", `ALTER TABLE logger_data ADD COLUMN imported_value REAL`},
		{"logger_data", "imported_unit", `ALTER TABLE logger_data ADD COLUMN imported_unit TEXT`},
		{"logger_data", "imported_kind", `ALTER TABLE logger_data ADD COLUMN imported_kind TEXT`},
		{"logger_data", "baro_corrected_m", `ALTER TABLE logger_data ADD COLUMN baro_corrected_m REAL`},
		{"logger_data", "correction_run_id", `ALTER TABLE logger_data ADD COLUMN correction_run_id INTEGER`},
	}

	for _, c := range columns {
		exists, err := columnExists(db, c.table, c.name)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := db.Exec(c.sql); err != nil {
				return fmt.Errorf("migration add %s.%s: %w", c.table, c.name, err)
			}
		}
	}

	if _, err := db.Exec(`UPDATE loggers SET role = 'water_level' WHERE role IS NULL OR role = ''`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE logger_data SET imported_value = level_m WHERE imported_value IS NULL`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE logger_data SET imported_unit = 'm' WHERE imported_unit IS NULL OR imported_unit = ''`); err != nil {
		return err
	}
	if _, err := db.Exec(`UPDATE logger_data SET imported_kind = 'manufacturer_compensated_level_m' WHERE imported_kind IS NULL OR imported_kind = ''`); err != nil {
		return err
	}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS baro_assignments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scope TEXT NOT NULL,
			scope_id INTEGER NOT NULL,
			barologger_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(scope, scope_id),
			FOREIGN KEY (barologger_id) REFERENCES loggers(id) ON DELETE CASCADE
		);
		CREATE TABLE IF NOT EXISTS correction_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			site_id INTEGER NOT NULL,
			logger_id INTEGER,
			barologger_id INTEGER NOT NULL,
			method TEXT NOT NULL,
			density_mode TEXT NOT NULL,
			density_kg_m3 REAL NOT NULL,
			match_mode TEXT NOT NULL,
			max_gap_seconds INTEGER NOT NULL,
			notes TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE,
			FOREIGN KEY (logger_id) REFERENCES loggers(id) ON DELETE SET NULL,
			FOREIGN KEY (barologger_id) REFERENCES loggers(id) ON DELETE CASCADE
		);
		INSERT OR IGNORE INTO schema_migrations(version) VALUES (1);
	`)
	return err
}

func columnExists(db *sql.DB, table string, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
