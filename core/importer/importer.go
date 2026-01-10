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

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	fmt.Printf("%s took %s\n", name, elapsed)
}

func ImportRecords(
	meta parsers.Metadata,
	recs []parsers.Record,
	siteID int,
	loggerID int,
) (*ImportResult, error) {

	defer timeTrack(time.Now(), "ImportRecords")

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
	const batchSize = 500

	for i := 0; i < len(recs); i += batchSize {
		end := i + batchSize
		if end > len(recs) {
			end = len(recs)
		}

		query := `
			INSERT OR IGNORE INTO logger_data (
				logger_id, timestamp, level_m, temp_c, sal_psu, ec_us
			) VALUES
		`

		args := make([]any, 0, (end-i)*6)

		for _, rec := range recs[i:end] {
			query += "(?, ?, ?, ?, ?, ?),"
			args = append(args,
				loggerID,
				rec.Timestamp,
				rec.LevelM,
				rec.TempC,
				rec.SalPSU,
				rec.EC,
			)
		}

		// remove trailing comma
		query = query[:len(query)-1]

		res, err := dbConn.Exec(query, args...)
		if err != nil {
			return nil, err
		}

		rows, _ := res.RowsAffected()
		inserted := int(rows)
		attempted := end - i

		result.Inserted += inserted
		result.Skipped += attempted - inserted
	}

	return result, nil
}
