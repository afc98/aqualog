package cmd

import (
	"aqualog/core/db"
	"aqualog/core/site"
	"fmt"

	"github.com/spf13/cobra"
)

var siteRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a site by ID or name",
	Run: func(cmd *cobra.Command, args []string) {
		ident, _ := cmd.Flags().GetString("site")
		id, err := db.ResolveSiteIdentifier(ident)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			fmt.Println("Refusing to remove site without --confirm.")
			return
		}
		if err := site.Remove(id); err != nil {
			fmt.Println("Failed to remove site:", err)
			return
		}
		fmt.Println("Site removed.")
	},
}

func init() {
	siteCmd.AddCommand(siteRemoveCmd)
	siteRemoveCmd.Flags().StringP("site", "s", "", "Site ID or name")
	siteRemoveCmd.Flags().Bool("confirm", false, "Confirm removal")
	siteRemoveCmd.MarkFlagRequired("site")
}
