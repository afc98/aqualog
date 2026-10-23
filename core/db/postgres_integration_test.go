package db_test

import (
	"aqualog/core/db"
	"aqualog/core/dbcopy"
	"aqualog/core/event"
	"aqualog/core/logger"
	"aqualog/core/manual"
	"aqualog/core/project"
	"aqualog/core/site"
	"os"
	"path/filepath"
	"testing"
)

func TestPostgresCoreCRUD(t *testing.T) {
	databaseURL := os.Getenv("AQUALOG_TEST_POSTGRES_URL")
	if databaseURL == "" {
		t.Skip("AQUALOG_TEST_POSTGRES_URL is not set")
	}
	db.SetPathOverride("")
	db.SetDatabaseURLOverride(databaseURL)
	t.Cleanup(func() { db.SetDatabaseURLOverride(""); _ = db.Close() })
	conn, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(`TRUNCATE projects RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}

	p, err := project.Add(project.AddParams{Name: "PostgreSQL project", Description: "integration"})
	if err != nil {
		t.Fatal(err)
	}
	s, err := site.Add(site.AddParams{ProjectID: p.ID, Name: "Site", Latitude: 51.5, Longitude: -0.1})
	if err != nil {
		t.Fatal(err)
	}
	l, err := logger.Add(logger.AddParams{SiteID: s.ID, Name: "Logger", Serial: "PG-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := event.Add(event.AddParams{LoggerID: l.ID, EventType: "installed", Timestamp: "20260731 06:00:00"}); err != nil {
		t.Fatal(err)
	}
	events, err := event.List(s.ID, l.ID)
	if err != nil || len(events) != 1 || events[0].Timestamp.Hour() != 6 {
		t.Fatalf("PostgreSQL event clock time: events=%v err=%v", events, err)
	}
	var dataType string
	if err := conn.QueryRow(`SELECT data_type FROM information_schema.columns WHERE table_name='logger_events' AND column_name='timestamp'`).Scan(&dataType); err != nil || dataType != "timestamp without time zone" {
		t.Fatalf("event timestamp type=%s err=%v", dataType, err)
	}
	if _, err := manual.Add(manual.AddParams{SiteID: s.ID, Timestamp: "20260102 03:04:05", WaterLevel: 1.25}); err != nil {
		t.Fatal(err)
	}

	projects, err := project.List()
	if err != nil {
		t.Fatal(err)
	}
	sites, err := site.List(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	loggers, err := logger.List(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	readings, err := manual.List(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || len(sites) != 1 || len(loggers) != 1 || len(readings) != 1 || loggers[0].ID != l.ID {
		t.Fatalf("unexpected PostgreSQL CRUD result: projects=%d sites=%d loggers=%d readings=%d", len(projects), len(sites), len(loggers), len(readings))
	}

	if _, err := conn.Exec(`TRUNCATE projects RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
	source, err := db.Open(t.Context(), db.Config{Backend: db.BackendSQLite, SQLitePath: filepath.Join(t.TempDir(), "source.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	if _, err := source.Exec(`
		INSERT INTO projects(id, name, description) VALUES (42, 'Migrated project', 'copy test');
		INSERT INTO sites(id, project_id, name) VALUES (43, 42, 'Migrated site');
	`); err != nil {
		t.Fatal(err)
	}
	if err := dbcopy.SQLiteToPostgres(t.Context(), source, conn); err != nil {
		t.Fatal(err)
	}
	var migratedName string
	if err := conn.QueryRow(`SELECT name FROM sites WHERE id = $1 AND project_id = $2`, 43, 42).Scan(&migratedName); err != nil {
		t.Fatal(err)
	}
	if migratedName != "Migrated site" {
		t.Fatalf("migrated site name = %q", migratedName)
	}
}
