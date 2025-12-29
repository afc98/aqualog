package importer

import (
	"aqualog/core/db"
	"aqualog/core/parsers"
	"database/sql"
	"fmt"
	"os"
	"time"
)

type ImportResult struct {
	LoggerID      int
	Inserted      int
	Skipped       int
	LoggerCreated bool
}

func ParseLoggerFile(fileType, filePath string) (parsers.Metadata, []parsers.Record, error) {
	if _, err := os.Stat(filePath); err != nil {
		return parsers.Metadata{}, nil, err
	}

	switch fileType {
	case "aquaread":
		data, err := parsers.ParseAquaread(filePath)
		if err != nil {
			return parsers.Metadata{}, nil, err
		}
		return data.Metadata, data.Records, nil

	case "solinst":
		data, err := parsers.ParseSolinst(filePath)
		if err != nil {
			return parsers.Metadata{}, nil, err
		}
		return data.Metadata, data.Records, nil

	default:
		return parsers.Metadata{}, nil, fmt.Errorf("unsupported file type")
	}
}

func ImportRecords(
	meta parsers.Metadata,
	recs []parsers.Record,
	siteID int,
	loggerID int,
) (*ImportResult, error) {

	dbConn, err := db.GetDB()
	if err != nil {
		return nil, err
	}
	defer dbConn.Close()

	result := &ImportResult{}

	// Resolve or create logger
	if loggerID == -1 {
		err = dbConn.QueryRow(`
			SELECT id FROM loggers
			WHERE serial_number=? AND site_id=?
		`, meta.SerialNumber, siteID).Scan(&loggerID)

		if err == sql.ErrNoRows {
			res, err := dbConn.Exec(`
				INSERT INTO loggers (site_id, name, serial_number)
				VALUES (?, ?, ?)
			`, siteID, meta.SiteIdent, meta.SerialNumber)
			if err != nil {
				return nil, err
			}
			id, _ := res.LastInsertId()
			loggerID = int(id)
			result.LoggerCreated = true
		} else if err != nil {
			return nil, err
		}
	}

	result.LoggerID = loggerID

	// Insert records
	for _, rec := range recs {
		var existingID int
		err := dbConn.QueryRow(`
			SELECT id FROM logger_data
			WHERE logger_id=? AND timestamp=?
		`, loggerID, rec.Timestamp).Scan(&existingID)

		if err == nil {
			result.Skipped++
			continue
		}
		if err != sql.ErrNoRows {
			return nil, err
		}

		_, err = dbConn.Exec(`
			INSERT INTO logger_data (logger_id, timestamp, level_m, temp_c, sal_psu, ec_us)
			VALUES (?, ?, ?, ?, ?, ?)
		`, loggerID,
			rec.Timestamp.Format(time.RFC3339),
			rec.LevelM,
			rec.TempC,
			rec.SalPSU,
			rec.EC,
		)
		if err != nil {
			return nil, err
		}

		result.Inserted++
	}

	return result, nil
}
