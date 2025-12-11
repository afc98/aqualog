package parsers

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// -----------------------------------------------------------------------------
// PARSER
// -----------------------------------------------------------------------------

func ParseSolinst(path string) (*ParsedFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var lines []string

	for scanner.Scan() {
		lines = append(lines, strings.TrimSpace(scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(lines) < 10 {
		return nil, errors.New("solinst: file too short or invalid")
	}

	// --------------------------------------------
	// STEP 1 — Extract metadata
	// --------------------------------------------
	md := Metadata{}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		lower := strings.ToLower(line)

		switch {
		case strings.HasPrefix(lower, "serial_number"):
			// Next line contains value
			if i+1 < len(lines) {
				md.SerialNumber = strings.TrimSpace(lines[i+1])
			}

		case strings.HasPrefix(lower, "location"):
			// Next line contains site name (Solinst "Location" is your site_ident)
			if i+1 < len(lines) {
				md.SiteIdent = strings.TrimSpace(lines[i+1])
			}
		}

		// Identify start of data header
		if strings.HasPrefix(lower, "date,time") {
			// Found header line index i
			return parseSolinstData(lines, i, md)
		}
	}

	return nil, errors.New("solinst: could not locate data header (Date,Time...)")
}

// -----------------------------------------------------------------------------
// STEP 2 — Parse data body
// -----------------------------------------------------------------------------

func parseSolinstData(lines []string, headerIndex int, md Metadata) (*ParsedFile, error) {

	headerLine := lines[headerIndex]
	headerCols := splitCSV(headerLine)

	// Column indices that we know exist in Solinst
	idxDate := indexOfContains(headerCols, "date")
	idxTime := indexOfContains(headerCols, "time")
	idxLevel := indexOfExact(headerCols, "LEVEL")
	idxTemp := indexOfContains(headerCols, "TEMPERATURE")
	idxEC := indexOfContains(headerCols, "CONDUCTIVITY") // optional

	if idxDate == -1 || idxTime == -1 || idxLevel == -1 || idxTemp == -1 {
		return nil, fmt.Errorf("solinst: required columns missing (date, time, level, temperature)")
	}

	var records []Record

	for _, line := range lines[headerIndex+1:] {
		if line == "" {
			continue
		}

		parts := splitCSV(line)
		if len(parts) < 4 {
			continue
		}

		// Timestamp (Date + Time)
		rawDate := parts[idxDate]
		rawTime := parts[idxTime]
		tsStr := fmt.Sprintf("%s %s", rawDate, rawTime)

		ts, err := parseSolinstTimestamp(tsStr)
		if err != nil {
			return nil, fmt.Errorf("solinst: invalid timestamp %q: %w", tsStr, err)
		}

		// Core values
		level := parseFloatSafe(parts[idxLevel])
		temp := parseFloatSafe(parts[idxTemp])

		rec := Record{
			Timestamp: ts,
			LevelM:    level,
			TempC:     temp,
			Zero:      0, // Solinst does not provide zero offset per record
			SalPSU:    nil,
			EC:        nil,
		}

		// Optional EC
		if idxEC != -1 && idxEC < len(parts) {
			ecVal := parseFloatPtr(parts[idxEC])
			rec.EC = ecVal
		}

		records = append(records, rec)
	}

	return &ParsedFile{
		Metadata: md,
		Records:  records,
	}, nil
}

func parseSolinstTimestamp(raw string) (time.Time, error) {
	layouts := []string{
		"02/01/2006 15:04:05", // dd/mm/yyyy
		"2006/01/02 15:04:05", // yyyy/mm/dd
	}

	for _, layout := range layouts {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts, nil
		}
	}

	return time.Time{}, fmt.Errorf("unrecognized date format: %s", raw)
}

// -----------------------------------------------------------------------------
// HELPERS
// -----------------------------------------------------------------------------

func splitCSV(s string) []string {
	// Solinst files are comma-separated
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parseFloatSafe(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// indexOfExact searches for an exact match (case-insensitive)
func indexOfExact(slice []string, target string) int {
	target = strings.ToLower(target)
	for i, v := range slice {
		if strings.ToLower(v) == target {
			return i
		}
	}
	return -1
}

// indexOfContains is more flexible (useful since solinst headers vary)
func indexOfContains(slice []string, target string) int {
	target = strings.ToLower(target)
	for i, v := range slice {
		if strings.Contains(strings.ToLower(v), target) {
			return i
		}
	}
	return -1
}
