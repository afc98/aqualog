package cmd

import (
	"aqualog/db"
	"aqualog/process"
	"fmt"

	"github.com/spf13/cobra"
)

var loggerProcessCmd = &cobra.Command{
	Use:   "process",
	Short: "Process raw logger data using manual readings.",
	Run: func(cmd *cobra.Command, args []string) {
		siteID, _ := cmd.Flags().GetInt("site")

		// Open database
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		err = process.ProcessLoggerData(database, siteID)
		if err != nil {
			fmt.Printf("Failed to process logger data for site %d, %w", siteID, err)
			return
		}
		fmt.Println("Successfully process logger data for site", siteID)
	},
}

func init() {
	loggerCmd.AddCommand(loggerProcessCmd)

	loggerProcessCmd.Flags().IntP("site", "s", 0, "Site ID")
	loggerListCmd.MarkFlagRequired("site")
}
