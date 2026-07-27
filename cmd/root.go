package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "aqualog",
	Short: "Manage environmental water-level logger data",
	Long:  `Aqualog manages projects, sites, loggers, manual readings, logger events, imports, processing, plotting, and exports for environmental logger data.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}
