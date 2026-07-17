package cmd

import (
	"aqualog/core/db"
	"aqualog/core/qa"
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show processing status for a site",
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		s, err := qa.SiteStatus(siteID)
		if err != nil {
			fmt.Println("Failed to get status:", err)
			return
		}
		fmt.Printf("Site: %s (ID %d)\n", s.SiteName, s.SiteID)
		fmt.Printf("Project: %s (ID %d)\n", s.ProjectName, s.ProjectID)
		fmt.Printf("Loggers: %d, Files: %d, Events: %d, Manual readings: %d\n", s.LoggerCount, s.FileCount, s.EventCount, s.ManualCount)
		fmt.Printf("Absolute-pressure rows: %d, Barometric rows: %d, Baro-corrected rows: %d\n", s.AbsoluteCount, s.BaroCount, s.BaroCorrected)
		fmt.Printf("Raw records: %d", s.RawCount)
		if s.RawStart.Valid {
			fmt.Printf(" (%s to %s)", s.RawStart.String, s.RawEnd.String)
		}
		fmt.Println()
		fmt.Printf("Corrected records: %d", s.CorrectedCount)
		if s.CorrectedStart.Valid {
			fmt.Printf(" (%s to %s)", s.CorrectedStart.String, s.CorrectedEnd.String)
		}
		fmt.Println()
		if s.CorrectedMissing {
			fmt.Println("Status: raw data present but corrected data missing; run process")
		} else {
			fmt.Println("Status: OK")
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringP("site", "s", "", "Site ID or name")
	statusCmd.MarkFlagRequired("site")
}
