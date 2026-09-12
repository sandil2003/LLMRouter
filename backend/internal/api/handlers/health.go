package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/llmrouter/backend/internal/database"
)

type HealthHandler struct {
	db *database.DB
}

func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
	Version  string `json:"version"`
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	dbStatus := "healthy"
	if err := h.db.Ping(); err != nil {
		dbStatus = "unreachable"
	}

	status := "ok"
	statusCode := http.StatusOK
	if dbStatus != "healthy" {
		status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status:   status,
		Database: dbStatus,
		Version:  "1.0.0",
	})
}
