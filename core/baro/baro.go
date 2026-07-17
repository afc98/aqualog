package baro

import (
	"aqualog/core/db"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"
)

const gravity = 9.80665

type CorrectOptions struct {
	SiteID       int
	LoggerID     int
	BarologgerID int
	DensityMode  string
	Density      float64
	MaxGap       time.Duration
	Notes        string
}

type CorrectResult struct {
	RunID     int
	Updated   int
	Skipped   int
	LoggerIDs []int
}

type Status struct {
	Assignments []Assignment
	Runs        []Run
}

type Assignment struct {
	Scope        string
	ScopeID      int
	BarologgerID int
	Barologger   string
}

type Run struct {
	ID           int
	SiteID       int
	LoggerID     sql.NullInt64
	BarologgerID int
	Method       string
	DensityMode  string
	Density      float64
	MaxGap       int
	CreatedAt    string
}

type pressurePoint struct {
	t        time.Time
	pressure float64
}

type waterRow struct {
	id       int
	loggerID int
	t        time.Time
	value    float64
	unit     string
	kind     string
	interval time.Duration
}

func Assign(scope string, scopeID int, barologgerID int) error {
	d, err := db.GetDB()
	if err != nil {
		return err
	}
	defer d.Close()
	if err := requireLoggerRole(d, barologgerID, "barometric"); err != nil {
		return err
	}
	_, err = d.Exec(`
		INSERT INTO baro_assignments(scope, scope_id, barologger_id)
		VALUES (?, ?, ?)
		ON CONFLICT(scope, scope_id) DO UPDATE SET
			barologger_id = excluded.barologger_id,
			created_at = CURRENT_TIMESTAMP
	`, scope, scopeID, barologgerID)
	return err
}

func requireLoggerRole(d *sql.DB, loggerID int, want string) error {
	role, err := loggerRole(d, loggerID)
	if err != nil {
		return err
	}
	if role == "" {
		role = "water_level"
	}
	if role == want {
		return nil
	}
	switch want {
	case "barometric":
		return fmt.Errorf("logger %d has role %q; expected a barometric logger", loggerID, role)
	case "water_level":
		return fmt.Errorf("logger %d has role %q; --logger must be a water logger. Use --barologger %d for the barometric logger, or omit --logger to correct all water loggers at the site", loggerID, role, loggerID)
	default:
		return fmt.Errorf("logger %d has role %q; expected %q", loggerID, role, want)
	}
}

func loggerRole(d *sql.DB, loggerID int) (string, error) {
	var role sql.NullString
	err := d.QueryRow(`SELECT role FROM loggers WHERE id = ?`, loggerID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("logger %d not found", loggerID)
	}
	if err != nil {
		return "", err
	}
	if role.Valid {
		return role.String, nil
	}
	return "", nil
}

func Correct(opts CorrectOptions) (CorrectResult, error) {
	if opts.DensityMode == "" {
		opts.DensityMode = "freshwater"
	}
	if opts.Density == 0 {
		opts.Density = densityForMode(opts.DensityMode)
	}

	d, err := db.GetDB()
	if err != nil {
		return CorrectResult{}, err
	}
	defer d.Close()

	if opts.BarologgerID == 0 {
		opts.BarologgerID, err = resolveAssignedBarologger(d, opts.SiteID)
		if err != nil {
			return CorrectResult{}, err
		}
	}
	if err := requireLoggerRole(d, opts.BarologgerID, "barometric"); err != nil {
		return CorrectResult{}, err
	}
	if opts.LoggerID > 0 {
		if err := requireLoggerRole(d, opts.LoggerID, "water_level"); err != nil {
			return CorrectResult{}, err
		}
	}
	baroRows, err := loadBaroRows(d, opts.BarologgerID)
	if err != nil {
		return CorrectResult{}, err
	}
	if len(baroRows) < 2 {
		return CorrectResult{}, fmt.Errorf("barologger %d has fewer than two pressure records", opts.BarologgerID)
	}

	waterRows, err := loadWaterRows(d, opts.SiteID, opts.LoggerID)
	if err != nil {
		return CorrectResult{}, err
	}
	if len(waterRows) == 0 {
		if opts.LoggerID > 0 {
			return CorrectResult{}, fmt.Errorf("logger %d has no absolute-pressure rows to correct", opts.LoggerID)
		}
		return CorrectResult{}, fmt.Errorf("no absolute-pressure water logger rows found")
	}
	if opts.MaxGap == 0 {
		opts.MaxGap = inferMaxGap(waterRows)
	}

	tx, err := d.Begin()
	if err != nil {
		return CorrectResult{}, err
	}
	defer tx.Rollback()

	runID, err := insertRun(tx, opts)
	if err != nil {
		return CorrectResult{}, err
	}

	result := CorrectResult{RunID: runID}
	seen := map[int]bool{}
	for _, row := range waterRows {
		baroPa, ok := interpolatePressure(baroRows, row.t, opts.MaxGap)
		if !ok {
			result.Skipped++
			continue
		}
		head, err := correctedHead(row, baroPa, opts.Density)
		if err != nil {
			result.Skipped++
			continue
		}
		if _, err := tx.Exec(`
			UPDATE logger_data
			SET baro_corrected_m = ?, level_m = ?, correction_run_id = ?
			WHERE id = ?
		`, head, head, runID, row.id); err != nil {
			return CorrectResult{}, err
		}
		result.Updated++
		if !seen[row.loggerID] {
			seen[row.loggerID] = true
			result.LoggerIDs = append(result.LoggerIDs, row.loggerID)
		}
	}

	if _, err := tx.Exec(`DELETE FROM corrected_data WHERE site_id = ?`, opts.SiteID); err != nil {
		return CorrectResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return CorrectResult{}, err
	}
	return result, nil
}

func SiteStatus(siteID int) (Status, error) {
	d, err := db.GetDB()
	if err != nil {
		return Status{}, err
	}
	defer d.Close()

	var out Status
	rows, err := d.Query(`
		SELECT a.scope, a.scope_id, a.barologger_id, l.name
		FROM baro_assignments a
		JOIN loggers l ON l.id = a.barologger_id
		WHERE (a.scope = 'site' AND a.scope_id = ?)
		   OR (a.scope = 'project' AND a.scope_id = (SELECT project_id FROM sites WHERE id = ?))
		ORDER BY a.scope
	`, siteID, siteID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.Scope, &a.ScopeID, &a.BarologgerID, &a.Barologger); err != nil {
			return out, err
		}
		out.Assignments = append(out.Assignments, a)
	}

	runRows, err := d.Query(`
		SELECT id, site_id, logger_id, barologger_id, method, density_mode, density_kg_m3, max_gap_seconds, created_at
		FROM correction_runs
		WHERE site_id = ?
		ORDER BY id DESC
		LIMIT 10
	`, siteID)
	if err != nil {
		return out, err
	}
	defer runRows.Close()
	for runRows.Next() {
		var r Run
		if err := runRows.Scan(&r.ID, &r.SiteID, &r.LoggerID, &r.BarologgerID, &r.Method, &r.DensityMode, &r.Density, &r.MaxGap, &r.CreatedAt); err != nil {
			return out, err
		}
		out.Runs = append(out.Runs, r)
	}
	return out, nil
}

func resolveAssignedBarologger(d *sql.DB, siteID int) (int, error) {
	var id int
	err := d.QueryRow(`SELECT barologger_id FROM baro_assignments WHERE scope = 'site' AND scope_id = ?`, siteID).Scan(&id)
	if err == nil {
		return id, nil
	}
	err = d.QueryRow(`
		SELECT barologger_id
		FROM baro_assignments
		WHERE scope = 'project'
		  AND scope_id = (SELECT project_id FROM sites WHERE id = ?)
	`, siteID).Scan(&id)
	if err == nil {
		return id, nil
	}
	return 0, fmt.Errorf("no barologger assigned for site %d", siteID)
}

func loadBaroRows(d *sql.DB, loggerID int) ([]pressurePoint, error) {
	rows, err := d.Query(`
		SELECT timestamp, imported_value, imported_kind
		FROM logger_data
		WHERE logger_id = ?
		  AND imported_kind IN ('barometric_pressure_mbar', 'barometric_pressure_kpa')
		ORDER BY timestamp
	`, loggerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []pressurePoint
	for rows.Next() {
		var ts, kind string
		var value float64
		if err := rows.Scan(&ts, &value, &kind); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, err
		}
		out = append(out, pressurePoint{t: t, pressure: pressurePa(value, kind)})
	}
	return out, rows.Err()
}

func loadWaterRows(d *sql.DB, siteID int, loggerID int) ([]waterRow, error) {
	args := []any{siteID}
	filter := ""
	if loggerID > 0 {
		filter = " AND d.logger_id = ?"
		args = append(args, loggerID)
	}
	rows, err := d.Query(`
		SELECT d.id, d.logger_id, d.timestamp, d.imported_value, d.imported_unit, d.imported_kind
		FROM logger_data d
		JOIN loggers l ON l.id = d.logger_id
		WHERE l.site_id = ?
		  AND d.imported_kind IN ('absolute_pressure_mbar', 'absolute_pressure_kpa', 'absolute_pressure_head_m')
	`+filter+`
		ORDER BY d.logger_id, d.timestamp
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []waterRow
	for rows.Next() {
		var r waterRow
		var ts string
		if err := rows.Scan(&r.id, &r.loggerID, &ts, &r.value, &r.unit, &r.kind); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, err
		}
		r.t = t
		out = append(out, r)
	}
	return out, rows.Err()
}

func insertRun(tx *sql.Tx, opts CorrectOptions) (int, error) {
	res, err := tx.Exec(`
		INSERT INTO correction_runs(site_id, logger_id, barologger_id, method, density_mode, density_kg_m3, match_mode, max_gap_seconds, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, opts.SiteID, nullInt(opts.LoggerID), opts.BarologgerID, "barometric", opts.DensityMode, opts.Density, "linear_interpolation", int(opts.MaxGap.Seconds()), opts.Notes)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

func nullInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

func densityForMode(mode string) float64 {
	switch mode {
	case "saline":
		return 1025
	case "none":
		return 1000
	default:
		return 1000
	}
}

func pressurePa(value float64, kind string) float64 {
	switch kind {
	case "barometric_pressure_mbar", "absolute_pressure_mbar":
		return value * 100
	case "barometric_pressure_kpa", "absolute_pressure_kpa":
		return value * 1000
	default:
		return value
	}
}

func correctedHead(row waterRow, baroPa float64, density float64) (float64, error) {
	switch row.kind {
	case "absolute_pressure_mbar", "absolute_pressure_kpa":
		return (pressurePa(row.value, row.kind) - baroPa) / (density * gravity), nil
	case "absolute_pressure_head_m":
		return row.value - (baroPa / (density * gravity)), nil
	default:
		return 0, fmt.Errorf("unsupported water measurement kind %s", row.kind)
	}
}

func interpolatePressure(points []pressurePoint, t time.Time, maxGap time.Duration) (float64, bool) {
	i := sort.Search(len(points), func(i int) bool {
		return !points[i].t.Before(t)
	})
	if i < len(points) && points[i].t.Equal(t) {
		return points[i].pressure, true
	}
	if i == 0 || i >= len(points) {
		return 0, false
	}
	prev := points[i-1]
	next := points[i]
	if t.Sub(prev.t) > maxGap || next.t.Sub(t) > maxGap {
		return 0, false
	}
	span := next.t.Sub(prev.t).Seconds()
	if span == 0 {
		return prev.pressure, true
	}
	f := t.Sub(prev.t).Seconds() / span
	return prev.pressure + (next.pressure-prev.pressure)*f, true
}

func inferMaxGap(rows []waterRow) time.Duration {
	if len(rows) < 2 {
		return time.Hour
	}
	var deltas []time.Duration
	for i := 1; i < len(rows); i++ {
		if rows[i].loggerID == rows[i-1].loggerID {
			d := rows[i].t.Sub(rows[i-1].t)
			if d > 0 {
				deltas = append(deltas, d)
			}
		}
	}
	if len(deltas) == 0 {
		return time.Hour
	}
	sort.Slice(deltas, func(i, j int) bool { return deltas[i] < deltas[j] })
	return time.Duration(math.Max(float64(deltas[len(deltas)/2]*2), float64(time.Minute)))
}
