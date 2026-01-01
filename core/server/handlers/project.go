package handlers

import (
	"encoding/json"
	"net/http"

	"aqualog/core/project"
)

func Projects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		projects, err := project.List()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, projects)

	case http.MethodPost:
		var req project.AddParams
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		p, err := project.Add(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, p)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
