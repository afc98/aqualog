package handlers

import (
	"encoding/json"
	"net/http"

	"aqualog/core/site"
)

func Sites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		q := r.URL.Query()

		projectID, err := parseOptionalInt(q.Get("project_id"))
		if err != nil {
			writeError(w, "invalid project_id", http.StatusBadRequest)
			return
		}

		if projectID == 0 {
			writeError(w, "project_id is required", http.StatusBadRequest)
			return
		}

		sites, err := site.List(projectID)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, sites)

	case http.MethodPost:
		var req site.AddParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, "invalid json", http.StatusBadRequest)
			return
		}

		s, err := site.Add(req)
		if err != nil {
			writeError(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, s)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
