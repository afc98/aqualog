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
		err := project.AddProject(name, desc)
		if err != nil {
			fmt.Println("Error adding project:", err)
			return
		}

	},
}

func init() {
	projectCmd.AddCommand(projectAddCmd)

	projectAddCmd.Flags().StringP("name", "n", "", "Name of the project")
	projectAddCmd.Flags().StringP("notes", "N", "", "Notes on the project")

	projectAddCmd.MarkFlagRequired("name")
}
