package parsers

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ParseInsitu parses a CSV file from an In-Situ datalogger.
func ParseInsitu(path string) (*ParsedFile, error) {
	return ParseInsituWithOptions(path, DateOptions{DateOrder: "auto"})
}

func ParseInsituWithOptions(path string, opts DateOptions) (*ParsedFile, error) {
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
		return nil, errors.New("insitu: file too short or invalid")
	}

	md := Metadata{}
	headerIndex := -1
	inLogData := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "Log Data:") {
			inLogData = true
			continue
		}

		// Identify start of data header - must be in Log Data section and have multiple columns
		if inLogData && strings.HasPrefix(line, "Date and Time") {
			parts := splitCSVInsitu(line)
			if len(parts) > 3 {
				headerIndex = i
				break
			}
		}

		parts := splitCSVInsitu(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])

		switch key {
		case "serial number":
			md.SerialNumber = val
		case "site":
			md.SiteIdent = val
		}
	}

	if headerIndex == -1 {
		return nil, errors.New("insitu: could not locate data header (Date and Time...)")
	}

	return parseInsituData(lines, headerIndex, md, normalizeDateOrder(opts.DateOrder))
}

func parseInsituData(lines []string, headerIndex int, md Metadata, dateOrder string) (*ParsedFile, error) {
	headerLine := lines[headerIndex]
	headerCols := splitCSVInsitu(headerLine)

	idxTimestamp := indexOfContainsInsitu(headerCols, "date")
	idxPressure := indexOfContainsInsitu(headerCols, "pressure (kpa)")
	idxBaro := indexOfContainsInsitu(headerCols, "barometric pressure")
	idxLevel := indexOfContainsInsitu(headerCols, "depth")
	if idxLevel == -1 {
		idxLevel = indexOfContainsInsitu(headerCols, "level")
	}
	idxTemp := indexOfContainsInsitu(headerCols, "temperature")

	// Optional columns
	idxEC := indexOfContainsInsitu(headerCols, "conductivity")
	idxSal := indexOfContainsInsitu(headerCols, "salinity")

	if idxTimestamp == -1 || idxTemp == -1 || (idxLevel == -1 && idxPressure == -1 && idxBaro == -1) {
		return nil, fmt.Errorf("insitu: required columns missing (date, pressure/depth/barometric pressure, temperature) - found: %v", headerCols)
	}
	if dateOrder == "auto" {
		detected, err := detectInsituDateOrder(lines[headerIndex+1:], idxTimestamp)
		if err != nil {
			return nil, err
		}
		dateOrder = detected
	}
	layouts := slashDateLayouts(dateOrder, true)

	var records []Record

	for _, line := range lines[headerIndex+1:] {
		if line == "" {
			continue
		}

		parts := splitCSVInsitu(line)
		if len(parts) <= idxTimestamp || len(parts) <= idxTemp {
			continue
		}

		tsStr := parts[idxTimestamp]
		ts, err := parseTimeWithLayouts(tsStr, layouts)
		if err != nil {
			continue
		}

		valueIdx := idxLevel
		unit := "m"
		kind := "manufacturer_compensated_depth_m"
		if idxBaro != -1 {
			valueIdx = idxBaro
			unit = "kPa"
			kind = "barometric_pressure_kpa"
		} else if idxPressure != -1 {
			valueIdx = idxPressure
			unit = "kPa"
			kind = "absolute_pressure_kpa"
		}
		if valueIdx == -1 || len(parts) <= valueIdx {
			continue
		}
		level := parseFloatInsitu(parts[valueIdx])
		temp := parseFloatInsitu(parts[idxTemp])

		rec := Record{
			Timestamp:       ts,
			LevelM:          level,
			TempC:           temp,
			Zero:            0,
			ImportedValue:   level,
			ImportedUnit:    unit,
			MeasurementKind: kind,
		}

		if idxEC != -1 && idxEC < len(parts) {
			rec.EC = parseFloatPtrInsitu(parts[idxEC])
		}
		if idxSal != -1 && idxSal < len(parts) {
			rec.SalPSU = parseFloatPtrInsitu(parts[idxSal])
		}

		records = append(records, rec)
	}

	return &ParsedFile{
		Metadata: md,
		Records:  records,
	}, nil
}

func detectInsituDateOrder(lines []string, dateIdx int) (string, error) {
	var dates []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := splitCSVInsitu(line)
		if dateIdx >= len(parts) {
			continue
		}
		date := strings.Split(strings.TrimSpace(parts[dateIdx]), " ")[0]
		dates = append(dates, date)
	}
	return detectSlashDateOrder(dates, "In-Situ")
}

func splitCSVInsitu(s string) []string {
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parseFloatInsitu(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func parseFloatPtrInsitu(s string) *float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

func indexOfContainsInsitu(slice []string, target string) int {
	target = strings.ToLower(target)
	for i, v := range slice {
		if strings.Contains(strings.ToLower(v), target) {
			return i
		}
	}
	return -1
}
