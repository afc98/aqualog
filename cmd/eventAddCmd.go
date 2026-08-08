package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/event"

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

		evt, err := event.Add(event.AddParams{
			LoggerID:  loggerID,
			EventType: eventType,
			Timestamp: timestamp,
			Notes:     notes,
		})
		if err != nil {
			fmt.Println("Failed to add event:", err)
			return
		}

		fmt.Printf(
			"Event added: ID %d (logger %d, %s, %s)\n",
			evt.ID,
			evt.LoggerID,
			evt.Timestamp.Format("2006-01-02 15:04:05"),
			evt.EventType,
		)
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
