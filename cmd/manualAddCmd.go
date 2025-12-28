package cmd

import (
	"aqualog/db"
	"aqualog/manual"
	"fmt"

	"github.com/spf13/cobra"
)

var manualAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a manual (to datum) water level reading",
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		waterLevel, _ := cmd.Flags().GetFloat64("waterlevel")
		timestamp, _ := cmd.Flags().GetString("time")
		notes, _ := cmd.Flags().GetString("notes")

		// Add manual reading
		err = manual.AddManualReading(siteID, waterLevel, timestamp, notes)
		if err != nil {
			fmt.Println("Error adding manual reading:", err)
			return
		}
	},
}

func init() {
	manualAddCmd.Flags().StringP("site", "s", "", "Site ID or name")
	manualAddCmd.Flags().Float64P("waterlevel", "w", 0, "Water level")
	manualAddCmd.Flags().StringP("time", "T", "", "Timestamp")
	manualAddCmd.Flags().StringP("notes", "N", "", "Notes")

	manualAddCmd.MarkFlagRequired("site")
	manualAddCmd.MarkFlagRequired("waterlevel")
	manualAddCmd.MarkFlagRequired("time")

	manualCmd.AddCommand(manualAddCmd)
}
