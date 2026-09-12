package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/llmrouter/backend/internal/database/repository"
	"github.com/llmrouter/backend/internal/models"
)

type LogsHandler struct {
	logRepo *repository.LogRepository
}

func NewLogsHandler(logRepo *repository.LogRepository) *LogsHandler {
	return &LogsHandler{logRepo: logRepo}
}

func (h *LogsHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	limit := 50
	if lStr := query.Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if oStr := query.Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	filter := models.LogFilter{
		ProviderID: query.Get("provider"),
		Status:     query.Get("status"),
		Limit:      limit,
		Offset:     offset,
	}

	logs, err := h.logRepo.GetLogs(r.Context(), filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(logs)
}
