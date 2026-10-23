package export

import (
	"aqualog/core/db"
	"aqualog/core/process"
	"aqualog/core/utils"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type LoggerCorrectedRow struct {
	ID        int
	SiteID    int
	Timestamp time.Time
	Value     float64
	TempC     *float64
	SalPSU    *float64
	ECUS      *float64
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
	TempC          *float64
	SalinityPSU    *float64
	ECUS           *float64
}

type Options struct {
	From   string
	To     string
	Format string
	Series []string
}

var defaultSeries = []string{"level", "salinity", "ec"}

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
	series, err := normalizeSeries(opts.Series)
	if err != nil {
		return err
	}
	switch opts.Format {
	case "", "csv":
		return writeCSVWithSeries(rows, fileName, series)
	case "json":
		return writeJSONWithSeries(rows, fileName, series)
	default:
		return fmt.Errorf("unsupported export format %q", opts.Format)
	}
}

func BuildExport(siteID int) ([]ExportRow, error) {
	dbConn, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

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
			Timestamp:      utils.FormatDatasetTime(row.Timestamp),
			SiteName:       site.Name,
			LoggerName:     name,
			LoggerModel:    model,
			LoggerSerial:   serial,
			Level:          rawValueAt(rawRows, row.Timestamp),
			ManualReading:  manual,
			CorrectedLevel: row.Value,
			TempC:          row.TempC,
			SalinityPSU:    row.SalPSU,
			ECUS:           row.ECUS,
		})
	}

	return out, nil
}

func WriteCSV(rows []ExportRow, fileName string) error {
	return writeCSVWithSeries(rows, fileName, defaultSeries)
}

func writeCSVWithSeries(rows []ExportRow, fileName string, series []string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"timestamp",
		"site_name",
		"logger_name",
		"logger_model",
		"logger_serial",
	}
	selected := seriesSet(series)
	if selected["level"] {
		header = append(header, "level", "manual_reading", "corrected_level")
	}
	if selected["temp"] {
		header = append(header, "temp_c")
	}
	if selected["salinity"] {
		header = append(header, "salinity_psu")
	}
	if selected["ec"] {
		header = append(header, "ec_us")
	}
	w.Write(header)

	for _, r := range rows {
		manual := ""
		if r.ManualReading != nil {
			manual = fmt.Sprintf("%.4f", *r.ManualReading)
		}

		record := []string{
			r.Timestamp,
			r.SiteName,
			r.LoggerName,
			r.LoggerModel,
			r.LoggerSerial,
		}
		if selected["level"] {
			record = append(record, fmt.Sprintf("%.4f", r.Level), manual, fmt.Sprintf("%.4f", r.CorrectedLevel))
		}
		if selected["temp"] {
			record = append(record, formatOptionalFloat(r.TempC))
		}
		if selected["salinity"] {
			record = append(record, formatOptionalFloat(r.SalinityPSU))
		}
		if selected["ec"] {
			record = append(record, formatOptionalFloat(r.ECUS))
		}
		w.Write(record)
	}

	return nil
}

func formatOptionalFloat(value *float64) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%.4f", *value)
}

func WriteJSON(rows []ExportRow, fileName string) error {
	return writeJSONWithSeries(rows, fileName, defaultSeries)
}

func writeJSONWithSeries(rows []ExportRow, fileName string, series []string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()
	selected := seriesSet(series)
	projected := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		item := map[string]any{
			"Timestamp": row.Timestamp, "SiteName": row.SiteName, "LoggerName": row.LoggerName,
			"LoggerModel": row.LoggerModel, "LoggerSerial": row.LoggerSerial,
		}
		if selected["level"] {
			item["Level"] = row.Level
			item["ManualReading"] = row.ManualReading
			item["CorrectedLevel"] = row.CorrectedLevel
		}
		if selected["temp"] {
			item["TempC"] = row.TempC
		}
		if selected["salinity"] {
			item["SalinityPSU"] = row.SalinityPSU
		}
		if selected["ec"] {
			item["ECUS"] = row.ECUS
		}
		projected = append(projected, item)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(projected)
}

func normalizeSeries(values []string) ([]string, error) {
	if len(values) == 0 {
		return append([]string(nil), defaultSeries...), nil
	}
	selected := make(map[string]bool, len(values))
	for _, rawValue := range values {
		value := strings.ToLower(strings.TrimSpace(rawValue))
		switch value {
		case "level", "temp", "salinity", "ec":
			selected[value] = true
		default:
			return nil, fmt.Errorf("unsupported export series %q; use level, temp, salinity, or ec", rawValue)
		}
	}
	ordered := make([]string, 0, len(selected))
	for _, value := range []string{"level", "temp", "salinity", "ec"} {
		if selected[value] {
			ordered = append(ordered, value)
		}
	}
	return ordered, nil
}

func seriesSet(series []string) map[string]bool {
	selected := make(map[string]bool, len(series))
	for _, value := range series {
		selected[value] = true
	}
	return selected
}

func filterRows(rows []ExportRow, opts Options) ([]ExportRow, error) {
	var from, to time.Time
	var err error
	if opts.From != "" {
		from, err = utils.ParseDatasetTime(opts.From)
		if err != nil {
			return nil, fmt.Errorf("invalid from timestamp: %w", err)
		}
	}
	if opts.To != "" {
		to, err = utils.ParseDatasetTime(opts.To)
		if err != nil {
			return nil, fmt.Errorf("invalid to timestamp: %w", err)
		}
	}
	if opts.From == "" && opts.To == "" {
		return rows, nil
	}
	out := make([]ExportRow, 0, len(rows))
	for _, row := range rows {
		t, err := utils.ParseDatasetTime(row.Timestamp)
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
			c.id,
			c.site_id,
			c.timestamp,
			c.corrected_value,
			d.temp_c,
			d.sal_psu,
			d.ec_us
		FROM corrected_data c
		JOIN logger_data d ON d.id = c.logger_data_id
		WHERE c.site_id = $1
		ORDER BY c.timestamp ASC
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

		t, err := utils.ParseDatasetTime(ts)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp %q: %w", ts, err)
		}
		row.Timestamp = t

		if tempC.Valid {
			value := tempC.Float64
			row.TempC = &value
		}

		if salPSU.Valid {
			value := salPSU.Float64
			row.SalPSU = &value
		}

		if ecUS.Valid {
			value := ecUS.Float64
			row.ECUS = &value
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
		WHERE e.logger_id IN (SELECT id FROM loggers WHERE site_id=$1)
		  AND e.timestamp <= $2
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
