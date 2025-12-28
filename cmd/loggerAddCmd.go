/*
Copyright © 2025 Arran Clarke
*/
package cmd

import (
	"aqualog/core/db"
	"aqualog/core/logger"
	"fmt"

	"github.com/spf13/cobra"
)

// projectAddCmd represents the projectAdd command
var loggerAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new logger to a site",
	Run: func(cmd *cobra.Command, args []string) {
		// Read flags
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		name, _ := cmd.Flags().GetString("name")
		model, _ := cmd.Flags().GetString("model")
		serial, _ := cmd.Flags().GetString("serial")

		//Add logger
		err = logger.AddLogger(siteID, name, model, serial)
		if err != nil {
			fmt.Println("Error adding logger:", err)
			return
		}
	},
}

func init() {
	loggerCmd.AddCommand(loggerAddCmd)

	loggerAddCmd.Flags().StringP("site", "s", "", "Site ID or name")
	loggerAddCmd.Flags().StringP("name", "n", "", "Logger name")
	loggerAddCmd.Flags().StringP("model", "m", "", "Logger model")
	loggerAddCmd.Flags().StringP("serial", "S", "", "Serial number")

	loggerAddCmd.MarkFlagRequired("site")
	loggerAddCmd.MarkFlagRequired("name")
}
