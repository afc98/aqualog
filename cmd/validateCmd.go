package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/qa"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate whether a site is ready for processing",
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		issues, err := qa.ValidateSite(siteID)
		if err != nil {
			fmt.Println("Validation failed:", err)
			return
		}
		if len(issues) == 0 {
			fmt.Println("No validation issues found.")
			return
		}
		for _, issue := range issues {
			fmt.Printf("[%s] %s\n", issue.Severity, issue.Message)
		}
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringP("site", "s", "", "Site ID or name")
	validateCmd.MarkFlagRequired("site")
}
