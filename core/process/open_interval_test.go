package process

import (
	"testing"
	"time"
)

func TestCarryForwardReprocessesOngoingInstallation(t *testing.T) {
	database := seedCarry(t)
	if _, err := database.Exec(`DELETE FROM logger_events WHERE event_type='removed' AND timestamp='2026-01-01T02:00:00Z'`); err != nil {
		t.Fatal(err)
	}
	result, err := CarryForward(database, 1, 2, carryTime("00:21:00"), time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Level != 16 || !result.Reprocessed {
		t.Fatalf("result=%+v", result)
	}
	var count int
	if err := database.QueryRow(`SELECT count(*) FROM corrected_data`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("corrected rows=%d; want 4", count)
	}
	var level float64
	if err := database.QueryRow(`SELECT corrected_value FROM corrected_data WHERE logger_data_id=4`).Scan(&level); err != nil {
		t.Fatal(err)
	}
	if level != 17 {
		t.Fatalf("last corrected level=%v; want 17", level)
	}
}

func TestProcessIncludesOngoingInstallationAndExcludesRemovalGap(t *testing.T) {
	raw := []LoggerRawRow{
		{ID: 1, LoggerID: 2, Timestamp: carryTime("00:01:00"), RawValue: 10},
		{ID: 2, LoggerID: 2, Timestamp: carryTime("00:02:00"), RawValue: 11},
		{ID: 3, LoggerID: 2, Timestamp: carryTime("00:15:00"), RawValue: 999},
		{ID: 4, LoggerID: 2, Timestamp: carryTime("00:21:00"), RawValue: 30},
		{ID: 5, LoggerID: 2, Timestamp: carryTime("00:22:00"), RawValue: 31},
	}
	events := []LoggerEvent{
		{LoggerID: 2, Timestamp: carryTime("00:00:00"), EventType: "installed"},
		{LoggerID: 2, Timestamp: carryTime("00:10:00"), EventType: "removed"},
		{LoggerID: 2, Timestamp: carryTime("00:20:00"), EventType: "installed"},
	}
	manual := []ManualMeasurement{{Timestamp: carryTime("00:01:00"), Level: 15}, {Timestamp: carryTime("00:21:00"), Level: 16}}
	rows, err := ComputeCorrectedRows(raw, manual, events)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 {
		t.Fatalf("rows=%+v", rows)
	}
	for _, row := range rows {
		if row.RawID == 3 {
			t.Fatal("removal gap included")
		}
	}
	if rows[3].Value != 17 {
		t.Fatalf("last row=%+v", rows[3])
	}
}
