package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/llmrouter/backend/internal/background"
	"github.com/llmrouter/backend/internal/database/repository"
	"github.com/llmrouter/backend/internal/langcache"
	"github.com/llmrouter/backend/internal/models"
	"github.com/llmrouter/backend/internal/registry"
	"github.com/llmrouter/backend/internal/router"
)

type RoutingHandler struct {
	routingRepo  *repository.RoutingRepository
	providerRepo *repository.ProviderRepository
	engine       *router.Engine
	syncWorker   *background.SyncWorker
}

func NewRoutingHandler(
	routingRepo *repository.RoutingRepository,
	providerRepo *repository.ProviderRepository,
	engine *router.Engine,
	syncWorker *background.SyncWorker,
) *RoutingHandler {
	return &RoutingHandler{
		routingRepo:  routingRepo,
		providerRepo: providerRepo,
		engine:       engine,
		syncWorker:   syncWorker,
	}
}

type RoutingConfigResponse struct {
	Strategy      models.RoutingStrategy  `json:"strategy"`
	Priorities    []models.ProviderConfig `json:"providers"`
	PolicyWeights registry.PolicyWeights  `json:"policy_weights"`
	CacheStats    langcache.CacheStats    `json:"cache_stats"`
	SyncStatus    background.SyncStatus   `json:"sync_status"`
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

	weights := registry.DefaultPolicyWeights()
	if h.engine != nil && h.engine.PolicyEngine() != nil {
		weights = h.engine.PolicyEngine().GetWeights()
	}

	var cacheStats langcache.CacheStats
	if h.engine != nil && h.engine.LangCache() != nil {
		cacheStats = h.engine.LangCache().GetStats(ctx)
	}

	var syncStatus background.SyncStatus
	if h.syncWorker != nil {
		syncStatus = h.syncWorker.GetStatus()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(RoutingConfigResponse{
		Strategy:      rule.Strategy,
		Priorities:    providers,
		PolicyWeights: weights,
		CacheStats:    cacheStats,
		SyncStatus:    syncStatus,
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

type UpdatePolicyRequest struct {
	Capability *float64 `json:"capability,omitempty"`
	Cost       *float64 `json:"cost,omitempty"`
	Latency    *float64 `json:"latency,omitempty"`
	Preset     *string  `json:"preset,omitempty"`
}

func (h *RoutingHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	var req UpdatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if h.engine == nil || h.engine.PolicyEngine() == nil {
		http.Error(w, "policy engine unavailable", http.StatusServiceUnavailable)
		return
	}

	current := h.engine.PolicyEngine().GetWeights()
	if req.Preset != nil && *req.Preset != "" {
		current = registry.PresetWeights(*req.Preset)
	}
	if req.Capability != nil {
		current.Capability = *req.Capability
	}
	if req.Cost != nil {
		current.Cost = *req.Cost
	}
	if req.Latency != nil {
		current.Latency = *req.Latency
	}

	h.engine.PolicyEngine().SetWeights(current)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(current)
}

func (h *RoutingHandler) GetRegistry(w http.ResponseWriter, r *http.Request) {
	if h.engine == nil || h.engine.ModelRegistry() == nil {
		http.Error(w, "model registry unavailable", http.StatusServiceUnavailable)
		return
	}

	modelsList := h.engine.ModelRegistry().GetAll()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(modelsList)
}

func (h *RoutingHandler) GetCacheStats(w http.ResponseWriter, r *http.Request) {
	if h.engine == nil || h.engine.LangCache() == nil {
		http.Error(w, "cache unavailable", http.StatusServiceUnavailable)
		return
	}

	stats := h.engine.LangCache().GetStats(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func (h *RoutingHandler) TriggerSync(w http.ResponseWriter, r *http.Request) {
	if h.syncWorker == nil {
		http.Error(w, "sync worker not available", http.StatusServiceUnavailable)
		return
	}

	go func() {
		_ = h.syncWorker.RunSync(r.Context())
	}()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Offline background sync & canary benchmarking started",
		"status":  "running",
	})
}
