package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/qa"
	"time"

	"github.com/spf13/cobra"
)

var gapsCmd = &cobra.Command{
	Use:   "gaps",
	Short: "Find timestamp gaps in logger data",
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		loggerID := 0
		loggerIdent, _ := cmd.Flags().GetString("logger")
		if loggerIdent != "" {
			loggerID, err = db.ResolveLoggerIdentifier(loggerIdent)
			if err != nil {
				fmt.Println("Logger not found:", err)
				return
			}
		}
		thresholdText, _ := cmd.Flags().GetString("threshold")
		var threshold time.Duration
		if thresholdText != "" {
			threshold, err = time.ParseDuration(thresholdText)
			if err != nil {
				fmt.Println("Invalid threshold:", err)
				return
			}
		}
		gaps, err := qa.FindGaps(siteID, loggerID, threshold)
		if err != nil {
			fmt.Println("Failed to find gaps:", err)
			return
		}
		if len(gaps) == 0 {
			fmt.Println("No gaps found.")
			return
		}
		for _, gap := range gaps {
			fmt.Printf("Logger %s (ID %d): %s to %s (%s)\n",
				gap.LoggerName,
				gap.LoggerID,
				gap.Start.Format(time.RFC3339),
				gap.End.Format(time.RFC3339),
				gap.Duration,
			)
		}
	},
}

func init() {
	rootCmd.AddCommand(gapsCmd)
	gapsCmd.Flags().StringP("site", "s", "", "Site ID or name")
	gapsCmd.Flags().StringP("logger", "l", "", "Logger ID or name")
	gapsCmd.Flags().String("threshold", "", "Gap threshold duration, e.g. 2h or 30m; defaults to twice the median interval")
	gapsCmd.MarkFlagRequired("site")
}
