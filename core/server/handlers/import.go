package handlers

import (
	"encoding/json"
	"net/http"

	"aqualog/core/importer"
)

type importRequest struct {
	FileType       string `json:"file_type"`
	FilePath       string `json:"file_path"`
	SiteID         int    `json:"site_id"`
	LoggerID       int    `json:"logger_id"`
	Replace        bool   `json:"replace"`
	TargetTimezone string `json:"target_timezone"`
}

func Import(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid json", http.StatusBadRequest)
		return
	}

	options := importer.ImportOptions{ReplaceExisting: req.Replace, TargetTimezone: req.TargetTimezone}
	meta, recs, err := importer.ParseLoggerFileWithOptions(req.FileType, req.FilePath, options)
	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := importer.ImportRecords(
		meta,
		recs,
		req.SiteID,
		req.LoggerID,
		req.FilePath,
		req.FileType,
		options,
	)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result.Warning = importer.TimezoneWarning(recs, req.TargetTimezone)
	writeJSON(w, result)
}
