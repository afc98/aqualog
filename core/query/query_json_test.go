package query

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDatasetJSONHasNoTimezoneSuffix(t *testing.T) {
	row := LoggerDataRow{Timestamp: time.Date(2026, 7, 31, 6, 0, 0, 0, time.UTC)}
	data, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"Timestamp":"2026-07-31T06:00:00"`) {
		t.Fatal(string(data))
	}
}
