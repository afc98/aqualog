package importer

import (
	"aqualog/core/parsers"
	"strings"
	"testing"
	"time"
)

func TestTimezoneImportDefaultAndTarget(t *testing.T) {
	source := time.Date(2026, 7, 31, 6, 0, 0, 0, time.FixedZone("source", 3600))
	rows := []parsers.Record{{Timestamp: source, TimezonePresent: true}}
	normal := normalizeImportedTimes(append([]parsers.Record(nil), rows...), nil)
	if got := normal[0].Timestamp.Format("2006-01-02 15:04:05"); got != "2026-07-31 06:00:00" {
		t.Fatal(got)
	}
	london, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	converted := normalizeImportedTimes(append([]parsers.Record(nil), rows...), london)
	if got := converted[0].Timestamp.Format("2006-01-02 15:04:05"); got != "2026-07-31 01:00:00" {
		t.Fatal(got)
	}
	if TimezoneWarning(normal, "") == "" || TimezoneWarning(converted, "America/New_York") == "" {
		t.Fatal("missing warning")
	}
}

func TestTimezoneImportWinterAndMixedRows(t *testing.T) {
	target, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	rows := []parsers.Record{
		{Timestamp: time.Date(2026, 1, 31, 6, 0, 0, 0, time.FixedZone("source", 3600)), TimezonePresent: true},
		{Timestamp: time.Date(2026, 1, 31, 8, 0, 0, 0, time.UTC)},
	}
	converted := normalizeImportedTimes(rows, target)
	if got := converted[0].Timestamp.Format("2006-01-02 15:04:05"); got != "2026-01-31 05:00:00" {
		t.Fatal(got)
	}
	if got := converted[1].Timestamp.Format("2006-01-02 15:04:05"); got != "2026-01-31 08:00:00" {
		t.Fatal(got)
	}
	if warning := TimezoneWarning(converted, "Europe/London"); !strings.Contains(warning, "1 imported timestamps") {
		t.Fatal(warning)
	}
}
