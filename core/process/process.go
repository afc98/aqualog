package process

import (
	"aqualog/core/utils"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

type LoggerRawRow struct {
	ID        int
	LoggerID  int
	Timestamp time.Time
	RawValue  float64
	TempC     float64
	SalPSU    *float64
	ECUS      *float64
}

type ManualMeasurement struct {
	Timestamp time.Time
	Level     float64
}

type LoggerEvent struct {
	LoggerID  int
	Timestamp time.Time
	EventType string
}

type TimeWindow struct {
	Start time.Time
	End   time.Time
}

type OffsetSegment struct {
	Start  time.Time
	End    time.Time
	Offset float64
}

type CorrectedRow struct {
	RawID     int
	Timestamp time.Time
	Value     float64
	TempC     float64
	SalPSU    *float64
	ECUS      *float64
}

func ProcessLoggerData(db *sql.DB, siteID int) error {

	defer utils.TimeTrack(time.Now(), "ProcessLoggerData")

	raw, err := LoadRawData(db, siteID)
	if err != nil {
		return err
	}

	manual, err := LoadManualMeasurements(db, siteID)
	if err != nil {
		return err
	}

	events, err := LoadLoggerEvents(db, siteID)
	if err != nil {
		return err
	}

	corrected, err := computeSiteRows(raw, manual, events)
	if err != nil {
		return err
	}
	return StoreCorrectedRows(db, siteID, corrected)
}

func computeSiteRows(raw []LoggerRawRow, manual []ManualMeasurement, events []LoggerEvent) ([]CorrectedRow, error) {
	rawByLogger := make(map[int][]LoggerRawRow)
	for _, row := range raw {
		rawByLogger[row.LoggerID] = append(rawByLogger[row.LoggerID], row)
	}
	eventsByLogger := make(map[int][]LoggerEvent)
	for _, event := range events {
		eventsByLogger[event.LoggerID] = append(eventsByLogger[event.LoggerID], event)
	}

	loggerIDs := make([]int, 0, len(rawByLogger))
	for loggerID := range rawByLogger {
		loggerIDs = append(loggerIDs, loggerID)
	}
	sort.Ints(loggerIDs)

	var corrected []CorrectedRow
	timestampOwner := make(map[int64]int)
	for _, loggerID := range loggerIDs {
		loggerRows, err := ComputeCorrectedRows(rawByLogger[loggerID], manual, eventsByLogger[loggerID])
		if err != nil {
			return nil, fmt.Errorf("logger %d: %w", loggerID, err)
		}
		for _, row := range loggerRows {
			timestampKey := row.Timestamp.UnixNano()
			if otherLoggerID, exists := timestampOwner[timestampKey]; exists {
				return nil, fmt.Errorf(
					"water-level loggers %d and %d both produce corrected data at %s; adjust their install/remove events so deployments do not overlap",
					otherLoggerID,
					loggerID,
					row.Timestamp.Format(time.RFC3339),
				)
			}
			timestampOwner[timestampKey] = loggerID
			corrected = append(corrected, row)
		}
	}
	sort.Slice(corrected, func(i, j int) bool { return corrected[i].Timestamp.Before(corrected[j].Timestamp) })

	return corrected, nil
}

type dataReader interface {
	Query(string, ...any) (*sql.Rows, error)
}

func LoadRawData(db dataReader, siteID int) ([]LoggerRawRow, error) {

	rows, err := db.Query(`
		SELECT
			d.id,
			d.logger_id,
			d.timestamp,
			d.level_m,
			d.temp_c,
			d.sal_psu,
			d.ec_us
		FROM logger_data d
		JOIN loggers l ON l.id = d.logger_id
		WHERE l.site_id = $1
		  AND COALESCE(NULLIF(l.role, ''), 'water_level') = 'water_level'
		ORDER BY d.logger_id ASC, d.timestamp ASC
	`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LoggerRawRow

	for rows.Next() {
		var r LoggerRawRow
		var ts string

		var tempC, salPSU, ecUS sql.NullFloat64

		if err := rows.Scan(
			&r.ID,
			&r.LoggerID,
			&ts,
			&r.RawValue,
			&tempC,
			&salPSU,
			&ecUS,
		); err != nil {
			return nil, err
		}

		t, err := utils.ParseDatasetTime(ts)
		if err != nil {
			return nil, err
		}

		r.Timestamp = t

		if tempC.Valid {
			r.TempC = tempC.Float64
		}
		if salPSU.Valid {
			value := salPSU.Float64
			r.SalPSU = &value
		}
		if ecUS.Valid {
			value := ecUS.Float64
			r.ECUS = &value
		}

		out = append(out, r)
	}

	return out, rows.Err()
}

func LoadManualMeasurements(db dataReader, siteID int) ([]ManualMeasurement, error) {

	rows, err := db.Query(`
		SELECT
			m.timestamp,
			m.value
		FROM manual_readings m
		WHERE m.site_id = $1
		ORDER BY m.timestamp ASC
	`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ManualMeasurement

	for rows.Next() {
		var r ManualMeasurement
		var ts string

		if err := rows.Scan(&ts, &r.Level); err != nil {
			return nil, err
		}

		t, err := utils.ParseDatasetTime(ts)
		if err != nil {
			return nil, err
		}

		r.Timestamp = t
		out = append(out, r)
	}

	return out, rows.Err()
}

func LoadLoggerEvents(db dataReader, siteID int) ([]LoggerEvent, error) {

	rows, err := db.Query(`
		SELECT
			e.logger_id,
			e.timestamp,
			e.event_type
		FROM logger_events e
		JOIN loggers l ON l.id = e.logger_id
		WHERE l.site_id = $1
		  AND COALESCE(NULLIF(l.role, ''), 'water_level') = 'water_level'
		ORDER BY e.logger_id ASC, e.timestamp ASC
	`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LoggerEvent

	for rows.Next() {
		var e LoggerEvent
		var ts string

		if err := rows.Scan(&e.LoggerID, &ts, &e.EventType); err != nil {
			return nil, err
		}

		t, err := utils.ParseDatasetTime(ts)
		if err != nil {
			return nil, err
		}

		e.Timestamp = t
		out = append(out, e)
	}

	return out, rows.Err()
}

func ComputeCorrectedRows(

	raw []LoggerRawRow,

	manual []ManualMeasurement,

	events []LoggerEvent,

) ([]CorrectedRow, error) {

	windows, err := buildValidWindows(events)

	if err != nil {

		return nil, err

	}

	var out []CorrectedRow

	for _, w := range windows {

		rawWin := filterRawToWindow(raw, w)

		if len(rawWin) < 2 {

			continue

		}

		interval := loggerInterval(rawWin)

		manualWin := filterManualToExtendedWindow(manual, w, interval)

		segments := buildOffsetSegments(rawWin, manualWin, interval, w)

		if len(segments) == 0 {

			continue // No segments to process for this window

		}

		segIdx := 0

		for _, r := range rawWin {

			for segIdx+1 < len(segments) &&

				!r.Timestamp.Before(segments[segIdx+1].Start) {

				segIdx++

			}

			seg := segments[segIdx]

			if r.Timestamp.Before(seg.Start) || !r.Timestamp.Before(seg.End) {

				continue

			}

			out = append(out, CorrectedRow{

				RawID: r.ID,

				Timestamp: r.Timestamp,

				Value: r.RawValue + seg.Offset,

				TempC: r.TempC,

				SalPSU: r.SalPSU,

				ECUS: r.ECUS,
			})

		}

	}

	return out, nil

}

func StoreCorrectedRows(
	db *sql.DB,
	siteID int,
	rows []CorrectedRow,
) error {

	// Start a single transaction for delete + insert
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	if err := storeCorrectedRows(tx, siteID, rows); err != nil {
		return err
	}
	return tx.Commit()
}

func storeCorrectedRows(tx *sql.Tx, siteID int, rows []CorrectedRow) error {
	// 1️⃣ Delete all existing corrected rows for this site
	if _, err := tx.Exec(`DELETE FROM corrected_data WHERE site_id = $1`, siteID); err != nil {
		return fmt.Errorf("failed to delete old corrected rows: %w", err)
	}

	// 2️⃣ Insert new corrected rows in batches
	const batchSize = 130 // safe for SQLite max_bind_vars

	for i := 0; i < len(rows); i += batchSize {
		end := i + batchSize
		if end > len(rows) {
			end = len(rows)
		}

		query := `
			INSERT INTO corrected_data (
				site_id, logger_data_id, timestamp, corrected_value, temp_c, sal_psu, ec_us
			) VALUES
		`
		args := make([]any, 0, (end-i)*7)

		for _, row := range rows[i:end] {
			base := len(args)
			query += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d),", base+1, base+2, base+3, base+4, base+5, base+6, base+7)
			args = append(args,
				siteID,
				row.RawID,
				row.Timestamp,
				row.Value,
				row.TempC,
				row.SalPSU,
				row.ECUS,
			)
		}

		// remove trailing comma
		query = query[:len(query)-1]

		if _, err := tx.Exec(query, args...); err != nil {
			return fmt.Errorf("failed to insert corrected rows: %w", err)
		}
	}

	return nil
}

/* ---------------- Window logic ---------------- */

func buildValidWindows(events []LoggerEvent) ([]TimeWindow, error) {

	var windows []TimeWindow
	var open *time.Time

	for _, e := range events {
		switch e.EventType {

		case "installed":
			if open != nil {
				return nil, fmt.Errorf("installation without removal at %s", e.Timestamp)
			}
			t := e.Timestamp
			open = &t

		case "removed":
			if open == nil {
				return nil, fmt.Errorf("removal without installation at %s", e.Timestamp)
			}
			windows = append(windows, TimeWindow{
				Start: *open,
				End:   e.Timestamp,
			})
			open = nil
		}
	}

	// A logger still installed remains valid through all available readings.
	// Do not require an invented future removal event to process its data.
	if open != nil {
		windows = append(windows, TimeWindow{
			Start: *open,
			End:   time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC),
		})
	}
	return windows, nil
}

/* ---------------- Filtering ---------------- */

func filterRawToWindow(rows []LoggerRawRow, w TimeWindow) []LoggerRawRow {
	var out []LoggerRawRow
	for _, r := range rows {
		if !r.Timestamp.Before(w.Start) && r.Timestamp.Before(w.End) {
			out = append(out, r)
		}
	}
	return out
}

func filterManualToExtendedWindow(
	rows []ManualMeasurement,
	w TimeWindow,
	interval time.Duration,
) []ManualMeasurement {

	start := w.Start.Add(-interval)
	end := w.End.Add(interval)

	var out []ManualMeasurement
	for _, r := range rows {
		if !r.Timestamp.Before(start) && !r.Timestamp.After(end) {
			out = append(out, r)
		}
	}
	return out
}

/* ---------------- Offset construction ---------------- */

func buildOffsetSegments(raw []LoggerRawRow, manual []ManualMeasurement, maxDist time.Duration, w TimeWindow) []OffsetSegment {
	if len(manual) == 0 {
		return nil
	}

	sort.Slice(manual, func(i, j int) bool {
		return manual[i].Timestamp.Before(manual[j].Timestamp)
	})

	type temporalOffset struct {
		Timestamp time.Time
		Offset    float64
	}
	var offsets []temporalOffset

	for _, m := range manual {
		rawVal, ok := nearestLoggerValue(raw, m.Timestamp, maxDist)
		if !ok {
			continue // Skip manual measurements without a corresponding raw value
		}
		offsets = append(offsets, temporalOffset{
			Timestamp: m.Timestamp,
			Offset:    m.Level - rawVal,
		})
	}

	if len(offsets) == 0 {
		return nil
	}

	var segments []OffsetSegment

	// Segment from start of window to first offset point
	segments = append(segments, OffsetSegment{
		Start:  w.Start,
		End:    offsets[0].Timestamp,
		Offset: offsets[0].Offset, // Apply the first offset backward
	})

	for i := 0; i < len(offsets)-1; i++ {
		segments = append(segments, OffsetSegment{
			Start:  offsets[i].Timestamp,
			End:    offsets[i+1].Timestamp,
			Offset: offsets[i+1].Offset, // Apply the next offset backward from its point
		})
	}

	// Segment from last offset point to end of window
	segments = append(segments, OffsetSegment{
		Start:  offsets[len(offsets)-1].Timestamp,
		End:    w.End,
		Offset: offsets[len(offsets)-1].Offset,
	})

	return segments
}

/* ---------------- Matching ---------------- */

func nearestLoggerValue(
	rows []LoggerRawRow,
	t time.Time,
	maxDist time.Duration,
) (float64, bool) {

	i := sort.Search(len(rows), func(i int) bool {
		return !rows[i].Timestamp.Before(t)
	})

	var best *LoggerRawRow

	if i < len(rows) {
		best = &rows[i]
	}
	if i > 0 {
		prev := &rows[i-1]
		if best == nil ||
			absDur(prev.Timestamp.Sub(t)) < absDur(best.Timestamp.Sub(t)) {
			best = prev
		}
	}

	if best == nil {
		return 0, false
	}

	if absDur(best.Timestamp.Sub(t)) > maxDist {
		return 0, false
	}

	return best.RawValue, true
}

func loggerInterval(rows []LoggerRawRow) time.Duration {

	deltas := make([]time.Duration, 0, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		deltas = append(deltas, rows[i].Timestamp.Sub(rows[i-1].Timestamp))
	}

	sort.Slice(deltas, func(i, j int) bool { return deltas[i] < deltas[j] })
	return deltas[len(deltas)/2]
}

func absDur(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
