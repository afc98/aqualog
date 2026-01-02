package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"aqualog/core/event"
)

func Events(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		q := r.URL.Query()

		siteID, err := parseOptionalInt(q.Get("site_id"))
		if err != nil {
			writeError(w, "invalid site_id", http.StatusBadRequest)
			return
		}

		loggerID, err := parseOptionalInt(q.Get("logger_id"))
		if err != nil {
			writeError(w, "invalid logger_id", http.StatusBadRequest)
			return
		}

		events, err := event.List(siteID, loggerID)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeSuccess(w, events)

	case http.MethodPost:
		var req event.AddParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, "invalid json", http.StatusBadRequest)
			return
		}

		ev, err := event.Add(req)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeSuccess(w, ev)

	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func EventByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/events/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, "invalid id", http.StatusBadRequest)
		return
	}

	switch r.Method {

	case http.MethodGet:
		ev, err := event.Get(id)
		if err != nil {
			writeError(w, err.Error(), http.StatusNotFound)
			return
		}
		writeSuccess(w, ev)

	case http.MethodDelete:
		if err := event.Delete(id); err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func parseOptionalInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}
