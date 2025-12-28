package cmd

import (
	"aqualog/db"
	"aqualog/process"
	"fmt"

	"github.com/spf13/cobra"
)

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Process raw logger data using manual readings.",
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}

		// Open database
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()
		fmt.Println("Processing logger data...")
		err = process.ProcessLoggerData(database, siteID)
		if err != nil {
			fmt.Printf("Failed to process logger data for site %d, %w", siteID, err)
			return
		}
		fmt.Println("Successfully process logger data for site", siteID)
	},
}

func init() {
	rootCmd.AddCommand(processCmd)

	processCmd.Flags().StringP("site", "s", "", "Site ID or name")
	processCmd.MarkFlagRequired("site")
}
