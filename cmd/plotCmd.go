package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"aqualog/core/db"
	aqlplot "aqualog/core/plot"

	"github.com/spf13/cobra"
)

var (
	plotSiteIdent string
	plotSiteID    int
	plotSeries    []string
	plotStart     string
	plotEnd       string
)

// plotCmd represents the "aqualog plot" command.
var plotCmd = &cobra.Command{
	Use:   "plot",
	Short: "Plot logger data for a site",
	Long: `Generate an interactive plot of one or more logger data series.
Example:
    aqualog plot --site 3 --series raw_level --series temp --start 2025-01-01 --end 2025-02-01`,
	RunE: func(cmd *cobra.Command, args []string) error {

		if plotSiteIdent == "" {
			return fmt.Errorf("--site is required")
		}

		plotSiteID, err := db.ResolveSiteIdentifier(plotSiteIdent)
		if err != nil {
			return fmt.Errorf("site not found: %w", err)
		}

		// Open database
		db, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return err
		}
		defer db.Close()

		// Parse optional time bounds
		var tStart, tEnd *time.Time
		if plotStart != "" {
			ts, err := time.Parse("2006-01-02", plotStart)
			if err != nil {
				return fmt.Errorf("invalid --start format, expected YYYY-MM-DD: %w", err)
			}
			tStart = &ts
		}
		if plotEnd != "" {
			te, err := time.Parse("2006-01-02", plotEnd)
			if err != nil {
				return fmt.Errorf("invalid --end format, expected YYYY-MM-DD: %w", err)
			}
			tEnd = &te
		}

		if len(plotSeries) == 0 {
			return fmt.Errorf("at least one --series name is required")
		}

		var allSeries []aqlplot.SeriesData

		for _, name := range plotSeries {
			s, err := loadSeriesFromDB(db, plotSiteID, name, tStart, tEnd)
			if err != nil {
				return fmt.Errorf("loading series %s: %w", name, err)
			}
			allSeries = append(allSeries, s)
		}

		// Call the plotting engine
		if err := aqlplot.PlotInteractive(allSeries); err != nil {
			return fmt.Errorf("plotting error: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(plotCmd)

	plotCmd.Flags().StringVar(&plotSiteIdent, "site", "", "Site ID or name")
	plotCmd.Flags().StringArrayVar(&plotSeries, "series", []string{}, "Series to plot (repeatable)")
	plotCmd.Flags().StringVar(&plotStart, "start", "", "Start date (YYYY-MM-DD)")
	plotCmd.Flags().StringVar(&plotEnd, "end", "", "End date (YYYY-MM-DD)")
}

func loadSeriesFromDB(db *sql.DB, siteID int, name string, start, end *time.Time) (aqlplot.SeriesData, error) {

	// TODO: Replace with your actual tables & schema.

	// map series name to a safe column name
	var columnName string
	switch name {
	case "level":
		columnName = "corrected_value"
	case "temp":
		columnName = "temp_c"
	case "ec":
		columnName = "ec_us"
	case "salinity":
		columnName = "sal_psu"
	default:
		return aqlplot.SeriesData{}, fmt.Errorf("unsupported series name %s. supported names are: level, temp, ec, salinity", name)
	}

	// default time bounds
	defaultStart, err := time.Parse("2006-01-02", "1900-01-01")
	if err != nil {
		return aqlplot.SeriesData{}, fmt.Errorf("unable to parse default start time: %w", err)
	}
	startTime := defaultStart.Format(time.RFC3339)

	defaultEnd, err := time.Parse("2006-01-02", "2200-01-01")
	if err != nil {
		return aqlplot.SeriesData{}, fmt.Errorf("unable to parse default end time: %w", err)
	}
	endTime := defaultEnd.Format(time.RFC3339)

	if start != nil {
		startTime = start.Format(time.RFC3339)
	}
	if end != nil {
		endTime = end.Format(time.RFC3339)
	}

	// Build query string with the validated column identifier inserted directly.
	queryStr := fmt.Sprintf(`
        SELECT timestamp, %s
        FROM corrected_data
        WHERE site_id=?
          AND timestamp >= ?
          AND timestamp <= ?
    `, columnName)

	rows, err := db.Query(queryStr, siteID, startTime, endTime)
	if err != nil {
		return aqlplot.SeriesData{}, err
	}
	defer rows.Close()

	var (
		timestamps []time.Time
		values     []float64
	)

	for rows.Next() {
		var ts time.Time
		var v float64
		if err := rows.Scan(&ts, &v); err != nil {
			return aqlplot.SeriesData{}, err
		}
		timestamps = append(timestamps, ts)
		values = append(values, v)
	}

	if err := rows.Err(); err != nil {
		return aqlplot.SeriesData{}, err
	}

	log.Printf("Loaded %d points for series %s", len(timestamps), name)

	return aqlplot.SeriesData{
		Name: name,
		X:    timestamps,
		Y:    values,
	}, nil
}
