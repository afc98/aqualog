package cmd

import (
	"github.com/spf13/cobra"
)

var loggerCmd = &cobra.Command{
	Use:   "logger",
	Short: "Manage loggers at site",
}

func init() {
	rootCmd.AddCommand(loggerCmd)
}
