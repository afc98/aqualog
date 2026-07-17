package parsers

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseAquaread(path string) (*ParsedFile, error) {
	return ParseAquareadWithOptions(path, DateOptions{DateOrder: "auto"})
}

func ParseAquareadWithOptions(path string, opts DateOptions) (*ParsedFile, error) {
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
	idxPressure := indexOf(dataHeader, "Pressure (mbar)")
	idxTemp := indexOf(dataHeader, "Temp (C)")
	idxZero := indexOf(dataHeader, "Zero")

	// Optional Columns
	idxSal := indexOf(dataHeader, "Sal (PSU)")
	idxEC := indexOf(dataHeader, "EC (uS 25C)")

	if idxTimestamp == -1 || (idxLevel == -1 && idxPressure == -1) || idxTemp == -1 {
		return nil, errors.New("one or more required columns (Date & Time, Level (m) or Pressure (mbar), Temp (C)) are missing")
	}

	var records []Record
	dateOrder := normalizeDateOrder(opts.DateOrder)
	if dateOrder == "auto" {
		detected, err := detectAquareadDateOrder(lines[4:], idxTimestamp)
		if err != nil {
			return nil, err
		}
		dateOrder = detected
	}
	layouts := append(slashDateLayouts(dateOrder, true), "02-Jan-06 15:04:05")

	for _, line := range lines[4:] {
		if line == "" {
			continue
		}

		parts := splitTSV(line)

		// Drop fractional seconds from timestamp
		tsRaw := parts[idxTimestamp]
		tsParts := strings.Split(tsRaw, ".")
		tsStr := tsParts[0]

		ts, err := parseTimeWithLayouts(tsStr, layouts)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp %q: %w", tsStr, err)
		}

		// Safely read Zero column - may be missing if row truncates trailing tabs
		zeroStr := ""
		if idxZero != -1 && idxZero < len(parts) {
			zeroStr = parts[idxZero]
		}
		valueIdx := idxLevel
		unit := "m"
		kind := "manufacturer_compensated_level_m"
		if idxPressure != -1 {
			valueIdx = idxPressure
			unit = "mbar"
			kind = "absolute_pressure_mbar"
			if idxSal == -1 && idxEC == -1 {
				kind = "barometric_pressure_mbar"
			}
		}
		value := parseFloat(parts[valueIdx])

		rec := Record{
			Timestamp:       ts,
			LevelM:          value,
			TempC:           parseFloat(parts[idxTemp]),
			Zero:            parseFloat(zeroStr),
			ImportedValue:   value,
			ImportedUnit:    unit,
			MeasurementKind: kind,
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

func detectAquareadDateOrder(lines []string, dateIdx int) (string, error) {
	var dates []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := splitTSV(line)
		if dateIdx >= len(parts) {
			continue
		}
		date := strings.Split(strings.TrimSpace(parts[dateIdx]), " ")[0]
		dates = append(dates, date)
	}
	return detectSlashDateOrder(dates, "Aquaread")
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
