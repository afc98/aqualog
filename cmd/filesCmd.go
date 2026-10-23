package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/files"
	"time"

	"github.com/spf13/cobra"
)

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "Manage imported logger files",
}

var filesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List imported logger files for a site",
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		loggerID := 0
		loggerIdent, _ := cmd.Flags().GetString("logger")
		if loggerIdent != "" {
			loggerID, err = db.ResolveLoggerIdentifier(loggerIdent)
			if err != nil {
				fmt.Println("Logger not found:", err)
				return
			}
		}
		rows, err := files.List(siteID, loggerID)
		if err != nil {
			fmt.Println("Failed to list files:", err)
			return
		}
		if len(rows) == 0 {
			fmt.Println("No imported files found.")
			return
		}
		for _, f := range rows {
			fmt.Printf("%d. logger=%s (%d), type=%s, kind=%s, unit=%s, rows=%d, imported=%s, file=%s\n",
				f.ID, f.Logger, f.LoggerID, f.Type, f.Kind, f.Unit, f.Rows, f.Timestamp.Format(time.RFC3339), f.Path)
		}
	},
}

var filesRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove an imported logger file and its rows",
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetInt("id")
		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			fmt.Println("Refusing to remove imported data without --confirm.")
			return
		}
		siteID, rows, err := files.Remove(id)
		if err != nil {
			fmt.Println("Failed to remove file:", err)
			return
		}
		fmt.Printf("Removed imported file %d and %d raw rows. Corrected data invalidated for site %d.\n", id, rows, siteID)
	},
}

func init() {
	rootCmd.AddCommand(filesCmd)
	filesCmd.AddCommand(filesListCmd)
	filesCmd.AddCommand(filesRemoveCmd)

	filesListCmd.Flags().StringP("site", "s", "", "Site ID or name")
	filesListCmd.Flags().StringP("logger", "l", "", "Logger ID or name")
	filesListCmd.MarkFlagRequired("site")

	filesRemoveCmd.Flags().Int("id", 0, "Imported file ID")
	filesRemoveCmd.Flags().Bool("confirm", false, "Confirm removal of imported data")
	filesRemoveCmd.MarkFlagRequired("id")
}
