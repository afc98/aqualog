package utils

import (
	"fmt"
	"time"
)

const datasetTimeLayout = "2006-01-02T15:04:05.999999999"

// FormatDatasetTime writes clock fields without declaring a timezone.
func FormatDatasetTime(t time.Time) string { return t.Format(datasetTimeLayout) }

// ParseDatasetTime accepts current clock-only output and legacy offset output.
// An offset on legacy output is discarded; its displayed clock fields remain.
func ParseDatasetTime(value string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339Nano, "2006-01-02T15:04:05.999999999",
		"2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), parsed.Nanosecond(), time.UTC), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid dataset timestamp %q", value)
}
