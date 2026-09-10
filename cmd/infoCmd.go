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
		info, err := db.ResolveConfig()
		if err != nil {
			fmt.Println("Database configuration: error:", err)
			return
		}
		fmt.Println("Database backend:", info.Backend)
		fmt.Println("Database location:", db.RedactedLocation(info))
		fmt.Println("Database source:", info.Source)
		fmt.Println("Supported import types: aquaread, solinst, insitu")
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
