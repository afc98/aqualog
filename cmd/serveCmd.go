package cmd

import (
	"fmt"

	"aqualog/core/db"
	"aqualog/core/server"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the aqualog HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := db.ResolveConfig()
		if err != nil {
			return err
		}
		if cfg.Backend != db.BackendSQLite {
			return fmt.Errorf("serve currently supports only the SQLite backend")
		}

		addr, _ := cmd.Flags().GetString("addr")

		srv := server.New(addr, server.Routes())

		fmt.Println("Starting server on", addr)
		return srv.Start()
	},
}

func init() {
	serveCmd.Flags().String("addr", "127.0.0.1:8080", "Listen address")
	rootCmd.AddCommand(serveCmd)
}
