/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"aqualog/core/db"
	"aqualog/core/site"
	"fmt"

	"github.com/spf13/cobra"
)

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

		// List sites
		sites, err := site.List(projectID)
		if err != nil {
			fmt.Println("Error listing sites:", err)
			return
		}

		fmt.Printf("Sites for project %d:\n", projectID)
		if len(sites) == 0 {
			fmt.Println(" No sites found.")
			return
		}

		for _, s := range sites {
			fmt.Printf(
				" %d. Name: %s, Latitude: %.6f, Longitude: %.6f\n",
				s.ID,
				s.Name,
				s.Latitude,
				s.Longitude,
			)
		}
	},
}

func init() {
	siteCmd.AddCommand(siteListCmd)
	siteListCmd.Flags().StringP("project", "p", "", "Project ID or name to list sites")
	siteListCmd.MarkFlagRequired("project")
}
