package export

import (
	"aqualog/db"
	"aqualog/process"
	"database/sql"
	"fmt"
	"time"
	"os"
	"encoding/csv"
)

type LoggerCorrectedRow struct {
	ID        int
	SiteID    int
	Timestamp time.Time
	Value     float64
	TempC     float64
	SalPSU    float64
	ECUS      float64
}

type ExportRow struct {
	Timestamp        string
	SiteName         string
	LoggerName       string
	LoggerModel      string
	LoggerSerial     string
	Level            float64
	ManualReading    *float64
	CorrectedLevel   float64
	SalinityPSU      float64
	ECUS             float64
}

func ExportData(dbConn *sql.DB, siteID int, fileName string) error {
	// Load data
	rawRows, err := process.LoadRawData(dbConn, siteID)
	if err != nil {
		return fmt.Errorf("load raw data: %w", err)
	}

	measurements, err := process.LoadManualMeasurements(dbConn, siteID)
	if err != nil {
		return fmt.Errorf("load manual measurements: %w", err)
	}

	correctedRows, err := loadCorrectedData(dbConn, siteID)
	if err != nil {
		return fmt.Errorf("load corrected data: %w", err)
	}

	site, err := db.GetSiteByID(dbConn, siteID)
	if err != nil {
		return fmt.Errorf("load site: %w", err)
	}

	// Open output file
	f, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("create export file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	if err := w.Write([]string{
		"timestamp",
		"site_name",
		"logger_name",
		"logger_model",
		"logger_serial",
		"level",
		"manual_reading",
		"corrected_level",
		"salinity_psu",
		"ec_us",
	}); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Iterate corrected data as canonical timeline
	for _, row := range correctedRows {
		manual := manualMeasurementAt(measurements, row.Timestamp)

		loggerName, loggerModel, loggerSerial, err := loggerAt(dbConn, siteID, row.Timestamp)
		if err != nil {
			return fmt.Errorf("resolve logger at %s: %w", row.Timestamp, err)
		}

		export := ExportRow{
			Timestamp:      row.Timestamp.Format(time.RFC3339),
			SiteName:       site.Name,
			LoggerName:     loggerName,
			LoggerModel:    loggerModel,
			LoggerSerial:   loggerSerial,
			Level:          rawValueAt(rawRows, row.Timestamp),
			ManualReading: manual,
			CorrectedLevel: row.Value,
			SalinityPSU:    row.SalPSU,
			ECUS:           row.ECUS,
		}

		manualStr := ""
		if export.ManualReading != nil {
			manualStr = fmt.Sprintf("%.4f", *export.ManualReading)
		}

		if err := w.Write([]string{
			export.Timestamp,
			export.SiteName,
			export.LoggerName,
			export.LoggerModel,
			export.LoggerSerial,
			fmt.Sprintf("%.4f", export.Level),
			manualStr,
			fmt.Sprintf("%.4f", export.CorrectedLevel),
			fmt.Sprintf("%.4f", export.SalinityPSU),
			fmt.Sprintf("%.4f", export.ECUS),
		}); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	return nil
}

func loadCorrectedData(dbConn *sql.DB, siteID int) ([]LoggerCorrectedRow, error) {
	rows, err := dbConn.Query(`
		SELECT
			id,
			site_id,
			timestamp,
			corrected_value,
			temp_c,
			sal_psu,
			ec_us
		FROM corrected_data
		WHERE site_id = ?
		ORDER BY timestamp ASC
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query corrected_data: %w", err)
	}
	defer rows.Close()

	var out []LoggerCorrectedRow

	for rows.Next() {
		var row LoggerCorrectedRow
		var ts string

		var tempC sql.NullFloat64
		var salPSU sql.NullFloat64
		var ecUS sql.NullFloat64

		if err := rows.Scan(
			&row.ID,
			&row.SiteID,
			&ts,
			&row.Value,
			&tempC,
			&salPSU,
			&ecUS,
		); err != nil {
			return nil, fmt.Errorf("scan logger_data: %w", err)
		}

		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp %q: %w", ts, err)
		}
		row.Timestamp = t

		if tempC.Valid {
			row.TempC = tempC.Float64
		}

		if salPSU.Valid {
			row.SalPSU = salPSU.Float64
		}

		if ecUS.Valid {
			row.ECUS = ecUS.Float64
		}

		out = append(out, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return out, nil
}

func manualMeasurementAt(rows []process.ManualMeasurement, t time.Time) *float64 {
	for _, r := range rows {
		if r.Timestamp.Equal(t) {
			v := r.Level
			return &v
		}
	}
	return nil
}

func rawValueAt(rows []process.LoggerRawRow, t time.Time) float64 {
	for _, r := range rows {
		if r.Timestamp.Equal(t) {
			return r.RawValue
		}
	}
	return 0
}

func loggerAt(db *sql.DB, siteID int, t time.Time) (name, model, serial string, err error) {
	row := db.QueryRow(`
		SELECT l.name, l.model, l.serial_number
		FROM logger_events e
		JOIN loggers l ON l.id = e.logger_id
		WHERE e.logger_id IN (SELECT id FROM loggers WHERE site_id=?)
		  AND e.timestamp <= ?
		  AND e.event_type IN ('installed', 'removed')
		ORDER BY e.timestamp DESC
		LIMIT 1
	`, siteID, t.Format(time.RFC3339))

	err = row.Scan(&name, &model, &serial)
	if err == sql.ErrNoRows {
		return "", "", "", nil
	}
	if err != nil {
		return "", "", "", err
	}

	return name, model, serial, nil
}
