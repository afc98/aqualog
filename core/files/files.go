package files

import (
	"aqualog/core/db"
	"database/sql"
	"fmt"
	"time"
)

type LoggerFile struct {
	ID        int
	LoggerID  int
	Logger    string
	Timestamp time.Time
	Path      string
	Type      string
	Rows      int
}

func List(siteID int, loggerID int) ([]LoggerFile, error) {
	d, err := db.GetDB()
	if err != nil {
		return nil, err
	}
	defer d.Close()

	args := []any{siteID}
	filter := ""
	if loggerID > 0 {
		filter = " AND l.id = ?"
		args = append(args, loggerID)
	}
	rows, err := d.Query(`
		SELECT f.id, f.logger_id, l.name, f.timestamp, f.file, f.type,
			(SELECT COUNT(*) FROM logger_data d WHERE d.logger_file_id = f.id)
		FROM logger_files f
		JOIN loggers l ON l.id = f.logger_id
		WHERE l.site_id = ?`+filter+`
		ORDER BY f.timestamp, f.id
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LoggerFile
	for rows.Next() {
		var f LoggerFile
		var ts string
		var typ sql.NullString
		if err := rows.Scan(&f.ID, &f.LoggerID, &f.Logger, &ts, &f.Path, &typ, &f.Rows); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05", ts)
			if err != nil {
				return nil, err
			}
		}
		f.Timestamp = t
		if typ.Valid {
			f.Type = typ.String
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func Remove(id int) (int, int, error) {
	d, err := db.GetDB()
	if err != nil {
		return 0, 0, err
	}
	defer d.Close()

	var siteID int
	err = d.QueryRow(`
		SELECT l.site_id
		FROM logger_files f
		JOIN loggers l ON l.id = f.logger_id
		WHERE f.id = ?
	`, id).Scan(&siteID)
	if err != nil {
		return 0, 0, err
	}

	tx, err := d.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM logger_data WHERE logger_file_id = ?`, id)
	if err != nil {
		return 0, 0, err
	}
	rows, _ := res.RowsAffected()
	if _, err := tx.Exec(`DELETE FROM logger_files WHERE id = ?`, id); err != nil {
		return 0, 0, err
	}
	if _, err := tx.Exec(`DELETE FROM corrected_data WHERE site_id = ?`, siteID); err != nil {
		return 0, 0, fmt.Errorf("failed to invalidate corrected data: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return siteID, int(rows), nil
}
