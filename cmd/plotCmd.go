package cmd

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/afc98/aqualog/core/db"
	aqlplot "github.com/afc98/aqualog/core/plot"

	"github.com/spf13/cobra"
)

const (
	plotTypeCorrected     = "corrected"
	plotTypeRaw           = "raw"
	plotTypeBaroCorrected = "baro"
	plotTypeUncompensated = "uncompensated"
	plotDefaultStart      = "1900-01-01T00:00:00Z"
	plotDefaultEnd        = "2200-01-01T00:00:00Z"
	standardWaterDensity  = 1000.0
	standardGravity       = 9.80665
)

var (
	plotSiteIdent string
	plotSeries    []string
	plotStart     string
	plotEnd       string
	plotType      string
	plotFormat    string
	plotOutput    string
	plotOpen      bool
	plotWidth     float64
	plotHeight    float64
)

// plotCmd represents the "aqualog plot" command.
var plotCmd = &cobra.Command{
	Use:   "plot",
	Short: "Plot logger data for a site",
	Long: `Generate a plot of one or more site data series.
Examples:
    aqualog plot --site 3 --type corrected --series level --series temp --format html --open
    aqualog plot --site 3 --type raw --series level --start 2025-01-01 --end 2025-02-01
    aqualog plot --site 3 --series uncompensated_level --output absolute-head.svg`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if plotSiteIdent == "" {
			return fmt.Errorf("--site is required")
		}

		siteID, err := db.ResolveSiteIdentifier(plotSiteIdent)
		if err != nil {
			return fmt.Errorf("site not found: %w", err)
		}

		if len(plotSeries) == 0 {
			return fmt.Errorf("at least one --series name is required")
		}

		startTime, err := parsePlotBound(plotStart, false)
		if err != nil {
			return fmt.Errorf("invalid --start value: %w", err)
		}
		endTime, err := parsePlotBound(plotEnd, true)
		if err != nil {
			return fmt.Errorf("invalid --end value: %w", err)
		}
		if !startTime.Before(endTime) {
			return fmt.Errorf("--start must be before --end")
		}

		dbConn, err := db.GetDB()
		if err != nil {
			return fmt.Errorf("database error: %w", err)
		}
		defer dbConn.Close()

		var allSeries []aqlplot.SeriesData
		for _, spec := range plotSeries {
			source, name, err := parsePlotSeriesSpec(plotType, spec)
			if err != nil {
				return err
			}
			s, err := loadSeriesFromDB(dbConn, siteID, source, name, startTime, endTime)
			if err != nil {
				return fmt.Errorf("loading series %s: %w", spec, err)
			}
			allSeries = append(allSeries, s)
		}

		output, err := aqlplot.Render(allSeries, aqlplot.Options{
			Title:  fmt.Sprintf("Aqualog Site %d", siteID),
			Format: plotFormat,
			Output: plotOutput,
			Open:   plotOpen,
			Width:  plotWidth,
			Height: plotHeight,
		})
		if err != nil {
			return fmt.Errorf("plotting error: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Wrote plot to %s\n", output)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(plotCmd)

	plotCmd.Flags().StringVar(&plotSiteIdent, "site", "", "Site ID or name")
	plotCmd.Flags().StringArrayVar(&plotSeries, "series", []string{}, "Series to plot (repeatable): level, temp, ec, salinity")
	plotCmd.Flags().StringVar(&plotType, "type", plotTypeCorrected, "Data source: corrected, raw, baro, or uncompensated")
	plotCmd.Flags().StringVar(&plotStart, "start", "", "Start date or timestamp (YYYY-MM-DD or RFC3339)")
	plotCmd.Flags().StringVar(&plotEnd, "end", "", "End date or timestamp (YYYY-MM-DD or RFC3339)")
	plotCmd.Flags().StringVar(&plotFormat, "format", "", "Output format: png, svg, pdf, jpg, tif, or html")
	plotCmd.Flags().StringVarP(&plotOutput, "output", "o", "", "Output file path")
	plotCmd.Flags().BoolVar(&plotOpen, "open", true, "Open the generated plot with the system viewer")
	plotCmd.Flags().Float64Var(&plotWidth, "width", 10, "Plot width in inches")
	plotCmd.Flags().Float64Var(&plotHeight, "height", 5, "Plot height in inches")
}

func parsePlotBound(value string, endOfDay bool) (time.Time, error) {
	if value == "" {
		if endOfDay {
			return time.Parse(time.RFC3339, plotDefaultEnd)
		}
		return time.Parse(time.RFC3339, plotDefaultStart)
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	t, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected YYYY-MM-DD or RFC3339: %w", err)
	}
	if endOfDay {
		return t.AddDate(0, 0, 1).Add(-time.Nanosecond), nil
	}
	return t, nil
}

func parsePlotSeriesSpec(defaultSource, spec string) (string, string, error) {
	source := strings.ToLower(strings.TrimSpace(defaultSource))
	name := strings.ToLower(strings.TrimSpace(spec))
	if source == "" {
		source = plotTypeCorrected
	}

	if strings.Contains(name, ":") {
		parts := strings.SplitN(name, ":", 2)
		source = parts[0]
		name = parts[1]
	}

	for _, prefix := range []string{"baro_corrected", "uncompensated", "uncorrected", "corrected", "raw", "baro"} {
		token := prefix + "_"
		if strings.HasPrefix(name, token) {
			source = prefix
			name = strings.TrimPrefix(name, token)
			break
		}
	}

	source = normalizePlotSource(source)
	name = normalizePlotSeriesName(name)
	if source == "" {
		return "", "", fmt.Errorf("unsupported plot type %q; use corrected, raw, baro, or uncompensated", defaultSource)
	}
	if name == "" {
		return "", "", fmt.Errorf("unsupported series %q; supported names are: level, temp, ec, salinity", spec)
	}
	return source, name, nil
}

func normalizePlotSource(source string) string {
	switch source {
	case plotTypeCorrected:
		return plotTypeCorrected
	case plotTypeRaw, "uncorrected":
		return plotTypeRaw
	case plotTypeBaroCorrected, "baro_corrected", "barometric":
		return plotTypeBaroCorrected
	case plotTypeUncompensated, "absolute":
		return plotTypeUncompensated
	default:
		return ""
	}
}

func normalizePlotSeriesName(name string) string {
	switch name {
	case "level", "water_level", "corrected_value", "raw_level", "baro_level", "uncompensated_level":
		return "level"
	case "temp", "temperature", "temperature_c", "temp_c":
		return "temp"
	case "ec", "conductivity", "ec_us":
		return "ec"
	case "sal", "salinity", "sal_psu":
		return "salinity"
	default:
		return ""
	}
}

func loadSeriesFromDB(dbConn *sql.DB, siteID int, source, name string, start, end time.Time) (aqlplot.SeriesData, error) {
	switch source {
	case plotTypeCorrected:
		return loadCorrectedPlotSeries(dbConn, siteID, name, start, end)
	case plotTypeRaw, plotTypeBaroCorrected, plotTypeUncompensated:
		return loadLoggerPlotSeries(dbConn, siteID, source, name, start, end)
	default:
		return aqlplot.SeriesData{}, fmt.Errorf("unsupported plot type %s", source)
	}
}

func loadCorrectedPlotSeries(dbConn *sql.DB, siteID int, name string, start, end time.Time) (aqlplot.SeriesData, error) {
	columnName, unit, err := correctedColumn(name)
	if err != nil {
		return aqlplot.SeriesData{}, err
	}
	rows, err := dbConn.Query(fmt.Sprintf(`
		SELECT timestamp, %s
		FROM corrected_data
		WHERE site_id = ?
		  AND timestamp >= ?
		  AND timestamp <= ?
		ORDER BY timestamp ASC
	`, columnName), siteID, start.Format(time.RFC3339), end.Format(time.RFC3339))
	if err != nil {
		return aqlplot.SeriesData{}, err
	}
	defer rows.Close()

	return scanNullablePlotRows(rows, "corrected "+name, unit)
}

func loadLoggerPlotSeries(dbConn *sql.DB, siteID int, source, name string, start, end time.Time) (aqlplot.SeriesData, error) {
	if source == plotTypeUncompensated && name == "level" {
		return loadUncompensatedLevelSeries(dbConn, siteID, start, end)
	}

	columnName, unit, err := loggerColumn(source, name)
	if err != nil {
		return aqlplot.SeriesData{}, err
	}
	where := ""
	if source == plotTypeUncompensated {
		where = "AND d.imported_kind IN ('absolute_pressure_mbar', 'absolute_pressure_kpa', 'absolute_pressure_head_m')"
	}
	rows, err := dbConn.Query(fmt.Sprintf(`
		SELECT d.timestamp, %s
		FROM logger_data d
		JOIN loggers l ON l.id = d.logger_id
		WHERE l.site_id = ?
		  AND d.timestamp >= ?
		  AND d.timestamp <= ?
		  %s
		ORDER BY d.timestamp ASC
	`, columnName, where), siteID, start.Format(time.RFC3339), end.Format(time.RFC3339))
	if err != nil {
		return aqlplot.SeriesData{}, err
	}
	defer rows.Close()

	return scanNullablePlotRows(rows, source+" "+name, unit)
}

func loadUncompensatedLevelSeries(dbConn *sql.DB, siteID int, start, end time.Time) (aqlplot.SeriesData, error) {
	rows, err := dbConn.Query(`
		SELECT d.timestamp, d.imported_value, d.imported_kind
		FROM logger_data d
		JOIN loggers l ON l.id = d.logger_id
		WHERE l.site_id = ?
		  AND d.timestamp >= ?
		  AND d.timestamp <= ?
		  AND d.imported_kind IN ('absolute_pressure_mbar', 'absolute_pressure_kpa', 'absolute_pressure_head_m')
		ORDER BY d.timestamp ASC
	`, siteID, start.Format(time.RFC3339), end.Format(time.RFC3339))
	if err != nil {
		return aqlplot.SeriesData{}, err
	}
	defer rows.Close()

	var timestamps []time.Time
	var values []float64
	for rows.Next() {
		var ts string
		var value sql.NullFloat64
		var kind sql.NullString
		if err := rows.Scan(&ts, &value, &kind); err != nil {
			return aqlplot.SeriesData{}, err
		}
		if !value.Valid || !kind.Valid {
			continue
		}
		t, err := parseDBTime(ts)
		if err != nil {
			return aqlplot.SeriesData{}, err
		}
		head, ok := uncompensatedHead(value.Float64, kind.String)
		if !ok {
			continue
		}
		timestamps = append(timestamps, t)
		values = append(values, head)
	}
	if err := rows.Err(); err != nil {
		return aqlplot.SeriesData{}, err
	}
	if len(timestamps) == 0 {
		return aqlplot.SeriesData{}, fmt.Errorf("series uncompensated level has no plottable points")
	}
	return aqlplot.SeriesData{Name: "uncompensated level", Unit: "m absolute head", X: timestamps, Y: values}, nil
}

func correctedColumn(name string) (string, string, error) {
	switch name {
	case "level":
		return "corrected_value", "m", nil
	case "temp":
		return "temp_c", "deg C", nil
	case "ec":
		return "ec_us", "uS", nil
	case "salinity":
		return "sal_psu", "PSU", nil
	default:
		return "", "", fmt.Errorf("unsupported series %s", name)
	}
}

func loggerColumn(source, name string) (string, string, error) {
	switch name {
	case "level":
		if source == plotTypeBaroCorrected {
			return "d.baro_corrected_m", "m", nil
		}
		return "d.level_m", "m", nil
	case "temp":
		return "d.temp_c", "deg C", nil
	case "ec":
		return "d.ec_us", "uS", nil
	case "salinity":
		return "d.sal_psu", "PSU", nil
	default:
		return "", "", fmt.Errorf("unsupported series %s", name)
	}
}

func scanNullablePlotRows(rows *sql.Rows, name, unit string) (aqlplot.SeriesData, error) {
	var timestamps []time.Time
	var values []float64
	for rows.Next() {
		var ts string
		var value sql.NullFloat64
		if err := rows.Scan(&ts, &value); err != nil {
			return aqlplot.SeriesData{}, err
		}
		if !value.Valid {
			continue
		}
		t, err := parseDBTime(ts)
		if err != nil {
			return aqlplot.SeriesData{}, err
		}
		timestamps = append(timestamps, t)
		values = append(values, value.Float64)
	}
	if err := rows.Err(); err != nil {
		return aqlplot.SeriesData{}, err
	}
	if len(timestamps) == 0 {
		return aqlplot.SeriesData{}, fmt.Errorf("series %s has no plottable points", name)
	}
	return aqlplot.SeriesData{Name: name, Unit: unit, X: timestamps, Y: values}, nil
}

func parseDBTime(value string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05-07:00", value); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02 15:04:05", value)
}

func uncompensatedHead(value float64, kind string) (float64, bool) {
	switch kind {
	case "absolute_pressure_mbar":
		return value * 100 / (standardWaterDensity * standardGravity), true
	case "absolute_pressure_kpa":
		return value * 1000 / (standardWaterDensity * standardGravity), true
	case "absolute_pressure_head_m":
		return value, true
	default:
		return 0, false
	}
}
