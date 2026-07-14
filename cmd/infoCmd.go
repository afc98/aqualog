package cmd

import (
	"aqualog/core/db"
	"fmt"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show aqualog runtime information",
	Run: func(cmd *cobra.Command, args []string) {
		path, err := db.Path()
		if err != nil {
			fmt.Println("Database path: error:", err)
			return
		}
		fmt.Println("Database path:", path)
		fmt.Println("Supported import types: aquaread, solinst, insitu")
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
