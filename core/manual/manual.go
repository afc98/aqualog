package manual

import (
	"aqualog/core/db"
	"aqualog/core/utils"
	"encoding/json"
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

func (row ManualReading) MarshalJSON() ([]byte, error) {
	type alias ManualReading
	return json.Marshal(struct {
		Timestamp string
		alias
	}{Timestamp: utils.FormatDatasetTime(row.Timestamp), alias: alias(row)})
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

	// Insert manual reading
	var id int
	err = database.QueryRow(`
	INSERT INTO manual_readings (site_id, timestamp, value, notes)
	VALUES ($1, $2, $3, $4)
	RETURNING id
	`,
		p.SiteID,
		parsedTime,
		p.WaterLevel,
		p.Notes,
	).Scan(&id)
	if err != nil {
		return ManualReading{}, fmt.Errorf("error adding manual reading: %w", err)
	}

	return ManualReading{
		ID:        id,
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
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Query projects
	rows, err := database.Query(`
	SELECT id, site_id, timestamp, value, notes 
	FROM manual_readings WHERE site_id=$1
	ORDER BY id
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("failed to query manual readings: %w", err)
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

	result, err := database.Exec(`
		DELETE FROM manual_readings
		WHERE id = $1
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

	var m ManualReading

	err = database.QueryRow(`
		SELECT id, site_id, timestamp, value, notes
		FROM manual_readings
		WHERE id = $1
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

	result, err := database.Exec(`
		UPDATE manual_readings
		SET timestamp = $1, value = $2, notes = $3
		WHERE id = $4
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
