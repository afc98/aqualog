package files

import (
	"database/sql"
	"github.com/afc98/aqualog/core/db"
	"path/filepath"
	"testing"
)

func TestRemoveDeletesSelectedFileAndInvalidatesOnlyItsSite(t *testing.T) {
	dbConn := openTestDB(t)
	projectID := insertID(t, dbConn, `INSERT INTO projects (name) VALUES (?)`, "Project")
	siteID := insertID(t, dbConn, `INSERT INTO sites (project_id, name) VALUES (?, ?)`, projectID, "Site 1")
	otherSiteID := insertID(t, dbConn, `INSERT INTO sites (project_id, name) VALUES (?, ?)`, projectID, "Site 2")
	loggerID := insertID(t, dbConn, `INSERT INTO loggers (site_id, name) VALUES (?, ?)`, siteID, "Logger 1")
	otherLoggerID := insertID(t, dbConn, `INSERT INTO loggers (site_id, name) VALUES (?, ?)`, otherSiteID, "Logger 2")
	fileID := insertID(t, dbConn, `INSERT INTO logger_files (logger_id, file) VALUES (?, ?)`, loggerID, "remove.tab")
	keepFileID := insertID(t, dbConn, `INSERT INTO logger_files (logger_id, file) VALUES (?, ?)`, loggerID, "keep.tab")
	otherFileID := insertID(t, dbConn, `INSERT INTO logger_files (logger_id, file) VALUES (?, ?)`, otherLoggerID, "other.tab")

	rawID1 := insertID(t, dbConn, `INSERT INTO logger_data (logger_id, logger_file_id, timestamp) VALUES (?, ?, ?)`, loggerID, fileID, "2026-01-01T00:00:00Z")
	insertID(t, dbConn, `INSERT INTO logger_data (logger_id, logger_file_id, timestamp) VALUES (?, ?, ?)`, loggerID, fileID, "2026-01-01T01:00:00Z")
	keepRawID := insertID(t, dbConn, `INSERT INTO logger_data (logger_id, logger_file_id, timestamp) VALUES (?, ?, ?)`, loggerID, keepFileID, "2026-01-01T02:00:00Z")
	otherRawID := insertID(t, dbConn, `INSERT INTO logger_data (logger_id, logger_file_id, timestamp) VALUES (?, ?, ?)`, otherLoggerID, otherFileID, "2026-01-01T03:00:00Z")

	insertID(t, dbConn, `INSERT INTO corrected_data (site_id, logger_data_id, timestamp, corrected_value) VALUES (?, ?, ?, ?)`, siteID, rawID1, "2026-01-01T00:00:00Z", 1.0)
	insertID(t, dbConn, `INSERT INTO corrected_data (site_id, logger_data_id, timestamp, corrected_value) VALUES (?, ?, ?, ?)`, siteID, keepRawID, "2026-01-01T02:00:00Z", 2.0)
	insertID(t, dbConn, `INSERT INTO corrected_data (site_id, logger_data_id, timestamp, corrected_value) VALUES (?, ?, ?, ?)`, otherSiteID, otherRawID, "2026-01-01T03:00:00Z", 3.0)
	if err := dbConn.Close(); err != nil {
		t.Fatal(err)
	}

	gotSiteID, removedRows, err := Remove(int(fileID))
	if err != nil {
		t.Fatal(err)
	}
	if gotSiteID != int(siteID) {
		t.Fatalf("site ID = %d; want %d", gotSiteID, siteID)
	}
	if removedRows != 2 {
		t.Fatalf("removed rows = %d; want 2", removedRows)
	}

	dbConn, err = db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()
	assertCount(t, dbConn, `SELECT COUNT(*) FROM logger_files WHERE id = ?`, 0, fileID)
	assertCount(t, dbConn, `SELECT COUNT(*) FROM logger_files WHERE id IN (?, ?)`, 2, keepFileID, otherFileID)
	assertCount(t, dbConn, `SELECT COUNT(*) FROM logger_data WHERE logger_file_id = ?`, 0, fileID)
	assertCount(t, dbConn, `SELECT COUNT(*) FROM logger_data WHERE logger_file_id IN (?, ?)`, 2, keepFileID, otherFileID)
	assertCount(t, dbConn, `SELECT COUNT(*) FROM corrected_data WHERE site_id = ?`, 0, siteID)
	assertCount(t, dbConn, `SELECT COUNT(*) FROM corrected_data WHERE site_id = ?`, 1, otherSiteID)
}

func TestRemoveMissingFileLeavesDataUntouched(t *testing.T) {
	dbConn := openTestDB(t)
	projectID := insertID(t, dbConn, `INSERT INTO projects (name) VALUES (?)`, "Project")
	siteID := insertID(t, dbConn, `INSERT INTO sites (project_id, name) VALUES (?, ?)`, projectID, "Site")
	loggerID := insertID(t, dbConn, `INSERT INTO loggers (site_id, name) VALUES (?, ?)`, siteID, "Logger")
	fileID := insertID(t, dbConn, `INSERT INTO logger_files (logger_id, file) VALUES (?, ?)`, loggerID, "keep.tab")
	insertID(t, dbConn, `INSERT INTO logger_data (logger_id, logger_file_id, timestamp) VALUES (?, ?, ?)`, loggerID, fileID, "2026-01-01T00:00:00Z")
	if err := dbConn.Close(); err != nil {
		t.Fatal(err)
	}

	if _, _, err := Remove(999999); err == nil {
		t.Fatal("expected an error for a missing file")
	}

	dbConn, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()
	assertCount(t, dbConn, `SELECT COUNT(*) FROM logger_files`, 1)
	assertCount(t, dbConn, `SELECT COUNT(*) FROM logger_data`, 1)
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db.SetPathOverride(filepath.Join(t.TempDir(), "aqualog.db"))
	t.Cleanup(func() { db.SetPathOverride("") })
	dbConn, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	return dbConn
}

func insertID(t *testing.T, dbConn *sql.DB, query string, args ...any) int64 {
	t.Helper()
	result, err := dbConn.Exec(query, args...)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func assertCount(t *testing.T, dbConn *sql.DB, query string, want int, args ...any) {
	t.Helper()
	var got int
	if err := dbConn.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("count = %d; want %d", got, want)
	}
}
