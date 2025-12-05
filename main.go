/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"aqualog/cmd"
	"aqualog/internal/db"
)

func main() {
	db.InitDB()
	db.Migrate()

	cmd.Execute()
}
