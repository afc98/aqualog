package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/project"

	"github.com/spf13/cobra"
)

var projectUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a project",
	Run: func(cmd *cobra.Command, args []string) {
		ident, _ := cmd.Flags().GetString("project")
		id, err := db.ResolveProjectIdentifier(ident)
		if err != nil {
			fmt.Println("Project not found:", err)
			return
		}
		name, _ := cmd.Flags().GetString("name")
		desc, _ := cmd.Flags().GetString("description")
		if name == "" && desc == "" {
			fmt.Println("Nothing to update.")
			return
		}
		if err := project.Update(id, name, desc); err != nil {
			fmt.Println("Failed to update project:", err)
			return
		}
		fmt.Println("Project updated.")
	},
}

func init() {
	projectCmd.AddCommand(projectUpdateCmd)
	projectUpdateCmd.Flags().StringP("project", "p", "", "Project ID or name")
	projectUpdateCmd.Flags().StringP("name", "n", "", "New project name")
	projectUpdateCmd.Flags().StringP("description", "d", "", "New project description")
	projectUpdateCmd.MarkFlagRequired("project")
}
