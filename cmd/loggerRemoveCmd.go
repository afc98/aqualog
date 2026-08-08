package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/logger"

	"github.com/spf13/cobra"
)

var loggerRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a logger by ID or name",
	Run: func(cmd *cobra.Command, args []string) {
		ident, _ := cmd.Flags().GetString("logger")
		id, err := db.ResolveLoggerIdentifier(ident)
		if err != nil {
			fmt.Println("Logger not found:", err)
			return
		}
		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			fmt.Println("Refusing to remove logger without --confirm.")
			return
		}
		if err := logger.Remove(id); err != nil {
			fmt.Println("Failed to remove logger:", err)
			return
		}
		fmt.Println("Logger removed.")
	},
}

func init() {
	loggerCmd.AddCommand(loggerRemoveCmd)
	loggerRemoveCmd.Flags().StringP("logger", "l", "", "Logger ID or name")
	loggerRemoveCmd.Flags().Bool("confirm", false, "Confirm removal")
	loggerRemoveCmd.MarkFlagRequired("logger")
}
