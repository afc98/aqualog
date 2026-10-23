package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/site"

	"github.com/spf13/cobra"
)

var siteUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a site",
	Run: func(cmd *cobra.Command, args []string) {
		ident, _ := cmd.Flags().GetString("site")
		id, err := db.ResolveSiteIdentifier(ident)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		name, _ := cmd.Flags().GetString("name")
		desc, _ := cmd.Flags().GetString("description")
		var latPtr, lonPtr *float64
		if cmd.Flags().Changed("lat") {
			lat, _ := cmd.Flags().GetFloat64("lat")
			latPtr = &lat
		}
		if cmd.Flags().Changed("lon") {
			lon, _ := cmd.Flags().GetFloat64("lon")
			lonPtr = &lon
		}
		if name == "" && desc == "" && latPtr == nil && lonPtr == nil {
			fmt.Println("Nothing to update.")
			return
		}
		if err := site.Update(id, name, latPtr, lonPtr, desc); err != nil {
			fmt.Println("Failed to update site:", err)
			return
		}
		fmt.Println("Site updated.")
	},
}

func init() {
	siteCmd.AddCommand(siteUpdateCmd)
	siteUpdateCmd.Flags().StringP("site", "s", "", "Site ID or name")
	siteUpdateCmd.Flags().StringP("name", "n", "", "New site name")
	siteUpdateCmd.Flags().Float64P("lat", "y", 0, "New latitude")
	siteUpdateCmd.Flags().Float64P("lon", "x", 0, "New longitude")
	siteUpdateCmd.Flags().StringP("description", "d", "", "New site description")
	siteUpdateCmd.MarkFlagRequired("site")
}
