package cmd

import (
	"aqualog/db"
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
			fmt.Println("Project not found:", err)
			return
		}

		// Open database
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		// Query projects
		rows, err := database.Query(`
		SELECT id, site_id, timestamp, value, notes 
		FROM manual_readings WHERE site_id=?
		ORDER BY id
		`, siteID)
		if err != nil {
			fmt.Println("Failed to query manual readings:", err)
			return
		}
		defer rows.Close()

		fmt.Println("Manual readings for site:")
		for rows.Next() {
			var id int
			var site, time, value, notes string
			rows.Scan(&id, &site, &time, &value, &notes)

			fmt.Printf(" %d. %s (time: %s, value: %s) notes: %s\n", id, site, time, value, notes)
		}
	},
}

func init() {
	manualCmd.AddCommand(manualListCmd)

	manualListCmd.Flags().StringP("site", "s", "", "Site ID or name")
	manualListCmd.MarkFlagRequired("site")
}
