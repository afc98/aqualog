package site

import (
	"aqualog/db"
	"fmt"
)

func AddSite(projectID int, name string, lat, lon float64) error {

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	_, err = database.Exec(
		`INSERT INTO sites (project_id, name, latitude, longitude) VALUES (?, ?, ?, ?)`,
		projectID, name, lat, lon,
	)
	if err != nil {
		return fmt.Errorf("Failed to add site: %w", err)
	}

	fmt.Println("Site added:", name)
	return nil
}

func ListSites(projectID int) error {

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Query projects
	rows, err := database.Query(`SELECT id, name, latitude, longitude FROM sites WHERE project_id=? ORDER by id`, projectID)
	if err != nil {
		return fmt.Errorf("Failed to query sites: %w", err)
	}
	defer rows.Close()

	fmt.Println("Sites for project:")
	for rows.Next() {
		var id int
		var name string
		var lat, lon float64
		rows.Scan(&id, &name, &lat, &lon)

		fmt.Printf(" %d. %s (%.6f, %.6f)\n", id, name, lat, lon)
	}
	return nil
}
