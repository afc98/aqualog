/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"aqualog/project"
	"fmt"

	"github.com/spf13/cobra"
)

// projectListCmd represents the projectList command
var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing projects",
	Long:  `Lists existing projects`,
	Run: func(cmd *cobra.Command, args []string) {
		err := project.ListProjects()
		if err != nil {
			fmt.Println("Error listing projects:", err)
			return
		}
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)
}
