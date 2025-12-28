package cmd

import (
	"aqualog/core/db"
	"aqualog/core/importer"
	"fmt"

	"github.com/spf13/cobra"
)

var loggerImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a logger file to a site",
	Run: func(cmd *cobra.Command, args []string) {
		// Read flags
		fileType, _ := cmd.Flags().GetString("type")
		filePath, _ := cmd.Flags().GetString("file")
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		loggerIdent, _ := cmd.Flags().GetString("logger")
		loggerID, err := db.ResolveLoggerIdentifier(loggerIdent)
		if err != nil {
			fmt.Println("Logger not found:", err)
			return
		}

		err = importer.ImportLoggerFile(fileType, filePath, siteID, loggerID)
		if err != nil {
			fmt.Println("Import failed:", err)
			return
		}
	},
}

func init() {
	loggerImportCmd.Flags().StringP("file", "f", "", "Logger file")
	loggerImportCmd.Flags().StringP("type", "t", "", "File type")
	loggerImportCmd.Flags().StringP("site", "s", "", "Site ID")
	loggerImportCmd.Flags().StringP("logger", "l", "", "Logger ID")

	loggerImportCmd.MarkFlagRequired(("file"))
	loggerImportCmd.MarkFlagRequired(("type"))
	loggerImportCmd.MarkFlagRequired(("site"))

	rootCmd.AddCommand(loggerImportCmd)
}
