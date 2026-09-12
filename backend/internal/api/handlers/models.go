package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/llmrouter/backend/internal/database/repository"
)

type ModelsHandler struct {
	modelRepo *repository.ModelRepository
}

func NewModelsHandler(modelRepo *repository.ModelRepository) *ModelsHandler {
	return &ModelsHandler{modelRepo: modelRepo}
}

func (h *ModelsHandler) List(w http.ResponseWriter, r *http.Request) {
	models, err := h.modelRepo.ListAllActive(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(models)
}
