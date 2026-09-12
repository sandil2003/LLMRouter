package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/llmrouter/backend/internal/database/repository"
)

type UsageHandler struct {
	usageRepo *repository.UsageRepository
}

func NewUsageHandler(usageRepo *repository.UsageRepository) *UsageHandler {
	return &UsageHandler{usageRepo: usageRepo}
}

func (h *UsageHandler) Get(w http.ResponseWriter, r *http.Request) {
	summary, err := h.usageRepo.GetSummary(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summary)
}
