package cmd

import (
	"aqualog/manual"
	"fmt"

	"github.com/spf13/cobra"
)

var manualUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a manual water level reading by its ID",
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetInt("id")
		newWaterLevel, _ := cmd.Flags().GetFloat64("waterlevel")
		tsString, _ := cmd.Flags().GetString("time")
		newNotes, _ := cmd.Flags().GetString("notes")

		// Update manual reading
		err := manual.UpdateManualReading(id, tsString, newWaterLevel, newNotes)
		if err != nil {
			fmt.Println("Error updating manual reading:", err)
			return
		}

	},
}

func init() {
	manualUpdateCmd.Flags().IntP("id", "i", 0, "Manual reading ID to update")
	manualUpdateCmd.Flags().StringP("time", "T", "", "Timestamp (YYYYMMDD HH:MM:SS)")
	manualUpdateCmd.Flags().Float64P("waterlevel", "w", -1, "Water level (or -1 to leave unchanged)")
	manualUpdateCmd.Flags().StringP("notes", "N", "", "Notes (leave blank to keep existing)")

	manualUpdateCmd.MarkFlagRequired("id")

	manualCmd.AddCommand(manualUpdateCmd)
}
