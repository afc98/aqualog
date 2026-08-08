package cmd

import (
	"github.com/afc98/aqualog/core/db"
	"os"

	"github.com/spf13/cobra"
)

var dbPath string

var rootCmd = &cobra.Command{
	Use:   "aqualog",
	Short: "Manage environmental water-level logger data",
	Long:  `Aqualog manages projects, sites, loggers, manual readings, logger events, imports, processing, plotting, and exports for environmental logger data.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		db.SetPathOverride(dbPath)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Path to the SQLite database")
}
