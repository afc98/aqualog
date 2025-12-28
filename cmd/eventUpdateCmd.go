package cmd

import (
	"aqualog/event"
	"fmt"

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

		err := event.EventUpdate(eventID, newEventType, tsString, newNotes)
		if err != nil {
			fmt.Printf("Failed to update event ID %d: %v\n", eventID, err)
			return
		}
	},
}

func init() {
	eventUpdateCmd.Flags().IntP("id", "i", 0, "Logger event ID to update")
	eventUpdateCmd.Flags().StringP("type", "t", "", "Event type (installed, moved, removed, other)")
	eventUpdateCmd.Flags().StringP("time", "T", "", "Timestamp (YYYYMMDD HH:MM:SS)")
	eventUpdateCmd.Flags().StringP("notes", "N", "", "Notes (blank keeps existing)")
	eventUpdateCmd.MarkFlagRequired("id")

	eventCmd.AddCommand(eventUpdateCmd)
}
