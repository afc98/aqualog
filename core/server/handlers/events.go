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
			http.Error(w, "invalid site_id", http.StatusBadRequest)
			return
		}

		loggerID, err := parseOptionalInt(q.Get("logger_id"))
		if err != nil {
			http.Error(w, "invalid logger_id", http.StatusBadRequest)
			return
		}

		events, err := event.List(siteID, loggerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, events)

	case http.MethodPost:
		var req event.AddParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		ev, err := event.Add(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, ev)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func EventByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/events/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}

	switch r.Method {

	case http.MethodGet:
		ev, err := event.Get(id)
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		writeJSON(w, ev)

	case http.MethodDelete:
		if err := event.Delete(id); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
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
