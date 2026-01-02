package handlers

import (
	"net/http"

	"aqualog/core/query"
)

func Data(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()

	siteID, err := parseOptionalInt(q.Get("site_id"))
	if err != nil || siteID == 0 {
		writeError(w, "invalid or missing site_id", http.StatusBadRequest)
		return
	}

	dataType := q.Get("type")
	if dataType == "" {
		dataType = "raw"
	}

	switch dataType {

	case "raw":
		rows, err := query.QueryLoggerData(siteID)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, rows)

	case "corrected":
		rows, err := query.QueryCorrectedData(siteID)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, rows)

	default:
		writeError(w, "invalid type (must be 'raw' or 'corrected')", http.StatusBadRequest)
	}
}
