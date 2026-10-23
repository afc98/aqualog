package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"github.com/afc98/aqualog/core/dbcopy"
	"sort"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{Use: "migrate", Short: "Migrate Aqualog data between database backends"}

func newSQLiteToPostgresCommand() *cobra.Command {
	var sourcePath string
	var appendMode bool
	command := &cobra.Command{
		Use: "sqlite-to-postgres", Short: "Copy SQLite to PostgreSQL; use --append to add projects to existing data", Args: cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			cfg, err := db.ResolveConfig()
			if err != nil {
				return err
			}
			if cfg.Backend != db.BackendPostgres {
				return fmt.Errorf("sqlite-to-postgres requires --database-url or AQUALOG_DATABASE_URL")
			}
			source, err := dbcopy.OpenSQLiteReadOnly(sourcePath)
			if err != nil {
				return err
			}
			defer source.Close()
			target, err := db.Open(command.Context(), cfg)
			if err != nil {
				return err
			}
			defer target.Close()
			if appendMode {
				result, err := dbcopy.AppendSQLiteToPostgres(command.Context(), source, target)
				if err != nil {
					return err
				}
				writeAppendResult(command, result)
				return nil
			}
			if err := dbcopy.SQLiteToPostgres(command.Context(), source, target); err != nil {
				return err
			}
			fmt.Fprintln(command.OutOrStdout(), "SQLite database copied to PostgreSQL successfully")
			return nil
		},
	}
	command.Flags().StringVar(&sourcePath, "source", "", "Path to the source SQLite database")
	command.Flags().BoolVar(&appendMode, "append", false, "Append all projects with new IDs; reject project-name conflicts (temporarily blocks destination writes)")
	_ = command.MarkFlagRequired("source")
	return command
}

func writeAppendResult(command *cobra.Command, result dbcopy.AppendResult) {
	fmt.Fprintln(command.OutOrStdout(), "SQLite database appended to PostgreSQL successfully")
	tables := make([]string, 0, len(result.Counts))
	for table := range result.Counts {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	for _, table := range tables {
		fmt.Fprintf(command.OutOrStdout(), "%s: %d rows imported\n", table, result.Counts[table])
	}
	ids := make([]int64, 0, len(result.ProjectIDs))
	for id := range result.ProjectIDs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, id := range ids {
		fmt.Fprintf(command.OutOrStdout(), "Project ID %d -> %d\n", id, result.ProjectIDs[id])
	}
}

func init() { migrateCmd.AddCommand(newSQLiteToPostgresCommand()); rootCmd.AddCommand(migrateCmd) }
