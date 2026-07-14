package qa

import (
	"aqualog/core/db"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

type Status struct {
	SiteID           int
	SiteName         string
	ProjectID        int
	ProjectName      string
	LoggerCount      int
	RawCount         int
	CorrectedCount   int
	ManualCount      int
	EventCount       int
	FileCount        int
	RawStart         sql.NullString
	RawEnd           sql.NullString
	CorrectedStart   sql.NullString
	CorrectedEnd     sql.NullString
	CorrectedMissing bool
}

type ValidationIssue struct {
	Severity string
	Message  string
}

type Gap struct {
	LoggerID   int
	LoggerName string
	Start      time.Time
	End        time.Time
	Duration   time.Duration
}

func SiteStatus(siteID int) (Status, error) {
	d, err := db.GetDB()
	if err != nil {
		return Status{}, err
	}
	defer d.Close()

	var s Status
	err = d.QueryRow(`
		SELECT s.id, s.name, p.id, p.name
		FROM sites s
		JOIN projects p ON p.id = s.project_id
		WHERE s.id = ?
	`, siteID).Scan(&s.SiteID, &s.SiteName, &s.ProjectID, &s.ProjectName)
	if err != nil {
		return Status{}, err
	}

	counts := []struct {
		query string
		dest  *int
	}{
		{`SELECT COUNT(*) FROM loggers WHERE site_id = ?`, &s.LoggerCount},
		{`SELECT COUNT(*) FROM logger_data WHERE logger_id IN (SELECT id FROM loggers WHERE site_id = ?)`, &s.RawCount},
		{`SELECT COUNT(*) FROM corrected_data WHERE site_id = ?`, &s.CorrectedCount},
		{`SELECT COUNT(*) FROM manual_readings WHERE site_id = ?`, &s.ManualCount},
		{`SELECT COUNT(*) FROM logger_events WHERE logger_id IN (SELECT id FROM loggers WHERE site_id = ?)`, &s.EventCount},
		{`SELECT COUNT(*) FROM logger_files WHERE logger_id IN (SELECT id FROM loggers WHERE site_id = ?)`, &s.FileCount},
	}
	for _, c := range counts {
		if err := d.QueryRow(c.query, siteID).Scan(c.dest); err != nil {
			return Status{}, err
		}
	}

	if err := d.QueryRow(`SELECT MIN(timestamp), MAX(timestamp) FROM logger_data WHERE logger_id IN (SELECT id FROM loggers WHERE site_id = ?)`, siteID).Scan(&s.RawStart, &s.RawEnd); err != nil {
		return Status{}, err
	}
	if err := d.QueryRow(`SELECT MIN(timestamp), MAX(timestamp) FROM corrected_data WHERE site_id = ?`, siteID).Scan(&s.CorrectedStart, &s.CorrectedEnd); err != nil {
		return Status{}, err
	}
	s.CorrectedMissing = s.RawCount > 0 && s.CorrectedCount == 0
	return s, nil
}

func ValidateSite(siteID int) ([]ValidationIssue, error) {
	status, err := SiteStatus(siteID)
	if err != nil {
		return nil, err
	}
	var issues []ValidationIssue
	add := func(sev, msg string) { issues = append(issues, ValidationIssue{Severity: sev, Message: msg}) }

	if status.LoggerCount == 0 {
		add("error", "site has no loggers")
	}
	if status.RawCount == 0 {
		add("error", "site has no raw logger data")
	}
	if status.ManualCount == 0 {
		add("error", "site has no manual readings for level correction")
	}
	if status.EventCount == 0 {
		add("error", "site has no logger install/remove events")
	}
	if status.CorrectedMissing {
		add("warning", "site has raw data but no corrected data; run process")
	}

	d, err := db.GetDB()
	if err != nil {
		return nil, err
	}
	defer d.Close()

	rows, err := d.Query(`
		SELECT e.timestamp, e.event_type, l.name
		FROM logger_events e
		JOIN loggers l ON l.id = e.logger_id
		WHERE l.site_id = ?
		ORDER BY e.timestamp
	`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	open := false
	for rows.Next() {
		var ts, typ, loggerName string
		if err := rows.Scan(&ts, &typ, &loggerName); err != nil {
			return nil, err
		}
		switch typ {
		case "installed":
			if open {
				add("error", fmt.Sprintf("logger %q has an install event before the previous install was removed at %s", loggerName, ts))
			}
			open = true
		case "removed":
			if !open {
				add("error", fmt.Sprintf("logger %q has a removal without a preceding install at %s", loggerName, ts))
			}
			open = false
		}
	}
	if open {
		add("error", "site has an install event without a matching removal event")
	}

	dupRows, err := d.Query(`SELECT name, COUNT(*) FROM loggers WHERE site_id = ? GROUP BY name HAVING COUNT(*) > 1`, siteID)
	if err != nil {
		return nil, err
	}
	defer dupRows.Close()
	for dupRows.Next() {
		var name string
		var count int
		if err := dupRows.Scan(&name, &count); err != nil {
			return nil, err
		}
		add("warning", fmt.Sprintf("logger name %q is used %d times at this site", name, count))
	}

	return issues, nil
}

func FindGaps(siteID int, loggerID int, threshold time.Duration) ([]Gap, error) {
	d, err := db.GetDB()
	if err != nil {
		return nil, err
	}
	defer d.Close()

	args := []any{siteID}
	filter := ""
	if loggerID > 0 {
		filter = " AND l.id = ?"
		args = append(args, loggerID)
	}
	rows, err := d.Query(`
		SELECT l.id, l.name, d.timestamp
		FROM logger_data d
		JOIN loggers l ON l.id = d.logger_id
		WHERE l.site_id = ?`+filter+`
		ORDER BY l.id, d.timestamp
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type point struct {
		loggerID int
		name     string
		t        time.Time
	}
	byLogger := map[int][]point{}
	for rows.Next() {
		var p point
		var ts string
		if err := rows.Scan(&p.loggerID, &p.name, &ts); err != nil {
			return nil, err
		}
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return nil, err
		}
		p.t = t
		byLogger[p.loggerID] = append(byLogger[p.loggerID], p)
	}

	var gaps []Gap
	for id, points := range byLogger {
		if len(points) < 2 {
			continue
		}
		limit := threshold
		if limit == 0 {
			var deltas []time.Duration
			for i := 1; i < len(points); i++ {
				deltas = append(deltas, points[i].t.Sub(points[i-1].t))
			}
			sort.Slice(deltas, func(i, j int) bool { return deltas[i] < deltas[j] })
			limit = deltas[len(deltas)/2] * 2
		}
		for i := 1; i < len(points); i++ {
			delta := points[i].t.Sub(points[i-1].t)
			if delta > limit {
				gaps = append(gaps, Gap{
					LoggerID:   id,
					LoggerName: points[i].name,
					Start:      points[i-1].t,
					End:        points[i].t,
					Duration:   delta,
				})
			}
		}
	}
	return gaps, nil
}
