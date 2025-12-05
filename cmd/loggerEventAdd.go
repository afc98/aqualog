package cmd

import (
	"aqualog/db"
	"fmt"

	"github.com/spf13/cobra"
)

var loggerEventAddCmd = &cobra.Command{
	Use:   "event",
	Short: "Add a logger event (installation, movement, removal)",
	Run: func(cmd *cobra.Command, args []string) {
		loggerID, _ := cmd.Flags().GetInt("logger")
		eventType, _ := cmd.Flags().GetString("type")
		timestamp, _ := cmd.Flags().GetString("time")
		notes, _ := cmd.Flags().GetString("notes")

		// Open database
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		_, err = database.Exec(`
		INSERT INTO logger_events (logger_id, timestamp, event_type, notes)
		VALUES (?, ?, ?, ?)
		`, loggerID, eventType, timestamp, notes)
		if err != nil {
			fmt.Println("Error adding event:", err)
			return
		}

		fmt.Println("Event added.")
	},
}

func init() {
	loggerEventAddCmd.Flags().IntP("logger", "l", 0, "Logger ID")
	loggerEventAddCmd.Flags().StringP("type", "t", "", "Event Type (install/move/remove)")
	loggerEventAddCmd.Flags().StringP("time", "i", "", "Timestamp for event")
	loggerEventAddCmd.Flags().StringP("notes", "n", "", "Notes")

	loggerEventAddCmd.MarkFlagRequired("logger")
	loggerEventAddCmd.MarkFlagRequired("type")
	loggerEventAddCmd.MarkFlagRequired("time")

	loggerCmd.AddCommand(loggerEventAddCmd)
}
