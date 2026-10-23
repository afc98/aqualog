package dbcopy

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// AppendResult describes a committed append. IDs are scoped to their table.
type AppendResult struct {
	Counts     map[string]int64
	ProjectIDs map[int64]int64
}

type appendReference struct {
	column, parent string
	nullable       bool
}

var appendReferences = map[string][]appendReference{
	"sites":            {{"project_id", "projects", false}},
	"loggers":          {{"site_id", "sites", false}},
	"logger_files":     {{"logger_id", "loggers", false}},
	"correction_runs":  {{"site_id", "sites", false}, {"logger_id", "loggers", true}, {"barologger_id", "loggers", false}},
	"logger_data":      {{"logger_id", "loggers", false}, {"logger_file_id", "logger_files", false}, {"correction_run_id", "correction_runs", true}},
	"logger_events":    {{"logger_id", "loggers", false}},
	"manual_readings":  {{"site_id", "sites", false}},
	"corrected_data":   {{"site_id", "sites", false}, {"logger_data_id", "logger_data", false}},
	"baro_assignments": {{"barologger_id", "loggers", false}},
}

// AppendSQLiteToPostgres copies all source records with new destination IDs.
// Existing rows are untouched. Failed transactions may consume identity values.
func AppendSQLiteToPostgres(ctx context.Context, source, target *sql.DB) (AppendResult, error) {
	failed := AppendResult{}
	if source == nil || target == nil {
		return failed, fmt.Errorf("source and target databases are required")
	}
	sourceTx, err := source.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return failed, fmt.Errorf("begin SQLite read transaction: %w", err)
	}
	defer sourceTx.Rollback()
	if err := validateSQLiteSource(ctx, sourceTx); err != nil {
		return failed, err
	}
	tx, err := target.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return failed, fmt.Errorf("begin PostgreSQL append transaction: %w", err)
	}
	defer tx.Rollback()
	names := make([]string, 0, len(copyOrder))
	for _, table := range copyOrder {
		names = append(names, table.name)
	}
	if _, err := tx.ExecContext(ctx, "LOCK TABLE "+strings.Join(names, ", ")+" IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		return failed, fmt.Errorf("lock PostgreSQL append tables: %w", err)
	}
	if err := checkAppendProjectNames(ctx, sourceTx, tx); err != nil {
		return failed, err
	}
	// Transaction-local tracking keeps validation limited to this import, even
	// when unrelated historical rows have unenforced, invalid references.
	if _, err := tx.ExecContext(ctx, "CREATE TEMP TABLE aqualog_append_rows (table_name TEXT NOT NULL, id INTEGER NOT NULL, PRIMARY KEY (table_name, id)) ON COMMIT DROP"); err != nil {
		return failed, fmt.Errorf("create append tracking table: %w", err)
	}
	result := AppendResult{Counts: map[string]int64{}}
	ids := map[string]map[int64]int64{}
	for _, refs := range appendReferences {
		for _, ref := range refs {
			ids[ref.parent] = map[int64]int64{}
		}
	}
	before := map[string]int64{}
	for _, table := range copyOrder {
		var count int64
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table.name).Scan(&count); err != nil {
			return failed, fmt.Errorf("count PostgreSQL table %s: %w", table.name, err)
		}
		before[table.name] = count
		copied, err := appendTable(ctx, sourceTx, tx, table, ids)
		if err != nil {
			return failed, err
		}
		result.Counts[table.name] = copied
	}
	if err := validateAppend(ctx, tx, before, result.Counts); err != nil {
		return failed, err
	}
	if err := tx.Commit(); err != nil {
		return failed, fmt.Errorf("commit PostgreSQL append: %w", err)
	}
	result.ProjectIDs = ids["projects"]
	return result, nil
}

func checkAppendProjectNames(ctx context.Context, source, target *sql.Tx) error {
	rows, err := source.QueryContext(ctx, "SELECT name FROM projects ORDER BY id")
	if err != nil {
		return fmt.Errorf("read source project names: %w", err)
	}
	defer rows.Close()
	seen, conflicts := map[string]bool{}, map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		var exists bool
		if err := target.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM projects WHERE name COLLATE \"C\" = $1)", name).Scan(&exists); err != nil {
			return fmt.Errorf("check project name: %w", err)
		}
		if exists || seen[name] {
			conflicts[name] = true
		}
		seen[name] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(conflicts) > 0 {
		names := make([]string, 0, len(conflicts))
		for name := range conflicts {
			names = append(names, fmt.Sprintf("%q", name))
		}
		sort.Strings(names)
		return fmt.Errorf("append rejected: conflicting project names (in source or destination): %s", strings.Join(names, ", "))
	}
	return nil
}

func remapAppendRow(table tableSpec, values []any, ids map[string]map[int64]int64) error {
	indexes := make(map[string]int, len(table.columns))
	for i, column := range table.columns {
		indexes[column] = i
	}
	refs := appendReferences[table.name]
	if table.name == "baro_assignments" {
		parent := ""
		switch values[indexes["scope"]] {
		case "project":
			parent = "projects"
		case "site":
			parent = "sites"
		default:
			return fmt.Errorf("table %s source row %v: unsupported scope %v", table.name, values[0], values[indexes["scope"]])
		}
		refs = append(append([]appendReference{}, refs...), appendReference{"scope_id", parent, false})
	}
	for _, ref := range refs {
		i := indexes[ref.column]
		if values[i] == nil && ref.nullable {
			continue
		}
		oldID, ok := values[i].(int64)
		newID, found := ids[ref.parent][oldID]
		if !ok || !found {
			return fmt.Errorf("table %s source row %v: %s=%v has no source record in %s", table.name, values[0], ref.column, values[i], ref.parent)
		}
		values[i] = newID
	}
	return nil
}

func appendTable(ctx context.Context, source, target *sql.Tx, table tableSpec, ids map[string]map[int64]int64) (int64, error) {
	rows, err := source.QueryContext(ctx, "SELECT "+strings.Join(table.columns, ", ")+" FROM "+table.name+" ORDER BY id")
	if err != nil {
		return 0, fmt.Errorf("read SQLite table %s: %w", table.name, err)
	}
	defer rows.Close()
	values, destinations := make([]any, len(table.columns)), make([]any, len(table.columns))
	placeholders := make([]string, len(table.columns)-1)
	for i := range values {
		destinations[i] = &values[i]
		if i > 0 {
			placeholders[i-1] = fmt.Sprintf("$%d", i)
		}
	}
	query := "WITH inserted AS (INSERT INTO " + table.name + " (" + strings.Join(table.columns[1:], ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ") RETURNING id) INSERT INTO pg_temp.aqualog_append_rows (table_name, id) SELECT '" + table.name + "', id FROM inserted RETURNING id"
	var count int64
	for rows.Next() {
		if err := rows.Scan(destinations...); err != nil {
			return 0, fmt.Errorf("read SQLite row from %s: %w", table.name, err)
		}
		oldID, ok := values[0].(int64)
		if !ok {
			return 0, fmt.Errorf("table %s: invalid source ID %v", table.name, values[0])
		}
		if err := remapAppendRow(table, values, ids); err != nil {
			return 0, err
		}
		var newID int64
		if err := target.QueryRowContext(ctx, query, values[1:]...).Scan(&newID); err != nil {
			return 0, fmt.Errorf("append table %s source row %d: %w", table.name, oldID, err)
		}
		if ids[table.name] != nil {
			ids[table.name][oldID] = newID
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("read SQLite table %s: %w", table.name, err)
	}
	return count, nil
}

func validateAppend(ctx context.Context, tx *sql.Tx, before, copied map[string]int64) error {
	for _, table := range copyOrder {
		var count int64
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table.name).Scan(&count); err != nil {
			return err
		}
		if count != before[table.name]+copied[table.name] {
			return fmt.Errorf("append row-count mismatch for %s: expected %d, found %d", table.name, before[table.name]+copied[table.name], count)
		}
		refs := appendReferences[table.name]
		if table.name == "baro_assignments" {
			refs = append(append([]appendReference{}, refs...), appendReference{"scope_id", "projects", false}, appendReference{"scope_id", "sites", false})
		}
		for _, ref := range refs {
			filter := "TRUE"
			if ref.nullable {
				filter = "c." + ref.column + " IS NOT NULL"
			}
			if ref.column == "scope_id" {
				scope := "project"
				if ref.parent == "sites" {
					scope = "site"
				}
				filter = "c.scope = '" + scope + "'"
			}
			query := "SELECT EXISTS (SELECT 1 FROM " + table.name + " c JOIN pg_temp.aqualog_append_rows imported ON imported.table_name = '" + table.name + "' AND imported.id = c.id LEFT JOIN " + ref.parent + " p ON p.id = c." + ref.column + " LEFT JOIN pg_temp.aqualog_append_rows parent_import ON parent_import.table_name = '" + ref.parent + "' AND parent_import.id = p.id WHERE " + filter + " AND parent_import.id IS NULL)"
			var invalid bool
			if err := tx.QueryRowContext(ctx, query).Scan(&invalid); err != nil {
				return fmt.Errorf("validate appended %s.%s: %w", table.name, ref.column, err)
			}
			if invalid {
				return fmt.Errorf("invalid appended relationship %s.%s to %s", table.name, ref.column, ref.parent)
			}
		}
	}
	return nil
}
