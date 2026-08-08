package cmd

import (
	"bufio"
	"fmt"
	"github.com/afc98/aqualog/core/manual"
	"os"
	"strings"

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
		existing, err := manual.Get(id)
		if err != nil {
			fmt.Println("Error fetching manual reading:", err)
			return
		}

		updated, err := manual.PrepareUpdate(existing, manual.UpdateParams{
			WaterLevel: newWaterLevel,
			Timestamp:  tsString,
			Notes:      newNotes,
		})
		if err != nil {
			fmt.Println("Error preparing update:", err)
			return
		}

		// Preview
		fmt.Printf("Are you sure you want to update manual reading %d?\n", id)
		fmt.Printf(
			"Existing: Site %d (%s) Water Level: %.2f Notes: %s\n",
			existing.SiteID,
			existing.Timestamp.Format("2006-01-02 15:04:05"),
			existing.WaterLevel,
			existing.Notes,
		)
		fmt.Printf(
			"     New: Site %d (%s) Water Level: %.2f Notes: %s\n",
			updated.SiteID,
			updated.Timestamp.Format("2006-01-02 15:04:05"),
			updated.WaterLevel,
			updated.Notes,
		)

		fmt.Print("Enter Y to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		if strings.TrimSpace(input) != "Y" {
			fmt.Println(" Cancelled.")
			return
		}

		// Apply once
		if err := manual.Update(id, updated); err != nil {
			fmt.Println("Error updating manual reading:", err)
			return
		}

		fmt.Println("Manual reading updated.")
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
