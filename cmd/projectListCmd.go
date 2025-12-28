/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"aqualog/db"
	"fmt"

	"github.com/spf13/cobra"
)

// projectListCmd represents the projectList command
var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing projects",
	Long:  `Lists existing projects`,
	Run: func(cmd *cobra.Command, args []string) {
		// Open database
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		// Query projects
		rows, err := database.Query(`SELECT id, name, description, created_at FROM  projects`)
		if err != nil {
			fmt.Println("Failed to query projects:", err)
			return
		}
		defer rows.Close()

		fmt.Println("Projects:")
		for rows.Next() {
			var id int
			var name, desc, created string
			rows.Scan(&id, &name, &desc, &created)

			fmt.Printf(" %d. %s - %s (created %s)\n", id, name, desc, created)
		}
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)
}
