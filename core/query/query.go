package query

import (
	"aqualog/core/db"
	"database/sql"
	"fmt"
	"time"
)

type LoggerDataRow struct {
	ID        int
	LoggerID  int
	Timestamp time.Time
	LevelM    float64
	TempC     sql.NullFloat64
	SalPSU    sql.NullFloat64
	ECUS      sql.NullFloat64
}

type CorrectedDataRow struct {
	ID             int
	SiteID         int
	Timestamp      time.Time
	CorrectedValue float64
	TempC          sql.NullFloat64
	SalPSU         sql.NullFloat64
	ECUS           sql.NullFloat64
}

type Options struct {
	From     string
	To       string
	LoggerID int
	Limit    int
}

func QueryLoggerData(siteID int) ([]LoggerDataRow, error) {
	return QueryLoggerDataWithOptions(siteID, Options{})
}

func QueryLoggerDataWithOptions(siteID int, opts Options) ([]LoggerDataRow, error) {

	dbConn, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer dbConn.Close()

	sqlText := `
	SELECT id, logger_id, timestamp, level_m, temp_c, sal_psu, ec_us
	FROM logger_data
	WHERE logger_id IN (
		SELECT id FROM loggers WHERE site_id = ?
	)
	`
	args := []any{siteID}
	if opts.LoggerID > 0 {
		sqlText += " AND logger_id = ?"
		args = append(args, opts.LoggerID)
	}
	if opts.From != "" {
		sqlText += " AND timestamp >= ?"
		args = append(args, opts.From)
	}
	if opts.To != "" {
		sqlText += " AND timestamp <= ?"
		args = append(args, opts.To)
	}
	sqlText += " ORDER BY timestamp ASC"
	if opts.Limit > 0 {
		sqlText += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	rows, err := dbConn.Query(sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("query logger_data: %w", err)
	}
	defer rows.Close()

	var loggerData []LoggerDataRow
	for rows.Next() {
		var row LoggerDataRow
		if err := rows.Scan(&row.ID, &row.LoggerID, &row.Timestamp, &row.LevelM, &row.TempC, &row.SalPSU, &row.ECUS); err != nil {
			return nil, fmt.Errorf("scan logger_data: %w", err)
		}
		loggerData = append(loggerData, row)
	}

	return loggerData, nil
}

func QueryCorrectedData(siteID int) ([]CorrectedDataRow, error) {
	return QueryCorrectedDataWithOptions(siteID, Options{})
}

func QueryCorrectedDataWithOptions(siteID int, opts Options) ([]CorrectedDataRow, error) {

	dbConn, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer dbConn.Close()

	sqlText := `
	SELECT id, site_id, timestamp, corrected_value, temp_c, sal_psu, ec_us
	FROM corrected_data
	WHERE site_id = ?
	`
	args := []any{siteID}
	if opts.From != "" {
		sqlText += " AND timestamp >= ?"
		args = append(args, opts.From)
	}
	if opts.To != "" {
		sqlText += " AND timestamp <= ?"
		args = append(args, opts.To)
	}
	sqlText += " ORDER BY timestamp ASC"
	if opts.Limit > 0 {
		sqlText += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	rows, err := dbConn.Query(sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("query corrected_data: %w", err)
	}
	defer rows.Close()

	var correctedData []CorrectedDataRow
	for rows.Next() {
		var row CorrectedDataRow
		if err := rows.Scan(&row.ID, &row.SiteID, &row.Timestamp, &row.CorrectedValue, &row.TempC, &row.SalPSU, &row.ECUS); err != nil {
			return nil, fmt.Errorf("scan corrected_data: %w", err)
		}
		correctedData = append(correctedData, row)
	}

	return correctedData, nil
}

func NullFloattoString(nf sql.NullFloat64) string {
	if nf.Valid {
		return fmt.Sprintf("%.2f", nf.Float64)
	}
	return "NULL"
}
