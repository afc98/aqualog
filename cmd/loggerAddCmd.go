/*
Copyright © 2025 Arran Clarke
*/
package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/logger"

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
		role, _ := cmd.Flags().GetString("role")

		lgr, err := logger.Add(logger.AddParams{
			SiteID: siteID,
			Name:   name,
			Model:  model,
			Serial: serial,
			Role:   role,
		})
		if err != nil {
			fmt.Println("Failed to add logger:", err)
			return
		}

		fmt.Printf(
			"Logger added: ID %d (site %d, %s, %s, %s, role: %s)\n",
			lgr.ID,
			lgr.SiteID,
			lgr.Name,
			lgr.Model,
			lgr.Serial,
			lgr.Role,
		)
	},
}

func init() {
	loggerCmd.AddCommand(loggerAddCmd)

	loggerAddCmd.Flags().StringP("site", "s", "", "Site ID or name")
	loggerAddCmd.Flags().StringP("name", "n", "", "Logger name")
	loggerAddCmd.Flags().StringP("model", "m", "", "Logger model")
	loggerAddCmd.Flags().StringP("serial", "S", "", "Serial number")
	loggerAddCmd.Flags().String("role", "water_level", "Logger role (water_level or barometric)")

	loggerAddCmd.MarkFlagRequired("site")
	loggerAddCmd.MarkFlagRequired("name")
}
