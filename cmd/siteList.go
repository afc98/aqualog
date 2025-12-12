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
var siteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing sites",
	Run: func(cmd *cobra.Command, args []string) {
		projectIdent, _ := cmd.Flags().GetString("project") // accepts "123" or "My Project"
		projectID, err := db.ResolveProjectIdentifier(projectIdent)
		if err != nil {
			fmt.Println("Project not found:", err)
			return
		}

		// Open database
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		// Query projects
		rows, err := database.Query(`SELECT id, name, latitude, longitude FROM sites WHERE project_id=? ORDER by id`, projectID)
		if err != nil {
			fmt.Println("Failed to query sites:", err)
			return
		}
		defer rows.Close()

		fmt.Println("Sites for project:")
		for rows.Next() {
			var id int
			var name string
			var lat, lon float64
			rows.Scan(&id, &name, &lat, &lon)

			fmt.Printf(" %d. %s (%.6f, %.6f)\n", id, name, lat, lon)
		}
	},
}

func init() {
	siteCmd.AddCommand(siteListCmd)
	siteListCmd.Flags().StringP("project", "p", "", "Project ID or name to list sites")
	siteListCmd.MarkFlagRequired("project")
}
