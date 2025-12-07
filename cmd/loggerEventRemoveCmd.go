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

var loggerEventRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a logger event by its ID",
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetInt("id")

		// Open database connection
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		// Fetch the event record
		var loggerID int
		var ts time.Time
		var eventType string
		var notes string

		err = database.QueryRow(`
			SELECT logger_id, timestamp, event_type, notes
			FROM logger_events
			WHERE id = ?
		`, id).Scan(&loggerID, &ts, &eventType, &notes)

		if err != nil {
			fmt.Println("No logger event found with that ID.")
			return
		}

		// Show details + confirmation prompt
		fmt.Printf("Are you sure you want to delete logger event %d?\n", id)
		fmt.Printf("%d. Logger %d (%s, %s) Notes: %s\n",
			id,
			loggerID,
			ts.Format("2006-01-02 15:04:05"),
			eventType,
			notes,
		)

		fmt.Print("Enter Y to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input != "Y" {
			fmt.Println("Cancelled.")
			return
		}

		// Perform delete
		result, err := database.Exec(`
			DELETE FROM logger_events
			WHERE id = ?
		`, id)
		if err != nil {
			fmt.Println("Error removing logger event:", err)
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			fmt.Println("No logger event removed — unexpected.")
			return
		}

		fmt.Println("Logger event removed.")
	},
}

func init() {
	loggerEventRemoveCmd.Flags().IntP("id", "i", 0, "Logger event ID to remove")
	loggerEventRemoveCmd.MarkFlagRequired("id")

	loggerEventAddCmd.AddCommand(loggerEventRemoveCmd)
}
