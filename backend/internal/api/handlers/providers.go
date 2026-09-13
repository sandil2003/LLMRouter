package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/llmrouter/backend/internal/circuitbreaker"
	"github.com/llmrouter/backend/internal/database/repository"
	"github.com/llmrouter/backend/internal/models"
	"github.com/llmrouter/backend/internal/providers"
	"github.com/llmrouter/backend/internal/providers/gemini"
	"github.com/llmrouter/backend/internal/providers/groq"
	"github.com/llmrouter/backend/internal/providers/openai"
	"github.com/llmrouter/backend/internal/providers/openrouter"
	"github.com/llmrouter/backend/internal/ratelimit"
)

type ProvidersHandler struct {
	providerRepo   *repository.ProviderRepository
	modelRepo      *repository.ModelRepository
	registry       *providers.Registry
	rateLimiter    *ratelimit.Tracker
	circuitBreaker *circuitbreaker.Manager
}

func NewProvidersHandler(
	providerRepo *repository.ProviderRepository,
	modelRepo *repository.ModelRepository,
	registry *providers.Registry,
	rateLimiter *ratelimit.Tracker,
	circuitBreaker *circuitbreaker.Manager,
) *ProvidersHandler {
	return &ProvidersHandler{
		providerRepo:   providerRepo,
		modelRepo:      modelRepo,
		registry:       registry,
		rateLimiter:    rateLimiter,
		circuitBreaker: circuitBreaker,
	}
}

type CreateProviderRequest struct {
	ID       string   `json:"id,omitempty"`
	Name     string   `json:"name"`
	Enabled  *bool    `json:"enabled,omitempty"`
	Priority int      `json:"priority"`
	BaseURL  string   `json:"base_url,omitempty"`
	APIKey   string   `json:"api_key,omitempty"`
	Models   []string `json:"models,omitempty"`
}

type UpdateProviderRequest struct {
	Name     *string  `json:"name,omitempty"`
	Enabled  *bool    `json:"enabled,omitempty"`
	Priority *int     `json:"priority,omitempty"`
	BaseURL  *string  `json:"base_url,omitempty"`
	APIKey   *string  `json:"api_key,omitempty"`
	Models   []string `json:"models,omitempty"`
}

func (h *ProvidersHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	configs, err := h.providerRepo.GetAll(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := make([]models.ProviderWithStatus, 0, len(configs))
	for _, cfg := range configs {
		cbState, fails := h.circuitBreaker.GetStatus(cfg.ID)
		isRL := h.rateLimiter.IsRateLimited(cfg.ID)
		resetIn := h.rateLimiter.RemainingDuration(cfg.ID).Milliseconds() / 1000

		healthState := models.HealthStateHealthy
		if !cfg.Enabled {
			healthState = models.HealthStateDown
		} else if isRL {
			healthState = models.HealthStateRateLimited
		} else if cbState == circuitbreaker.StateOpen {
			healthState = models.HealthStateDown
		} else if cbState == circuitbreaker.StateHalfOpen || fails > 0 {
			healthState = models.HealthStateDegraded
		}

		modelsList, _ := h.modelRepo.ListByProvider(ctx, cfg.ID)
		activeModelNames := make([]string, 0, len(modelsList))
		for _, m := range modelsList {
			if m.Enabled {
				activeModelNames = append(activeModelNames, m.Name)
			}
		}

		activeKey, _ := h.registry.Credentials().GetAPIKey(ctx, cfg.ID)

		result = append(result, models.ProviderWithStatus{
			ProviderConfig:    cfg,
			HealthState:       healthState,
			RateLimited:       isRL,
			RateLimitResetIn:  resetIn,
			CircuitState:      string(cbState),
			ConsecutiveErrors: fails,
			ActiveModels:      activeModelNames,
			HasAPIKey:         activeKey != "",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *ProvidersHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	id := req.ID
	if id == "" {
		id = strings.ToLower(strings.ReplaceAll(req.Name, " ", "-")) + "-" + uuid.NewString()[:8]
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	priority := req.Priority
	if priority <= 0 {
		priority = 1
	}

	cfg := models.ProviderConfig{
		ID:       id,
		Name:     req.Name,
		Enabled:  enabled,
		Priority: priority,
		BaseURL:  req.BaseURL,
	}

	ctx := r.Context()
	if err := h.providerRepo.Create(ctx, &cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Register API key in secure credStore
	if strings.TrimSpace(req.APIKey) != "" {
		_ = h.registry.UpdateAPIKey(ctx, id, strings.TrimSpace(req.APIKey))
	}

	// Save associated models
	if len(req.Models) > 0 {
		_ = h.modelRepo.ReplaceProviderModels(ctx, id, req.Models)
	}

	// Instantiate and register in memory
	h.registerProviderInstance(ctx, cfg, req.APIKey)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(cfg)
}

func (h *ProvidersHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "provider id required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	existing, err := h.providerRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, "provider not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var req UpdateProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		existing.Priority = *req.Priority
	}
	if req.BaseURL != nil {
		existing.BaseURL = *req.BaseURL
	}

	if err := h.providerRepo.Update(ctx, existing); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var apiKey string
	if req.APIKey != nil {
		apiKey = strings.TrimSpace(*req.APIKey)
		_ = h.registry.UpdateAPIKey(ctx, id, apiKey)
	} else {
		apiKey, _ = h.registry.Credentials().GetAPIKey(ctx, id)
	}

	if req.Models != nil {
		_ = h.modelRepo.ReplaceProviderModels(ctx, id, req.Models)
	}

	h.registerProviderInstance(ctx, *existing, apiKey)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(existing)
}

func (h *ProvidersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "provider id required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.providerRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, "provider not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.registry.Unregister(id)
	h.rateLimiter.Clear(id)
	h.circuitBreaker.Reset(id)

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProvidersHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, exists := h.registry.Get(id)
	if !exists {
		http.Error(w, "provider instance not active or registered", http.StatusNotFound)
		return
	}

	start := time.Now()
	err := p.HealthCheck(r.Context())
	latency := time.Since(start).Milliseconds()

	success := err == nil
	msg := "Connection successful"
	if err != nil {
		msg = err.Error()
	} else {
		// Clear any rate-limit or circuit breaker errors if test passes
		h.circuitBreaker.RecordSuccess(id)
		h.rateLimiter.Clear(id)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(models.TestConnectionResult{
		Success:   success,
		LatencyMs: latency,
		Message:   msg,
	})
}

func (h *ProvidersHandler) registerProviderInstance(ctx context.Context, cfg models.ProviderConfig, apiKey string) {
	if apiKey == "" {
		apiKey, _ = h.registry.Credentials().GetAPIKey(ctx, cfg.ID)
	}

	nameLower := strings.ToLower(cfg.Name)
	var inst providers.Provider

	switch {
	case strings.Contains(nameLower, "gemini"):
		inst = gemini.New(cfg.ID, apiKey, cfg.BaseURL)
	case strings.Contains(nameLower, "groq"):
		inst = groq.New(cfg.ID, apiKey, cfg.BaseURL)
	case strings.Contains(nameLower, "openrouter"):
		inst = openrouter.New(cfg.ID, apiKey, cfg.BaseURL)
	default:
		inst = openai.New(cfg.ID, cfg.Name, apiKey, cfg.BaseURL)
	}

	h.registry.Register(inst)
}
