package query

import (
	"database/sql"
	"fmt"
)

func QueryLoggerData(dbConn *sql.DB, siteID int) error {
	rows, err := dbConn.Query(`
	SELECT id, logger_id, timestamp, level_m, temp_c, sal_psu, ec_us
	FROM logger_data
	WHERE logger_id IN (
		SELECT id FROM loggers WHERE site_id = ?
	)
	ORDER BY timestamp ASC;
	`, siteID)
	if err != nil {
		return fmt.Errorf("query logger_data: %w", err)
	}
	defer rows.Close()
	fmt.Println("ID, LoggerID, Timestamp, Level(m), Temp(C), Sal(PSU), EC(uS)")
	for rows.Next() {
		// Process each row
		var id int
		var loggerID int
		var timestamp string
		var levelM float64
		var tempC sql.NullFloat64
		var salPSU sql.NullFloat64
		var ecUS sql.NullFloat64

		if err := rows.Scan(&id, &loggerID, &timestamp, &levelM, &tempC, &salPSU, &ecUS); err != nil {
			return fmt.Errorf("scan logger_data: %w", err)
		}

		// Print data to SDTOUT
		fmt.Printf("%d, %d, %s, %.2f, %v, %v, %v\n",
			id, loggerID, timestamp, levelM,
			NullFloattoString(tempC),
			NullFloattoString(salPSU),
			NullFloattoString(ecUS),
		)
	}
	return nil
}

func QueryCorrectedData(dbConn *sql.DB, siteID int) error {
	rows, err := dbConn.Query(`
	SELECT id, site_id, timestamp, corrected_value, temp_c, sal_psu, ec_us
	FROM corrected_data
	WHERE site_id = ?
	ORDER BY timestamp ASC;
	`, siteID)
	if err != nil {
		return fmt.Errorf("query corrected_data: %w", err)
	}
	defer rows.Close()
	fmt.Println("ID, SiteID, Timestamp, Level(m), Temp(C), Sal(PSU), EC(uS)")
	for rows.Next() {
		// Process each row
		var id int
		var siteID int
		var timestamp string
		var levelM float64
		var tempC sql.NullFloat64
		var salPSU sql.NullFloat64
		var ecUS sql.NullFloat64

		if err := rows.Scan(&id, &siteID, &timestamp, &levelM, &tempC, &salPSU, &ecUS); err != nil {
			return fmt.Errorf("scan logger_data: %w", err)
		}

		// Print data to SDTOUT
		fmt.Printf("%d, %d, %s, %.2f, %v, %v, %v\n",
			id, siteID, timestamp, levelM,
			NullFloattoString(tempC),
			NullFloattoString(salPSU),
			NullFloattoString(ecUS),
		)
	}
	return nil
}

func NullFloattoString(nf sql.NullFloat64) string {
	if nf.Valid {
		return fmt.Sprintf("%.2f", nf.Float64)
	}
	return "NULL"
}
