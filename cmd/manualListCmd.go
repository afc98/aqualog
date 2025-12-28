package cmd

import (
	"aqualog/core/db"
	"aqualog/core/manual"
	"fmt"

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
		err = manual.ListManualReadings(siteID)
		if err != nil {
			fmt.Println("Error listing manual readings:", err)
			return
		}
	},
}

func init() {
	manualCmd.AddCommand(manualListCmd)

	manualListCmd.Flags().StringP("site", "s", "", "Site ID or name")
	manualListCmd.MarkFlagRequired("site")
}
