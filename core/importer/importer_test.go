package importer

import (
	"aqualog/core/db"
	"aqualog/core/parsers"
	"path/filepath"
	"testing"
	"time"
)

func TestImportRecordsWithAutoDetectedLogger(t *testing.T) {
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
			Timestamp:       time.Date(2026, 7, 14, 10, 5, 0, 0, time.UTC),
			LevelM:          1.25,
			ImportedValue:   1.25,
			ImportedUnit:    "m",
			MeasurementKind: "manufacturer_compensated_level_m",
		}},
		int(siteID),
		-1,
		"logger.tab",
		"solinst",
		ImportOptions{},
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
}
