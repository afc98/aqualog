package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql schema_postgres.sql
var schemaFS embed.FS

type Backend string

const (
	BackendSQLite   Backend = "sqlite"
	BackendPostgres Backend = "postgres"
)

type PathSource string

const (
	PathSourceFlag    PathSource = "flag"
	PathSourceEnv     PathSource = "env"
	PathSourceDefault PathSource = "default"
)

type PathInfo struct {
	Path   string
	Source PathSource
}
type Config struct {
	Backend                 Backend
	SQLitePath, DatabaseURL string
	Source                  PathSource
}

var pathOverride, databaseURLOverride string
var sharedMu sync.Mutex
var sharedDB *sql.DB
var sharedConfig Config

func SetPathOverride(path string) {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	pathOverride = strings.TrimSpace(path)
	closeSharedLocked()
}
func SetDatabaseURLOverride(value string) {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	databaseURLOverride = strings.TrimSpace(value)
	closeSharedLocked()
}

func ResolveConfig() (Config, error) {
	return resolveConfig(pathOverride, databaseURLOverride, os.Getenv, runtime.GOOS, os.UserHomeDir)
}

func resolveConfig(pathFlag, urlFlag string, getenv func(string) string, goos string, userHomeDir func() (string, error)) (Config, error) {
	pathFlag, urlFlag = strings.TrimSpace(pathFlag), strings.TrimSpace(urlFlag)
	if pathFlag != "" && urlFlag != "" {
		return Config{}, errors.New("--db and --database-url cannot be used together")
	}
	if urlFlag != "" {
		return postgresConfig(urlFlag, PathSourceFlag)
	}
	if pathFlag != "" {
		return Config{Backend: BackendSQLite, SQLitePath: filepath.Clean(pathFlag), Source: PathSourceFlag}, nil
	}
	pathEnv, urlEnv := strings.TrimSpace(getenv("AQUALOG_DB")), strings.TrimSpace(getenv("AQUALOG_DATABASE_URL"))
	if pathEnv != "" && urlEnv != "" {
		return Config{}, errors.New("AQUALOG_DB and AQUALOG_DATABASE_URL cannot both be set")
	}
	if urlEnv != "" {
		return postgresConfig(urlEnv, PathSourceEnv)
	}
	if pathEnv != "" {
		return Config{Backend: BackendSQLite, SQLitePath: filepath.Clean(pathEnv), Source: PathSourceEnv}, nil
	}
	dir, err := defaultDataDir(getenv, goos, userHomeDir)
	if err != nil {
		return Config{}, err
	}
	return Config{Backend: BackendSQLite, SQLitePath: filepath.Join(dir, "aqualog.db"), Source: PathSourceDefault}, nil
}

func postgresConfig(value string, source PathSource) (Config, error) {
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" || strings.TrimPrefix(u.Path, "/") == "" {
		return Config{}, errors.New("database URL must be a valid postgres:// or postgresql:// URL with host and database")
	}
	return Config{Backend: BackendPostgres, DatabaseURL: value, Source: source}, nil
}

func GetDB() (*sql.DB, error) {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	cfg, err := ResolveConfig()
	if err != nil {
		return nil, err
	}
	if sharedDB != nil && sharedConfig == cfg {
		if err := sharedDB.Ping(); err == nil {
			return sharedDB, nil
		}
		closeSharedLocked()
	}
	sharedDB, err = Open(context.Background(), cfg)
	if err == nil {
		sharedConfig = cfg
	}
	return sharedDB, err
}

func Close() error {
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if sharedDB == nil {
		return nil
	}
	err := sharedDB.Close()
	sharedDB = nil
	sharedConfig = Config{}
	return err
}

func closeSharedLocked() {
	if sharedDB != nil {
		_ = sharedDB.Close()
	}
	sharedDB = nil
	sharedConfig = Config{}
}

func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	var driver, dsn string
	switch cfg.Backend {
	case BackendSQLite:
		if cfg.SQLitePath == "" {
			return nil, errors.New("SQLite path is empty")
		}
		if err := os.MkdirAll(filepath.Dir(cfg.SQLitePath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
		driver, dsn = "sqlite3", cfg.SQLitePath
	case BackendPostgres:
		driver, dsn = "pgx", cfg.DatabaseURL
	default:
		return nil, fmt.Errorf("unsupported database backend %q", cfg.Backend)
	}
	database, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s database: %w", cfg.Backend, err)
	}
	if cfg.Backend == BackendPostgres {
		database.SetMaxOpenConns(4)
		database.SetMaxIdleConns(2)
		database.SetConnMaxIdleTime(5 * time.Minute)
	}
	if err := database.PingContext(ctx); err != nil {
		database.Close()
		return nil, fmt.Errorf("failed to connect to %s database: %w", cfg.Backend, err)
	}
	if err := applySchema(ctx, database, cfg.Backend); err != nil {
		database.Close()
		return nil, err
	}
	return database, nil
}

func ResolvePath() (PathInfo, error) {
	cfg, err := ResolveConfig()
	if err != nil {
		return PathInfo{}, err
	}
	if cfg.Backend != BackendSQLite {
		return PathInfo{}, errors.New("selected database backend does not use a filesystem path")
	}
	return PathInfo{Path: cfg.SQLitePath, Source: cfg.Source}, nil
}

func Path() (string, error) {
	info, err := ResolvePath()
	if err != nil {
		return "", err
	}
	return info.Path, nil
}

func resolvePath(override string, getenv func(string) string, goos string, userHomeDir func() (string, error)) (PathInfo, error) {
	cfg, err := resolveConfig(override, "", getenv, goos, userHomeDir)
	if err != nil {
		return PathInfo{}, err
	}
	return PathInfo{Path: cfg.SQLitePath, Source: cfg.Source}, nil
}

func defaultDataDir(getenv func(string) string, goos string, userHomeDir func() (string, error)) (string, error) {
	switch goos {
	case "windows":
		if dir := strings.TrimSpace(getenv("LOCALAPPDATA")); dir != "" {
			return filepath.Join(dir, "Aqualog"), nil
		}
		home, err := userHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve user data directory: %w", err)
		}
		return filepath.Join(home, "AppData", "Local", "Aqualog"), nil
	case "darwin":
		home, err := userHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve user data directory: %w", err)
		}
		return filepath.Join(home, "Library", "Application Support", "Aqualog"), nil
	default:
		if dir := strings.TrimSpace(getenv("XDG_DATA_HOME")); dir != "" {
			return filepath.Join(dir, "aqualog"), nil
		}
		home, err := userHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to resolve user data directory: %w", err)
		}
		return filepath.Join(home, ".local", "share", "aqualog"), nil
	}
}

func applySchema(ctx context.Context, database *sql.DB, backend Backend) error {
	name := "schema.sql"
	if backend == BackendPostgres {
		name = "schema_postgres.sql"
	}
	b, err := schemaFS.ReadFile(name)
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}
	if backend == BackendPostgres {
		conn, err := database.Conn(ctx)
		if err != nil {
			return fmt.Errorf("acquire schema connection: %w", err)
		}
		defer conn.Close()
		if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtext('aqualog_schema'))`); err != nil {
			return fmt.Errorf("lock schema: %w", err)
		}
		defer conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock(hashtext('aqualog_schema'))`)
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin schema migration: %w", err)
		}
		defer tx.Rollback()
		for _, statement := range strings.Split(string(b), ";") {
			if strings.TrimSpace(statement) == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("failed to apply %s schema: %w", backend, err)
			}
		}
		if err := migratePostgresDatasetTimes(ctx, tx); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit schema migration: %w", err)
		}
		return nil
	}
	if _, err := database.ExecContext(ctx, string(b)); err != nil {
		return fmt.Errorf("failed to apply %s schema: %w", backend, err)
	}
	return migrateSQLite(database)
}

// migratePostgresDatasetTimes preserves the UTC clock fields stored by the
// previous CLI while removing timezone conversion from dataset columns.
func migratePostgresDatasetTimes(ctx context.Context, tx *sql.Tx) error {
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		return fmt.Errorf("read PostgreSQL schema version: %w", err)
	}
	if version >= 2 {
		return nil
	}
	for _, table := range []string{"logger_data", "logger_events", "manual_readings", "corrected_data"} {
		var dataType string
		err := tx.QueryRowContext(ctx, `SELECT data_type FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = $1 AND column_name = 'timestamp'`, table).Scan(&dataType)
		if err != nil {
			return fmt.Errorf("inspect %s.timestamp: %w", table, err)
		}
		if dataType == "timestamp with time zone" {
			statement := fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN timestamp TYPE TIMESTAMP WITHOUT TIME ZONE USING timestamp AT TIME ZONE 'UTC'`, table)
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("migrate %s.timestamp: %w", table, err)
			}
		} else if dataType != "timestamp without time zone" {
			return fmt.Errorf("unsupported %s.timestamp type %q", table, dataType)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES (2)`); err != nil {
		return fmt.Errorf("record PostgreSQL timestamp migration: %w", err)
	}
	return nil
}

func migrateSQLite(database *sql.DB) error {
	type columnMigration struct{ table, name, sql string }
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
		exists, err := columnExists(database, c.table, c.name)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := database.Exec(c.sql); err != nil {
				return fmt.Errorf("migration add %s.%s: %w", c.table, c.name, err)
			}
		}
	}
	_, err := database.Exec(`
		UPDATE loggers SET role = 'water_level' WHERE role IS NULL OR role = '';
		UPDATE logger_data SET imported_value = level_m WHERE imported_value IS NULL;
		UPDATE logger_data SET imported_unit = 'm' WHERE imported_unit IS NULL OR imported_unit = '';
		UPDATE logger_data SET imported_kind = 'manufacturer_compensated_level_m' WHERE imported_kind IS NULL OR imported_kind = '';
		INSERT OR IGNORE INTO schema_migrations(version) VALUES (1);
	`)
	return err
}

func columnExists(database *sql.DB, table, column string) (bool, error) {
	rows, err := database.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func RedactedLocation(cfg Config) string {
	if cfg.Backend == BackendSQLite {
		return cfg.SQLitePath
	}
	u, err := url.Parse(cfg.DatabaseURL)
	if err != nil {
		return "<invalid PostgreSQL URL>"
	}
	u.User = nil
	return u.Redacted()
}
