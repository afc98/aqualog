package qa

import (
	"database/sql"
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/utils"
	"sort"
	"time"
)

type Status struct {
	SiteID             int
	SiteName           string
	ProjectID          int
	ProjectName        string
	LoggerCount        int
	RawCount           int
	CorrectedCount     int
	ManualCount        int
	EventCount         int
	FileCount          int
	RawStart           sql.NullString
	RawEnd             sql.NullString
	CorrectedStart     sql.NullString
	CorrectedEnd       sql.NullString
	CorrectedMissing   bool
	AbsoluteCount      int
	BaroCount          int
	AvailableBaroCount int
	BaroCorrected      int
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

	var s Status
	err = d.QueryRow(`
		SELECT s.id, s.name, p.id, p.name
		FROM sites s
		JOIN projects p ON p.id = s.project_id
		WHERE s.id = $1
	`, siteID).Scan(&s.SiteID, &s.SiteName, &s.ProjectID, &s.ProjectName)
	if err != nil {
		return Status{}, err
	}

	counts := []struct {
		query string
		dest  *int
	}{
		{`SELECT COUNT(*) FROM loggers WHERE site_id = $1`, &s.LoggerCount},
		{`SELECT COUNT(*) FROM logger_data WHERE logger_id IN (SELECT id FROM loggers WHERE site_id = $1 AND COALESCE(NULLIF(role, ''), 'water_level') = 'water_level')`, &s.RawCount},
		{`SELECT COUNT(*) FROM corrected_data WHERE site_id = $1`, &s.CorrectedCount},
		{`SELECT COUNT(*) FROM manual_readings WHERE site_id = $1`, &s.ManualCount},
		{`SELECT COUNT(*) FROM logger_events WHERE logger_id IN (SELECT id FROM loggers WHERE site_id = $1 AND COALESCE(NULLIF(role, ''), 'water_level') = 'water_level')`, &s.EventCount},
		{`SELECT COUNT(*) FROM logger_files WHERE logger_id IN (SELECT id FROM loggers WHERE site_id = $1)`, &s.FileCount},
		{`SELECT COUNT(*) FROM logger_data WHERE logger_id IN (SELECT id FROM loggers WHERE site_id = $1) AND imported_kind IN ('absolute_pressure_mbar', 'absolute_pressure_kpa', 'absolute_pressure_head_m')`, &s.AbsoluteCount},
		{`SELECT COUNT(*) FROM logger_data WHERE logger_id IN (SELECT id FROM loggers WHERE site_id = $1) AND imported_kind IN ('barometric_pressure_mbar', 'barometric_pressure_kpa')`, &s.BaroCount},
		{`SELECT COUNT(*) FROM logger_data WHERE logger_id IN (SELECT id FROM loggers WHERE site_id = $1) AND baro_corrected_m IS NOT NULL`, &s.BaroCorrected},
	}
	for _, c := range counts {
		if err := d.QueryRow(c.query, siteID).Scan(c.dest); err != nil {
			return Status{}, err
		}
	}
	if err := d.QueryRow(`
		SELECT COUNT(*)
		FROM logger_data d
		WHERE d.imported_kind IN ('barometric_pressure_mbar', 'barometric_pressure_kpa')
		  AND d.logger_id IN (
			SELECT id FROM loggers
			WHERE site_id = $1 AND role = 'barometric'
			UNION
			SELECT barologger_id FROM baro_assignments
			WHERE scope = 'site' AND scope_id = $2
			UNION
			SELECT a.barologger_id
			FROM baro_assignments a
			JOIN sites s ON s.project_id = a.scope_id
			WHERE a.scope = 'project' AND s.id = $3
			UNION
			SELECT barologger_id FROM correction_runs WHERE site_id = $4
		  )
	`, siteID, siteID, siteID, siteID).Scan(&s.AvailableBaroCount); err != nil {
		return Status{}, err
	}

	if err := d.QueryRow(`SELECT MIN(timestamp), MAX(timestamp) FROM logger_data WHERE logger_id IN (SELECT id FROM loggers WHERE site_id = $1)`, siteID).Scan(&s.RawStart, &s.RawEnd); err != nil {
		return Status{}, err
	}
	if err := d.QueryRow(`SELECT MIN(timestamp), MAX(timestamp) FROM corrected_data WHERE site_id = $1`, siteID).Scan(&s.CorrectedStart, &s.CorrectedEnd); err != nil {
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
	if status.AbsoluteCount > 0 && status.BaroCorrected == 0 {
		add("error", "site has absolute-pressure logger data but no barometric correction")
	}
	if status.AbsoluteCount > 0 && status.AvailableBaroCount == 0 {
		add("error", "site has absolute-pressure logger data but no barometric records available through the site, an assignment, or correction history")
	}

	d, err := db.GetDB()
	if err != nil {
		return nil, err
	}

	rows, err := d.Query(`
		SELECT e.logger_id, e.timestamp, e.event_type, l.name
		FROM logger_events e
		JOIN loggers l ON l.id = e.logger_id
		WHERE l.site_id = $1
		  AND COALESCE(NULLIF(l.role, ''), 'water_level') = 'water_level'
		ORDER BY e.logger_id, e.timestamp
	`, siteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type loggerState struct {
		name string
		open bool
	}
	states := make(map[int]loggerState)
	for rows.Next() {
		var loggerID int
		var ts, typ, loggerName string
		if err := rows.Scan(&loggerID, &ts, &typ, &loggerName); err != nil {
			return nil, err
		}
		state := states[loggerID]
		state.name = loggerName
		switch typ {
		case "installed":
			if state.open {
				add("error", fmt.Sprintf("logger %q has an install event before the previous install was removed at %s", loggerName, ts))
			}
			state.open = true
		case "removed":
			if !state.open {
				add("error", fmt.Sprintf("logger %q has a removal without a preceding install at %s", loggerName, ts))
			}
			state.open = false
		}
		states[loggerID] = state
	}
	for _, state := range states {
		if state.open {
			add("error", fmt.Sprintf("logger %q has an install event without a matching removal event", state.name))
		}
	}

	dupRows, err := d.Query(`SELECT name, COUNT(*) FROM loggers WHERE site_id = $1 GROUP BY name HAVING COUNT(*) > 1`, siteID)
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

	args := []any{siteID}
	filter := ""
	if loggerID > 0 {
		filter = " AND l.id = $2"
		args = append(args, loggerID)
	}
	rows, err := d.Query(`
		SELECT l.id, l.name, d.timestamp
		FROM logger_data d
		JOIN loggers l ON l.id = d.logger_id
		WHERE l.site_id = $1`+filter+`
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
		t, err := utils.ParseDatasetTime(ts)
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
