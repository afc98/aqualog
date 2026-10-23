package process

import (
	"database/sql"
	"github.com/afc98/aqualog/core/db"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessLoggerDataExcludesBarometricRows(t *testing.T) {
	database := newProcessTestDB(t)
	seedProcessSite(t, database, false)

	if err := ProcessLoggerData(database, 1); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := database.QueryRow(`
		SELECT COUNT(*)
		FROM corrected_data c
		JOIN logger_data d ON d.id = c.logger_data_id
		WHERE c.site_id = 1 AND d.logger_id = 2
	`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("water-level corrected row count = %d; want 2", count)
	}
	if err := database.QueryRow(`
		SELECT COUNT(*)
		FROM corrected_data c
		JOIN logger_data d ON d.id = c.logger_data_id
		WHERE c.site_id = 1 AND d.logger_id = 1
	`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("barometric corrected row count = %d; want 0", count)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM corrected_data WHERE site_id = 1 AND (sal_psu IS NOT NULL OR ec_us IS NOT NULL)`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("corrected rows with fabricated optional values = %d; want 0", count)
	}
}

func TestProcessLoggerDataPreservesRealZeroOptionalValues(t *testing.T) {
	database := newProcessTestDB(t)
	seedProcessSite(t, database, true)
	if _, err := database.Exec(`UPDATE logger_data SET sal_psu = 0, ec_us = 0 WHERE logger_id = 2`); err != nil {
		t.Fatal(err)
	}
	if err := ProcessLoggerData(database, 1); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM corrected_data WHERE site_id = 1 AND sal_psu = 0 AND ec_us = 0`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("corrected rows preserving measured zeroes = %d; want 2", count)
	}
}

func TestProcessLoggerDataCombinesNonOverlappingWaterLoggers(t *testing.T) {
	database := newProcessTestDB(t)
	seedProcessSite(t, database, true)
	_, err := database.Exec(`
		INSERT INTO loggers(id, site_id, name, role) VALUES (3, 1, 'Water 2', 'water_level');
		INSERT INTO logger_files(id, logger_id, file) VALUES (3, 3, 'water2.csv');
		INSERT INTO logger_data(logger_id, logger_file_id, timestamp, level_m) VALUES
			(3, 3, '2026-01-01T01:01:00Z', 20),
			(3, 3, '2026-01-01T01:02:00Z', 21);
		INSERT INTO logger_events(logger_id, timestamp, event_type) VALUES
			(3, '2026-01-01T01:00:00Z', 'installed'),
			(3, '2026-01-01T01:10:00Z', 'removed');
		INSERT INTO manual_readings(site_id, timestamp, value) VALUES
			(1, '2026-01-01T01:01:00Z', 25);
	`)
	if err != nil {
		t.Fatal(err)
	}

	if err := ProcessLoggerData(database, 1); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM corrected_data WHERE site_id = 1`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("corrected row count = %d; want 4", count)
	}
}

func TestProcessLoggerDataRejectsOverlappingWaterLoggersBeforeStorage(t *testing.T) {
	database := newProcessTestDB(t)
	seedProcessSite(t, database, true)
	_, err := database.Exec(`
		INSERT INTO loggers(id, site_id, name, role) VALUES (3, 1, 'Water 2', 'water_level');
		INSERT INTO logger_files(id, logger_id, file) VALUES (3, 3, 'water2.csv');
		INSERT INTO logger_data(logger_id, logger_file_id, timestamp, level_m) VALUES
			(3, 3, '2026-01-01T00:01:00Z', 20),
			(3, 3, '2026-01-01T00:02:00Z', 21);
		INSERT INTO logger_events(logger_id, timestamp, event_type) VALUES
			(3, '2026-01-01T00:00:00Z', 'installed'),
			(3, '2026-01-01T00:10:00Z', 'removed');
		INSERT INTO corrected_data(site_id, logger_data_id, timestamp, corrected_value)
		VALUES (1, 3, '2025-12-31T23:00:00Z', 999);
	`)
	if err != nil {
		t.Fatal(err)
	}

	err = ProcessLoggerData(database, 1)
	if err == nil || !strings.Contains(err.Error(), "both produce corrected data at 2026-01-01T00:01:00Z") {
		t.Fatalf("overlap error = %v", err)
	}

	var count int
	var value float64
	if err := database.QueryRow(`SELECT COUNT(*), corrected_value FROM corrected_data WHERE site_id = 1`).Scan(&count, &value); err != nil {
		t.Fatal(err)
	}
	if count != 1 || value != 999 {
		t.Fatalf("stored corrected data changed after overlap: count=%d value=%v", count, value)
	}
}

func newProcessTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db.SetPathOverride(filepath.Join(t.TempDir(), "aqualog.db"))
	t.Cleanup(func() { db.SetPathOverride("") })
	database, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func seedProcessSite(t *testing.T, database *sql.DB, waterOnly bool) {
	t.Helper()
	_, err := database.Exec(`
		INSERT INTO projects(id, name) VALUES (1, 'Project');
		INSERT INTO sites(id, project_id, name) VALUES (1, 1, 'Site');
		INSERT INTO loggers(id, site_id, name, role) VALUES (2, 1, 'Water 1', 'water_level');
		INSERT INTO logger_files(id, logger_id, file) VALUES (2, 2, 'water1.csv');
		INSERT INTO logger_data(logger_id, logger_file_id, timestamp, level_m) VALUES
			(2, 2, '2026-01-01T00:01:00Z', 10),
			(2, 2, '2026-01-01T00:02:00Z', 11);
		INSERT INTO logger_events(logger_id, timestamp, event_type) VALUES
			(2, '2026-01-01T00:00:00Z', 'installed'),
			(2, '2026-01-01T00:10:00Z', 'removed');
		INSERT INTO manual_readings(site_id, timestamp, value) VALUES
			(1, '2026-01-01T00:01:00Z', 15);
	`)
	if err != nil {
		t.Fatal(err)
	}
	if waterOnly {
		return
	}
	_, err = database.Exec(`
		INSERT INTO loggers(id, site_id, name, role) VALUES (1, 1, 'Barometer', 'barometric');
		INSERT INTO logger_files(id, logger_id, file) VALUES (1, 1, 'baro.csv');
		INSERT INTO logger_data(logger_id, logger_file_id, timestamp, level_m) VALUES
			(1, 1, '2026-01-01T00:01:00Z', 1000),
			(1, 1, '2026-01-01T00:02:00Z', 1001);
	`)
	if err != nil {
		t.Fatal(err)
	}
}
