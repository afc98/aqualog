package logger

import (
	"aqualog/db"
	"fmt"
)

func AddLogger(siteID int, name, model, serial string) error {

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Insert project
	_, err = database.Exec(
		`INSERT INTO loggers (site_id, name, model, serial_number) VALUES (?, ?, ?, ?)`,
		siteID, name, model, serial,
	)
	if err != nil {
		return fmt.Errorf("Failed to add logger: %w", err)
	}

	fmt.Printf("Logger '%s' added to site %d\n", name, siteID)
	return nil
}

func ListLoggers(siteID int) error {

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Query projects
	rows, err := database.Query(`
	SELECT id, name, model, serial_number 
	FROM loggers WHERE site_id=?
	ORDER BY id
	`, siteID)
	if err != nil {
		return fmt.Errorf("Failed to query loggers: %w", err)
	}
	defer rows.Close()

	fmt.Println("Logger for site:")
	for rows.Next() {
		var id int
		var name, model, serial string
		rows.Scan(&id, &name, &model, &serial)

		fmt.Printf(" %d. %s (model: %s, serial: %s)\n", id, name, model, serial)
	}

	return nil

}
