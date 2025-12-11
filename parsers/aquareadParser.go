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

func ParseAquaread(path string) (*ParsedFile, error) {
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

	if len(lines) < 5 {
		return nil, errors.New("No data found in file")
	}

	metaHeader := splitTSV(lines[0])
	metaValues := splitTSV(lines[1])

	if len(metaHeader) != len(metaValues) {
		return nil, errors.New("Metadata header/value count mismatch")
	}

	md := Metadata{}

	for i, h := range metaHeader {
		key := strings.ToLower(strings.ReplaceAll(h, " ", "_"))
		val := metaValues[i]

		switch key {
		case "site_ident":
			md.SiteIdent = val
		case "serial_number":
			md.SerialNumber = val
		case "latitude":
			md.Latitude = parseFloat(val)
		case "longitude":
			md.Longitude = parseFloat(val)
		case "altitude":
			md.Altitude = parseFloat(val)
		}
	}

	dataHeader := splitTSV(lines[3])

	idxTimestamp := indexOf(dataHeader, "Date & Time")
	idxLevel := indexOf(dataHeader, "Level (m)")
	idxTemp := indexOf(dataHeader, "Temp (C)")
	idxZero := indexOf(dataHeader, "Zero")

	// Optional Columns
	idxSal := indexOf(dataHeader, "Sal (PSU)")
	idxEC := indexOf(dataHeader, "EC (uS 25C)")

	if idxTimestamp == -1 || idxLevel == -1 || idxTemp == -1 || idxZero == -1 {
		return nil, errors.New("One or more required columns (Date & Time, Level (m), Temp (C) and Zero) are missing")
	}

	var records []Record
	layout := "02/01/2006 15:04:05" // dd/mm/yyyy hh:mm:ss

	for _, line := range lines[4:] {
		if line == "" {
			continue
		}

		parts := splitTSV(line)

		// Drop fractional seconds from timestamp
		tsRaw := parts[idxTimestamp]
		tsParts := strings.Split(tsRaw, ".")
		tsStr := tsParts[0]

		ts, err := time.Parse(layout, tsStr)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp %q: %w", tsStr, err)
		}

		// Safely read Zero column - may be missing if row truncates trailing tabs
		zeroStr := ""
		if idxZero < len(parts) {
			zeroStr = parts[idxZero]
		}

		rec := Record{
			Timestamp: ts,
			LevelM:    parseFloat(parts[idxLevel]),
			TempC:     parseFloat(parts[idxTemp]),
			Zero:      parseFloat(zeroStr),
		}

		// Optional salinity
		if idxSal != -1 && idxSal < len(parts) {
			val := parseFloatPtr(parts[idxSal])
			rec.SalPSU = val
		}

		// Optional EC
		if idxEC != -1 && idxEC < len(parts) {
			val := parseFloatPtr(parts[idxEC])
			rec.EC = val
		}

		records = append(records, rec)
	}

	return &ParsedFile{
		Metadata: md,
		Records:  records,
	}, nil
}

func splitTSV(s string) []string {
	return strings.Split(s, "\t")
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func indexOf(slice []string, target string) int {
	for i, v := range slice {
		if v == target {
			return i
		}
	}
	return -1
}
