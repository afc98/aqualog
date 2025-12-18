/*
Copyright © 2025 Arran Clarke
*/
package cmd

import (
	"aqualog/db"
	"fmt"

	"github.com/spf13/cobra"
)

// siteAddCmd represents the siteAdd command
var siteAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new site to a project",
	Run: func(cmd *cobra.Command, args []string) {
		projectIdent, _ := cmd.Flags().GetString("project") // accepts "123" or "My Project"
		projectID, err := db.ResolveProjectIdentifier(projectIdent)
		if err != nil {
			fmt.Println("Project not found:", err)
			return
		}
		name, _ := cmd.Flags().GetString("name")
		lat, _ := cmd.Flags().GetFloat64("lat")
		lon, _ := cmd.Flags().GetFloat64("lon")

		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		_, err = database.Exec(
			`INSERT INTO sites (project_id, name, latitude, longitude) VALUES (?, ?, ?, ?)`,
			projectID, name, lat, lon,
		)
		if err != nil {
			fmt.Println("Failed to add site:", err)
			return
		}

		fmt.Println("Site added:", name)
	},
}

func init() {
	siteCmd.AddCommand(siteAddCmd)
	
	siteAddCmd.Flags().StringP("project", "p", "", "Project ID or name")
	siteAddCmd.Flags().StringP("name", "n", "", "Name of the site")
	siteAddCmd.Flags().Float64P("lat", "y", 0, "Latitude of the site")
	siteAddCmd.Flags().Float64P("lon", "x", 0, "Longitude of the site")

	siteAddCmd.MarkFlagRequired("project")
	siteAddCmd.MarkFlagRequired("name")
}
