package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"aqualog/core/manual"
)

func ManualReadings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		q := r.URL.Query()

		siteID, err := parseOptionalInt(q.Get("site_id"))
		if err != nil {
			http.Error(w, "invalid site_id", http.StatusBadRequest)
			return
		}

		if siteID == 0 {
			http.Error(w, "site_id is required", http.StatusBadRequest)
			return
		}

		readings, err := manual.List(siteID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, readings)

	case http.MethodPost:
		var req manual.AddParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		reading, err := manual.Add(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, reading)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func ManualReadingByID(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/manual/")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	switch r.Method {

	case http.MethodGet:
		reading, err := manual.Get(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, reading)

	case http.MethodDelete:
		if err := manual.Delete(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
