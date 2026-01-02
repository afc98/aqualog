package handlers

import (
	"encoding/json"
	"net/http"

	"aqualog/core/export"
)

type exportRequest struct {
	FilePath string `json:"file_path"`
	SiteID   int    `json:"site_id"`
}

func Export(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req exportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid json", http.StatusBadRequest)
		return
	}

	err := export.ExportData(req.SiteID, req.FilePath)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
