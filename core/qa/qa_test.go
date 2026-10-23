package qa

import (
	"github.com/afc98/aqualog/core/db"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSiteIgnoresBarometricLoggerEvents(t *testing.T) {
	db.SetPathOverride(filepath.Join(t.TempDir(), "aqualog.db"))
	t.Cleanup(func() { db.SetPathOverride("") })
	database, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	_, err = database.Exec(`
		INSERT INTO projects(id, name) VALUES (1, 'Project');
		INSERT INTO sites(id, project_id, name) VALUES (1, 1, 'Site');
		INSERT INTO loggers(id, site_id, name, role) VALUES
			(1, 1, 'Water', 'water_level'),
			(2, 1, 'Barometer', 'barometric');
		INSERT INTO logger_files(id, logger_id, file) VALUES (1, 1, 'water.csv');
		INSERT INTO logger_data(logger_id, logger_file_id, timestamp, level_m) VALUES
			(1, 1, '2026-01-01T00:01:00Z', 10),
			(1, 1, '2026-01-01T00:02:00Z', 11);
		INSERT INTO manual_readings(site_id, timestamp, value) VALUES (1, '2026-01-01T00:01:00Z', 15);
		INSERT INTO logger_events(logger_id, timestamp, event_type) VALUES
			(1, '2026-01-01T00:00:00Z', 'installed'),
			(1, '2026-01-01T00:10:00Z', 'removed'),
			(2, '2026-01-01T00:00:00Z', 'installed'),
			(2, '2026-01-01T00:01:00Z', 'installed');
	`)
	if err != nil {
		t.Fatal(err)
	}

	issues, err := ValidateSite(1)
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range issues {
		if issue.Severity == "error" || strings.Contains(issue.Message, "Barometer") {
			t.Fatalf("unexpected validation issue: [%s] %s", issue.Severity, issue.Message)
		}
	}
}

func TestValidateSiteRecognizesCrossSiteCorrectionBarometer(t *testing.T) {
	db.SetPathOverride(filepath.Join(t.TempDir(), "aqualog.db"))
	t.Cleanup(func() { db.SetPathOverride("") })
	database, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	_, err = database.Exec(`
		INSERT INTO projects(id, name) VALUES (1, 'Project');
		INSERT INTO sites(id, project_id, name) VALUES (1, 1, 'Baro site'), (2, 1, 'Water site');
		INSERT INTO loggers(id, site_id, name, role) VALUES
			(1, 1, 'Barometer', 'barometric'),
			(2, 2, 'Water', 'water_level');
		INSERT INTO logger_files(id, logger_id, file) VALUES (1, 1, 'baro.csv'), (2, 2, 'water.csv');
		INSERT INTO logger_data(logger_id, logger_file_id, timestamp, level_m, imported_value, imported_unit, imported_kind, baro_corrected_m) VALUES
			(1, 1, '2026-01-01T00:00:00Z', 1010, 1010, 'mbar', 'barometric_pressure_mbar', NULL),
			(2, 2, '2026-01-01T00:01:00Z', 0.2, 1012, 'mbar', 'absolute_pressure_mbar', 0.2);
		INSERT INTO manual_readings(site_id, timestamp, value) VALUES (2, '2026-01-01T00:01:00Z', 1.2);
		INSERT INTO logger_events(logger_id, timestamp, event_type) VALUES
			(2, '2026-01-01T00:00:00Z', 'installed'),
			(2, '2026-01-01T00:10:00Z', 'removed');
		INSERT INTO correction_runs(site_id, logger_id, barologger_id, method, density_mode, density_kg_m3, match_mode, max_gap_seconds)
		VALUES (2, NULL, 1, 'barometric', 'freshwater', 1000, 'linear_interpolation', 600);
	`)
	if err != nil {
		t.Fatal(err)
	}

	status, err := SiteStatus(2)
	if err != nil {
		t.Fatal(err)
	}
	if status.BaroCount != 0 || status.AvailableBaroCount != 1 {
		t.Fatalf("barometric counts = local %d available %d; want 0 and 1", status.BaroCount, status.AvailableBaroCount)
	}
	issues, err := ValidateSite(2)
	if err != nil {
		t.Fatal(err)
	}
	for _, issue := range issues {
		if strings.Contains(issue.Message, "no barometric records") {
			t.Fatalf("unexpected validation issue: %s", issue.Message)
		}
	}

	if _, err := database.Exec(`DELETE FROM logger_data WHERE logger_id = 1`); err != nil {
		t.Fatal(err)
	}
	status, err = SiteStatus(2)
	if err != nil {
		t.Fatal(err)
	}
	if status.AvailableBaroCount != 0 {
		t.Fatalf("available barometric rows = %d after removing pressure data; want 0", status.AvailableBaroCount)
	}
	issues, err = ValidateSite(2)
	if err != nil {
		t.Fatal(err)
	}
	foundMissing := false
	for _, issue := range issues {
		foundMissing = foundMissing || strings.Contains(issue.Message, "no barometric records")
	}
	if !foundMissing {
		t.Fatal("expected missing barometric records validation error")
	}
}
