package handlers

import (
	"net/http"
	"strconv"

	"aqualog/core/db"
	"aqualog/core/process"
)

func Process(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	siteID, err := strconv.Atoi(r.URL.Query().Get("site_id"))
	if err != nil {
		writeError(w, "invalid site_id", http.StatusBadRequest)
		return
	}

	database, err := db.GetDB()
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer database.Close()

	if err := process.ProcessLoggerData(database, siteID); err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
