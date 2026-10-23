package cmd

import (
	"aqualog/core/db"
	"aqualog/core/process"
	"aqualog/core/utils"
	"fmt"
	"github.com/spf13/cobra"
	"time"
)

var manualCarryForwardCmd = &cobra.Command{
	Use:   "carry-forward",
	Short: "Create an estimated reading from the previous logger interval",
	RunE: func(cmd *cobra.Command, args []string) error {
		site, _ := cmd.Flags().GetString("site")
		logger, _ := cmd.Flags().GetString("logger")
		stamp, _ := cmd.Flags().GetString("time")
		reprocess, _ := cmd.Flags().GetBool("reprocess")
		maxLookback, _ := cmd.Flags().GetDuration("max-lookback")
		siteID, err := db.ResolveSiteIdentifier(site)
		if err != nil {
			return err
		}
		loggerID, err := db.ResolveLoggerIdentifier(logger)
		if err != nil {
			return err
		}
		timestamp, err := utils.ParseDatasetTime(stamp)
		if err != nil {
			return err
		}
		database, err := db.GetDB()
		if err != nil {
			return err
		}
		result, err := process.CarryForward(database, siteID, loggerID, timestamp, maxLookback, reprocess)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Saved estimated reading %d: %.6f m at %s, carried from %s. Reprocessed: %t\n", result.ReadingID, result.Level, utils.FormatDatasetTime(result.Timestamp), utils.FormatDatasetTime(result.SourceTime), result.Reprocessed)
		return nil
	},
}

func init() {
	manualCmd.AddCommand(manualCarryForwardCmd)
	manualCarryForwardCmd.Flags().String("site", "", "Site ID or name")
	manualCarryForwardCmd.Flags().String("logger", "", "Water-level logger ID or name")
	manualCarryForwardCmd.Flags().String("time", "", "Time for the estimated reading (YYYY-MM-DDTHH:MM:SS)")
	manualCarryForwardCmd.Flags().Bool("reprocess", true, "Reprocess the site after adding the reading")
	manualCarryForwardCmd.Flags().Duration("max-lookback", time.Hour, "Maximum age of the source reading, for example 1h or 24h")
	for _, name := range []string{"site", "logger", "time"} {
		_ = manualCarryForwardCmd.MarkFlagRequired(name)
	}
}
