package cmd

import (
	"aqualog/db"
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var manualUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a manual water level reading by its ID",
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetInt("id")
		newWaterLevel, _ := cmd.Flags().GetFloat64("waterlevel")
		tsString, _ := cmd.Flags().GetString("time")
		newNotes, _ := cmd.Flags().GetString("notes")

		// Open DB
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		// Fetch existing row
		var siteID int
		var existingTS time.Time
		var existingValue float64
		var existingNotes string

		err = database.QueryRow(`
			SELECT site_id, timestamp, value, notes
			FROM manual_readings
			WHERE id = ?
		`, id).Scan(&siteID, &existingTS, &existingValue, &existingNotes)
		if err != nil {
			fmt.Println("Manual reading not found with that ID.")
			return
		}

		// Apply defaults BEFORE preview
		finalTS := existingTS
		finalValue := existingValue
		finalNotes := existingNotes

		// Parse timestamp if provided
		if tsString != "" {
			t, err := time.Parse("20060102 15:04:05", tsString)
			if err != nil {
				fmt.Println("Invalid timestamp format. Use: YYYYMMDD HH:MM:SS")
				return
			}
			finalTS = t
		}

		// Apply water level unless -1
		if newWaterLevel != -1 {
			finalValue = newWaterLevel
		}

		if newNotes != "" {
			finalNotes = newNotes
		}

		// Preview
		fmt.Printf("Are you sure you want to update manual reading %d?\n", id)
		fmt.Printf("Existing: %d. Site %d (%s, %.3f) Notes: %s\n",
			id, siteID, existingTS.Format("2006-01-02 15:04:05"), existingValue, existingNotes)
		fmt.Printf("     New: %d. Site %d (%s, %.3f) Notes: %s\n",
			id, siteID, finalTS.Format("2006-01-02 15:04:05"), finalValue, finalNotes)
		fmt.Print("Enter Y to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input != "Y" {
			fmt.Println("Cancelled.")
			return
		}

		// Perform update
		result, err := database.Exec(`
			UPDATE manual_readings
			SET timestamp = ?, value = ?, notes = ?
			WHERE id = ?
		`, finalTS, finalValue, finalNotes, id)

		if err != nil {
			fmt.Println("Error updating manual reading:", err)
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			fmt.Println("No manual reading updated — unexpected.")
			return
		}

		fmt.Println("Manual reading updated.")
	},
}

func init() {
	manualUpdateCmd.Flags().IntP("id", "i", 0, "Manual reading ID to update")
	manualUpdateCmd.Flags().StringP("time", "T", "", "Timestamp (YYYYMMDD HH:MM:SS)")
	manualUpdateCmd.Flags().Float64P("waterlevel", "w", -1, "Water level (or -1 to leave unchanged)")
	manualUpdateCmd.Flags().StringP("notes", "N", "", "Notes (leave blank to keep existing)")
	
	manualUpdateCmd.MarkFlagRequired("id")

	manualCmd.AddCommand(manualUpdateCmd)
}
