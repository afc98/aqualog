/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"aqualog/db"
	"aqualog/logger"
	"fmt"

	"github.com/spf13/cobra"
)

var loggerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List existing loggers for a site",
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}

		//List loggers
		err = logger.ListLoggers(siteID)
		if err != nil {
			fmt.Println("Error listing loggers:", err)
			return
		}

	},
}

func init() {
	loggerCmd.AddCommand(loggerListCmd)

	loggerListCmd.Flags().StringP("site", "s", "", "Site ID or name")
	loggerListCmd.MarkFlagRequired("site")
}
