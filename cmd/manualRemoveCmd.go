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

var manualRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a manual water level reading by its ID",
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetInt("id")

		// Open database
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		// Fetch the manual reading so we can display it
		var siteID int
		var ts time.Time
		var value float64
		var notes string

		err = database.QueryRow(`
			SELECT site_id, timestamp, value, notes
			FROM manual_readings
			WHERE id = ?
		`, id).Scan(&siteID, &ts, &value, &notes)

		if err != nil {
			fmt.Println("Manual reading not found with that ID.")
			return
		}

		// Show details and request confirmation
		fmt.Printf("Are you sure you want to delete manual reading %d?\n", id)
		fmt.Printf("%d. Site %d (%s, %.3f) Notes: %s\n",
			id, siteID, ts.Format("2006-01-02 15:04:05"), value, notes)
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
			DELETE FROM manual_readings
			WHERE id = ?
		`, id)
		if err != nil {
			fmt.Println("Error removing manual reading:", err)
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			fmt.Println("No manual reading removed — unexpected.")
			return
		}

		fmt.Println("Manual reading removed.")
	},
}

func init() {
	manualRemoveCmd.Flags().IntP("id", "i", 0, "Manual reading ID to remove")
	manualRemoveCmd.MarkFlagRequired("id")

	manualCmd.AddCommand(manualRemoveCmd)
}
