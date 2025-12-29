package logger

import (
	"aqualog/core/db"
	"fmt"
)

type AddParams struct {
	SiteID int
	Name   string
	Model  string
	Serial string
}

type Logger struct {
	ID     int
	SiteID int
	Name   string
	Model  string
	Serial string
}

func Add(p AddParams) (Logger, error) {
	// Open database
	database, err := db.GetDB()
	if err != nil {
		return Logger{}, fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Insert logger
	result, err := database.Exec(
		`INSERT INTO loggers (site_id, name, model, serial_number) VALUES (?, ?, ?, ?)`,
		p.SiteID, p.Name, p.Model, p.Serial,
	)
	if err != nil {
		return Logger{}, fmt.Errorf("Failed to add logger: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Logger{}, err
	}

	return Logger{
		ID:     int(id),
		SiteID: p.SiteID,
		Name:   p.Name,
		Model:  p.Model,
		Serial: p.Serial,
	}, nil
}

func List(SiteID int) ([]Logger, error) {
	if SiteID == 0 {
		return nil, fmt.Errorf("must specify site")
	}

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer database.Close()

	// Query projects
	rows, err := database.Query(`
	SELECT id, site_id, name, model, serial_number 
	FROM loggers WHERE site_id=?
	ORDER BY id
	`, SiteID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	loggers := []Logger{}

	for rows.Next() {
		var l Logger
		if err := rows.Scan(
			&l.ID,
			&l.SiteID,
			&l.Name,
			&l.Model,
			&l.Serial,
		); err != nil {
			return nil, err
		}
		loggers = append(loggers, l)
	}

	return loggers, nil
}
