package importer

import (
	"aqualog/core/db"
	"aqualog/core/parsers"
	"database/sql"
	"fmt"
	"os"
)

func ImportLoggerFile(fileType, filePath string, siteID, loggerID int) error {

	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("Could not open file: %w", err)
	}
	var meta parsers.Metadata
	var recs []parsers.Record
	switch fileType {
	case "aquaread":
		fmt.Printf("Importing Aquaread file: %s (site %d)\n", filePath, siteID)
		parsedData, err := parsers.ParseAquaread(filePath)
		if err != nil {
			return fmt.Errorf("Failed to parse aquaread data")
		}
		meta = parsedData.Metadata
		recs = parsedData.Records
	case "solinst":
		fmt.Printf("Importing Solinst file: %s (site %d)\n", filePath, siteID)
		parsedData, err := parsers.ParseSolinst(filePath)
		if err != nil {
			return fmt.Errorf("Failed to parse solinst data")
		}
		meta = parsedData.Metadata
		recs = parsedData.Records
	case "in-situ":
		return fmt.Errorf("In-Situ logger filetype support is not yet implemented")
	default:
		return fmt.Errorf("Logger file type not recognised")
	}

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Check if logger already exists using its serial number if a logger ID has not been provided
	if loggerID == -1 {
		err = database.QueryRow(`
		SELECT id FROM loggers
		WHERE serial_number=? AND site_id=?
		`, meta.SerialNumber, siteID).Scan(&loggerID)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("No logger with matching metadata found in database.")
		}
		if err == nil {
			fmt.Println("Found existing logger, ID:", loggerID)
		}

		// Add a new logger if it doesn't already exist
		if err == sql.ErrNoRows {
			fmt.Println("Adding new logger based on file metadata")
			res, err := database.Exec(`
				INSERT INTO loggers (site_id, name, serial_number)
				VALUES (?, ?, ?)
			`, siteID, meta.SiteIdent, meta.SerialNumber)
			if err != nil {
				return fmt.Errorf("Unable to create a new logger entry from file metadata")
			}
			loggerID64, _ := res.LastInsertId()
			loggerID = int(loggerID64)
		}
	}
	// Insert records into logger_data
	fmt.Printf("Inserting %d records for logger %d\n", len(recs), loggerID)
	skipped := 0
	for _, rec := range recs {
		// Check if record already exists
		var existingID int
		err := database.QueryRow(`
		SELECT id FROM logger_data
		WHERE logger_id=? AND timestamp=?
		`, loggerID, rec.Timestamp).Scan(&existingID)

		if err == nil {
			// Record already exists, skip it
			skipped++
			continue
		} else if err != sql.ErrNoRows {
			// Database error
			return fmt.Errorf("Error checking for existing record at %s: %v\n", rec.Timestamp, err)
		}

		// Record doesn't exist, insert it
		_, err = database.Exec(`
			INSERT INTO logger_data (logger_id, timestamp, level_m, temp_c, sal_psu, ec_us)
			VALUES (?, ?, ?, ?, ?, ?)
		`, loggerID, rec.Timestamp.Format("2006-01-02 15:04:05Z"), rec.LevelM, rec.TempC, rec.SalPSU, rec.EC)
		if err != nil {
			return fmt.Errorf("Failed to insert record at %s: %v\n", rec.Timestamp, err)
		}
	}
	fmt.Printf("Successfully imported %d records (%d duplicate rows skipped)\n", len(recs)-skipped, skipped)
	return nil
}
