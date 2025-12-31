package site

import (
	"aqualog/core/db"
	"fmt"
)

type AddParams struct {
	ProjectID int
	Name      string
	Latitude  float64
	Longitude float64
}

type Site struct {
	ID        int
	ProjectID int
	Name      string
	Latitude  float64
	Longitude float64
}

func Add(p AddParams) (Site, error) {

	// Open database
	database, err := db.GetDB()
	if err != nil {
		return Site{}, fmt.Errorf("Database error: %w", err)
	}
	defer database.Close()

	result, err := database.Exec(
		`INSERT INTO sites (project_id, name, latitude, longitude) VALUES (?, ?, ?, ?)`,
		p.ProjectID, p.Name, p.Latitude, p.Longitude,
	)
	if err != nil {
		return Site{}, fmt.Errorf("failed to add site: %w", err)
	}

	// Get the ID of the newly added site
	id, err := result.LastInsertId()
	if err != nil {
		return Site{}, fmt.Errorf("failed to get site ID: %w", err)
	}

	return Site{
		ID:        int(id),
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
	defer database.Close()

	// Query sites
	rows, err := database.Query(`SELECT id, name, latitude, longitude FROM sites WHERE project_id = ? ORDER BY id`, ProjectID)
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
