package cmd

import (
	"aqualog/db"
	"aqualog/query"
	"fmt"

	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query logger or corrected data for a site.",
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		typeFlag, _ := cmd.Flags().GetString("type")

		// Open database
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()
		fmt.Println("Fetching data...")
		switch typeFlag {
		case "raw":
			err = query.QueryLoggerData(database, siteID)
		case "corrected":
			err = query.QueryCorrectedData(database, siteID)
		default:
			fmt.Println("Invalid type specified. Use 'raw' or 'corrected'.")
			return
		}
		if err != nil {
			fmt.Printf("Failed to query logger data for site %d, %w", siteID, err)
			return
		}
		fmt.Println("Successfully queried logger data for site", siteID)
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)

	queryCmd.Flags().StringP("site", "s", "", "Site ID or name")
	queryCmd.Flags().StringP("type", "t", "", "Type of data to query (raw or corrected)")

	queryCmd.MarkFlagRequired("site")
	queryCmd.MarkFlagRequired("type")
}
