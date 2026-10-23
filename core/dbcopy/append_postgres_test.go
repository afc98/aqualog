package dbcopy_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/dbcopy"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var schemaCounter atomic.Int64

func postgresTarget(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("AQUALOG_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("AQUALOG_TEST_POSTGRES_URL is not set")
	}
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close() })
	schema := fmt.Sprintf("append_test_%d_%d", time.Now().UnixNano(), schemaCounter.Add(1))
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
	})
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	target, err := db.Open(t.Context(), db.Config{Backend: db.BackendPostgres, DatabaseURL: u.String()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { target.Close() })
	return target
}

func execSQL(t *testing.T, database *sql.DB, query string) {
	t.Helper()
	if _, err := database.Exec(query); err != nil {
		t.Fatal(err)
	}
}

func sqliteSource(t *testing.T, name string) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.db")
	source, err := db.Open(t.Context(), db.Config{Backend: db.BackendSQLite, SQLitePath: path})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { source.Close() })
	if _, err := source.Exec("INSERT INTO projects(id,name,description) VALUES (42, ?, 'description')", name); err != nil {
		t.Fatal(err)
	}
	execSQL(t, source, `
INSERT INTO sites(id,project_id,name,latitude,longitude) VALUES (43,42,'Shared site',51.5,-0.1);
INSERT INTO loggers(id,site_id,name,role) VALUES (44,43,'Water','water_level'),(45,43,'Baro','barometric');
INSERT INTO logger_files(id,logger_id,file,type,measurement_kind,measurement_unit) VALUES (46,44,'original/path.csv','solinst','pressure','kPa');
INSERT INTO correction_runs(id,site_id,logger_id,barologger_id,method,density_mode,density_kg_m3,match_mode,max_gap_seconds) VALUES (47,43,44,45,'barometric','freshwater',1000,'nearest',600),(48,43,NULL,45,'barometric','freshwater',1000,'nearest',600);
INSERT INTO logger_data(id,logger_id,logger_file_id,timestamp,level_m,imported_value,imported_unit,imported_kind,baro_corrected_m,correction_run_id,temp_c) VALUES (49,44,46,'2026-01-01T00:00:00Z',1.2,110,'kPa','pressure',1.1,47,0),(50,44,46,'2026-01-01T01:00:00Z',NULL,NULL,NULL,NULL,NULL,NULL,NULL);
INSERT INTO logger_events(id,logger_id,timestamp,event_type,notes) VALUES (51,44,'2026-01-01T00:00:00Z','installed','event');
INSERT INTO manual_readings(id,site_id,timestamp,value,notes) VALUES (52,43,'2026-01-01T00:00:00Z',12.5,'manual');
INSERT INTO corrected_data(id,site_id,logger_data_id,timestamp,corrected_value) VALUES (53,43,49,'2026-01-01T00:00:00Z',12.5);
INSERT INTO baro_assignments(id,scope,scope_id,barologger_id) VALUES (54,'project',42,45),(55,'site',43,45);
`)
	return source
}

var tables = []string{"projects", "sites", "loggers", "logger_files", "correction_runs", "logger_data", "logger_events", "manual_readings", "corrected_data", "baro_assignments"}

func snapshot(t *testing.T, target *sql.DB, maxIDs map[string]int64) string {
	t.Helper()
	data := map[string]string{}
	for _, table := range tables {
		query := "SELECT COALESCE(json_agg(t ORDER BY id)::text, '[]') FROM " + table + " t"
		if maxIDs != nil {
			query += fmt.Sprintf(" WHERE id <= %d", maxIDs[table])
		}
		var value string
		if err := target.QueryRow(query).Scan(&value); err != nil {
			t.Fatal(err)
		}
		data[table] = value
	}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestAppendPostgresRemapsWholeDatabase(t *testing.T) {
	target := postgresTarget(t)
	first := sqliteSource(t, "First")
	if err := dbcopy.SQLiteToPostgres(t.Context(), first, target); err != nil {
		t.Fatal(err)
	}
	maxIDs := map[string]int64{}
	for _, table := range tables {
		var id int64
		if err := target.QueryRow("SELECT MAX(id) FROM " + table).Scan(&id); err != nil {
			t.Fatal(err)
		}
		maxIDs[table] = id
	}
	before := snapshot(t, target, maxIDs)
	second := sqliteSource(t, "Second")
	execSQL(t, second, "INSERT INTO projects(id,name) VALUES (100,'Third'); INSERT INTO sites(id,project_id,name) VALUES (101,100,'Third site'); INSERT INTO baro_assignments(id,scope,scope_id,barologger_id) VALUES (102,'project',100,45)")
	result, err := dbcopy.AppendSQLiteToPostgres(t.Context(), second, target)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int64{"projects": 2, "sites": 2, "loggers": 2, "logger_files": 1, "correction_runs": 2, "logger_data": 2, "logger_events": 1, "manual_readings": 1, "corrected_data": 1, "baro_assignments": 3}
	if !reflect.DeepEqual(result.Counts, want) {
		t.Fatalf("counts = %v", result.Counts)
	}
	if len(result.ProjectIDs) != 2 || result.ProjectIDs[42] <= 42 {
		t.Fatalf("project mappings = %v", result.ProjectIDs)
	}
	if got := snapshot(t, target, maxIDs); got != before {
		t.Fatal("existing records changed")
	}
	// Verify every mapped column, not just relationship existence. The default
	// migration left a complete overlapping set of IDs in the destination.
	mapped := map[string]map[int64]int64{}
	for _, table := range tables {
		mapped[table] = map[int64]int64{}
		sourceRows, err := second.Query("SELECT id FROM " + table + " ORDER BY id")
		if err != nil {
			t.Fatal(err)
		}
		var oldIDs []int64
		for sourceRows.Next() {
			var id int64
			if err := sourceRows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			oldIDs = append(oldIDs, id)
		}
		if err := sourceRows.Err(); err != nil {
			t.Fatal(err)
		}
		sourceRows.Close()
		rows, err := target.Query("SELECT id FROM "+table+" WHERE id > $1 ORDER BY id", maxIDs[table])
		if err != nil {
			t.Fatal(err)
		}
		i := 0
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				t.Fatal(err)
			}
			if i >= len(oldIDs) {
				t.Fatal("extra row")
			}
			mapped[table][oldIDs[i]] = id
			i++
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
		if i != len(oldIDs) {
			t.Fatal("missing rows")
		}
	}
	references := map[string]map[string]string{
		"sites": {"project_id": "projects"}, "loggers": {"site_id": "sites"}, "logger_files": {"logger_id": "loggers"},
		"correction_runs": {"site_id": "sites", "logger_id": "loggers", "barologger_id": "loggers"},
		"logger_data":     {"logger_id": "loggers", "logger_file_id": "logger_files", "correction_run_id": "correction_runs"},
		"logger_events":   {"logger_id": "loggers"}, "manual_readings": {"site_id": "sites"},
		"corrected_data": {"site_id": "sites", "logger_data_id": "logger_data"}, "baro_assignments": {"barologger_id": "loggers"},
	}
	for table, columns := range references {
		for oldID, newID := range mapped[table] {
			for column, parent := range columns {
				var oldRef, newRef sql.NullInt64
				if err := second.QueryRow("SELECT "+column+" FROM "+table+" WHERE id = ?", oldID).Scan(&oldRef); err != nil {
					t.Fatal(err)
				}
				if err := target.QueryRow("SELECT "+column+" FROM "+table+" WHERE id = $1", newID).Scan(&newRef); err != nil {
					t.Fatal(err)
				}
				if oldRef.Valid != newRef.Valid || (oldRef.Valid && mapped[parent][oldRef.Int64] != newRef.Int64) {
					t.Fatalf("incorrect %s.%s remapping", table, column)
				}
			}
		}
	}
	for oldID, newID := range mapped["baro_assignments"] {
		var scope string
		var oldScope, newScope int64
		if err := second.QueryRow("SELECT scope,scope_id FROM baro_assignments WHERE id = ?", oldID).Scan(&scope, &oldScope); err != nil {
			t.Fatal(err)
		}
		if err := target.QueryRow("SELECT scope_id FROM baro_assignments WHERE id = $1", newID).Scan(&newScope); err != nil {
			t.Fatal(err)
		}
		parent := "projects"
		if scope == "site" {
			parent = "sites"
		}
		if newScope != mapped[parent][oldScope] {
			t.Fatal("incorrect assignment scope")
		}
	}
	var file string
	var level float64
	var zero sql.NullFloat64
	if err := target.QueryRow("SELECT f.file,d.level_m,d.temp_c FROM logger_data d JOIN logger_files f ON f.id=d.logger_file_id WHERE d.id=$1", mapped["logger_data"][49]).Scan(&file, &level, &zero); err != nil {
		t.Fatal(err)
	}
	if file != "original/path.csv" || level != 1.2 || !zero.Valid || zero.Float64 != 0 {
		t.Fatal("values not preserved")
	}
	execSQL(t, target, "INSERT INTO projects(name) VALUES ('After append')")
	if err := dbcopy.SQLiteToPostgres(t.Context(), second, target); err == nil || !strings.Contains(err.Error(), "must be empty") {
		t.Fatalf("default migration: %v", err)
	}
	before = snapshot(t, target, nil)
	if _, err := dbcopy.AppendSQLiteToPostgres(t.Context(), second, target); err == nil || !strings.Contains(err.Error(), "conflicting project names") {
		t.Fatalf("repeat import: %v", err)
	}
	if snapshot(t, target, nil) != before {
		t.Fatal("repeat import changed rows")
	}
}

func TestAppendPostgresFailuresRollback(t *testing.T) {
	for _, tc := range []struct{ name, change, want string }{
		{"name conflict", "UPDATE projects SET name='Existing'", "conflicting project names"},
		{"duplicate source names", "INSERT INTO projects(id,name) VALUES (99,'Incoming')", "conflicting project names"},
		{"schema", "UPDATE schema_migrations SET version=999", "schema version"},
		{"missing reference", "UPDATE corrected_data SET logger_data_id=999", "logger_data_id=999"},
		{"invalid scope", "UPDATE baro_assignments SET scope='unknown' WHERE id=55", "unsupported scope"},
		{"missing scope", "UPDATE baro_assignments SET scope_id=999 WHERE id=55", "scope_id=999"},
		{"unenforced reference", "UPDATE logger_data SET correction_run_id=999 WHERE id=49", "correction_run_id=999"},
		{"late constraint", "UPDATE manual_readings SET value=-1", "manual_readings"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := postgresTarget(t)
			execSQL(t, target, "INSERT INTO projects(name) VALUES ('Existing')")
			source := sqliteSource(t, "Incoming")
			source.SetMaxOpenConns(1)
			execSQL(t, source, "PRAGMA foreign_keys=OFF")
			if tc.name == "late constraint" {
				execSQL(t, target, "ALTER TABLE manual_readings ADD CHECK (value >= 0)")
			}
			execSQL(t, source, tc.change)
			before := snapshot(t, target, nil)
			result, err := dbcopy.AppendSQLiteToPostgres(t.Context(), source, target)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v; want %s", err, tc.want)
			}
			if result.Counts != nil || result.ProjectIDs != nil {
				t.Fatal("failed append returned success data")
			}
			if snapshot(t, target, nil) != before {
				t.Fatal("failed append changed rows")
			}
		})
	}
}

func TestAppendPostgresEmptyAndHistoricalData(t *testing.T) {
	target := postgresTarget(t)
	source := sqliteSource(t, "Incoming")
	if _, err := dbcopy.AppendSQLiteToPostgres(t.Context(), source, target); err != nil {
		t.Fatal(err)
	}
	// The relationship is deliberately not enforced by the schema.
	execSQL(t, target, "UPDATE logger_data SET correction_run_id=999")
	second := sqliteSource(t, "Other")
	if _, err := dbcopy.AppendSQLiteToPostgres(t.Context(), second, target); err != nil {
		t.Fatal(err)
	}
}

func TestAppendPostgresCancellationAndConcurrency(t *testing.T) {
	target := postgresTarget(t)
	source := sqliteSource(t, "Incoming")
	lock, err := target.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Rollback()
	if _, err := lock.Exec("LOCK TABLE projects IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
	defer cancel()
	if _, err := dbcopy.AppendSQLiteToPostgres(ctx, source, target); err == nil {
		t.Fatal("expected cancellation")
	}
	if err := lock.Rollback(); err != nil {
		t.Fatal(err)
	}
	for _, sameName := range []bool{true, false} {
		t.Run(fmt.Sprint(sameName), func(t *testing.T) {
			target := postgresTarget(t)
			first := sqliteSource(t, "One")
			name := "Two"
			if sameName {
				name = "One"
			}
			second := sqliteSource(t, name)
			start := make(chan struct{})
			results := make(chan error, 2)
			for _, source := range []*sql.DB{first, second} {
				go func(source *sql.DB) {
					<-start
					_, err := dbcopy.AppendSQLiteToPostgres(t.Context(), source, target)
					results <- err
				}(source)
			}
			close(start)
			succeeded := 0
			for range 2 {
				err := <-results
				if err == nil {
					succeeded++
				} else if !sameName || !strings.Contains(err.Error(), "conflicting project names") {
					t.Fatal(err)
				}
			}
			want := 2
			if sameName {
				want = 1
			}
			if succeeded != want {
				t.Fatalf("success count=%d", succeeded)
			}
		})
	}
}
