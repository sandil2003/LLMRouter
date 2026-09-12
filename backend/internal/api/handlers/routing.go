package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/llmrouter/backend/internal/database/repository"
	"github.com/llmrouter/backend/internal/models"
)

type RoutingHandler struct {
	routingRepo  *repository.RoutingRepository
	providerRepo *repository.ProviderRepository
}

func NewRoutingHandler(routingRepo *repository.RoutingRepository, providerRepo *repository.ProviderRepository) *RoutingHandler {
	return &RoutingHandler{
		routingRepo:  routingRepo,
		providerRepo: providerRepo,
	}
}

type RoutingConfigResponse struct {
	Strategy   models.RoutingStrategy  `json:"strategy"`
	Priorities []models.ProviderConfig `json:"providers"`
}

func (h *RoutingHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rule, err := h.routingRepo.GetActiveRule(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	providers, err := h.providerRepo.GetAll(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(RoutingConfigResponse{
		Strategy:   rule.Strategy,
		Priorities: providers,
	})
}

func (h *RoutingHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req models.RoutingUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if req.Strategy != nil {
		if err := h.routingRepo.UpdateStrategy(ctx, *req.Strategy); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if len(req.Priorities) > 0 {
		if err := h.routingRepo.BatchUpdatePriorities(ctx, req.Priorities); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	h.Get(w, r)
}
