package cmd

import (
	"aqualog/db"
	"aqualog/export"
	"fmt"

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

		fmt.Println("Exporting data...")
		err = export.ExportData(siteID, fileName)
		if err != nil {
			fmt.Printf("Failed to export logger data for site %d, %w", siteID, err)
			return
		}
		fmt.Println("Successfully exported logger data for site", siteID)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringP("site", "s", "", "Site ID or name")
	exportCmd.Flags().StringP("file", "f", "", "Output file")

	exportCmd.MarkFlagRequired("site")
	exportCmd.MarkFlagRequired("file")
}
