package manual

import (
	"aqualog/core/db"
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func AddManualReading(siteID int, waterLevel float64, timestamp, notes string) error {
	// Open database
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Parse timestamp
	parsedTime, err := time.Parse("20060102 15:04:05", timestamp)
	if err != nil {
		return fmt.Errorf("Invalid timestamp format. Use: YYYYMMDD HH:MM:SS")
	}
	// Insert manual reading
	_, err = database.Exec(`
	INSERT INTO manual_readings (site_id, timestamp, value, notes)
	VALUES (?, ?, ?, ?)
	`, siteID, parsedTime, waterLevel, notes)
	if err != nil {
		return fmt.Errorf("Error adding manual reading: %w", err)
	}
	fmt.Println("Manual reading added")
	return nil
}

func ListManualReadings(siteID int) error {

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Query projects
	rows, err := database.Query(`
	SELECT id, site_id, timestamp, value, notes 
	FROM manual_readings WHERE site_id=?
	ORDER BY id
	`, siteID)
	if err != nil {
		return fmt.Errorf("Failed to query manual readings: %w", err)
	}
	defer rows.Close()

	fmt.Println("Manual readings for site:")
	for rows.Next() {
		var id int
		var site, time, value, notes string
		rows.Scan(&id, &site, &time, &value, &notes)

		fmt.Printf(" %d. %s (time: %s, value: %s) notes: %s\n", id, site, time, value, notes)
	}
	return nil
}

func RemoveManualReading(id int) error {

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Fetch the manual reading so we can display it
	var siteID int
	var ts time.Time
	var value float64
	var notes string

	err = database.QueryRow(`
		SELECT site_id, timestamp, value, notes
		FROM manual_readings
		WHERE id = ?
	`, id).Scan(&siteID, &ts, &value, &notes)

	if err != nil {
		return fmt.Errorf("Manual reading not found with that ID.")
	}

	// Show details and request confirmation
	fmt.Printf("Are you sure you want to delete manual reading %d?\n", id)
	fmt.Printf("%d. Site %d (%s, %.3f) Notes: %s\n",
		id, siteID, ts.Format("2006-01-02 15:04:05"), value, notes)
	fmt.Print("Enter Y to confirm: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input != "Y" {
		return fmt.Errorf("Cancelled.")
	}

	// Perform delete
	result, err := database.Exec(`
		DELETE FROM manual_readings
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("Error removing manual reading: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("No manual reading removed — unexpected.")
	}

	fmt.Println("Manual reading removed.")
	return nil
}

func UpdateManualReading(id int, tsString string, newWaterLevel float64, newNotes string) error {

	// Open DB
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Fetch existing row
	var siteID int
	var existingTS time.Time
	var existingValue float64
	var existingNotes string

	err = database.QueryRow(`
		SELECT site_id, timestamp, value, notes
		FROM manual_readings
		WHERE id = ?
	`, id).Scan(&siteID, &existingTS, &existingValue, &existingNotes)
	if err != nil {
		return fmt.Errorf("Manual reading not found with that ID.")
	}

	// Apply defaults BEFORE preview
	finalTS := existingTS
	finalValue := existingValue
	finalNotes := existingNotes

	// Parse timestamp if provided
	if tsString != "" {
		t, err := time.Parse("20060102 15:04:05", tsString)
		if err != nil {
			return fmt.Errorf("Invalid timestamp format. Use: YYYYMMDD HH:MM:SS")
		}
		finalTS = t
	}

	// Apply water level unless -1
	if newWaterLevel != -1 {
		finalValue = newWaterLevel
	}

	if newNotes != "" {
		finalNotes = newNotes
	}

	// Preview
	fmt.Printf("Are you sure you want to update manual reading %d?\n", id)
	fmt.Printf("Existing: %d. Site %d (%s, %.3f) Notes: %s\n",
		id, siteID, existingTS.Format("2006-01-02 15:04:05"), existingValue, existingNotes)
	fmt.Printf("     New: %d. Site %d (%s, %.3f) Notes: %s\n",
		id, siteID, finalTS.Format("2006-01-02 15:04:05"), finalValue, finalNotes)
	fmt.Print("Enter Y to confirm: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input != "Y" {
		return fmt.Errorf("Cancelled.")
	}

	// Perform update
	result, err := database.Exec(`
		UPDATE manual_readings
		SET timestamp = ?, value = ?, notes = ?
		WHERE id = ?
	`, finalTS, finalValue, finalNotes, id)

	if err != nil {
		return fmt.Errorf("Error updating manual reading: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("No manual reading updated — unexpected.")
	}

	fmt.Println("Manual reading updated.")
	return nil
}
