// Package dbcopy contains explicit, one-shot database copy operations.
package dbcopy

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const CurrentSQLiteSchemaVersion = 1

type tableSpec struct {
	name    string
	columns []string
}

// copyOrder respects all foreign-key dependencies. schema_migrations is owned
// by each backend's migration runner and is deliberately not copied.
var copyOrder = []tableSpec{
	{"projects", []string{"id", "name", "description", "created_at"}},
	{"sites", []string{"id", "project_id", "name", "latitude", "longitude", "description"}},
	{"loggers", []string{"id", "site_id", "name", "model", "serial_number", "role", "depth_offset_cm"}},
	{"logger_files", []string{"id", "logger_id", "timestamp", "file", "type", "measurement_kind", "measurement_unit"}},
	{"correction_runs", []string{"id", "site_id", "logger_id", "barologger_id", "method", "density_mode", "density_kg_m3", "match_mode", "max_gap_seconds", "notes", "created_at"}},
	{"logger_data", []string{"id", "logger_id", "logger_file_id", "timestamp", "level_m", "imported_value", "imported_unit", "imported_kind", "baro_corrected_m", "correction_run_id", "temp_c", "sal_psu", "ec_us"}},
	{"logger_events", []string{"id", "logger_id", "timestamp", "event_type", "notes"}},
	{"manual_readings", []string{"id", "site_id", "timestamp", "value", "notes"}},
	{"corrected_data", []string{"id", "site_id", "logger_data_id", "timestamp", "corrected_value", "temp_c", "sal_psu", "ec_us"}},
	{"baro_assignments", []string{"id", "scope", "scope_id", "barologger_id", "created_at"}},
}

// OpenSQLiteReadOnly opens an existing SQLite database without permitting the
// migration process (or SQLite connection setup) to create or modify it.
func OpenSQLiteReadOnly(path string) (*sql.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("SQLite source path is required")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve SQLite source path: %w", err)
	}
	if _, err := os.Stat(absolute); err != nil {
		return nil, fmt.Errorf("access SQLite source: %w", err)
	}
	uriPath := filepath.ToSlash(absolute)
	if filepath.VolumeName(absolute) != "" && !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}
	sourceURL := &url.URL{Scheme: "file", Path: uriPath}
	dsn := sourceURL.String() + "?mode=ro&_query_only=1"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQLite source: %w", err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// SQLiteToPostgres copies a current SQLite database into an initialized, empty
// PostgreSQL database. All target changes occur in one transaction.
func SQLiteToPostgres(ctx context.Context, source, target *sql.DB) (err error) {
	if source == nil || target == nil {
		return fmt.Errorf("source and target databases are required")
	}
	sourceTx, err := source.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("begin SQLite read transaction: %w", err)
	}
	defer sourceTx.Rollback()
	if err := validateSQLiteSource(ctx, sourceTx); err != nil {
		return err
	}

	tx, err := target.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin PostgreSQL copy transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = requireEmptyTarget(ctx, tx); err != nil {
		return err
	}

	counts := make(map[string]int64, len(copyOrder))
	for _, table := range copyOrder {
		if counts[table.name], err = copyTable(ctx, sourceTx, tx, table); err != nil {
			return err
		}
	}
	if err = resetIdentitySequences(ctx, tx); err != nil {
		return err
	}
	if err = validateTarget(ctx, tx, counts); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit PostgreSQL copy: %w", err)
	}
	return nil
}

func validateSQLiteSource(ctx context.Context, source *sql.Tx) error {
	var version int
	if err := source.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version); err != nil {
		return fmt.Errorf("source is not a supported Aqualog SQLite database: %w", err)
	}
	if version != CurrentSQLiteSchemaVersion {
		return fmt.Errorf("SQLite source schema version is %d; expected %d", version, CurrentSQLiteSchemaVersion)
	}
	for _, table := range copyOrder {
		rows, err := source.QueryContext(ctx, "PRAGMA table_info("+table.name+")")
		if err != nil {
			return fmt.Errorf("inspect SQLite table %s: %w", table.name, err)
		}
		found := map[string]bool{}
		for rows.Next() {
			var cid, notNull, pk int
			var name, typ string
			var defaultValue any
			if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
				rows.Close()
				return fmt.Errorf("inspect SQLite table %s: %w", table.name, err)
			}
			found[name] = true
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("inspect SQLite table %s: %w", table.name, err)
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("inspect SQLite table %s: %w", table.name, err)
		}
		for _, column := range table.columns {
			if !found[column] {
				return fmt.Errorf("SQLite source table %s is missing column %s", table.name, column)
			}
		}
	}
	return nil
}

func requireEmptyTarget(ctx context.Context, tx *sql.Tx) error {
	for _, table := range copyOrder {
		var exists bool
		query := "SELECT EXISTS (SELECT 1 FROM " + table.name + " LIMIT 1)"
		if err := tx.QueryRowContext(ctx, query).Scan(&exists); err != nil {
			return fmt.Errorf("inspect PostgreSQL table %s: %w", table.name, err)
		}
		if exists {
			return fmt.Errorf("PostgreSQL destination must be empty; table %s contains data", table.name)
		}
	}
	return nil
}

func copyTable(ctx context.Context, source *sql.Tx, tx *sql.Tx, table tableSpec) (int64, error) {
	columnList := strings.Join(table.columns, ", ")
	rows, err := source.QueryContext(ctx, "SELECT "+columnList+" FROM "+table.name+" ORDER BY id")
	if err != nil {
		return 0, fmt.Errorf("read SQLite table %s: %w", table.name, err)
	}
	defer rows.Close()

	values := make([]any, len(table.columns))
	destinations := make([]any, len(values))
	placeholders := make([]string, len(values))
	for i := range values {
		destinations[i] = &values[i]
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	insert := "INSERT INTO " + table.name + " (" + columnList + ") VALUES (" + strings.Join(placeholders, ", ") + ")"
	var count int64
	for rows.Next() {
		if err := rows.Scan(destinations...); err != nil {
			return 0, fmt.Errorf("read SQLite row from %s: %w", table.name, err)
		}
		if _, err := tx.ExecContext(ctx, insert, values...); err != nil {
			return 0, fmt.Errorf("copy row %d into PostgreSQL table %s: %w", count+1, table.name, err)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("read SQLite table %s: %w", table.name, err)
	}
	return count, nil
}

func resetIdentitySequences(ctx context.Context, tx *sql.Tx) error {
	for _, table := range copyOrder {
		// setval(..., false) makes the next generated ID 1 for an empty table.
		query := "SELECT setval(pg_get_serial_sequence('" + table.name + "', 'id'), COALESCE(MAX(id), 1), MAX(id) IS NOT NULL) FROM " + table.name
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("reset PostgreSQL identity for %s: %w", table.name, err)
		}
	}
	return nil
}

func validateTarget(ctx context.Context, tx *sql.Tx, expected map[string]int64) error {
	for _, table := range copyOrder {
		var actual int64
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table.name).Scan(&actual); err != nil {
			return fmt.Errorf("validate PostgreSQL table %s: %w", table.name, err)
		}
		if actual != expected[table.name] {
			return fmt.Errorf("row-count mismatch for %s: copied %d, found %d", table.name, expected[table.name], actual)
		}
	}

	checks := []struct{ child, parent, join, present string }{
		{"sites", "projects", "c.project_id = p.id", "TRUE"},
		{"loggers", "sites", "c.site_id = p.id", "TRUE"},
		{"logger_files", "loggers", "c.logger_id = p.id", "TRUE"},
		{"logger_data", "loggers", "c.logger_id = p.id", "TRUE"},
		{"logger_data", "logger_files", "c.logger_file_id = p.id", "TRUE"},
		{"logger_data", "correction_runs", "c.correction_run_id = p.id", "c.correction_run_id IS NOT NULL"},
		{"logger_events", "loggers", "c.logger_id = p.id", "TRUE"},
		{"manual_readings", "sites", "c.site_id = p.id", "TRUE"},
		{"corrected_data", "sites", "c.site_id = p.id", "TRUE"},
		{"corrected_data", "logger_data", "c.logger_data_id = p.id", "TRUE"},
		{"baro_assignments", "loggers", "c.barologger_id = p.id", "TRUE"},
		{"correction_runs", "sites", "c.site_id = p.id", "TRUE"},
		{"correction_runs", "loggers", "c.barologger_id = p.id", "TRUE"},
		{"correction_runs", "loggers", "c.logger_id = p.id", "c.logger_id IS NOT NULL"},
	}
	for _, check := range checks {
		query := "SELECT EXISTS (SELECT 1 FROM " + check.child + " c LEFT JOIN " + check.parent + " p ON " + check.join + " WHERE " + check.present + " AND p.id IS NULL)"
		var invalid bool
		if err := tx.QueryRowContext(ctx, query).Scan(&invalid); err != nil {
			return fmt.Errorf("validate relationship %s to %s: %w", check.child, check.parent, err)
		}
		if invalid {
			return fmt.Errorf("invalid relationship from %s to %s after copy", check.child, check.parent)
		}
	}
	return nil
}
