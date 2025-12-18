package cmd

import (
	"aqualog/db"
	"fmt"

	"github.com/spf13/cobra"
)

var eventListCmd = &cobra.Command{
	Use:   "list",
	Short: "List logger events (filter by site and/or logger)",
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID := 0
		if siteIdent != "" {
			var err error
			siteID, err = db.ResolveSiteIdentifier(siteIdent)
			if err != nil {
				fmt.Println("Site not found:", err)
				return
			}
		}

		loggerIdent, _ := cmd.Flags().GetString("logger")
		loggerID := 0
		if loggerIdent != "" {
			var err error
			loggerID, err = db.ResolveLoggerIdentifier(loggerIdent)
			if err != nil {
				fmt.Println("Logger not found:", err)
				return
			}
		}

		// Must specify at least one filter
		if siteID == 0 && loggerID == 0 {
			fmt.Println("You must specify --site <id/name> and/or --logger <id/name>")
			return
		}

		// Build SQL dynamically
		query := `
			SELECT 
				e.id,
				e.logger_id,
				e.timestamp,
				e.event_type,
				e.notes
			FROM logger_events e
			JOIN loggers l ON e.logger_id = l.id
			WHERE 1=1
		`
		params := []any{}

		if siteID != 0 {
			query += " AND l.site_id = ?"
			params = append(params, siteID)
		}
		if loggerID != 0 {
			query += " AND l.id = ?"
			params = append(params, loggerID)
		}

		query += " ORDER BY e.timestamp"

		// Open DB
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		rows, err := database.Query(query, params...)
		if err != nil {
			fmt.Println("Failed to query logger events:", err)
			return
		}
		defer rows.Close()

		if siteID != 0 && loggerID != 0 {
			fmt.Printf("Logger events for site %d and logger %d:\n", siteID, loggerID)
		} else if siteID != 0 {
			fmt.Printf("Logger events for site %d:\n", siteID)
		} else {
			fmt.Printf("Logger events for logger %d:\n", loggerID)
		}

		found := false
		for rows.Next() {
			found = true

			var id int
			var logID int
			var timestamp string
			var eventType string
			var notes string

			err := rows.Scan(&id, &logID, &timestamp, &eventType, &notes)
			if err != nil {
				fmt.Println("Row scan error:", err)
				continue
			}

			fmt.Printf(" %d. Logger %d (time: %s, event: %s) notes: %s\n",
				id, logID, timestamp, eventType, notes)
		}

		if !found {
			fmt.Println(" No events found.")
		}
	},
}

func init() {
	eventCmd.AddCommand(eventListCmd)

	eventListCmd.Flags().StringP("site", "s", "", "Filter by Site ID")
	eventListCmd.Flags().StringP("logger", "l", "", "Filter by Logger ID")
}
