package cmd

import (
	"aqualog/db"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var manualAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a manual (to datum) water level reading",
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Project not found:", err)
			return
		}
		waterLevel, _ := cmd.Flags().GetFloat64("waterlevel")
		timestamp, _ := cmd.Flags().GetString("time")
		notes, _ := cmd.Flags().GetString("notes")

		// Parse and validate timestamp
		parsedTime, err := time.Parse("20060102 15:04:05", timestamp)
		if err != nil {
			fmt.Println("Invalid timestamp format. Use: YYYYMMDD HH:MM:SS")
			return
		}

		// Open database
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		_, err = database.Exec(`
		INSERT INTO manual_readings (site_id, timestamp, value, notes)
		VALUES (?, ?, ?, ?)
		`, siteID, parsedTime, waterLevel, notes)
		if err != nil {
			fmt.Println("Error adding manual reading:", err)
			return
		}
		fmt.Println("Manual reading added")
	},
}

func init() {
	manualAddCmd.Flags().StringP("site", "s", "", "Site ID or name")
	manualAddCmd.Flags().Float64P("waterlevel", "w", 0, "Water level")
	manualAddCmd.Flags().StringP("time", "t", "", "Timestamp")
	manualAddCmd.Flags().StringP("notes", "n", "", "Notes")

	manualAddCmd.MarkFlagRequired("site")
	manualAddCmd.MarkFlagRequired("waterlevel")
	manualAddCmd.MarkFlagRequired("timestamp")

	manualCmd.AddCommand(manualAddCmd)
}
