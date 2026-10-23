package process

import (
	"aqualog/core/utils"
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"
)

type CarryForwardResult struct {
	ReadingID   int
	SourceTime  time.Time
	Timestamp   time.Time
	Level       float64
	Reprocessed bool
}

// CarryForward computes the previous deployment from current inputs, rather
// than trusting potentially stale corrected_data. All writes are atomic.
func CarryForward(database *sql.DB, siteID, loggerID int, timestamp time.Time, maxLookback time.Duration, reprocess bool) (CarryForwardResult, error) {
	var result CarryForwardResult
	if maxLookback <= 0 {
		return result, fmt.Errorf("maximum look-back must be greater than zero")
	}
	tx, err := database.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	var role string
	if err := tx.QueryRow(`SELECT COALESCE(NULLIF(role, ''), 'water_level') FROM loggers WHERE id=$1 AND site_id=$2`, loggerID, siteID).Scan(&role); err != nil {
		return result, fmt.Errorf("water logger does not belong to this site: %w", err)
	}
	if role != "water_level" {
		return result, fmt.Errorf("select a water-level logger")
	}
	raw, err := LoadRawData(tx, siteID)
	if err != nil {
		return result, err
	}
	manual, err := LoadManualMeasurements(tx, siteID)
	if err != nil {
		return result, err
	}
	events, err := LoadLoggerEvents(tx, siteID)
	if err != nil {
		return result, err
	}
	for _, m := range manual {
		if m.Timestamp.Equal(timestamp) {
			return result, fmt.Errorf("a manual reading already exists at this time; edit it instead")
		}
	}
	var loggerRaw []LoggerRawRow
	var loggerEvents []LoggerEvent
	for _, r := range raw {
		if r.LoggerID == loggerID {
			loggerRaw = append(loggerRaw, r)
		}
	}
	for _, e := range events {
		if e.LoggerID == loggerID {
			loggerEvents = append(loggerEvents, e)
		}
	}
	previous, current, err := carryWindows(loggerEvents, timestamp)
	if err != nil {
		return result, err
	}
	if timestamp.Sub(previous.End) >= maxLookback {
		return result, fmt.Errorf("the previous interval is outside the maximum look-back of %s", maxLookback)
	}
	previousRows, err := ComputeCorrectedRows(loggerRaw, manual, []LoggerEvent{
		{LoggerID: loggerID, Timestamp: previous.Start, EventType: "installed"},
		{LoggerID: loggerID, Timestamp: previous.End, EventType: "removed"},
	})
	if err != nil {
		return result, err
	}
	if len(previousRows) == 0 {
		return result, fmt.Errorf("the previous interval has no corrected level; check its logger data and manual readings")
	}
	source := previousRows[len(previousRows)-1]
	if timestamp.Sub(source.Timestamp) > maxLookback {
		return result, fmt.Errorf("the last corrected reading in the previous interval (%s) is older than the maximum look-back of %s", utils.FormatDatasetTime(source.Timestamp), maxLookback)
	}
	if math.IsNaN(source.Value) || math.IsInf(source.Value, 0) {
		return result, fmt.Errorf("the previous corrected level is not finite")
	}
	currentRaw := filterRawToWindow(loggerRaw, current)
	if len(currentRaw) < 2 {
		return result, fmt.Errorf("the current interval needs at least two logger readings")
	}
	if _, ok := nearestLoggerValue(currentRaw, timestamp, loggerInterval(currentRaw)); !ok {
		return result, fmt.Errorf("the chosen time is too far from a logger reading in the current interval")
	}
	notes := fmt.Sprintf("Estimated reading: carried forward %.10g m from logger %d corrected level at %s, before removal at %s. Assumes no water-level change before reinstallation at %s; not a field measurement.", source.Value, loggerID, utils.FormatDatasetTime(source.Timestamp), utils.FormatDatasetTime(previous.End), utils.FormatDatasetTime(current.Start))
	if err := tx.QueryRow(`INSERT INTO manual_readings(site_id, timestamp, value, notes) VALUES ($1,$2,$3,$4) RETURNING id`, siteID, timestamp, source.Value, notes).Scan(&result.ReadingID); err != nil {
		return result, err
	}
	if reprocess {
		manual = append(manual, ManualMeasurement{Timestamp: timestamp, Level: source.Value})
		corrected, err := computeSiteRows(raw, manual, events)
		if err != nil {
			return result, err
		}
		// Do not report successful reprocessing if the new anchor cannot
		// produce any output in its installation interval.
		found := false
		currentIDs := make(map[int]bool)
		for _, r := range currentRaw {
			currentIDs[r.ID] = true
		}
		for _, r := range corrected {
			if currentIDs[r.RawID] {
				found = true
				break
			}
		}
		if !found {
			return result, fmt.Errorf("the current interval produces no corrected data; check its logger readings and installation events")
		}
		if err := storeCorrectedRows(tx, siteID, corrected); err != nil {
			return result, err
		}
	}
	if err := tx.Commit(); err != nil {
		return result, err
	}
	result.SourceTime, result.Timestamp, result.Level, result.Reprocessed = source.Timestamp, timestamp, source.Value, reprocess
	return result, nil
}

func carryWindows(events []LoggerEvent, timestamp time.Time) (TimeWindow, TimeWindow, error) {
	sort.Slice(events, func(i, j int) bool { return events[i].Timestamp.Before(events[j].Timestamp) })
	windows, err := buildValidWindows(events)
	if err != nil {
		return TimeWindow{}, TimeWindow{}, err
	}
	for i, w := range windows {
		if !timestamp.Before(w.Start) && timestamp.Before(w.End) {
			if i == 0 {
				return TimeWindow{}, TimeWindow{}, fmt.Errorf("there is no previous interval for this logger")
			}
			return windows[i-1], w, nil
		}
	}
	return TimeWindow{}, TimeWindow{}, fmt.Errorf("choose a time after installation and before removal for this logger")
}
