package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/manual"

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
		man, err := manual.Add(manual.AddParams{
			SiteID:     siteID,
			WaterLevel: waterLevel,
			Timestamp:  timestamp,
			Notes:      notes,
		})
		if err != nil {
			fmt.Println("Failed to add manual reading:", err)
			return
		}

		fmt.Printf(
			"Manual reading added: ID %d (site %d, %s)\n",
			man.ID,
			man.SiteID,
			man.Timestamp.Format("2006-01-02 15:04:05"),
		)
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
