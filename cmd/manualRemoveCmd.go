package cmd

import (
	"aqualog/core/manual"
	"fmt"

	"github.com/spf13/cobra"
)

var manualRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a manual water level reading by its ID",
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetInt("id")

		// Remove manual reading
		err := manual.RemoveManualReading(id)
		if err != nil {
			fmt.Println("Error removing manual reading:", err)
			return
		}

	},
}

func init() {
	manualRemoveCmd.Flags().IntP("id", "i", 0, "Manual reading ID to remove")
	manualRemoveCmd.MarkFlagRequired("id")

	manualCmd.AddCommand(manualRemoveCmd)
}
