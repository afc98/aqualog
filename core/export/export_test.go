package export

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestWriteCSVFormatsMissingAndZeroOptionalValues(t *testing.T) {
	zero := 0.0
	path := filepath.Join(t.TempDir(), "export.csv")
	rows := []ExportRow{
		{Timestamp: "2026-01-01T00:00:00Z", SalinityPSU: nil, ECUS: nil},
		{Timestamp: "2026-01-01T00:01:00Z", SalinityPSU: &zero, ECUS: &zero},
	}
	if err := WriteCSV(rows, path); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	records, err := csv.NewReader(file).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if records[1][8] != "" || records[1][9] != "" {
		t.Fatalf("missing optional CSV values = %q, %q; want blanks", records[1][8], records[1][9])
	}
	if records[2][8] != "0.0000" || records[2][9] != "0.0000" {
		t.Fatalf("measured zero CSV values = %q, %q; want formatted zeroes", records[2][8], records[2][9])
	}
}

func TestWriteJSONFormatsMissingAndZeroOptionalValues(t *testing.T) {
	zero := 0.0
	path := filepath.Join(t.TempDir(), "export.json")
	rows := []ExportRow{{SalinityPSU: nil, ECUS: nil}, {SalinityPSU: &zero, ECUS: &zero}}
	if err := WriteJSON(rows, path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded[0]["SalinityPSU"] != nil || decoded[0]["ECUS"] != nil {
		t.Fatalf("missing optional JSON values = %#v", decoded[0])
	}
	if decoded[1]["SalinityPSU"] != float64(0) || decoded[1]["ECUS"] != float64(0) {
		t.Fatalf("measured zero JSON values = %#v", decoded[1])
	}
}

func TestWritersIncludeOnlySelectedMeasurementSeries(t *testing.T) {
	temperature := 23.5
	row := ExportRow{Timestamp: "2026-01-01T00:00:00Z", TempC: &temperature}
	csvPath := filepath.Join(t.TempDir(), "temperature.csv")
	if err := writeCSVWithSeries([]ExportRow{row}, csvPath, []string{"temp"}); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(file).ReadAll()
	_ = file.Close()
	if err != nil {
		t.Fatal(err)
	}
	wantHeader := []string{"timestamp", "site_name", "logger_name", "logger_model", "logger_serial", "temp_c"}
	if len(records[0]) != len(wantHeader) {
		t.Fatalf("CSV header = %v; want %v", records[0], wantHeader)
	}
	for i := range wantHeader {
		if records[0][i] != wantHeader[i] {
			t.Fatalf("CSV header = %v; want %v", records[0], wantHeader)
		}
	}
	if records[1][5] != "23.5000" {
		t.Fatalf("temperature CSV value = %q; want 23.5000", records[1][5])
	}

	jsonPath := filepath.Join(t.TempDir(), "temperature.json")
	if err := writeJSONWithSeries([]ExportRow{row}, jsonPath, []string{"temp"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded[0]["TempC"] != 23.5 {
		t.Fatalf("temperature JSON value = %#v; want 23.5", decoded[0]["TempC"])
	}
	for _, excluded := range []string{"Level", "ManualReading", "CorrectedLevel", "SalinityPSU", "ECUS"} {
		if _, exists := decoded[0][excluded]; exists {
			t.Fatalf("excluded JSON field %q is present", excluded)
		}
	}
}

func TestNormalizeSeriesDefaultsAndRejectsUnknown(t *testing.T) {
	got, err := normalizeSeries(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != "level" || got[1] != "salinity" || got[2] != "ec" {
		t.Fatalf("default series = %v", got)
	}
	if _, err := normalizeSeries([]string{"oxygen"}); err == nil {
		t.Fatal("expected unsupported series error")
	}
}

func TestLoadCorrectedDataUsesNullableSourceMeasurements(t *testing.T) {
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	_, err = database.Exec(`
		CREATE TABLE logger_data(id INTEGER PRIMARY KEY, temp_c REAL, sal_psu REAL, ec_us REAL);
		CREATE TABLE corrected_data(id INTEGER PRIMARY KEY, site_id INTEGER, logger_data_id INTEGER, timestamp TEXT, corrected_value REAL, temp_c REAL, sal_psu REAL, ec_us REAL);
		INSERT INTO logger_data(id, temp_c, sal_psu, ec_us) VALUES (1, 20, NULL, NULL), (2, 21, 0, 0);
		INSERT INTO corrected_data(id, site_id, logger_data_id, timestamp, corrected_value, temp_c, sal_psu, ec_us) VALUES
			(1, 1, 1, '2026-01-01T00:00:00Z', 1, 20, 0, 0),
			(2, 1, 2, '2026-01-01T00:01:00Z', 2, 21, 0, 0);
	`)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := loadCorrectedData(database, 1)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].SalPSU != nil || rows[0].ECUS != nil {
		t.Fatalf("legacy fabricated zeroes were not restored to null: %#v", rows[0])
	}
	if rows[1].SalPSU == nil || *rows[1].SalPSU != 0 || rows[1].ECUS == nil || *rows[1].ECUS != 0 {
		t.Fatalf("real source zeroes were not preserved: %#v", rows[1])
	}
}
