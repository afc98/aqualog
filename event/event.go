package event

import (
	"aqualog/db"
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// EventAdd adds an event to the logger_events table
func EventAdd(loggerID int, eventType, timestamp, notes string) error {

	// Check that a valid event type is being used
	eventType = strings.ToLower(eventType)
	validEvents := map[string]struct{}{
		"installed": {},
		"moved":     {},
		"removed":   {},
		"other":     {},
	}

	// Parse and validate timestamp
	parsedTime, err := time.Parse("20060102 15:04:05", timestamp)
	if err != nil {
		fmt.Println("Invalid timestamp format. Use: YYYYMMDD HH:MM:SS")
		return err
	}

	if _, ok := validEvents[eventType]; !ok {
		fmt.Println("Unsupported event type. Use: installed, moved, removed, other")
		return fmt.Errorf("unsupported event type: %s", eventType)
	}

	// Open database
	database, err := db.GetDB()
	if err != nil {
		fmt.Println("Database error:", err)
		return err
	}
	defer database.Close()

	// Insert event into database
	_, err = database.Exec(`
		INSERT INTO logger_events (logger_id, timestamp, event_type, notes)
		VALUES (?, ?, ?, ?)
		`, loggerID, parsedTime, eventType, notes)
	if err != nil {
		fmt.Println("Error adding event:", err)
		return err
	}

	fmt.Println("Event added.")

	return nil
}

func EventList(siteID, loggerID int) error {
	// Must specify at least one filter
	if siteID == 0 && loggerID == 0 {
		return fmt.Errorf("You must specify --site <id/name> and/or --logger <id/name>")
	}

	// Build SQL dynamically
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
		query += " AND l.site_id = ?"
		params = append(params, siteID)
	}
	if loggerID != 0 {
		query += " AND l.id = ?"
		params = append(params, loggerID)
	}

	query += " ORDER BY e.timestamp"

	// Open DB
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	rows, err := database.Query(query, params...)
	if err != nil {
		return fmt.Errorf("Failed to query logger events: %w", err)
	}
	defer rows.Close()

	if siteID != 0 && loggerID != 0 {
		fmt.Printf("Logger events for site %d and logger %d:\n", siteID, loggerID)
	} else if siteID != 0 {
		fmt.Printf("Logger events for site %d:\n", siteID)
	} else {
		fmt.Printf("Logger events for logger %d:\n", loggerID)
	}

	found := false
	for rows.Next() {
		found = true

		var id int
		var logID int
		var timestamp string
		var eventType string
		var notes string

		err := rows.Scan(&id, &logID, &timestamp, &eventType, &notes)
		if err != nil {
			fmt.Println("Row scan error:", err)
			continue
		}

		fmt.Printf(" %d. Logger %d (time: %s, event: %s) notes: %s\n",
			id, logID, timestamp, eventType, notes)
	}

	if !found {
		fmt.Println(" No events found.")
	}

	return nil
}

func EventRemove(id int) error {
	// Open database connection
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Fetch the event record
	var loggerID int
	var ts time.Time
	var eventType string
	var notes string

	err = database.QueryRow(`
		SELECT logger_id, timestamp, event_type, notes
		FROM logger_events
		WHERE id = ?
	`, id).Scan(&loggerID, &ts, &eventType, &notes)

	if err != nil {
		return fmt.Errorf("No logger event found with that ID.")
	}

	// Show details + confirmation prompt
	fmt.Printf("Are you sure you want to delete logger event %d?\n", id)
	fmt.Printf("%d. Logger %d (%s, %s) Notes: %s\n",
		id,
		loggerID,
		ts.Format("2006-01-02 15:04:05"),
		eventType,
		notes,
	)

	fmt.Print("Enter Y to confirm: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input != "Y" {
		return fmt.Errorf("Cancelled.")
	}

	// Perform delete
	result, err := database.Exec(`
		DELETE FROM logger_events
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("Error removing logger event: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("No logger event removed — unexpected.")
	}

	fmt.Println("Logger event removed.")

	return nil
}

func EventUpdate(id int, eventType, timestamp, notes string) error {
	// Normalize event type if provided
	eventType = strings.ToLower(eventType)

	validEvents := map[string]struct{}{
		"installed": {},
		"moved":     {},
		"removed":   {},
		"other":     {},
	}

	// Open DB
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Fetch existing row
	var loggerID int
	var existingTS time.Time
	var existingType string
	var existingNotes string

	err = database.QueryRow(`
		SELECT logger_id, timestamp, event_type, notes
		FROM logger_events
		WHERE id = ?
	`, id).Scan(&loggerID, &existingTS, &existingType, &existingNotes)

	if err != nil {
		return fmt.Errorf("Logger event not found with that ID.")
	}

	// Defaults
	finalTS := existingTS
	finalType := existingType
	finalNotes := existingNotes

	// Parse timestamp if provided
	if timestamp != "" {
		t, err := time.Parse("20060102 15:04:05", timestamp)
		if err != nil {
			return fmt.Errorf("Invalid timestamp. Use: YYYYMMDD HH:MM:SS")
		}
		finalTS = t
	}

	// Apply event type if provided
	if eventType != "" {
		if _, ok := validEvents[eventType]; !ok {
			return fmt.Errorf("Unsupported event type. Use: installed, moved, removed, other")
		}
		finalType = eventType
	}

	// Apply notes if provided
	if notes != "" {
		finalNotes = notes
	}

	// Preview before applying
	fmt.Printf("Are you sure you want to update logger event %d?\n", id)
	fmt.Printf("Existing: Logger %d (%s, %s) Notes: %s\n",
		loggerID,
		existingTS.Format("2006-01-02 15:04:05"),
		existingType,
		existingNotes,
	)
	fmt.Printf("     New: Logger %d (%s, %s) Notes: %s\n",
		loggerID,
		finalTS.Format("2006-01-02 15:04:05"),
		finalType,
		finalNotes,
	)
	fmt.Print("Enter Y to confirm: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input != "Y" {
		return fmt.Errorf("Cancelled.")
	}

	// Perform update
	result, err := database.Exec(`
		UPDATE logger_events
		SET timestamp = ?, event_type = ?, notes = ?
		WHERE id = ?
	`, finalTS, finalType, finalNotes, id)

	if err != nil {
		return fmt.Errorf("Error updating logger event: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("No logger event updated — unexpected.")
	}

	fmt.Println("Logger event updated.")
	return nil
}
