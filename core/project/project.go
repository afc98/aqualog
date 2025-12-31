package project

import (
	"aqualog/core/db"
	"fmt"
	"time"
)

type AddParams struct {
	Name        string
	Description string
}

type Project struct {
	ID          int
	Name        string
	Description string
	CreatedAt   time.Time
}

func Add(p AddParams) (Project, error) {

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return Project{}, fmt.Errorf("database error: %w", err)
	}

	// Insert project
	result, err := database.Exec(`
	INSERT INTO projects (name, description)
	VALUES (?, ?)
	`,
		p.Name,
		p.Description,
	)
	if err != nil {
		return Project{}, fmt.Errorf("error adding project: %w", err)
	}

	// Get id of new project
	id, err := result.LastInsertId()
	if err != nil {
		return Project{}, fmt.Errorf("error getting last insert ID: %w", err)
	}

	return Project{
		ID:          int(id),
		Name:        p.Name,
		Description: p.Description,
		CreatedAt:   time.Now(),
	}, nil
}

func List() error {

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	// Query projects
	rows, err := database.Query(`SELECT id, name, description, created_at FROM  projects`)
	if err != nil {
		return fmt.Errorf("failed to query projects: %w", err)
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
