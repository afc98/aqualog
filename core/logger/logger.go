package logger

import (
	"aqualog/core/db"
	"database/sql"
	"fmt"
)

type AddParams struct {
	SiteID int
	Name   string
	Model  string
	Serial string
	Role   string
}

type Logger struct {
	ID     int
	SiteID int
	Name   string
	Model  string
	Serial string
	Role   string
}

func Add(p AddParams) (Logger, error) {
	// Open database
	database, err := db.GetDB()
	if err != nil {
		return Logger{}, fmt.Errorf("database error: %w", err)
	}
	defer database.Close()

	role := p.Role
	if role == "" {
		role = "water_level"
	}

	// Insert logger
	result, err := database.Exec(
		`INSERT INTO loggers (site_id, name, model, serial_number, role) VALUES (?, ?, ?, ?, ?)`,
		p.SiteID, p.Name, p.Model, p.Serial, role,
	)
	if err != nil {
		return Logger{}, fmt.Errorf("failed to add logger: %w", err)
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
		Role:   role,
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
	SELECT id, site_id, name, model, serial_number, role
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
		var model sql.NullString
		var serial sql.NullString
		var role sql.NullString
		if err := rows.Scan(
			&l.ID,
			&l.SiteID,
			&l.Name,
			&model,
			&serial,
			&role,
		); err != nil {
			return nil, err
		}
		if model.Valid {
			l.Model = model.String
		}
		if serial.Valid {
			l.Serial = serial.String
		}
		if role.Valid {
			l.Role = role.String
		}
		loggers = append(loggers, l)
	}

	return loggers, nil
}

func Update(id int, name string, model string, serial string, role string) error {
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	defer database.Close()
	if name != "" {
		if _, err := database.Exec(`UPDATE loggers SET name = ? WHERE id = ?`, name, id); err != nil {
			return err
		}
	}
	if model != "" {
		if _, err := database.Exec(`UPDATE loggers SET model = ? WHERE id = ?`, model, id); err != nil {
			return err
		}
	}
	if serial != "" {
		if _, err := database.Exec(`UPDATE loggers SET serial_number = ? WHERE id = ?`, serial, id); err != nil {
			return err
		}
	}
	if role != "" {
		if _, err := database.Exec(`UPDATE loggers SET role = ? WHERE id = ?`, role, id); err != nil {
			return err
		}
	}
	return nil
}

func Remove(id int) error {
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	defer database.Close()
	_, err = database.Exec(`DELETE FROM loggers WHERE id = ?`, id)
	return err
}
