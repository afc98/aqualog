package query

import (
	"aqualog/core/db"
	"path/filepath"
	"testing"
	"time"
)

func TestCorrectedQueryIncludesSourceLogger(t *testing.T) {
	db.SetPathOverride(filepath.Join(t.TempDir(), "provenance.db"))
	t.Cleanup(func() { db.SetPathOverride("") })
	database, err := db.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(`INSERT INTO projects(id,name) VALUES(1,'Project');
 INSERT INTO sites(id,project_id,name) VALUES(1,1,'Site');
 INSERT INTO loggers(id,site_id,name) VALUES(52,1,'Ardley Quarry'),(55,1,'Ardley Quarry');
 INSERT INTO logger_files(id,logger_id,file) VALUES(1,52,'old.csv'),(2,55,'new.csv');`)
	if err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = database.Exec(`INSERT INTO logger_data(id,logger_id,logger_file_id,timestamp,level_m) VALUES(1,52,1,$1,10),(2,55,2,$1,20)`, stamp)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(`INSERT INTO corrected_data(site_id,logger_data_id,timestamp,corrected_value) VALUES(1,2,$1,104.24)`, stamp)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := QueryCorrectedDataWithOptions(1, Options{From: "2025-01-01", To: "2027-01-01", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].LoggerID != 55 || rows[0].CorrectedValue != 104.24 {
		t.Fatalf("rows=%+v", rows)
	}
}
