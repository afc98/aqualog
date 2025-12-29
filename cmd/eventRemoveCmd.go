package cmd

import (
	"aqualog/core/event"
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var eventRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a logger event by its ID",
	Run: func(cmd *cobra.Command, args []string) {
		eventID, _ := cmd.Flags().GetInt("id")

		existing, err := event.Get(eventID)
		if err != nil {
			fmt.Println("Error fetching event:", err)
			return
		}

		// Show details
		fmt.Printf("Are you sure you want to delete logger event %d?\n", eventID)
		fmt.Printf(
			"%d. Logger %d (%s, %s) Notes: %s\n",
			existing.ID,
			existing.LoggerID,
			existing.Timestamp.Format("2006-01-02 15:04:05"),
			existing.EventType,
			existing.Notes,
		)

		fmt.Print("Enter Y to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		if strings.TrimSpace(input) != "Y" {
			fmt.Println("Cancelled.")
			return
		}

		if err := event.Delete(eventID); err != nil {
			fmt.Println("Error removing event:", err)
			return
		}

		fmt.Println("Logger event removed.")
	},
}

func init() {
	eventRemoveCmd.Flags().IntP("id", "i", 0, "Logger event ID to remove")
	eventRemoveCmd.MarkFlagRequired("id")

	eventCmd.AddCommand(eventRemoveCmd)
}
