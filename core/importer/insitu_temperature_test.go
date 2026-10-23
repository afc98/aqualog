package importer

import (
	"aqualog/core/db"
	"aqualog/core/parsers"
	"path/filepath"
	"testing"
	"time"
)

func TestImportMissingBarometricTemperature(t *testing.T) {
	db.SetPathOverride(filepath.Join(t.TempDir(), "aqualog.db"))
	t.Cleanup(func() { db.SetPathOverride("") })

	dbConn, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	projectResult, err := dbConn.Exec(`INSERT INTO projects (name) VALUES (?)`, "Test project")
	if err != nil {
		t.Fatal(err)
	}
	projectID, err := projectResult.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	siteResult, err := dbConn.Exec(
		`INSERT INTO sites (project_id, name) VALUES (?, ?)`,
		projectID,
		"Test site",
	)
	if err != nil {
		t.Fatal(err)
	}
	siteID, err := siteResult.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if err := dbConn.Close(); err != nil {
		t.Fatal(err)
	}

	result, err := ImportRecords(
		parsers.Metadata{SiteIdent: "Test logger", SerialNumber: "SN-001"},
		[]parsers.Record{{
			Timestamp:          time.Date(2026, 7, 14, 10, 5, 0, 0, time.UTC),
			TemperatureMissing: true,
			ImportedValue:      1004.232,
			ImportedUnit:       "mbar",
			MeasurementKind:    "barometric_pressure_mbar",
		}},
		int(siteID),
		-1,
		"logger.tab",
		"solinst",
		ImportOptions{Role: "barometric"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !result.LoggerCreated {
		t.Fatal("expected automatic logger creation")
	}
	if result.Inserted != 1 {
		t.Fatalf("inserted = %d; want 1", result.Inserted)
	}
	conn, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var missing bool
	var pressure float64
	var unit string
	if err := conn.QueryRow("SELECT temp_c IS NULL, imported_value, imported_unit FROM logger_data WHERE logger_id = ?", result.LoggerID).Scan(&missing, &pressure, &unit); err != nil {
		t.Fatal(err)
	}
	if !missing || pressure != 1004.232 || unit != "mbar" {
		t.Fatalf("missing=%v pressure=%v unit=%s", missing, pressure, unit)
	}
}
