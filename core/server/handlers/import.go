package handlers

import (
	"encoding/json"
	"net/http"

	"aqualog/core/importer"
)

type importRequest struct {
	FileType string `json:"file_type"`
	FilePath string `json:"file_path"`
	SiteID   int    `json:"site_id"`
	LoggerID int    `json:"logger_id"`
}

func Import(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", 400)
		return
	}

	meta, recs, err := importer.ParseLoggerFile(req.FileType, req.FilePath)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	result, err := importer.ImportRecords(
		meta,
		recs,
		req.SiteID,
		req.LoggerID,
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	writeJSON(w, result)
}
