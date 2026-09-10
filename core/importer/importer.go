package importer

import (
	"aqualog/core/db"
	"aqualog/core/parsers"
	"aqualog/core/utils"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"
)

var ErrFileAlreadyRecorded = errors.New("file already recorded for logger")

type ImportResult struct {
	LoggerID      int
	Inserted      int
	Skipped       int
	Replaced      int
	LoggerCreated bool
}

type ImportOptions struct {
	ReplaceExisting bool
	Role            string
	MeasurementKind string
	DateOrder       string
}

type File struct {
	LoggerID int
	Path     string
	Type     string
}

func ParseLoggerFile(fileType, filePath string) (parsers.Metadata, []parsers.Record, error) {
	return ParseLoggerFileWithOptions(fileType, filePath, ImportOptions{})
}

func ParseLoggerFileWithOptions(fileType, filePath string, options ImportOptions) (parsers.Metadata, []parsers.Record, error) {
	if _, err := os.Stat(filePath); err != nil {
		return parsers.Metadata{}, nil, err
	}

	switch fileType {
	case "aquaread":
		data, err := parsers.ParseAquareadWithOptions(filePath, parsers.DateOptions{DateOrder: options.DateOrder})
		if err != nil {
			return parsers.Metadata{}, nil, err
		}
		return data.Metadata, data.Records, nil

	case "solinst":
		data, err := parsers.ParseSolinstWithOptions(filePath, parsers.SolinstOptions{DateOrder: options.DateOrder})
		if err != nil {
			return parsers.Metadata{}, nil, err
		}
		return data.Metadata, data.Records, nil

	case "insitu":
		data, err := parsers.ParseInsituWithOptions(filePath, parsers.DateOptions{DateOrder: options.DateOrder})
		if err != nil {
			return parsers.Metadata{}, nil, err
		}
		return data.Metadata, data.Records, nil

	default:
		return parsers.Metadata{}, nil, fmt.Errorf("unsupported file type")
	}
}

func LoggerFileExists(loggerID int, filePath string) (bool, error) {
	dbConn, err := db.GetDB()
	if err != nil {
		return false, err
	}

	var exists bool
	err = dbConn.QueryRow(
		`SELECT EXISTS(
			SELECT 1
			FROM logger_files
			WHERE logger_id = $1 AND file = $2
		)`,
		loggerID, filePath,
	).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}

func loggerFileExistsTx(tx *sql.Tx, loggerID int, filePath string) (bool, error) {
	var exists bool
	err := tx.QueryRow(
		`SELECT EXISTS(
			SELECT 1
			FROM logger_files
			WHERE logger_id = $1 AND file = $2
		)`,
		loggerID, filePath,
	).Scan(&exists)

	return exists, err
}

func insertLoggerFileTx(
	tx *sql.Tx,
	loggerID int,
	filePath string,
	fileType string,
	measurementKind string,
	measurementUnit string,
) (int, error) {

	var id int
	err := tx.QueryRow(
		`INSERT INTO logger_files (logger_id, file, type, measurement_kind, measurement_unit)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		loggerID, filePath, fileType, measurementKind, measurementUnit,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func firstMeasurement(recs []parsers.Record) (string, string) {
	if len(recs) == 0 {
		return "", ""
	}
	return recs[0].MeasurementKind, recs[0].ImportedUnit
}

func ImportRecords(
	meta parsers.Metadata,
	recs []parsers.Record,
	siteID int,
	loggerID int,
	filePath string,
	fileType string,
	options ImportOptions,
) (*ImportResult, error) {

	defer utils.TimeTrack(time.Now(), "ImportRecords")

	// Advisory pre-flight check for explicitly selected loggers. Auto-detected
	// loggers are checked after the logger has been resolved or created.
	if loggerID != -1 {
		exists, err := LoggerFileExists(loggerID, filePath)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrFileAlreadyRecorded
		}
	}

	dbConn, err := db.GetDB()
	if err != nil {
		return nil, err
	}

	tx, err := dbConn.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	result := &ImportResult{}
	role := options.Role
	if role == "" {
		role = "water_level"
		if len(recs) > 0 && isBarometricKind(recs[0].MeasurementKind) {
			role = "barometric"
		}
	}

	// Resolve or create logger (inside transaction)
	if loggerID == -1 {
		err = tx.QueryRow(`
			SELECT id
			FROM loggers
			WHERE serial_number = $1 AND site_id = $2
		`, meta.SerialNumber, siteID).Scan(&loggerID)

		if err == sql.ErrNoRows {
			err := tx.QueryRow(`
				INSERT INTO loggers (site_id, name, serial_number, role)
				VALUES ($1, $2, $3, $4)
				RETURNING id
			`, siteID, meta.SiteIdent, meta.SerialNumber, role).Scan(&loggerID)
			if err != nil {
				return nil, err
			}
			result.LoggerCreated = true
		} else if err != nil {
			return nil, err
		}
	}
	// Use the active transaction here. Opening another connection after logger
	// resolution can deadlock with this transaction's SQLite lock.
	exists, err := loggerFileExistsTx(tx, loggerID, filePath)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrFileAlreadyRecorded
	}
	if _, err := tx.Exec(`UPDATE loggers SET role = $1 WHERE id = $2`, role, loggerID); err != nil {
		return nil, err
	}

	result.LoggerID = loggerID

	// Insert logger_files FIRST
	kind, unit := firstMeasurement(recs)
	if options.MeasurementKind != "" {
		kind = options.MeasurementKind
	}
	loggerFileID, err := insertLoggerFileTx(
		tx,
		loggerID,
		filePath,
		fileType,
		kind,
		unit,
	)
	if err != nil {
		return nil, err
	}

	// SQLite max_bind_vars ~= 999
	const batchSize = 130

	for i := 0; i < len(recs); i += batchSize {
		end := i + batchSize
		if end > len(recs) {
			end = len(recs)
		}

		query := fmt.Sprintf(`
			INSERT INTO logger_data (
				logger_id,
				logger_file_id,
				timestamp,
				level_m,
				imported_value,
				imported_unit,
				imported_kind,
				temp_c,
				sal_psu,
				ec_us
			) VALUES
		`)

		args := make([]any, 0, (end-i)*10)

		for _, rec := range recs[i:end] {
			recKind := rec.MeasurementKind
			if options.MeasurementKind != "" {
				recKind = options.MeasurementKind
			}
			level := rec.LevelM
			if isBarometricKind(recKind) || isAbsolutePressureKind(recKind) {
				level = 0
			}
			base := len(args)
			query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d),",
				base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9, base+10)
			args = append(args,
				loggerID,
				loggerFileID,
				rec.Timestamp,
				level,
				rec.ImportedValue,
				rec.ImportedUnit,
				recKind,
				rec.TempC,
				rec.SalPSU,
				rec.EC,
			)
		}

		query = query[:len(query)-1]
		if options.ReplaceExisting {
			replaced, err := countExistingRecordsTx(tx, loggerID, recs[i:end])
			if err != nil {
				return nil, err
			}
			result.Replaced += replaced
			result.Inserted += (end - i) - replaced

			query += `
				ON CONFLICT(logger_id, timestamp) DO UPDATE SET
					logger_file_id = excluded.logger_file_id,
					level_m = excluded.level_m,
					imported_value = excluded.imported_value,
					imported_unit = excluded.imported_unit,
					imported_kind = excluded.imported_kind,
					temp_c = excluded.temp_c,
					sal_psu = excluded.sal_psu,
					ec_us = excluded.ec_us
			`
		} else {
			query += ` ON CONFLICT(logger_id, timestamp) DO NOTHING`
		}

		res, err := tx.Exec(query, args...)
		if err != nil {
			return nil, err
		}

		if !options.ReplaceExisting {
			rows, _ := res.RowsAffected()
			inserted := int(rows)
			attempted := end - i

			result.Inserted += inserted
			result.Skipped += attempted - inserted
		}
	}

	if result.Replaced > 0 {
		if _, err := tx.Exec(`DELETE FROM corrected_data WHERE site_id = $1`, siteID); err != nil {
			return nil, fmt.Errorf("failed to invalidate corrected data: %w", err)
		}
	}

	// Commit only after everything succeeds
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return result, nil
}

func countExistingRecordsTx(tx *sql.Tx, loggerID int, recs []parsers.Record) (int, error) {
	if len(recs) == 0 {
		return 0, nil
	}

	query := `
		SELECT COUNT(*)
		FROM logger_data
		WHERE logger_id = $1
		  AND timestamp IN (
	`
	args := make([]any, 0, len(recs)+1)
	args = append(args, loggerID)

	for i, rec := range recs {
		query += fmt.Sprintf("$%d,", i+2)
		args = append(args, rec.Timestamp)
	}

	query = query[:len(query)-1] + ")"

	var count int
	if err := tx.QueryRow(query, args...).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func isBarometricKind(kind string) bool {
	return kind == "barometric_pressure_mbar" || kind == "barometric_pressure_kpa"
}

func isAbsolutePressureKind(kind string) bool {
	return kind == "absolute_pressure_mbar" || kind == "absolute_pressure_kpa" || kind == "absolute_pressure_head_m"
}
