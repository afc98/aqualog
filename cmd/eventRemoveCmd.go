package cmd

import (
	"aqualog/core/event"
	"fmt"

	"github.com/spf13/cobra"
)

var eventRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a logger event by its ID",
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetInt("id")

		err := event.EventRemove(id)
		if err != nil {
			fmt.Printf("Failed to remove event ID %d: %v\n", id, err)
			return
		}
	},
}

func init() {
	eventRemoveCmd.Flags().IntP("id", "i", 0, "Logger event ID to remove")
	eventRemoveCmd.MarkFlagRequired("id")

	eventCmd.AddCommand(eventRemoveCmd)
}
