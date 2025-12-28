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

		// List sites
		err = site.ListSites(projectID)
		if err != nil {
			fmt.Println("Error listing sites:", err)
			return
		}

	},
}

func init() {
	siteCmd.AddCommand(siteListCmd)
	siteListCmd.Flags().StringP("project", "p", "", "Project ID or name to list sites")
	siteListCmd.MarkFlagRequired("project")
}
