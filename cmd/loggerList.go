/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"aqualog/db"
	"fmt"

	"github.com/spf13/cobra"
)

var loggerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing loggers for a site",
	Long:  `Lists existing projects`,
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Project not found:", err)
			return
		}
		// Open database
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		// Query projects
		rows, err := database.Query(`
		SELECT id, name, model, serial_number 
		FROM loggers WHERE site_id=?
		ORDER BY id
		`, siteID)
		if err != nil {
			fmt.Println("Failed to query loggers:", err)
			return
		}
		defer rows.Close()

		fmt.Println("Logger for site:")
		for rows.Next() {
			var id int
			var name, model, serial string
			rows.Scan(&id, &name, &model, &serial)

			fmt.Printf(" %d. %s (model: %s, serial: %s)\n", id, name, model, serial)
		}
	},
}

func init() {
	loggerCmd.AddCommand(loggerListCmd)

	loggerListCmd.Flags().StringP("site", "s", "", "Site ID or name")
	loggerListCmd.MarkFlagRequired("site")
}
