package cmd

import (
	"bytes"
	"github.com/afc98/aqualog/core/baro"
	"github.com/afc98/aqualog/core/db"
	"strings"
	"testing"
)

func TestRunBaroCorrectBatchDoesNotLockDatabase(t *testing.T) {
	db.SetPathOverride(t.TempDir() + `\aqualog.db`)
	t.Cleanup(func() { db.SetPathOverride("") })

	conn, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec(`
		INSERT INTO projects(id, name) VALUES (1, 'Batch project');
		INSERT INTO sites(id, project_id, name) VALUES
			(1, 1, 'Site 1'),
			(2, 1, 'Site 2');
		INSERT INTO loggers(id, site_id, name, role) VALUES
			(1, 1, 'Barologger', 'barometric'),
			(2, 1, 'Water 1', 'water_level'),
			(3, 2, 'Water 2', 'water_level');
		INSERT INTO logger_files(id, logger_id, file, measurement_kind, measurement_unit) VALUES
			(1, 1, 'baro.csv', 'barometric_pressure_kpa', 'kPa'),
			(2, 2, 'water1.csv', 'absolute_pressure_kpa', 'kPa'),
			(3, 3, 'water2.csv', 'absolute_pressure_kpa', 'kPa');
		INSERT INTO logger_data(logger_id, logger_file_id, timestamp, level_m, imported_value, imported_unit, imported_kind) VALUES
			(1, 1, '2026-01-01T00:00:00Z', 100, 100, 'kPa', 'barometric_pressure_kpa'),
			(1, 1, '2026-01-01T00:10:00Z', 101, 101, 'kPa', 'barometric_pressure_kpa'),
			(2, 2, '2026-01-01T00:05:00Z', 110, 110, 'kPa', 'absolute_pressure_kpa'),
			(3, 3, '2026-01-01T00:05:00Z', 111, 111, 'kPa', 'absolute_pressure_kpa');
		INSERT INTO baro_assignments(scope, scope_id, barologger_id) VALUES ('project', 1, 1);
	`)
	if err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := runBaroCorrectBatch(1, baro.CorrectOptions{}, &output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "database is locked") {
		t.Fatalf("batch output contains lock failure: %s", output.String())
	}
	for _, siteName := range []string{"Site 1: run=", "Site 2: run="} {
		if !strings.Contains(output.String(), siteName) {
			t.Fatalf("batch output missing %q: %s", siteName, output.String())
		}
	}

	conn, err = db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var runCount, correctedCount int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM correction_runs WHERE site_id IN (1, 2)`).Scan(&runCount); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(`SELECT COUNT(*) FROM logger_data WHERE logger_id IN (2, 3) AND baro_corrected_m IS NOT NULL AND correction_run_id IS NOT NULL`).Scan(&correctedCount); err != nil {
		t.Fatal(err)
	}
	if runCount != 2 {
		t.Fatalf("correction run count = %d; want 2", runCount)
	}
	if correctedCount != 2 {
		t.Fatalf("corrected row count = %d; want 2", correctedCount)
	}
}
