package process

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

type LoggerRawRow struct {
	ID        int
	LoggerID  int
	siteID    int
	Timestamp time.Time
	RawValue  float64
	tempC     float64
	salPSU    float64
	ecUS      float64
}

type ManualMeasurement struct {
	ID        int
	SiteID    int
	Timestamp time.Time
	Level     float64 // water level relative to datum
}

type LoggerEvent struct {
	ID        int
	LoggerID  int
	SiteID    int
	Type      string
	Timestamp time.Time
	Notes     string
}

type OffsetSegment struct {
	Start  time.Time
	End    time.Time
	Offset float64
	Source string
}

func ProcessLoggerData(db *sql.DB, siteID int) error {
	// Load all logger_data rows for site
	rawRows, err := loadRawData(db, siteID)
	if err != nil {
		return err
	}

	// Load manual measurements and logger events
	measurements, err := loadManualMeasurements(db, siteID)
	if err != nil {
		return err
	}
	events, err := loadLoggerEvents(db, siteID)
	if err != nil {
		return err
	}

	// Build offset segments
	segments, err := buildOffsetSegments(rawRows, measurements, events)
	if err != nil {
		return err
	}

	// Walk through each row
	for _, row := range rawRows {
		seg := findSegmentForTime(segments, row.Timestamp)
		if seg == nil {
			continue
		}
		correctedLevel := row.RawValue + seg.Offset
		insertCorrectedRow(db, siteID, row.Timestamp, correctedLevel, row.tempC, row.salPSU, row.ecUS)
	}
	return nil
}

func loadRawData(db *sql.DB, siteID int) ([]LoggerRawRow, error) {
	rows, err := db.Query(`
		SELECT id, logger_id, timestamp, level_m, temp_c, sal_psu, ec_us
		FROM logger_data
		WHERE logger_id IN (
			SELECT id FROM loggers WHERE site_id=?
		)
		ORDER BY timestamp ASC
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query logger_data: %w", err)
	}
	defer rows.Close()

	var out []LoggerRawRow

	for rows.Next() {
		var row LoggerRawRow
		var ts string

		// Allow nullable fields
		var tempC sql.NullFloat64
		var salPSU sql.NullFloat64
		var ecUS sql.NullFloat64

		if err := rows.Scan(
			&row.ID,
			&row.LoggerID,
			&ts,
			&row.RawValue,
			&tempC,
			&salPSU,
			&ecUS,
		); err != nil {
			return nil, fmt.Errorf("scan logger_data: %w", err)
		}

		t, err := time.Parse("2006-01-02T15:04:05Z", ts)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp %q: %w", ts, err)
		}
		row.Timestamp = t

		// Convert NullFloat64 -> float64 (use 0 for NULL or choose sentinel)
		if tempC.Valid {
			row.tempC = tempC.Float64
		} else {
			row.tempC = 0
		}

		if salPSU.Valid {
			row.salPSU = salPSU.Float64
		} else {
			row.salPSU = 0 // or NaN, or leave as 0
		}

		if ecUS.Valid {
			row.ecUS = ecUS.Float64
		} else {
			row.ecUS = 0
		}

		out = append(out, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return out, nil
}

func loadManualMeasurements(db *sql.DB, siteID int) ([]ManualMeasurement, error) {
	rows, err := db.Query(`
	SELECT id, site_id, timestamp, value
	FROM manual_readings
	WHERE site_id=?
	ORDER BY timestamp ASC
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query manual_readings: %w", err)
	}
	defer rows.Close()

	var out []ManualMeasurement

	for rows.Next() {
		var m ManualMeasurement
		var ts string

		if err := rows.Scan(&m.ID, &m.SiteID, &ts, &m.Level); err != nil {
			return nil, fmt.Errorf("scan manual_measurements: %w", err)
		}

		t, err := time.Parse("2006-01-02T15:04:05Z", ts)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp %q: %w", ts, err)
		}
		m.Timestamp = t
		out = append(out, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return out, nil
}

func loadLoggerEvents(db *sql.DB, siteID int) ([]LoggerEvent, error) {
	rows, err := db.Query(`
        SELECT id, logger_id, event_type, timestamp, notes
        FROM logger_events
        WHERE logger_id IN (
		SELECT id FROM loggers WHERE site_id=?)
        ORDER BY timestamp ASC
    `, siteID)
	if err != nil {
		return nil, fmt.Errorf("query logger_events: %w", err)
	}
	defer rows.Close()

	var events []LoggerEvent

	for rows.Next() {
		var e LoggerEvent
		var ts string

		if err := rows.Scan(&e.ID, &e.LoggerID, &e.Type, &ts, &e.Notes); err != nil {
			return nil, fmt.Errorf("scan logger_events: %w", err)
		}
		t, err := time.Parse("2006-01-02T15:04:05Z", ts)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp %q: %w", ts, err)
		}
		e.Timestamp = t

		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return events, nil
}

func buildOffsetSegments(
	raw []LoggerRawRow,
	manual []ManualMeasurement,
	events []LoggerEvent,
) ([]OffsetSegment, error) {

	if len(raw) == 0 {
		return nil, fmt.Errorf("no raw data available")
	}

	// Find raw value at or nearest before timestamp
	getRawAt := func(ts time.Time) *LoggerRawRow {
		// simple binary search
		var prev *LoggerRawRow
		for i := range raw {
			if raw[i].Timestamp.After(ts) {
				return prev
			}
			prev = &raw[i]
		}
		return prev
	}

	var segments []OffsetSegment

	// Determine the timeline breakpoints
	// Combine events and manual measurements into a chronological list
	type marker struct {
		Time   time.Time
		Kind   string // "event" or "manual"
		Event  *LoggerEvent
		Manual *ManualMeasurement
	}

	var markers []marker

	for i := range events {
		markers = append(markers, marker{
			Time:  events[i].Timestamp,
			Kind:  "event",
			Event: &events[i],
		})
	}

	for i := range manual {
		markers = append(markers, marker{
			Time:   manual[i].Timestamp,
			Kind:   "manual",
			Manual: &manual[i],
		})
	}

	sort.Slice(markers, func(i, j int) bool {
		return markers[i].Time.Before(markers[j].Time)
	})

	// State while we walk the timeline
	var (
		currentStart  time.Time
		currentOffset float64
		haveOffset    bool
		loggerActive  bool
	)

	// Initialize with first install event
	for _, m := range markers {
		if m.Kind == "event" && m.Event.Type == "installed" {
			loggerActive = true
			currentStart = m.Time

			// Try to set offset from nearest raw
			if r := getRawAt(m.Time); r != nil {
				currentOffset = 0 // baseline offset
				haveOffset = true
			} else {
				currentOffset = 0
				haveOffset = true
			}
			break
		}
	}

	if !loggerActive {
		return nil, fmt.Errorf("no install event found")
	}

	// Walk through markers and create segments
	for i := range markers {
		m := markers[i]

		if m.Time.Before(currentStart) {
			continue
		}

		switch m.Kind {

		case "manual":
			// Close the segment up to this manual timestamp
			if haveOffset && !m.Time.Equal(currentStart) {
				segments = append(segments, OffsetSegment{
					Start:  currentStart,
					End:    m.Time,
					Offset: currentOffset,
					Source: "previous",
				})
			}

			// Calculate new offset from manual measurement
			if r := getRawAt(m.Time); r != nil {
				currentOffset = m.Manual.Level - r.RawValue
				haveOffset = true
			}
			currentStart = m.Time

		case "event":
			switch m.Event.Type {
			case "installed":
				// Already handled during initialization; skip
				continue

			case "moved":
				// End previous segment
				if haveOffset {
					segments = append(segments, OffsetSegment{
						Start:  currentStart,
						End:    m.Time,
						Offset: currentOffset,
						Source: "previous",
					})
				}
				// Start new segment with unknown offset
				haveOffset = false
				currentStart = m.Time

			case "removed":
				if haveOffset {
					segments = append(segments, OffsetSegment{
						Start:  currentStart,
						End:    m.Time,
						Offset: currentOffset,
						Source: "end",
					})
				}
				loggerActive = false
				return segments, nil
			}
		}
	}

	// Close final segment up to end of raw data
	if loggerActive && haveOffset {
		segments = append(segments, OffsetSegment{
			Start:  currentStart,
			End:    raw[len(raw)-1].Timestamp,
			Offset: currentOffset,
			Source: "final",
		})
	}

	return segments, nil
}

func findSegmentForTime(segments []OffsetSegment, ts time.Time) *OffsetSegment {
	for i := range segments {
		seg := &segments[i]

		// Condition: ts ∈ [Start, End]
		if (ts.Equal(seg.Start) || ts.After(seg.Start)) &&
			(ts.Equal(seg.End) || ts.Before(seg.End)) {

			return seg
		}
	}
	return nil
}

func insertCorrectedRow(
	db *sql.DB,
	siteID int,
	ts time.Time,
	correctedLevel float64,
	tempC float64,
	salPSU float64,
	ecUS float64,
) error {

	_, err := db.Exec(`
        INSERT OR REPLACE INTO corrected_data (site_id, timestamp, corrected_value, temp_c, sal_psu, ec_us)
        VALUES (?, ?, ?, ?, ?, ?)
    `, siteID, ts, correctedLevel, tempC, salPSU, ecUS)

	return err
}
