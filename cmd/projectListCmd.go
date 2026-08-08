package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/project"

	"github.com/spf13/cobra"
)

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing projects",
	Long:  `Lists existing projects`,
	Run: func(cmd *cobra.Command, args []string) {

		// List projects
		projects, err := project.List()
		if err != nil {
			fmt.Println("Error listing projects:", err)
			return
		}

		fmt.Println("Existing projects:")
		if len(projects) == 0 {
			fmt.Println(" No projects found.")
			return
		}

		for _, p := range projects {
			fmt.Printf(
				" %d. Name: %s, Description: %s, Created At: %s\n",
				p.ID,
				p.Name,
				p.Description,
				p.CreatedAt.Format("2006-01-02 15:04:05"),
			)
		}
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)
}
