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
	idStr := strings.TrimPrefix(r.URL.Path, "/event/")
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

	case http.MethodPut:
		var req event.UpdateParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, "invalid json", http.StatusBadRequest)
			return
		}

		existing, err := event.Get(id)
		if err != nil {
			writeError(w, "error getting event", http.StatusNotFound)
			return
		}

		updated, err := event.PrepareUpdate(existing, req)
		if err != nil {
			writeError(w, "error preparing update", http.StatusBadRequest)
			return
		}

		if err := event.Update(id, updated); err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeSuccess(w, updated)

	default:
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

