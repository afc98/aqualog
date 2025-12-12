package cmd

import (
	"github.com/spf13/cobra"
)

var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "Manage events at site",
}

func init() {
	rootCmd.AddCommand(eventCmd)
}
