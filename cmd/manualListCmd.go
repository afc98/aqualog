package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/manual"

	"github.com/spf13/cobra"
)

var manualListCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing manual readings for a site",
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}

		// List manual readings
		mans, err := manual.List(siteID)
		if err != nil {
			fmt.Println("Error retrieving manual readings:", err)
			return
		}

		if siteID != 0 {
			fmt.Printf("Manual readings for site %d:\n", siteID)
		} else {
			fmt.Println(" No manual readings found.")
			return
		}

		for _, m := range mans {
			fmt.Printf(
				" %d. Site %d (time: %s, water level: %.2f) notes: %s\n",
				m.ID,
				m.SiteID,
				m.Timestamp.Format("2006-01-02 15:04:05"),
				m.WaterLevel,
				m.Notes,
			)
		}
	},
}

func init() {
	manualCmd.AddCommand(manualListCmd)

	manualListCmd.Flags().StringP("site", "s", "", "Site ID or name")
	manualListCmd.MarkFlagRequired("site")
}
