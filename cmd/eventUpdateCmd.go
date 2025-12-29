package cmd

import (
	"aqualog/core/event"
	"bufio"
	"fmt"
	"os"
	"strings"

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

		existing, err := event.Get(eventID)
		if err != nil {
			fmt.Println("Error fetching event:", err)
			return
		}

		updated, err := event.PrepareUpdate(existing, event.UpdateParams{
			EventType: newEventType,
			Timestamp: tsString,
			Notes:     newNotes,
		})
		if err != nil {
			fmt.Println("Error preparing update:", err)
			return
		}

		// Preview
		fmt.Printf("Are you sure you want to update logger event %d?\n", eventID)
		fmt.Printf(
			"Existing: Logger %d (%s, %s) Notes: %s\n",
			existing.LoggerID,
			existing.Timestamp.Format("2006-01-02 15:04:05"),
			existing.EventType,
			existing.Notes,
		)
		fmt.Printf(
			"     New: Logger %d (%s, %s) Notes: %s\n",
			updated.LoggerID,
			updated.Timestamp.Format("2006-01-02 15:04:05"),
			updated.EventType,
			updated.Notes,
		)

		fmt.Print("Enter Y to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		if strings.TrimSpace(input) != "Y" {
			fmt.Println("Cancelled.")
			return
		}

		// Apply once
		if err := event.Update(eventID, updated); err != nil {
			fmt.Println("Error updating event:", err)
			return
		}

		fmt.Println("Logger event updated.")
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
