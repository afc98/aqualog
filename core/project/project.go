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

func List() ([]Project, error) {

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Query projects
	rows, err := database.Query(`SELECT id, name, description, created_at FROM  projects`)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects: %w", err)
	}
	defer rows.Close()

	projects := []Project{}
	for rows.Next() {
		var project Project
		if err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Description,
			&project.CreatedAt,
		); err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	return projects, nil
}

func Update(id int, name string, description string) error {
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	defer database.Close()

	if name != "" {
		if _, err := database.Exec(`UPDATE projects SET name = ? WHERE id = ?`, name, id); err != nil {
			return err
		}
	}
	if description != "" {
		if _, err := database.Exec(`UPDATE projects SET description = ? WHERE id = ?`, description, id); err != nil {
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
	_, err = database.Exec(`DELETE FROM projects WHERE id = ?`, id)
	return err
}
