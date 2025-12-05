package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite3", "./aqualog.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	DB.Exec("PRAGMA foreign_keys = ON;")
}
