package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/export"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export data in csv format for a site.",
	Run: func(cmd *cobra.Command, args []string) {
		fileName, _ := cmd.Flags().GetString("file")
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		format, _ := cmd.Flags().GetString("format")

		fmt.Println("Exporting data...")
		err = export.ExportDataWithOptions(siteID, fileName, export.Options{
			From:   from,
			To:     to,
			Format: format,
		})
		if err != nil {
			fmt.Printf("Failed to export logger data for site %d: %v\n", siteID, err)
			return
		}
		fmt.Println("Successfully exported logger data for site", siteID)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringP("site", "s", "", "Site ID or name")
	exportCmd.Flags().StringP("file", "f", "", "Output file")
	exportCmd.Flags().String("from", "", "Start timestamp filter in RFC3339 format")
	exportCmd.Flags().String("to", "", "End timestamp filter in RFC3339 format")
	exportCmd.Flags().String("format", "csv", "Output format: csv or json")

	exportCmd.MarkFlagRequired("site")
	exportCmd.MarkFlagRequired("file")
}
