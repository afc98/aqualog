package cmd

import (
	"database/sql"
	"math"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestParsePlotSeriesSpecAliases(t *testing.T) {
	tests := []struct {
		defaultSource string
		spec          string
		wantSource    string
		wantName      string
	}{
		{plotTypeCorrected, "level", plotTypeCorrected, "level"},
		{plotTypeCorrected, "raw_level", plotTypeRaw, "level"},
		{plotTypeCorrected, "uncorrected_level", plotTypeRaw, "level"},
		{plotTypeCorrected, "baro_corrected_level", plotTypeBaroCorrected, "level"},
		{plotTypeBaroCorrected, "level", plotTypeBaroCorrected, "level"},
		{plotTypeCorrected, "pressure", plotTypePressure, "pressure"},
		{plotTypeCorrected, "atmospheric_pressure", plotTypePressure, "pressure"},
		{plotTypeCorrected, "uncompensated:level", plotTypeUncompensated, "level"},
		{plotTypeRaw, "temperature", plotTypeRaw, "temp"},
	}

	for _, tt := range tests {
		gotSource, gotName, err := parsePlotSeriesSpec(tt.defaultSource, tt.spec)
		if err != nil {
			t.Fatalf("parsePlotSeriesSpec(%q, %q): %v", tt.defaultSource, tt.spec, err)
		}
		if gotSource != tt.wantSource || gotName != tt.wantName {
			t.Fatalf("parsePlotSeriesSpec(%q, %q) = %q, %q; want %q, %q",
				tt.defaultSource, tt.spec, gotSource, gotName, tt.wantSource, tt.wantName)
		}
	}
}

func TestLoadBarometricPressureSeriesConvertsToKPA(t *testing.T) {
	dbConn := newPlotTestDB(t)
	insertPlotTestSite(t, dbConn)
	_, err := dbConn.Exec(`
		INSERT INTO loggers(id, site_id, name, role) VALUES (2, 1, 'Barometer', 'barometric');
		INSERT INTO logger_files(id, logger_id, file) VALUES (2, 2, 'baro.csv');
		INSERT INTO logger_data(logger_id, logger_file_id, timestamp, imported_value, imported_kind) VALUES
			(2, 2, '2025-01-01T00:00:00Z', 1013.2, 'barometric_pressure_mbar'),
			(2, 2, '2025-01-01T01:00:00Z', 101.5, 'barometric_pressure_kpa'),
			(2, 2, '2025-01-01T02:00:00Z', 999, 'absolute_pressure_mbar');
	`)
	if err != nil {
		t.Fatal(err)
	}
	series, err := loadBarometricPressureSeries(dbConn, 2, "Barometer", mustTime(t, "2025-01-01T00:00:00Z"), mustTime(t, "2025-01-02T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if series.Unit != "kPa" || len(series.Y) != 2 || math.Abs(series.Y[0]-101.32) > 1e-9 || series.Y[1] != 101.5 {
		t.Fatalf("pressure series = unit %q values %v; want kPa [101.32 101.5]", series.Unit, series.Y)
	}
}

func TestResolvePressureBarologgerPrecedence(t *testing.T) {
	dbConn := newPlotTestDB(t)
	_, err := dbConn.Exec(`
		INSERT INTO sites(id, project_id, name) VALUES (1, 1, 'Water site'), (2, 1, 'Baro site');
		INSERT INTO loggers(id, site_id, name, role) VALUES
			(1, 1, 'Local barometer', 'barometric'),
			(2, 2, 'Site assignment', 'barometric'),
			(3, 2, 'Project assignment', 'barometric'),
			(4, 2, 'Recent run', 'barometric');
		INSERT INTO baro_assignments(scope, scope_id, barologger_id) VALUES
			('site', 1, 2), ('project', 1, 3);
		INSERT INTO correction_runs(id, site_id, barologger_id) VALUES (1, 1, 4);
	`)
	if err != nil {
		t.Fatal(err)
	}

	assertResolved := func(wantID int, ident string) {
		t.Helper()
		gotID, _, err := resolvePressureBarologger(dbConn, 1, ident)
		if err != nil {
			t.Fatal(err)
		}
		if gotID != wantID {
			t.Fatalf("resolved logger = %d; want %d", gotID, wantID)
		}
	}
	assertResolved(4, "4")
	assertResolved(2, "")
	if _, err := dbConn.Exec(`DELETE FROM baro_assignments WHERE scope = 'site'`); err != nil {
		t.Fatal(err)
	}
	assertResolved(3, "")
	if _, err := dbConn.Exec(`DELETE FROM baro_assignments WHERE scope = 'project'`); err != nil {
		t.Fatal(err)
	}
	assertResolved(4, "")
	if _, err := dbConn.Exec(`DELETE FROM correction_runs`); err != nil {
		t.Fatal(err)
	}
	assertResolved(1, "")
}

func TestLoadCorrectedPlotSeriesSkipsNulls(t *testing.T) {
	dbConn := newPlotTestDB(t)
	insertPlotTestSite(t, dbConn)

	_, err := dbConn.Exec(`
		INSERT INTO corrected_data(site_id, logger_data_id, timestamp, corrected_value, temp_c, sal_psu, ec_us)
		VALUES
			(1, 1, '2025-01-01T00:00:00Z', 10.0, NULL, NULL, NULL),
			(1, 2, '2025-01-02T00:00:00Z', 11.0, 22.5, NULL, NULL)
	`)
	if err != nil {
		t.Fatal(err)
	}

	series, err := loadCorrectedPlotSeries(dbConn, 1, "temp", mustTime(t, "2025-01-01T00:00:00Z"), mustTime(t, "2025-01-03T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if len(series.X) != 1 || len(series.Y) != 1 {
		t.Fatalf("point count = %d/%d; want 1/1", len(series.X), len(series.Y))
	}
	if series.Y[0] != 22.5 {
		t.Fatalf("temperature = %v; want 22.5", series.Y[0])
	}
}

func TestLoadUncompensatedLevelSeriesConvertsPressureToHead(t *testing.T) {
	dbConn := newPlotTestDB(t)
	insertPlotTestSite(t, dbConn)

	_, err := dbConn.Exec(`
		INSERT INTO logger_data(logger_id, logger_file_id, timestamp, level_m, imported_value, imported_unit, imported_kind)
		VALUES
			(1, 1, '2025-01-01T00:00:00Z', 0, 98.0665, 'kPa', 'absolute_pressure_kpa'),
			(1, 1, '2025-01-02T00:00:00Z', 0, 5.5, 'm', 'absolute_pressure_head_m')
	`)
	if err != nil {
		t.Fatal(err)
	}

	series, err := loadUncompensatedLevelSeries(dbConn, 1, mustTime(t, "2025-01-01T00:00:00Z"), mustTime(t, "2025-01-03T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if len(series.Y) != 2 {
		t.Fatalf("point count = %d; want 2", len(series.Y))
	}
	if math.Abs(series.Y[0]-10) > 1e-9 {
		t.Fatalf("converted kPa head = %v; want 10", series.Y[0])
	}
	if series.Y[1] != 5.5 {
		t.Fatalf("head_m value = %v; want 5.5", series.Y[1])
	}
}

func newPlotTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbConn, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = dbConn.Close() })
	_, err = dbConn.Exec(`
		CREATE TABLE sites(id INTEGER PRIMARY KEY, project_id INTEGER NOT NULL, name TEXT NOT NULL);
		CREATE TABLE loggers(id INTEGER PRIMARY KEY, site_id INTEGER NOT NULL, name TEXT NOT NULL, role TEXT);
		CREATE TABLE logger_files(id INTEGER PRIMARY KEY, logger_id INTEGER NOT NULL, file TEXT NOT NULL);
		CREATE TABLE logger_data(
			id INTEGER PRIMARY KEY,
			logger_id INTEGER NOT NULL,
			logger_file_id INTEGER NOT NULL,
			timestamp DATETIME NOT NULL,
			level_m REAL,
			imported_value REAL,
			imported_unit TEXT,
			imported_kind TEXT,
			baro_corrected_m REAL,
			correction_run_id INTEGER,
			temp_c REAL,
			sal_psu REAL,
			ec_us REAL
		);
		CREATE TABLE corrected_data(
			id INTEGER PRIMARY KEY,
			site_id INTEGER NOT NULL,
			logger_data_id INTEGER NOT NULL,
			timestamp DATETIME NOT NULL,
			corrected_value REAL NOT NULL,
			temp_c REAL,
			sal_psu REAL,
			ec_us REAL
		);
		CREATE TABLE baro_assignments(scope TEXT NOT NULL, scope_id INTEGER NOT NULL, barologger_id INTEGER NOT NULL);
		CREATE TABLE correction_runs(id INTEGER PRIMARY KEY, site_id INTEGER NOT NULL, barologger_id INTEGER NOT NULL);
	`)
	if err != nil {
		t.Fatal(err)
	}
	return dbConn
}

func insertPlotTestSite(t *testing.T, dbConn *sql.DB) {
	t.Helper()
	_, err := dbConn.Exec(`
		INSERT INTO sites(id, project_id, name) VALUES (1, 1, 'Site 1');
		INSERT INTO loggers(id, site_id, name, role) VALUES (1, 1, 'Logger 1', 'water_level');
		INSERT INTO logger_files(id, logger_id, file) VALUES (1, 1, 'logger.csv');
	`)
	if err != nil {
		t.Fatal(err)
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
