package cmd

import (
	"github.com/spf13/cobra"
)

var manualCmd = &cobra.Command{
	Use:   "manual",
	Short: "Manage manual (to datum) water level measurements",
}

func init() {
	rootCmd.AddCommand(manualCmd)
}
