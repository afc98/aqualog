/*
Copyright © 2025 Arran Clarke
*/
package cmd

import (
	"fmt"
	"aqualog/internal/db"

	"github.com/spf13/cobra"
)

// projectAddCmd represents the projectAdd command
var projectAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new project",
	Long:  `Add a new project that can contain multiple data logger sites`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		desc, _ := cmd.Flags().GetString("description")

		_, err != db.DB.Exec(
			`INSERT INTO projects (name, description) VALUES (?, ?)`,
			name, desc,
		)

		fmt.Println("Project created:", name)

		// Later: insert into SQLite
	},
}

func init() {
	projectCmd.AddCommand(projectAddCmd)

	projectAddCmd.Flags().StringP("name", "n", "", "Name of the project")
	projectAddCmd.Flags().StringP("description", "d", "", "Description of the project")

	projectAddCmd.MarkFlagRequired("name")
}
