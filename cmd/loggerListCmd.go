package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/logger"

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

		loggers, err := logger.List(siteID)
		if err != nil {
			fmt.Println("Failed to list loggers:", err)
			return
		}

		if len(loggers) == 0 {
			fmt.Println("No loggers found.")
			return
		} else {
			fmt.Printf("Loggers for site %d:\n", siteID)
		}

		for _, l := range loggers {
			fmt.Printf(
				" %d. %s (model: %s, serial: %s, role: %s)\n",
				l.ID,
				l.Name,
				l.Model,
				l.Serial,
				l.Role,
			)
		}
	},
}

func init() {
	loggerCmd.AddCommand(loggerListCmd)

	loggerListCmd.Flags().StringP("site", "s", "", "Site ID or name")
	loggerListCmd.MarkFlagRequired("site")
}
