package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/event"

	"github.com/spf13/cobra"
)

var eventListCmd = &cobra.Command{
	Use:   "list",
	Short: "List logger events (filter by site and/or logger)",
	RunE: func(cmd *cobra.Command, args []string) error {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID := 0
		if siteIdent != "" {
			var err error
			siteID, err = db.ResolveSiteIdentifier(siteIdent)
			if err != nil {
				fmt.Println("Site not found:", err)
				return nil
			}
		}

		loggerIdent, _ := cmd.Flags().GetString("logger")
		loggerID := 0
		if loggerIdent != "" {
			var err error
			loggerID, err = db.ResolveLoggerIdentifier(loggerIdent)
			if err != nil {
				fmt.Println("Logger not found:", err)
				return nil
			}
		}

		events, err := event.List(siteID, loggerID)
		if err != nil {
			return err
		}

		if siteID != 0 && loggerID != 0 {
			fmt.Printf("Logger events for site %d and logger %d:\n", siteID, loggerID)
		} else if siteID != 0 {
			fmt.Printf("Logger events for site %d:\n", siteID)
		} else {
			fmt.Printf("Logger events for logger %d:\n", loggerID)
		}

		if len(events) == 0 {
			fmt.Println(" No events found.")
			return nil
		}

		for _, e := range events {
			fmt.Printf(
				" %d. Logger %d (time: %s, event: %s) notes: %s\n",
				e.ID,
				e.LoggerID,
				e.Timestamp.Format("2006-01-02 15:04:05"),
				e.EventType,
				e.Notes,
			)
		}

		return nil
	},
}

func init() {
	eventCmd.AddCommand(eventListCmd)

	eventListCmd.Flags().StringP("site", "s", "", "Filter by Site ID")
	eventListCmd.Flags().StringP("logger", "l", "", "Filter by Logger ID")
}
