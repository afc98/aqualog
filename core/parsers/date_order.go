package parsers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type DateOptions struct {
	DateOrder string
}

func normalizeDateOrder(order string) string {
	switch strings.ToLower(strings.TrimSpace(order)) {
	case "mdy", "us":
		return "mdy"
	case "dmy", "uk", "":
		return "dmy"
	case "auto":
		return "auto"
	default:
		return order
	}
}

var slashDatePattern = regexp.MustCompile(`^(\d{1,2})/(\d{1,2})/(\d{2}|\d{4})$`)

func detectSlashDateOrder(dates []string, label string) (string, error) {
	ambiguous := ""
	for _, raw := range dates {
		date := strings.TrimSpace(raw)
		matches := slashDatePattern.FindStringSubmatch(date)
		if matches == nil {
			continue
		}
		first, _ := strconv.Atoi(matches[1])
		second, _ := strconv.Atoi(matches[2])
		if first > 12 {
			return "dmy", nil
		}
		if second > 12 {
			return "mdy", nil
		}
		if ambiguous == "" {
			ambiguous = date
		}
	}
	if ambiguous != "" {
		return "", fmt.Errorf("ambiguous %s date format %q; re-run with --date-order dmy or --date-order mdy", label, ambiguous)
	}
	return "dmy", nil
}

func slashDateLayouts(dateOrder string, includeSeconds bool) []string {
	if includeSeconds {
		switch normalizeDateOrder(dateOrder) {
		case "mdy":
			return []string{
				"01/02/2006 15:04:05",
				"1/2/2006 15:04:05",
				"01/02/06 15:04:05",
				"1/2/06 15:04:05",
				"01/02/2006 03:04:05 pm",
				"1/2/2006 03:04:05 pm",
				"01/02/2006 3:04:05 pm",
				"1/2/2006 3:04:05 pm",
				"01/02/2006 03:04:05 PM",
				"1/2/2006 03:04:05 PM",
				"01/02/2006 3:04:05 PM",
				"1/2/2006 3:04:05 PM",
				"01/02/06 03:04:05 pm",
				"1/2/06 03:04:05 pm",
				"01/02/06 3:04:05 pm",
				"1/2/06 3:04:05 pm",
				"01/02/06 03:04:05 PM",
				"1/2/06 03:04:05 PM",
				"01/02/06 3:04:05 PM",
				"1/2/06 3:04:05 PM",
			}
		default:
			return []string{
				"02/01/2006 15:04:05",
				"2/1/2006 15:04:05",
				"02/01/06 15:04:05",
				"2/1/06 15:04:05",
				"02/01/2006 03:04:05 pm",
				"2/1/2006 03:04:05 pm",
				"02/01/2006 3:04:05 pm",
				"2/1/2006 3:04:05 pm",
				"02/01/2006 03:04:05 PM",
				"2/1/2006 03:04:05 PM",
				"02/01/2006 3:04:05 PM",
				"2/1/2006 3:04:05 PM",
				"02/01/06 03:04:05 pm",
				"2/1/06 03:04:05 pm",
				"02/01/06 3:04:05 pm",
				"2/1/06 3:04:05 pm",
				"02/01/06 03:04:05 PM",
				"2/1/06 03:04:05 PM",
				"02/01/06 3:04:05 PM",
				"2/1/06 3:04:05 PM",
			}
		}
	}
	switch normalizeDateOrder(dateOrder) {
	case "mdy":
		return []string{"01/02/2006", "01/02/06"}
	default:
		return []string{"02/01/2006", "02/01/06"}
	}
}

func parseTimeWithLayouts(raw string, layouts []string) (time.Time, error) {
	var lastErr error
	candidates := []string{raw}
	upper := strings.ReplaceAll(strings.ReplaceAll(raw, " am", " AM"), " pm", " PM")
	if upper != raw {
		candidates = append(candidates, upper)
	}
	for _, candidate := range candidates {
		for _, layout := range layouts {
			t, err := time.Parse(layout, candidate)
			if err == nil {
				return t, nil
			}
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}
