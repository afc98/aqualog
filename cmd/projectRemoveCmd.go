package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/project"

	"github.com/spf13/cobra"
)

var projectRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a project by ID or name",
	Run: func(cmd *cobra.Command, args []string) {
		ident, _ := cmd.Flags().GetString("project")
		id, err := db.ResolveProjectIdentifier(ident)
		if err != nil {
			fmt.Println("Project not found:", err)
			return
		}
		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			fmt.Println("Refusing to remove project without --confirm.")
			return
		}
		if err := project.Remove(id); err != nil {
			fmt.Println("Failed to remove project:", err)
			return
		}
		fmt.Println("Project removed.")
	},
}

func init() {
	projectCmd.AddCommand(projectRemoveCmd)
	projectRemoveCmd.Flags().StringP("project", "p", "", "Project ID or name")
	projectRemoveCmd.Flags().Bool("confirm", false, "Confirm removal")
	projectRemoveCmd.MarkFlagRequired("project")
}
