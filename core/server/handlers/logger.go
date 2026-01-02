package handlers

import (
	"encoding/json"
	"net/http"

	"aqualog/core/logger"
)

func Loggers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		q := r.URL.Query()

		siteID, err := parseOptionalInt(q.Get("site_id"))
		if err != nil {
			writeError(w, "invalid site_id", http.StatusBadRequest)
			return
		}

		if siteID == 0 {
			writeError(w, "site_id is required", http.StatusBadRequest)
			return
		}

		loggers, err := logger.List(siteID)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, loggers)

	case http.MethodPost:
		var req logger.AddParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, "invalid json", http.StatusBadRequest)
			return
		}

		l, err := logger.Add(req)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, l)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
