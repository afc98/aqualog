/*
Copyright © 2025 Arran Clarke
*/
package cmd

import (
	"aqualog/db"
	"fmt"

	"github.com/spf13/cobra"
)

// projectAddCmd represents the projectAdd command
var projectAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new project",
	Long:  `Add a new project that can contain multiple data logger sites`,
	Run: func(cmd *cobra.Command, args []string) {
		// Read flags
		name, _ := cmd.Flags().GetString("name")
		desc, _ := cmd.Flags().GetString("description")

		// Open database
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		// Insert project
		_, err = database.Exec(
			`INSERT INTO projects (name, description) VALUES (?, ?)`,
			name, desc,
		)
		if err != nil {
			fmt.Println("Failed to create project:", err)
			return
		}

		fmt.Println("Project created:", name)

		// Later: insert into SQLite
	},
}

func init() {
	projectCmd.AddCommand(projectAddCmd)

	projectAddCmd.Flags().StringP("name", "n", "", "Name of the project")
	projectAddCmd.Flags().StringP("notes", "N", "", "Notes on the project")
	
	projectAddCmd.MarkFlagRequired("name")
}
