package cmd

import (
	"aqualog/core/db"
	"aqualog/core/dbcopy"
	"fmt"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{Use: "migrate", Short: "Migrate Aqualog data between database backends"}

func newSQLiteToPostgresCommand() *cobra.Command {
	var sourcePath string
	command := &cobra.Command{
		Use: "sqlite-to-postgres", Short: "Copy an Aqualog SQLite database into an empty PostgreSQL database", Args: cobra.NoArgs,
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
			if err := dbcopy.SQLiteToPostgres(command.Context(), source, target); err != nil {
				return err
			}
			fmt.Fprintln(command.OutOrStdout(), "SQLite database copied to PostgreSQL successfully")
			return nil
		},
	}
	command.Flags().StringVar(&sourcePath, "source", "", "Path to the source SQLite database")
	_ = command.MarkFlagRequired("source")
	return command
}

func init() { migrateCmd.AddCommand(newSQLiteToPostgresCommand()); rootCmd.AddCommand(migrateCmd) }
