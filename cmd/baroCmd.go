package cmd

import (
	"errors"
	"fmt"
	"github.com/afc98/aqualog/core/baro"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/importer"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var baroCmd = &cobra.Command{
	Use:   "baro",
	Short: "Import, assign, and apply barometric correction",
}

var baroImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a barometric logger file",
	Run: func(cmd *cobra.Command, args []string) {
		fileType, _ := cmd.Flags().GetString("type")
		filePath, _ := cmd.Flags().GetString("file")
		siteIdent, _ := cmd.Flags().GetString("site")
		loggerIdent, _ := cmd.Flags().GetString("logger")
		replaceExisting, _ := cmd.Flags().GetBool("replace")
		dateOrder, _ := cmd.Flags().GetString("date-order")

		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		loggerID := -1
		if loggerIdent != "" {
			loggerID, err = db.ResolveLoggerIdentifier(loggerIdent)
			if err != nil {
				fmt.Println("Logger not found:", err)
				return
			}
		}
		options := importer.ImportOptions{
			ReplaceExisting: replaceExisting,
			Role:            "barometric",
			DateOrder:       dateOrder,
		}
		meta, recs, err := importer.ParseLoggerFileWithOptions(fileType, filePath, options)
		if err != nil {
			fmt.Println("Failed to parse barometric file:", err)
			return
		}
		kind := ""
		if len(recs) > 0 {
			kind = recs[0].MeasurementKind
		}
		if kind != "barometric_pressure_mbar" && kind != "barometric_pressure_kpa" {
			fmt.Printf("Warning: parsed measurement kind is %q; importing as barometric because baro import was requested.\n", kind)
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
		fmt.Printf("Imported %d barometric records (%d skipped, %d replaced). Logger ID: %d\n", result.Inserted, result.Skipped, result.Replaced, result.LoggerID)
	},
}

var baroImportDirCmd = &cobra.Command{
	Use:   "import-dir",
	Short: "Import matching barometric logger files from a directory",
	Run: func(cmd *cobra.Command, args []string) {
		fileType, _ := cmd.Flags().GetString("type")
		dir, _ := cmd.Flags().GetString("dir")
		pattern, _ := cmd.Flags().GetString("pattern")
		siteIdent, _ := cmd.Flags().GetString("site")
		loggerIdent, _ := cmd.Flags().GetString("logger")
		replaceExisting, _ := cmd.Flags().GetBool("replace")
		dateOrder, _ := cmd.Flags().GetString("date-order")

		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		loggerID := -1
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

		options := importer.ImportOptions{
			ReplaceExisting: replaceExisting,
			Role:            "barometric",
			DateOrder:       dateOrder,
		}
		totalInserted, totalSkipped, totalReplaced, failed := 0, 0, 0, 0
		for _, filePath := range matches {
			meta, recs, err := importer.ParseLoggerFileWithOptions(fileType, filePath, options)
			if err != nil {
				failed++
				fmt.Printf("FAILED %s: %v\n", filePath, err)
				continue
			}
			kind := ""
			if len(recs) > 0 {
				kind = recs[0].MeasurementKind
			}
			if kind != "barometric_pressure_mbar" && kind != "barometric_pressure_kpa" {
				fmt.Printf("Warning %s: parsed measurement kind is %q; importing as barometric because baro import-dir was requested.\n", filePath, kind)
			}
			result, err := importer.ImportRecords(meta, recs, siteID, loggerID, filePath, fileType, options)
			if err != nil {
				if errors.Is(err, importer.ErrFileAlreadyRecorded) {
					totalSkipped += len(recs)
					fmt.Printf("SKIP %s: file already recorded for logger\n", filePath)
					continue
				}
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

var baroAssignCmd = &cobra.Command{
	Use:   "assign",
	Short: "Assign a default barologger to a site or project",
	Run: func(cmd *cobra.Command, args []string) {
		baroIdent, _ := cmd.Flags().GetString("barologger")
		barologgerID, err := db.ResolveLoggerIdentifier(baroIdent)
		if err != nil {
			fmt.Println("Barologger not found:", err)
			return
		}
		siteIdent, _ := cmd.Flags().GetString("site")
		projectIdent, _ := cmd.Flags().GetString("project")
		switch {
		case siteIdent != "":
			siteID, err := db.ResolveSiteIdentifier(siteIdent)
			if err != nil {
				fmt.Println("Site not found:", err)
				return
			}
			err = baro.Assign("site", siteID, barologgerID)
		case projectIdent != "":
			projectID, err := db.ResolveProjectIdentifier(projectIdent)
			if err != nil {
				fmt.Println("Project not found:", err)
				return
			}
			err = baro.Assign("project", projectID, barologgerID)
		default:
			fmt.Println("Specify --site or --project.")
			return
		}
		if err != nil {
			fmt.Println("Assignment failed:", err)
			return
		}
		fmt.Println("Barologger assigned.")
	},
}

var baroCorrectCmd = &cobra.Command{
	Use:   "correct",
	Short: "Apply barometric correction to a site or logger",
	Run: func(cmd *cobra.Command, args []string) {
		opts, ok := baroOptionsFromFlags(cmd)
		if !ok {
			return
		}
		result, err := baro.Correct(opts)
		if err != nil {
			fmt.Println("Correction failed:", err)
			return
		}
		fmt.Printf("Correction run %d complete: updated=%d skipped=%d\n", result.RunID, result.Updated, result.Skipped)
	},
}

var baroCorrectBatchCmd = &cobra.Command{
	Use:   "correct-batch",
	Short: "Apply barometric correction to all sites in a project",
	Run: func(cmd *cobra.Command, args []string) {
		projectIdent, _ := cmd.Flags().GetString("project")
		projectID, err := db.ResolveProjectIdentifier(projectIdent)
		if err != nil {
			fmt.Println("Project not found:", err)
			return
		}
		conn, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer conn.Close()
		rows, err := conn.Query(`SELECT id, name FROM sites WHERE project_id = ? ORDER BY id`, projectID)
		if err != nil {
			fmt.Println("Failed to list sites:", err)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var siteID int
			var name string
			if err := rows.Scan(&siteID, &name); err != nil {
				fmt.Println("Failed to read site:", err)
				continue
			}
			opts, ok := baroOptionsFromFlags(cmd)
			if !ok {
				return
			}
			opts.SiteID = siteID
			result, err := baro.Correct(opts)
			if err != nil {
				fmt.Printf("%s: failed: %v\n", name, err)
				continue
			}
			fmt.Printf("%s: run=%d updated=%d skipped=%d\n", name, result.RunID, result.Updated, result.Skipped)
		}
	},
}

var baroStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show barometric assignments and recent correction runs",
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		status, err := baro.SiteStatus(siteID)
		if err != nil {
			fmt.Println("Status failed:", err)
			return
		}
		if len(status.Assignments) == 0 {
			fmt.Println("No barologger assignment found.")
		}
		for _, a := range status.Assignments {
			fmt.Printf("Assignment: %s %d -> %s (ID %d)\n", a.Scope, a.ScopeID, a.Barologger, a.BarologgerID)
		}
		if len(status.Runs) == 0 {
			fmt.Println("No correction runs found.")
		}
		for _, r := range status.Runs {
			fmt.Printf("Run %d: site=%d logger=%v barologger=%d density=%s %.1f max_gap=%ds created=%s\n", r.ID, r.SiteID, r.LoggerID, r.BarologgerID, r.DensityMode, r.Density, r.MaxGap, r.CreatedAt)
		}
	},
}

func init() {
	rootCmd.AddCommand(baroCmd)
	baroCmd.AddCommand(baroImportCmd, baroImportDirCmd, baroAssignCmd, baroCorrectCmd, baroCorrectBatchCmd, baroStatusCmd)

	baroImportCmd.Flags().StringP("file", "f", "", "Barometric logger file")
	baroImportCmd.Flags().StringP("type", "t", "", "File type (solinst, aquaread or insitu)")
	baroImportCmd.Flags().StringP("site", "s", "", "Site ID or name")
	baroImportCmd.Flags().StringP("logger", "l", "", "Logger ID or name")
	baroImportCmd.Flags().String("date-order", "auto", "Slash date order for imports: auto, dmy, or mdy")
	baroImportCmd.Flags().Bool("replace", false, "Replace existing records for matching timestamps")
	baroImportCmd.MarkFlagRequired("file")
	baroImportCmd.MarkFlagRequired("type")
	baroImportCmd.MarkFlagRequired("site")

	baroImportDirCmd.Flags().StringP("dir", "d", "", "Directory containing barometric logger files")
	baroImportDirCmd.Flags().StringP("type", "t", "", "File type (solinst, aquaread or insitu)")
	baroImportDirCmd.Flags().StringP("site", "s", "", "Site ID or name")
	baroImportDirCmd.Flags().StringP("logger", "l", "", "Logger ID or name")
	baroImportDirCmd.Flags().String("date-order", "auto", "Slash date order for imports: auto, dmy, or mdy")
	baroImportDirCmd.Flags().String("pattern", "*", "File glob pattern within the directory")
	baroImportDirCmd.Flags().Bool("replace", false, "Replace existing records for matching timestamps")
	baroImportDirCmd.MarkFlagRequired("dir")
	baroImportDirCmd.MarkFlagRequired("type")
	baroImportDirCmd.MarkFlagRequired("site")

	baroAssignCmd.Flags().StringP("site", "s", "", "Site ID or name")
	baroAssignCmd.Flags().StringP("project", "p", "", "Project ID or name")
	baroAssignCmd.Flags().StringP("barologger", "b", "", "Barometric logger ID or name")
	baroAssignCmd.MarkFlagRequired("barologger")

	addCorrectionFlags(baroCorrectCmd)
	baroCorrectCmd.Flags().StringP("site", "s", "", "Site ID or name")
	baroCorrectCmd.Flags().StringP("logger", "l", "", "Water logger ID or name")
	baroCorrectCmd.MarkFlagRequired("site")

	addCorrectionFlags(baroCorrectBatchCmd)
	baroCorrectBatchCmd.Flags().StringP("project", "p", "", "Project ID or name")
	baroCorrectBatchCmd.MarkFlagRequired("project")

	baroStatusCmd.Flags().StringP("site", "s", "", "Site ID or name")
	baroStatusCmd.MarkFlagRequired("site")
}

func addCorrectionFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("barologger", "b", "", "Barometric logger ID or name")
	cmd.Flags().String("density", "freshwater", "Density mode: freshwater, saline, fixed, or none")
	cmd.Flags().Float64("rho", 0, "Fixed density kg/m3 when --density fixed")
	cmd.Flags().String("max-gap", "", "Maximum barometric match gap, e.g. 10m or 1h; defaults to twice the water logger interval")
	cmd.Flags().String("notes", "", "Correction run notes")
}

func baroOptionsFromFlags(cmd *cobra.Command) (baro.CorrectOptions, bool) {
	var opts baro.CorrectOptions
	var err error
	siteIdent, _ := cmd.Flags().GetString("site")
	if siteIdent != "" {
		opts.SiteID, err = db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return opts, false
		}
	}
	loggerIdent, _ := cmd.Flags().GetString("logger")
	if loggerIdent != "" {
		opts.LoggerID, err = db.ResolveLoggerIdentifier(loggerIdent)
		if err != nil {
			fmt.Println("Logger not found:", err)
			return opts, false
		}
	}
	baroIdent, _ := cmd.Flags().GetString("barologger")
	if baroIdent != "" {
		opts.BarologgerID, err = db.ResolveLoggerIdentifier(baroIdent)
		if err != nil {
			fmt.Println("Barologger not found:", err)
			return opts, false
		}
	}
	opts.DensityMode, _ = cmd.Flags().GetString("density")
	opts.Density, _ = cmd.Flags().GetFloat64("rho")
	if opts.DensityMode == "fixed" && opts.Density == 0 {
		fmt.Println("--rho is required when --density fixed")
		return opts, false
	}
	maxGap, _ := cmd.Flags().GetString("max-gap")
	if maxGap != "" {
		opts.MaxGap, err = time.ParseDuration(maxGap)
		if err != nil {
			fmt.Println("Invalid max gap:", err)
			return opts, false
		}
	}
	opts.Notes, _ = cmd.Flags().GetString("notes")
	return opts, true
}
