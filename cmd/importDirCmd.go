package cmd

import (
	"aqualog/core/db"
	"aqualog/core/importer"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
)

var importDirCmd = &cobra.Command{
	Use:   "import-dir",
	Short: "Import all matching logger files from a directory",
	Run: func(cmd *cobra.Command, args []string) {
		fileType, _ := cmd.Flags().GetString("type")
		dir, _ := cmd.Flags().GetString("dir")
		pattern, _ := cmd.Flags().GetString("pattern")
		replaceExisting, _ := cmd.Flags().GetBool("replace")
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		loggerID := -1
		loggerIdent, _ := cmd.Flags().GetString("logger")
		if loggerIdent != "" {
			loggerID, err = db.ResolveLoggerIdentifier(loggerIdent)
			if err != nil {
				fmt.Println("Logger not found:", err)
				return
			}
		}
		matches, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			fmt.Println("Invalid pattern:", err)
			return
		}
		if len(matches) == 0 {
			fmt.Println("No files matched.")
			return
		}
		totalInserted, totalSkipped, totalReplaced, failed := 0, 0, 0, 0
		for _, filePath := range matches {
			meta, recs, err := importer.ParseLoggerFile(fileType, filePath)
			if err != nil {
				failed++
				fmt.Printf("FAILED %s: %v\n", filePath, err)
				continue
			}
			result, err := importer.ImportRecords(meta, recs, siteID, loggerID, filePath, fileType, importer.ImportOptions{ReplaceExisting: replaceExisting})
			if err != nil {
				failed++
				fmt.Printf("FAILED %s: %v\n", filePath, err)
				continue
			}
			totalInserted += result.Inserted
			totalSkipped += result.Skipped
			totalReplaced += result.Replaced
			fmt.Printf("OK %s: inserted=%d skipped=%d replaced=%d logger=%d\n", filePath, result.Inserted, result.Skipped, result.Replaced, result.LoggerID)
		}
		fmt.Printf("Summary: files=%d failed=%d inserted=%d skipped=%d replaced=%d\n", len(matches), failed, totalInserted, totalSkipped, totalReplaced)
	},
}

func init() {
	rootCmd.AddCommand(importDirCmd)
	importDirCmd.Flags().StringP("dir", "d", "", "Directory containing logger files")
	importDirCmd.Flags().StringP("type", "t", "", "File type (solinst, aquaread or insitu)")
	importDirCmd.Flags().StringP("site", "s", "", "Site ID or name")
	importDirCmd.Flags().StringP("logger", "l", "", "Logger ID or name")
	importDirCmd.Flags().String("pattern", "*", "File glob pattern within the directory")
	importDirCmd.Flags().Bool("replace", false, "Replace existing records for matching logger timestamps")
	importDirCmd.MarkFlagRequired("dir")
	importDirCmd.MarkFlagRequired("type")
	importDirCmd.MarkFlagRequired("site")
}
