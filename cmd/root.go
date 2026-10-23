package cmd

import (
	"github.com/afc98/aqualog/core/db"
	"os"

	"github.com/spf13/cobra"
)

var dbPath string
var databaseURL string

var rootCmd = &cobra.Command{
	Use:   "aqualog",
	Short: "Manage environmental water-level logger data",
	Long:  `Aqualog manages projects, sites, loggers, manual readings, logger events, imports, processing, plotting, and exports for environmental logger data.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		db.SetPathOverride(dbPath)
		db.SetDatabaseURLOverride(databaseURL)
		_, err := db.ResolveConfig()
		return err
	},
}

func Execute() {
	err := rootCmd.Execute()
	_ = db.Close()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Path to the SQLite database")
	rootCmd.PersistentFlags().StringVar(&databaseURL, "database-url", "", "PostgreSQL connection URL")
}
