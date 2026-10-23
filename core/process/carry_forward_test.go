package process

import (
	"database/sql"
	"strings"
	"testing"
	"time"
)

func carryTime(value string) time.Time {
	t, _ := time.Parse(time.RFC3339, "2026-01-01T"+value+"Z")
	return t
}

func seedCarry(t *testing.T) *sql.DB {
	t.Helper()
	database := newProcessTestDB(t)
	seedProcessSite(t, database, true)
	_, err := database.Exec(`
 INSERT INTO logger_events(logger_id,timestamp,event_type) VALUES
 (2,'2026-01-01T00:20:00Z','installed'),(2,'2026-01-01T02:00:00Z','removed');
 INSERT INTO logger_data(logger_id,logger_file_id,timestamp,level_m) VALUES
 (2,2,'2026-01-01T00:21:00Z',30),(2,2,'2026-01-01T00:22:00Z',31);
 INSERT INTO corrected_data(site_id,logger_data_id,timestamp,corrected_value)
 VALUES(1,1,'2026-01-01T00:01:00Z',999);
 `)
	if err != nil {
		t.Fatal(err)
	}
	return database
}

func TestCarryForwardUsesFreshPreviousIntervalAndReprocesses(t *testing.T) {
	database := seedCarry(t)
	result, err := CarryForward(database, 1, 2, carryTime("00:21:00"), time.Hour, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.Level != 16 || !result.SourceTime.Equal(carryTime("00:02:00")) || !result.Reprocessed {
		t.Fatalf("result: %+v", result)
	}
	var notes string
	if err := database.QueryRow(`SELECT notes FROM manual_readings WHERE id=$1`, result.ReadingID).Scan(&notes); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notes, "Estimated reading") || !strings.Contains(notes, "not a field measurement") || !strings.Contains(notes, "00:02:00") {
		t.Fatal(notes)
	}
	var value float64
	if err := database.QueryRow(`SELECT corrected_value FROM corrected_data WHERE logger_data_id=3`).Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != 16 {
		t.Fatalf("post-install corrected level = %v", value)
	}
	var count int
	database.QueryRow(`SELECT count(*) FROM logger_events`).Scan(&count)
	if count != 4 {
		t.Fatalf("events changed: %d", count)
	}
	if _, err := CarryForward(database, 1, 2, carryTime("00:21:00"), time.Hour, true); err == nil {
		t.Fatal("accepted duplicate reading")
	}
}

func TestCarryForwardLookbackIncludesGapAndSourceAge(t *testing.T) {
	for _, test := range []struct {
		name  string
		limit time.Duration
		ok    bool
	}{
		{"outside_previous_interval", 10 * time.Minute, false},
		{"interval_recent_but_reading_old", 18 * time.Minute, false},
		{"exact_boundary", 19 * time.Minute, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			database := seedCarry(t)
			_, err := CarryForward(database, 1, 2, carryTime("00:21:00"), test.limit, false)
			if (err == nil) != test.ok {
				t.Fatalf("error = %v", err)
			}
			var count int
			database.QueryRow(`SELECT count(*) FROM manual_readings`).Scan(&count)
			expected := 1
			if test.ok {
				expected = 2
			}
			if count != expected {
				t.Fatalf("readings=%d", count)
			}
			var value float64
			database.QueryRow(`SELECT corrected_value FROM corrected_data`).Scan(&value)
			if value != 999 {
				t.Fatal("manual-only operation changed processed results")
			}
		})
	}
}

func TestCarryForwardFailureRollsBackReadingAndCorrectedData(t *testing.T) {
	database := seedCarry(t)
	// Force storage failure after the reading is inserted and old corrected
	// rows are deleted; the transaction must restore both tables.
	if _, err := database.Exec(`CREATE TRIGGER reject_corrected BEFORE INSERT ON corrected_data BEGIN SELECT RAISE(ABORT, 'test storage failure'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := CarryForward(database, 1, 2, carryTime("00:21:00"), time.Hour, true); err == nil {
		t.Fatal("expected processing failure")
	}
	var count int
	database.QueryRow(`SELECT count(*) FROM manual_readings`).Scan(&count)
	if count != 1 {
		t.Fatalf("partial reading saved: %d", count)
	}
	var value float64
	database.QueryRow(`SELECT corrected_value FROM corrected_data`).Scan(&value)
	if value != 999 {
		t.Fatal("existing corrected data changed")
	}
}

func TestCarryForwardDoesNotSkipEmptyPreviousInterval(t *testing.T) {
	database := seedCarry(t)
	if _, err := database.Exec(`INSERT INTO logger_events(logger_id,timestamp,event_type) VALUES (2,'2026-01-01T00:11:00Z','installed'),(2,'2026-01-01T00:12:00Z','removed')`); err != nil {
		t.Fatal(err)
	}
	if _, err := CarryForward(database, 1, 2, carryTime("00:21:00"), time.Hour, false); err == nil || !strings.Contains(err.Error(), "previous interval has no corrected level") {
		t.Fatalf("error = %v", err)
	}
}

func TestCarryForwardRejectsGapAndFirstInterval(t *testing.T) {
	for _, stamp := range []string{"00:01:00", "00:15:00", "02:00:00"} {
		t.Run(stamp, func(t *testing.T) {
			database := seedCarry(t)
			if _, err := CarryForward(database, 1, 2, carryTime(stamp), time.Hour, false); err == nil {
				t.Fatal("accepted invalid interval")
			}
		})
	}
}
