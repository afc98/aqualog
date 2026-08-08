package export

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/process"
	"os"
	"time"
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
	Timestamp      string
	SiteName       string
	LoggerName     string
	LoggerModel    string
	LoggerSerial   string
	Level          float64
	ManualReading  *float64
	CorrectedLevel float64
	SalinityPSU    float64
	ECUS           float64
}

type Options struct {
	From   string
	To     string
	Format string
}

func ExportData(siteID int, fileName string) error {
	return ExportDataWithOptions(siteID, fileName, Options{Format: "csv"})
}

func ExportDataWithOptions(siteID int, fileName string, opts Options) error {
	rows, err := BuildExport(siteID)
	if err != nil {
		return err
	}
	rows, err = filterRows(rows, opts)
	if err != nil {
		return err
	}
	switch opts.Format {
	case "", "csv":
		return WriteCSV(rows, fileName)
	case "json":
		return WriteJSON(rows, fileName)
	default:
		return fmt.Errorf("unsupported export format %q", opts.Format)
	}
}

func BuildExport(siteID int) ([]ExportRow, error) {
	dbConn, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer dbConn.Close()

	rawRows, err := process.LoadRawData(dbConn, siteID)
	if err != nil {
		return nil, err
	}

	measurements, err := process.LoadManualMeasurements(dbConn, siteID)
	if err != nil {
		return nil, err
	}

	correctedRows, err := loadCorrectedData(dbConn, siteID)
	if err != nil {
		return nil, err
	}

	site, err := db.GetSiteByID(dbConn, siteID)
	if err != nil {
		return nil, err
	}

	var out []ExportRow

	for _, row := range correctedRows {
		manual := manualMeasurementAt(measurements, row.Timestamp)

		name, model, serial, err := loggerAt(dbConn, siteID, row.Timestamp)
		if err != nil {
			return nil, err
		}

		out = append(out, ExportRow{
			Timestamp:      row.Timestamp.Format(time.RFC3339),
			SiteName:       site.Name,
			LoggerName:     name,
			LoggerModel:    model,
			LoggerSerial:   serial,
			Level:          rawValueAt(rawRows, row.Timestamp),
			ManualReading:  manual,
			CorrectedLevel: row.Value,
			SalinityPSU:    row.SalPSU,
			ECUS:           row.ECUS,
		})
	}

	return out, nil
}

func WriteCSV(rows []ExportRow, fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	w.Write([]string{
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
	})

	for _, r := range rows {
		manual := ""
		if r.ManualReading != nil {
			manual = fmt.Sprintf("%.4f", *r.ManualReading)
		}

		w.Write([]string{
			r.Timestamp,
			r.SiteName,
			r.LoggerName,
			r.LoggerModel,
			r.LoggerSerial,
			fmt.Sprintf("%.4f", r.Level),
			manual,
			fmt.Sprintf("%.4f", r.CorrectedLevel),
			fmt.Sprintf("%.4f", r.SalinityPSU),
			fmt.Sprintf("%.4f", r.ECUS),
		})
	}

	return nil
}

func WriteJSON(rows []ExportRow, fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

func filterRows(rows []ExportRow, opts Options) ([]ExportRow, error) {
	var from, to time.Time
	var err error
	if opts.From != "" {
		from, err = time.Parse(time.RFC3339, opts.From)
		if err != nil {
			return nil, fmt.Errorf("invalid from timestamp: %w", err)
		}
	}
	if opts.To != "" {
		to, err = time.Parse(time.RFC3339, opts.To)
		if err != nil {
			return nil, fmt.Errorf("invalid to timestamp: %w", err)
		}
	}
	if opts.From == "" && opts.To == "" {
		return rows, nil
	}
	out := make([]ExportRow, 0, len(rows))
	for _, row := range rows {
		t, err := time.Parse(time.RFC3339, row.Timestamp)
		if err != nil {
			return nil, err
		}
		if opts.From != "" && t.Before(from) {
			continue
		}
		if opts.To != "" && t.After(to) {
			continue
		}
		out = append(out, row)
	}
	return out, nil
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

	var nullName, nullModel, nullSerial sql.NullString
	err = row.Scan(&nullName, &nullModel, &nullSerial)
	if err == sql.ErrNoRows {
		return "", "", "", nil
	}
	if err != nil {
		return "", "", "", err
	}

	if nullName.Valid {
		name = nullName.String
	}
	if nullModel.Valid {
		model = nullModel.String
	}
	if nullSerial.Valid {
		serial = nullSerial.String
	}

	return name, model, serial, nil
}
