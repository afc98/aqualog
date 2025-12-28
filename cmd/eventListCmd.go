package cmd

import (
	"aqualog/core/db"
	"aqualog/core/event"
	"fmt"

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

		err := event.EventList(siteID, loggerID)
		if err != nil {
			fmt.Printf("Failed to list logger events: %v\n", err)
			return err
		}
		return nil
	},
}

func init() {
	eventCmd.AddCommand(eventListCmd)

	eventListCmd.Flags().StringP("site", "s", "", "Filter by Site ID")
	eventListCmd.Flags().StringP("logger", "l", "", "Filter by Logger ID")
}
