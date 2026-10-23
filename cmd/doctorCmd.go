package cmd

import (
	"fmt"
	"github.com/afc98/aqualog/core/db"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check local aqualog runtime configuration",
	Run: func(cmd *cobra.Command, args []string) {
		info, err := db.ResolveConfig()
		if err != nil {
			fmt.Println("Database configuration: error:", err)
			return
		}
		fmt.Println("Database backend:", info.Backend)
		fmt.Println("Database location:", db.RedactedLocation(info))
		fmt.Println("Database source:", info.Source)
		conn, err := db.GetDB()
		if err != nil {
			fmt.Println("Database open: error:", err)
			return
		}
		fmt.Println("Database open: OK")
		var count int
		if err := conn.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
			fmt.Println("Schema check: error:", err)
			return
		}
		fmt.Printf("Schema check: OK (migration version rows: %d)\n", count)
		if info.Backend == db.BackendSQLite {
			testFile := filepath.Join(filepath.Dir(info.SQLitePath), ".write-test")
			if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
				fmt.Println("Database directory writable: error:", err)
				return
			}
			_ = os.Remove(testFile)
			fmt.Println("Database directory writable: OK")
		}
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
