package server

import (
	"net/http"

	"aqualog/core/server/handlers"
)

func Routes() http.Handler {
	mux := http.NewServeMux()

	// Projects
	mux.HandleFunc("/project", handlers.Projects)

	// Sites
	mux.HandleFunc("/site", handlers.Sites)

	// Loggers
	mux.HandleFunc("/logger", handlers.Loggers)

	// Manual Readings
	mux.HandleFunc("/manual", handlers.ManualReadings)
	mux.HandleFunc("/manual/", handlers.ManualReadingByID)

	// Events
	mux.HandleFunc("/event", handlers.Events)
	mux.HandleFunc("/event/", handlers.EventByID)

	// Import
	mux.HandleFunc("/import", handlers.Import)

	// Process
	mux.HandleFunc("/process", handlers.Process)

	// Export
	mux.HandleFunc("/export", handlers.Export)

	// Data Query
	mux.HandleFunc("/data", handlers.Data)

	return mux
}
