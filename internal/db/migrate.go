package db

import "log"

func Migrate() {
	createProjects := `
	CREATE TABLE IF NOT EXISTS projects (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT UNIQUE NOT NULL,
	description TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := DB.Exec(createProjects)
	if err != nil {
		log.Fatal("Failed to create project table:", err)
	}
}
