package project

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
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
	var id int
	err = database.QueryRow(`
	INSERT INTO projects (name, description)
	VALUES ($1, $2)
	RETURNING id
	`,
		p.Name,
		p.Description,
	).Scan(&id)
	if err != nil {
		return Project{}, fmt.Errorf("error adding project: %w", err)
	}

	return Project{
		ID:          id,
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

	if name != "" {
		if _, err := database.Exec(`UPDATE projects SET name = $1 WHERE id = $2`, name, id); err != nil {
			return err
		}
	}
	if description != "" {
		if _, err := database.Exec(`UPDATE projects SET description = $1 WHERE id = $2`, description, id); err != nil {
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
	_, err = database.Exec(`DELETE FROM projects WHERE id = $1`, id)
	return err
}
