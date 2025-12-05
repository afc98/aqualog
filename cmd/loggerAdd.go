/*
Copyright © 2025 Arran Clarke
*/
package cmd

import (
	"aqualog/db"
	"fmt"

	"github.com/spf13/cobra"
)

// projectAddCmd represents the projectAdd command
var loggerAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new logger to a site",
	Run: func(cmd *cobra.Command, args []string) {
		// Read flags
		siteID, _ := cmd.Flags().GetInt("site")
		name, _ := cmd.Flags().GetString("name")
		model, _ := cmd.Flags().GetString("model")
		serial, _ := cmd.Flags().GetString("serial")

		// Open database
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		// Insert project
		_, err = database.Exec(
			`INSERT INTO loggers (site_id, name, model, serial_number) VALUES (?, ?, ?, ?)`,
			siteID, name, model, serial,
		)
		if err != nil {
			fmt.Println("Failed to add logger:", err)
			return
		}

		fmt.Printf("Logger '%s' added to site %d\n", name, siteID)
	},
}

func init() {
	loggerCmd.AddCommand(loggerAddCmd)

	loggerAddCmd.Flags().IntP("site", "s", 0, "Site ID")
	loggerAddCmd.Flags().StringP("name", "n", "", "Logger name")
	loggerAddCmd.Flags().StringP("model", "m", "", "Logger model")
	loggerAddCmd.Flags().StringP("serial", "e", "", "Serial number")

	loggerAddCmd.MarkFlagRequired("site")
	loggerAddCmd.MarkFlagRequired("name")
}
