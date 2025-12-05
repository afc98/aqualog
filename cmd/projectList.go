/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"aqualog/internal/db"
	"fmt"

	"github.com/spf13/cobra"
)

// projectListCmd represents the projectList command
var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing projects",
	Long:  `Lists existing projects`,
	Run: func(cmd *cobra.Command, args []string) {
		rows, err := db.DB.Query(`SELECT id, name, description, created_at FROM  projects`)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer rows.Close()

		fmt.Println("Projects:")
		for rows.Next() {
			var id int
			var name, desc, created string
			rows.Scan(&id, &name, &desc, &created)

			fmt.Printf(" %d. %s - %s (created %s)\n", id, name, desc, created)
		}
	},
}

func init() {
	projectCmd.AddCommand(projectListCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// projectListCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// projectListCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
