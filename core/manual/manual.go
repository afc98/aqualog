package manual

import (
	"aqualog/core/db"
	"fmt"
	"time"
)

type AddParams struct {
	SiteID     int
	WaterLevel float64
	Timestamp  string
	Notes      string
}

type ManualReading struct {
	ID         int
	SiteID     int
	WaterLevel float64
	Timestamp  time.Time
	Notes      string
}

type UpdateParams struct {
	Timestamp  string
	WaterLevel float64
	Notes      string
}

func Add(p AddParams) (ManualReading, error) {

	// Parse string timestamp
	parsedTime, err := time.Parse("20060102 15:04:05", p.Timestamp)
	if err != nil {
		return ManualReading{}, fmt.Errorf(
			"invalid timestamp format (use YYYYMMDD HH:MM:SS)",
		)
	}

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return ManualReading{}, fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Insert manual reading
	result, err := database.Exec(`
	INSERT INTO manual_readings (site_id, timestamp, value, notes)
	VALUES (?, ?, ?, ?)
	`,
		p.SiteID,
		parsedTime,
		p.WaterLevel,
		p.Notes,
	)
	if err != nil {
		return ManualReading{}, fmt.Errorf("error adding manual reading: %w", err)
	}

	// Get id of new manual reading
	id, err := result.LastInsertId()
	if err != nil {
		return ManualReading{}, err
	}

	return ManualReading{
		ID:        int(id),
		SiteID:    p.SiteID,
		Timestamp: parsedTime,
		Notes:     p.Notes,
	}, nil
}

func List(siteID int) ([]ManualReading, error) {
	if siteID == 0 {
		return nil, fmt.Errorf("must specify site")
	}

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Query projects
	rows, err := database.Query(`
	SELECT id, site_id, timestamp, value, notes 
	FROM manual_readings WHERE site_id=?
	ORDER BY id
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("Failed to query manual readings: %w", err)
	}
	defer rows.Close()

	manual_readings := []ManualReading{}

	for rows.Next() {
		var m ManualReading
		if err := rows.Scan(
			&m.ID,
			&m.SiteID,
			&m.Timestamp,
			&m.WaterLevel,
			&m.Notes,
		); err != nil {
			return nil, err
		}
		manual_readings = append(manual_readings, m)
	}

	return manual_readings, nil
}

func Delete(id int) error {
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	defer database.Close()

	result, err := database.Exec(`
		DELETE FROM manual_readings
		WHERE id = ?
	`, id)

	if err != nil {
		return fmt.Errorf("error removing manual reading: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no manual reading with that ID")
	}

	return nil
}

func Get(id int) (ManualReading, error) {
	database, err := db.GetDB()
	if err != nil {
		return ManualReading{}, fmt.Errorf("database error: %w", err)
	}
	defer database.Close()

	var m ManualReading

	err = database.QueryRow(`
		SELECT id, site_id, timestamp, value, notes
		FROM manual_readings
		WHERE id = ?
	`, id).Scan(
		&m.ID,
		&m.SiteID,
		&m.Timestamp,
		&m.WaterLevel,
		&m.Notes,
	)

	if err != nil {
		return ManualReading{}, fmt.Errorf("manual reading not found")
	}

	return m, nil
}

func PrepareUpdate(existing ManualReading, p UpdateParams) (ManualReading, error) {
	final := existing

	// Timestamp
	if p.Timestamp != "" {
		ts, err := time.Parse("20060102 15:04:05", p.Timestamp)
		if err != nil {
			return ManualReading{}, fmt.Errorf(
				"invalid timestamp format (use YYYYMMDD HH:MM:SS)",
			)
		}
		final.Timestamp = ts
	}

	// Water Level
	if p.WaterLevel != -1 {
		final.WaterLevel = p.WaterLevel
	}

	// Notes
	if p.Notes != "" {
		final.Notes = p.Notes
	}

	return final, nil
}

func Update(id int, updated ManualReading) error {

	// Open DB
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	result, err := database.Exec(`
		UPDATE manual_readings
		SET timestamp = ?, value = ?, notes = ?
		WHERE id = ?
	;`,
		updated.Timestamp,
		updated.WaterLevel,
		updated.Notes,
		id,
	)

	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no manual reading updated")
	}

	return nil
}
