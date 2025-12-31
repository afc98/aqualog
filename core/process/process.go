package process

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

type LoggerRawRow struct {
	Timestamp time.Time
	RawValue  float64
	TempC     float64
	SalPSU    float64
	ECUS      float64
}

type ManualMeasurement struct {
	Timestamp time.Time
	Level     float64
}

type LoggerEvent struct {
	Timestamp time.Time
	EventType string
}

type OffsetSegment struct {
	Start  time.Time
	End    *time.Time
	Offset float64
}

type CorrectedRow struct {
	Timestamp time.Time
	Value     float64
	TempC     float64
	SalPSU    float64
	ECUS      float64
}

func ProcessLoggerData(db *sql.DB, siteID int) error {

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

	corrected, err := ComputeCorrectedRows(raw, manual, events)
	if err != nil {
		return err
	}

	return StoreCorrectedRows(db, siteID, corrected)
}

func LoadRawData(db *sql.DB, siteID int) ([]LoggerRawRow, error) {

	rows, err := db.Query(`
		SELECT
			d.timestamp,
			d.level_m,
			d.temp_c,
			d.sal_psu,
			d.ec_us
		FROM logger_data d
		JOIN loggers l ON l.id = d.logger_id
		WHERE l.site_id = ?
		ORDER BY d.timestamp ASC
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
			&ts,
			&r.RawValue,
			&tempC,
			&salPSU,
			&ecUS,
		); err != nil {
			return nil, err
		}

		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, err
		}

		r.Timestamp = t

		if tempC.Valid {
			r.TempC = tempC.Float64
		}
		if salPSU.Valid {
			r.SalPSU = salPSU.Float64
		}
		if ecUS.Valid {
			r.ECUS = ecUS.Float64
		}

		out = append(out, r)
	}

	return out, rows.Err()
}

func LoadManualMeasurements(db *sql.DB, siteID int) ([]ManualMeasurement, error) {

	rows, err := db.Query(`
		SELECT
			m.timestamp,
			m.value
		FROM manual_readings m
		WHERE m.site_id = ?
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

		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, err
		}

		r.Timestamp = t
		out = append(out, r)
	}

	return out, rows.Err()
}

func LoadLoggerEvents(db *sql.DB, siteID int) ([]LoggerEvent, error) {

	rows, err := db.Query(`
		SELECT
			e.timestamp,
			e.event_type
		FROM logger_events e
		JOIN loggers l ON l.id = e.logger_id
		WHERE l.site_id = ?
		ORDER BY e.timestamp ASC
	`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LoggerEvent

	for rows.Next() {
		var e LoggerEvent
		var ts string

		if err := rows.Scan(&ts, &e.EventType); err != nil {
			return nil, err
		}

		t, err := time.Parse(time.RFC3339, ts)
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

	if len(manual) == 0 {
		return nil, fmt.Errorf("no manual measurements available")
	}

	segments, err := buildOffsetSegments(raw, manual, events)
	if err != nil {
		return nil, err
	}

	var out []CorrectedRow

	for _, r := range raw {
		seg := findSegmentForTime(segments, r.Timestamp)
		if seg == nil {
			continue
		}

		out = append(out, CorrectedRow{
			Timestamp: r.Timestamp,
			Value:     r.RawValue + seg.Offset,
			TempC:     r.TempC,
			SalPSU:    r.SalPSU,
			ECUS:      r.ECUS,
		})
	}

	return out, nil
}

func StoreCorrectedRows(
	db *sql.DB,
	siteID int,
	rows []CorrectedRow,
) error {

	for _, r := range rows {
		_, err := db.Exec(`
			INSERT OR REPLACE INTO corrected_data
			(site_id, timestamp, corrected_value, temp_c, sal_psu, ec_us)
			VALUES (?, ?, ?, ?, ?, ?)
		`,
			siteID,
			r.Timestamp.Format(time.RFC3339),
			r.Value,
			r.TempC,
			r.SalPSU,
			r.ECUS,
		)
		if err != nil {
			return fmt.Errorf("insert corrected row at %s: %w", r.Timestamp, err)
		}
	}

	return nil
}

func buildOffsetSegments(
	raw []LoggerRawRow,
	manual []ManualMeasurement,
	events []LoggerEvent,
) ([]OffsetSegment, error) {

	sort.Slice(manual, func(i, j int) bool {
		return manual[i].Timestamp.Before(manual[j].Timestamp)
	})

	var segments []OffsetSegment

	for i, m := range manual {

		rawVal, ok := rawValueAt(raw, m.Timestamp)
		if !ok {
			continue
		}

		offset := m.Level - rawVal

		seg := OffsetSegment{
			Start:  m.Timestamp,
			Offset: offset,
		}

		if i+1 < len(manual) {
			end := manual[i+1].Timestamp
			seg.End = &end
		}

		segments = append(segments, seg)
	}

	return segments, nil
}

func findSegmentForTime(
	segments []OffsetSegment,
	t time.Time,
) *OffsetSegment {

	for _, s := range segments {
		if t.Before(s.Start) {
			continue
		}
		if s.End != nil && !t.Before(*s.End) {
			continue
		}
		return &s
	}
	return nil
}

func rawValueAt(rows []LoggerRawRow, t time.Time) (float64, bool) {
	for _, r := range rows {
		if r.Timestamp.Equal(t) {
			return r.RawValue, true
		}
	}
	return 0, false
}
