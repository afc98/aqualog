package cmd

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/query"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query logger or corrected data for a site.",
	Run: func(cmd *cobra.Command, args []string) {
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}
		typeFlag, _ := cmd.Flags().GetString("type")
		format, _ := cmd.Flags().GetString("format")
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		limit, _ := cmd.Flags().GetInt("limit")
		loggerID := 0
		loggerIdent, _ := cmd.Flags().GetString("logger")
		if loggerIdent != "" {
			loggerID, err = db.ResolveLoggerIdentifier(loggerIdent)
			if err != nil {
				fmt.Println("Logger not found:", err)
				return
			}
		}
		opts := query.Options{From: from, To: to, LoggerID: loggerID, Limit: limit}

		switch typeFlag {
		case "raw":
			result, err := query.QueryLoggerDataWithOptions(siteID, opts)
			if err != nil {
				fmt.Printf("Failed to query logger data for site %d: %v\n", siteID, err)
				return
			}
			printRawQuery(result, format)
		case "corrected":
			result, err := query.QueryCorrectedDataWithOptions(siteID, opts)
			if err != nil {
				fmt.Printf("Failed to query corrected data for site %d: %v\n", siteID, err)
				return
			}
			printCorrectedQuery(result, format)
		default:
			fmt.Println("Invalid type specified. Use 'raw' or 'corrected'.")
			return
		}
		if err != nil {
			fmt.Printf("Failed to query logger data for site %d: %v\n", siteID, err)
			return
		}
		fmt.Println("Successfully queried logger data for site", siteID)
	},
}

func init() {
	rootCmd.AddCommand(queryCmd)

	queryCmd.Flags().StringP("site", "s", "", "Site ID or name")
	queryCmd.Flags().StringP("type", "t", "", "Type of data to query (raw or corrected)")
	queryCmd.Flags().String("from", "", "Start timestamp filter")
	queryCmd.Flags().String("to", "", "End timestamp filter")
	queryCmd.Flags().StringP("logger", "l", "", "Logger ID or name for raw data")
	queryCmd.Flags().Int("limit", 0, "Maximum rows to return")
	queryCmd.Flags().String("format", "table", "Output format: table, csv, or json")

	queryCmd.MarkFlagRequired("site")
	queryCmd.MarkFlagRequired("type")
}

func printRawQuery(rows []query.LoggerDataRow, format string) {
	switch format {
	case "json":
		_ = json.NewEncoder(os.Stdout).Encode(rows)
	case "csv":
		w := csv.NewWriter(os.Stdout)
		defer w.Flush()
		w.Write([]string{"id", "logger_id", "timestamp", "level_m", "imported_value", "imported_unit", "imported_kind", "baro_corrected_m", "correction_run_id", "temp_c", "sal_psu", "ec_us"})
		for _, row := range rows {
			w.Write([]string{strconv.Itoa(row.ID), strconv.Itoa(row.LoggerID), row.Timestamp.Format("2006-01-02 15:04:05"), fmt.Sprintf("%.4f", row.LevelM), nullFloat(row.ImportedValue), nullString(row.ImportedUnit), nullString(row.ImportedKind), nullFloat(row.BaroCorrectedM), nullInt(row.CorrectionRun), query.NullFloattoString(row.TempC), query.NullFloattoString(row.SalPSU), query.NullFloattoString(row.ECUS)})
		}
	default:
		fmt.Println("Logger Data:")
		for _, row := range rows {
			fmt.Printf("ID: %d, LoggerID: %d, Timestamp: %s, Level(m): %.2f, Imported: %s %s (%s), BaroCorrected(m): %s, Run: %s, Temp(C): %s, Sal(PSU): %s, EC(uS): %s\n",
				row.ID, row.LoggerID, row.Timestamp.Format("2006-01-02 15:04:05"), row.LevelM,
				nullFloat(row.ImportedValue),
				nullString(row.ImportedUnit),
				nullString(row.ImportedKind),
				nullFloat(row.BaroCorrectedM),
				nullInt(row.CorrectionRun),
				query.NullFloattoString(row.TempC),
				query.NullFloattoString(row.SalPSU),
				query.NullFloattoString(row.ECUS))
		}
	}
}

func nullFloat(n sql.NullFloat64) string {
	if n.Valid {
		return fmt.Sprintf("%.4f", n.Float64)
	}
	return ""
}

func nullString(n sql.NullString) string {
	if n.Valid {
		return n.String
	}
	return ""
}

func nullInt(n sql.NullInt64) string {
	if n.Valid {
		return strconv.FormatInt(n.Int64, 10)
	}
	return ""
}

func printCorrectedQuery(rows []query.CorrectedDataRow, format string) {
	switch format {
	case "json":
		_ = json.NewEncoder(os.Stdout).Encode(rows)
	case "csv":
		w := csv.NewWriter(os.Stdout)
		defer w.Flush()
		w.Write([]string{"id", "site_id", "timestamp", "corrected_value", "temp_c", "sal_psu", "ec_us"})
		for _, row := range rows {
			w.Write([]string{strconv.Itoa(row.ID), strconv.Itoa(row.SiteID), row.Timestamp.Format("2006-01-02 15:04:05"), fmt.Sprintf("%.4f", row.CorrectedValue), query.NullFloattoString(row.TempC), query.NullFloattoString(row.SalPSU), query.NullFloattoString(row.ECUS)})
		}
	default:
		fmt.Println("Corrected Data:")
		for _, row := range rows {
			fmt.Printf("ID: %d, SiteID: %d, Timestamp: %s, Corrected Value: %.2f, Temp(C): %s, Sal(PSU): %s, EC(uS): %s\n",
				row.ID, row.SiteID, row.Timestamp.Format("2006-01-02 15:04:05"), row.CorrectedValue,
				query.NullFloattoString(row.TempC),
				query.NullFloattoString(row.SalPSU),
				query.NullFloattoString(row.ECUS))
		}
	}
}
