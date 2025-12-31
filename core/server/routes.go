package server

import (
	"net/http"

	"aqualog/core/server/handlers"
)

func Routes() http.Handler {
	mux := http.NewServeMux()

	// Events
	mux.HandleFunc("/events", handlers.Events)
	mux.HandleFunc("/events/", handlers.EventByID)

	// Import
	mux.HandleFunc("/import", handlers.Import)

	// Process
	mux.HandleFunc("/process", handlers.Process)

	// Export
	//mux.HandleFunc("/export", handlers.Export)

	return mux
}
