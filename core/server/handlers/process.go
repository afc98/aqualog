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
		http.Error(w, "invalid site_id", 400)
		return
	}

	database, err := db.GetDB()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer database.Close()

	if err := process.ProcessLoggerData(database, siteID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
