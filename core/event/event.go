package event

import (
	"encoding/json"
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/utils"
	"strings"
	"time"
)

type AddParams struct {
	LoggerID  int
	EventType string
	Timestamp string
	Notes     string
}

type Event struct {
	ID        int
	LoggerID  int
	Timestamp time.Time
	EventType string
	Notes     string
}

func (row Event) MarshalJSON() ([]byte, error) {
	type alias Event
	return json.Marshal(struct {
		Timestamp string
		alias
	}{Timestamp: utils.FormatDatasetTime(row.Timestamp), alias: alias(row)})
}

type UpdateParams struct {
	EventType string
	Timestamp string
	Notes     string
}

func Add(p AddParams) (Event, error) {
	eventType := strings.ToLower(p.EventType)

	validEvents := map[string]struct{}{
		"installed": {},
		"moved":     {},
		"removed":   {},
		"other":     {},
	}

	if _, ok := validEvents[eventType]; !ok {
		return Event{}, fmt.Errorf(
			"unsupported event type: %s (use installed, moved, removed, other)",
			eventType,
		)
	}

	// Parse string timestamp
	parsedTime, err := time.Parse("20060102 15:04:05", p.Timestamp)
	if err != nil {
		return Event{}, fmt.Errorf(
			"invalid timestamp format (use YYYYMMDD HH:MM:SS)",
		)
	}

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return Event{}, fmt.Errorf("database error: %w", err)
	}

	// Insert event
	var id int
	err = database.QueryRow(`
		INSERT INTO logger_events (logger_id, timestamp, event_type, notes)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`,
		p.LoggerID,
		parsedTime,
		eventType,
		p.Notes,
	).Scan(&id)
	if err != nil {
		return Event{}, fmt.Errorf("error adding event: %w", err)
	}

	return Event{
		ID:        id,
		LoggerID:  p.LoggerID,
		Timestamp: parsedTime,
		EventType: eventType,
		Notes:     p.Notes,
	}, nil
}

func List(siteID, loggerID int) ([]Event, error) {
	if siteID == 0 && loggerID == 0 {
		return nil, fmt.Errorf("must specify site and/or logger")
	}

	query := `
		SELECT
			e.id,
			e.logger_id,
			e.timestamp,
			e.event_type,
			e.notes
		FROM logger_events e
		JOIN loggers l ON e.logger_id = l.id
		WHERE 1=1
	`
	params := []any{}

	if siteID != 0 {
		query += fmt.Sprintf(" AND l.site_id = $%d", len(params)+1)
		params = append(params, siteID)
	}
	if loggerID != 0 {
		query += fmt.Sprintf(" AND l.id = $%d", len(params)+1)
		params = append(params, loggerID)
	}

	query += " ORDER BY e.timestamp"

	database, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	rows, err := database.Query(query, params...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	events := []Event{}

	for rows.Next() {
		var e Event
		if err := rows.Scan(
			&e.ID,
			&e.LoggerID,
			&e.Timestamp,
			&e.EventType,
			&e.Notes,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, nil
}

func Delete(id int) error {
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	result, err := database.Exec(`
		DELETE FROM logger_events
		WHERE id = $1
	`, id)

	if err != nil {
		return fmt.Errorf("error removing logger event: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no logger event found with that ID")
	}

	return nil
}

func Get(id int) (Event, error) {
	database, err := db.GetDB()
	if err != nil {
		return Event{}, fmt.Errorf("database error: %w", err)
	}

	var e Event

	err = database.QueryRow(`
		SELECT id, logger_id, timestamp, event_type, notes
		FROM logger_events
		WHERE id = $1
	`, id).Scan(
		&e.ID,
		&e.LoggerID,
		&e.Timestamp,
		&e.EventType,
		&e.Notes,
	)

	if err != nil {
		return Event{}, fmt.Errorf("logger event not found")
	}

	return e, nil
}

func PrepareUpdate(existing Event, p UpdateParams) (Event, error) {
	final := existing

	// Timestamp
	if p.Timestamp != "" {
		ts, err := time.Parse("20060102 15:04:05", p.Timestamp)
		if err != nil {
			return Event{}, fmt.Errorf(
				"invalid timestamp format (use YYYYMMDD HH:MM:SS)",
			)
		}
		final.Timestamp = ts
	}

	// Event type
	if p.EventType != "" {
		et := strings.ToLower(p.EventType)
		validEvents := map[string]struct{}{
			"installed": {},
			"moved":     {},
			"removed":   {},
			"other":     {},
		}
		if _, ok := validEvents[et]; !ok {
			return Event{}, fmt.Errorf("unsupported event type: %s", et)
		}
		final.EventType = et
	}

	// Notes
	if p.Notes != "" {
		final.Notes = p.Notes
	}

	return final, nil
}

func Update(id int, updated Event) error {

	// Open DB
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	result, err := database.Exec(`
		UPDATE logger_events
		SET timestamp = $1, event_type = $2, notes = $3
		WHERE id = $4
	`,
		updated.Timestamp,
		updated.EventType,
		updated.Notes,
		id,
	)

	if err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("no event updated")
	}

	return nil
}
