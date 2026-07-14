package cmd

import (
	"aqualog/core/db"
	"aqualog/core/logger"
	"fmt"

	"github.com/spf13/cobra"
)

var loggerUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a logger",
	Run: func(cmd *cobra.Command, args []string) {
		ident, _ := cmd.Flags().GetString("logger")
		id, err := db.ResolveLoggerIdentifier(ident)
		if err != nil {
			fmt.Println("Logger not found:", err)
			return
		}
		name, _ := cmd.Flags().GetString("name")
		model, _ := cmd.Flags().GetString("model")
		serial, _ := cmd.Flags().GetString("serial")
		if name == "" && model == "" && serial == "" {
			fmt.Println("Nothing to update.")
			return
		}
		if err := logger.Update(id, name, model, serial); err != nil {
			fmt.Println("Failed to update logger:", err)
			return
		}
		fmt.Println("Logger updated.")
	},
}

func init() {
	loggerCmd.AddCommand(loggerUpdateCmd)
	loggerUpdateCmd.Flags().StringP("logger", "l", "", "Logger ID or name")
	loggerUpdateCmd.Flags().StringP("name", "n", "", "New logger name")
	loggerUpdateCmd.Flags().StringP("model", "m", "", "New logger model")
	loggerUpdateCmd.Flags().StringP("serial", "S", "", "New logger serial")
	loggerUpdateCmd.MarkFlagRequired("logger")
}
