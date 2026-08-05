package cmd

import (
	"aqualog/core/db"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check local aqualog runtime configuration",
	Run: func(cmd *cobra.Command, args []string) {
		info, err := db.ResolvePath()
		if err != nil {
			fmt.Println("Database path: error:", err)
			return
		}
		path := info.Path
		fmt.Println("Database path:", path)
		fmt.Println("Database source:", info.Source)
		conn, err := db.GetDB()
		if err != nil {
			fmt.Println("Database open: error:", err)
			return
		}
		defer conn.Close()
		fmt.Println("Database open: OK")
		var count int
		if err := conn.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table'`).Scan(&count); err != nil {
			fmt.Println("Schema check: error:", err)
			return
		}
		fmt.Printf("Schema check: OK (%d tables)\n", count)
		testFile := filepath.Join(filepath.Dir(path), ".write-test")
		if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
			fmt.Println("Database directory writable: error:", err)
			return
		}
		_ = os.Remove(testFile)
		fmt.Println("Database directory writable: OK")
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
