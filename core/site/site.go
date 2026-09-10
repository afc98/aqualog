package site

import (
	"aqualog/core/db"
	"fmt"
)

type AddParams struct {
	ProjectID   int
	Name        string
	Latitude    float64
	Longitude   float64
	Description string
}

type Site struct {
	ID          int
	ProjectID   int
	Name        string
	Latitude    float64
	Longitude   float64
	Description string
}

func Add(p AddParams) (Site, error) {

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return Site{}, fmt.Errorf("Database error: %w", err)
	}

	var id int
	err = database.QueryRow(
		`INSERT INTO sites (project_id, name, latitude, longitude) VALUES ($1, $2, $3, $4) RETURNING id`,
		p.ProjectID, p.Name, p.Latitude, p.Longitude,
	).Scan(&id)
	if err != nil {
		return Site{}, fmt.Errorf("failed to add site: %w", err)
	}

	return Site{
		ID:        id,
		ProjectID: p.ProjectID,
		Name:      p.Name,
		Latitude:  p.Latitude,
		Longitude: p.Longitude,
	}, nil
}

func List(ProjectID int) ([]Site, error) {

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Query sites
	rows, err := database.Query(`SELECT id, name, latitude, longitude FROM sites WHERE project_id = $1 ORDER BY id`, ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sites: %w", err)
	}
	defer rows.Close()

	sites := []Site{}
	for rows.Next() {
		var site Site
		if err := rows.Scan(
			&site.ID,
			&site.Name,
			&site.Latitude,
			&site.Longitude,
		); err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}

	return sites, nil
}

func Update(id int, name string, latitude *float64, longitude *float64, description string) error {
	database, err := db.GetDB()
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if name != "" {
		if _, err := database.Exec(`UPDATE sites SET name = $1 WHERE id = $2`, name, id); err != nil {
			return err
		}
	}
	if latitude != nil {
		if _, err := database.Exec(`UPDATE sites SET latitude = $1 WHERE id = $2`, *latitude, id); err != nil {
			return err
		}
	}
	if longitude != nil {
		if _, err := database.Exec(`UPDATE sites SET longitude = $1 WHERE id = $2`, *longitude, id); err != nil {
			return err
		}
	}
	if description != "" {
		if _, err := database.Exec(`UPDATE sites SET description = $1 WHERE id = $2`, description, id); err != nil {
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
	_, err = database.Exec(`DELETE FROM sites WHERE id = $1`, id)
	return err
}
