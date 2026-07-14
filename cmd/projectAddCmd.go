/*
Copyright © 2025 Arran Clarke
*/
package cmd

import (
	"aqualog/core/project"
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

		// Add project
		proj, err := project.Add(project.AddParams{
			Name:        name,
			Description: desc,
		})
		if err != nil {
			fmt.Println("Failed to add project:", err)
			return
		}

		fmt.Printf(
			"Project added: ID %d, Name: %s, Notes: %s, Created At: %s\n",
			proj.ID,
			proj.Name,
			proj.Description,
			proj.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	},
}

func init() {
	projectCmd.AddCommand(projectAddCmd)

	projectAddCmd.Flags().StringP("name", "n", "", "Name of the project")
	projectAddCmd.Flags().StringP("description", "d", "", "Description of the project")

	projectAddCmd.MarkFlagRequired("name")
}
