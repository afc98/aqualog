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
		info, err := db.ResolvePath()
		if err != nil {
			fmt.Println("Database path: error:", err)
			return
		}
		fmt.Println("Database path:", info.Path)
		fmt.Println("Database source:", info.Source)
		fmt.Println("Supported import types: aquaread, solinst, insitu")
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
