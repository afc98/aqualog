package cmd

import (
	"aqualog/core/db"
	"aqualog/core/event"
	"fmt"

	"github.com/spf13/cobra"
)

var eventAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a logger event (installed, moved, removed, other)",
	Run: func(cmd *cobra.Command, args []string) {
		loggerIdent, _ := cmd.Flags().GetString("logger")
		loggerID, err := db.ResolveLoggerIdentifier(loggerIdent)
		if err != nil {
			fmt.Println("Logger not found:", err)
			return
		}
		eventType, _ := cmd.Flags().GetString("type")
		timestamp, _ := cmd.Flags().GetString("time")
		notes, _ := cmd.Flags().GetString("notes")

		err = event.EventAdd(loggerID, eventType, timestamp, notes)
		if err != nil {
			fmt.Printf("Failed to add event for logger %d, %w", loggerID, err)
			return
		}
		fmt.Println("Successfully added event for logger", loggerID)
	},
}

func init() {
	eventAddCmd.Flags().StringP("logger", "l", "", "Logger ID or name")
	eventAddCmd.Flags().StringP("type", "t", "", "Event Type (install/move/remove)")
	eventAddCmd.Flags().StringP("time", "T", "", "Timestamp for event")
	eventAddCmd.Flags().StringP("notes", "N", "", "Notes")

	eventAddCmd.MarkFlagRequired("logger")
	eventAddCmd.MarkFlagRequired("type")
	eventAddCmd.MarkFlagRequired("time")

	eventCmd.AddCommand(eventAddCmd)
}
