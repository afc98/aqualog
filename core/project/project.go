package project

import (
	"aqualog/core/db"
	"fmt"
)

func AddProject(name string, description string) error {
	// Open database
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Insert project
	_, err = database.Exec(
		`INSERT INTO projects (name, description) VALUES (?, ?)`,
		name, description,
	)
	if err != nil {
		return fmt.Errorf("Failed to create project: %w", err)
	}

	fmt.Println("Project created:", name)
	return nil
}

func ListProjects() error {

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	// Query projects
	rows, err := database.Query(`SELECT id, name, description, created_at FROM  projects`)
	if err != nil {
		return fmt.Errorf("Failed to query projects: %w", err)
	}
	defer rows.Close()

	fmt.Println("Projects:")
	for rows.Next() {
		var id int
		var name, desc, created string
		rows.Scan(&id, &name, &desc, &created)

		fmt.Printf(" %d. %s - %s (created %s)\n", id, name, desc, created)
	}
	return nil
}
