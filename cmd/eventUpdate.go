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

var eventUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a logger event by its ID",
	Run: func(cmd *cobra.Command, args []string) {

		eventID, _ := cmd.Flags().GetInt("id")
		newEventType, _ := cmd.Flags().GetString("type")
		tsString, _ := cmd.Flags().GetString("time")
		newNotes, _ := cmd.Flags().GetString("notes")

		// Normalize event type if provided
		newEventType = strings.ToLower(newEventType)

		validEvents := map[string]struct{}{
			"installed": {},
			"moved":     {},
			"removed":   {},
			"other":     {},
		}

		// Open DB
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		// Fetch existing row
		var loggerID int
		var existingTS time.Time
		var existingType string
		var existingNotes string

		err = database.QueryRow(`
			SELECT logger_id, timestamp, event_type, notes
			FROM logger_events
			WHERE id = ?
		`, eventID).Scan(&loggerID, &existingTS, &existingType, &existingNotes)

		if err != nil {
			fmt.Println("Logger event not found with that ID.")
			return
		}

		// Defaults
		finalTS := existingTS
		finalType := existingType
		finalNotes := existingNotes

		// Parse timestamp if provided
		if tsString != "" {
			t, err := time.Parse("20060102 15:04:05", tsString)
			if err != nil {
				fmt.Println("Invalid timestamp. Use: YYYYMMDD HH:MM:SS")
				return
			}
			finalTS = t
		}

		// Apply event type if provided
		if newEventType != "" {
			if _, ok := validEvents[newEventType]; !ok {
				fmt.Println("Unsupported event type. Use: installed, moved, removed, other")
				return
			}
			finalType = newEventType
		}

		// Apply notes if provided
		if newNotes != "" {
			finalNotes = newNotes
		}

		// Preview before applying
		fmt.Printf("Are you sure you want to update logger event %d?\n", eventID)
		fmt.Printf("Existing: Logger %d (%s, %s) Notes: %s\n",
			loggerID,
			existingTS.Format("2006-01-02 15:04:05"),
			existingType,
			existingNotes,
		)
		fmt.Printf("     New: Logger %d (%s, %s) Notes: %s\n",
			loggerID,
			finalTS.Format("2006-01-02 15:04:05"),
			finalType,
			finalNotes,
		)
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
			UPDATE logger_events
			SET timestamp = ?, event_type = ?, notes = ?
			WHERE id = ?
		`, finalTS, finalType, finalNotes, eventID)

		if err != nil {
			fmt.Println("Error updating logger event:", err)
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			fmt.Println("No logger event updated — unexpected.")
			return
		}

		fmt.Println("Logger event updated.")
	},
}

func init() {
	eventUpdateCmd.Flags().IntP("id", "i", 0, "Logger event ID to update")
	eventUpdateCmd.Flags().StringP("type", "t", "", "Event type (installed, moved, removed, other)")
	eventUpdateCmd.Flags().StringP("time", "m", "", "Timestamp (YYYYMMDD HH:MM:SS)")
	eventUpdateCmd.Flags().StringP("notes", "n", "", "Notes (blank keeps existing)")

	eventUpdateCmd.MarkFlagRequired("id")

	eventCmd.AddCommand(eventUpdateCmd)
}
