package cmd

import (
	"aqualog/core/db"
	"aqualog/core/importer"
	"errors"
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
		replaceExisting, _ := cmd.Flags().GetBool("replace")
		role, _ := cmd.Flags().GetString("role")
		measurementKind, _ := cmd.Flags().GetString("measurement-kind")
		dateOrder, _ := cmd.Flags().GetString("date-order")
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		loggerIdent, _ := cmd.Flags().GetString("logger")
		loggerID, err := db.ResolveLoggerIdentifier(loggerIdent)
		if err != nil {
			loggerID = -1
		}

		options := importer.ImportOptions{
			ReplaceExisting: replaceExisting,
			Role:            role,
			MeasurementKind: measurementKind,
			DateOrder:       dateOrder,
		}
		meta, recs, err := importer.ParseLoggerFileWithOptions(fileType, filePath, options)
		if err != nil {
			fmt.Println("Failed to parse logger file:", err)
			return
		}

		result, err := importer.ImportRecords(meta, recs, siteID, loggerID, filePath, fileType, options)
		if err != nil {
			if errors.Is(err, importer.ErrFileAlreadyRecorded) {
				fmt.Println("Import skipped: file already recorded for logger")
				return
			}
			fmt.Println("Import failed:", err)
			return
		}

		fmt.Printf(
			"Imported %d records (%d skipped, %d replaced). Logger ID: %d\n",
			result.Inserted,
			result.Skipped,
			result.Replaced,
			result.LoggerID,
		)
	},
}

func init() {
	loggerImportCmd.Flags().StringP("file", "f", "", "Logger file")
	loggerImportCmd.Flags().StringP("type", "t", "", "File type (solinst, aquaread or insitu)")
	loggerImportCmd.Flags().StringP("site", "s", "", "Site ID")
	loggerImportCmd.Flags().StringP("logger", "l", "", "Logger ID")
	loggerImportCmd.Flags().String("role", "", "Logger role (water_level or barometric); inferred when omitted")
	loggerImportCmd.Flags().String("measurement-kind", "", "Override parsed measurement kind")
	loggerImportCmd.Flags().String("date-order", "auto", "Slash date order for imports: auto, dmy, or mdy")
	loggerImportCmd.Flags().Bool("replace", false, "Replace existing records for matching logger timestamps")

	loggerImportCmd.MarkFlagRequired(("file"))
	loggerImportCmd.MarkFlagRequired(("type"))
	loggerImportCmd.MarkFlagRequired(("site"))

	rootCmd.AddCommand(loggerImportCmd)
}
