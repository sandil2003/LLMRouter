package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/llmrouter/backend/internal/circuitbreaker"
	"github.com/llmrouter/backend/internal/database"
	"github.com/llmrouter/backend/internal/database/repository"
	"github.com/llmrouter/backend/internal/models"
	"github.com/llmrouter/backend/internal/providers"
	"github.com/llmrouter/backend/internal/ratelimit"
)

func setupTestProvidersHandler(t *testing.T) (*ProvidersHandler, *database.DB) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to init in-memory db: %v", err)
	}

	providerRepo := repository.NewProviderRepository(db)
	modelRepo := repository.NewModelRepository(db)
	registry := providers.NewRegistry(nil)
	rateLimiter := ratelimit.NewTracker(time.Minute)
	circuitBreaker := circuitbreaker.NewManager(3, 30*time.Second)

	h := NewProvidersHandler(providerRepo, modelRepo, registry, rateLimiter, circuitBreaker)
	return h, db
}

func TestProvidersHandler_ModelManagementAndDiscovery(t *testing.T) {
	h, db := setupTestProvidersHandler(t)
	defer db.Close()
	ctx := context.Background()

	// 1. Create a provider
	p := &models.ProviderConfig{
		ID:       "gemini-test",
		Name:     "Gemini",
		Enabled:  true,
		Priority: 1,
		BaseURL:  "https://generativelanguage.googleapis.com/v1beta",
	}
	if err := h.providerRepo.Create(ctx, p); err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	// 2. Test DiscoverModels endpoint
	reqDiscover := httptest.NewRequest(http.MethodGet, "/api/providers/discover-models?name=Gemini", nil)
	wDiscover := httptest.NewRecorder()
	h.DiscoverModels(wDiscover, reqDiscover)

	if wDiscover.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from DiscoverModels, got %d", wDiscover.Code)
	}

	var discoverRes providers.DiscoverModelsResult
	if err := json.NewDecoder(wDiscover.Body).Decode(&discoverRes); err != nil {
		t.Fatalf("failed to decode discover response: %v", err)
	}
	if len(discoverRes.Models) == 0 {
		t.Fatalf("expected non-empty discovered models")
	}

	// 3. Test AddModel endpoint
	addBody, _ := json.Marshal(map[string]string{"model": "gemini-2.5-flash"})
	rAdd := chi.NewRouter()
	rAdd.Post("/api/providers/{id}/models", h.AddModel)

	reqAdd := httptest.NewRequest(http.MethodPost, "/api/providers/gemini-test/models", bytes.NewReader(addBody))
	wAdd := httptest.NewRecorder()
	rAdd.ServeHTTP(wAdd, reqAdd)

	if wAdd.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from AddModel, got %d: %s", wAdd.Code, wAdd.Body.String())
	}

	var addRes struct {
		ProviderID   string   `json:"provider_id"`
		ActiveModels []string `json:"active_models"`
	}
	if err := json.NewDecoder(wAdd.Body).Decode(&addRes); err != nil {
		t.Fatalf("failed to decode AddModel response: %v", err)
	}
	if len(addRes.ActiveModels) != 1 || addRes.ActiveModels[0] != "gemini-2.5-flash" {
		t.Fatalf("expected active_models ['gemini-2.5-flash'], got %v", addRes.ActiveModels)
	}

	// 4. Test GetAvailableModels endpoint
	rAvailable := chi.NewRouter()
	rAvailable.Get("/api/providers/{id}/available-models", h.GetAvailableModels)

	reqAvail := httptest.NewRequest(http.MethodGet, "/api/providers/gemini-test/available-models", nil)
	wAvail := httptest.NewRecorder()
	rAvailable.ServeHTTP(wAvail, reqAvail)

	if wAvail.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from GetAvailableModels, got %d: %s", wAvail.Code, wAvail.Body.String())
	}

	var availRes providers.DiscoverModelsResult
	if err := json.NewDecoder(wAvail.Body).Decode(&availRes); err != nil {
		t.Fatalf("failed to decode GetAvailableModels response: %v", err)
	}

	foundActive := false
	for _, m := range availRes.Models {
		if m.ID == "gemini-2.5-flash" && m.IsActive {
			foundActive = true
			break
		}
	}
	if !foundActive {
		t.Errorf("expected gemini-2.5-flash to be marked active in available models")
	}

	// 5. Test RemoveModel endpoint
	rRemove := chi.NewRouter()
	rRemove.Delete("/api/providers/{id}/models/{model}", h.RemoveModel)

	reqRemove := httptest.NewRequest(http.MethodDelete, "/api/providers/gemini-test/models/gemini-2.5-flash", nil)
	wRemove := httptest.NewRecorder()
	rRemove.ServeHTTP(wRemove, reqRemove)

	if wRemove.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from RemoveModel, got %d: %s", wRemove.Code, wRemove.Body.String())
	}

	var removeRes struct {
		ProviderID   string   `json:"provider_id"`
		ActiveModels []string `json:"active_models"`
	}
	if err := json.NewDecoder(wRemove.Body).Decode(&removeRes); err != nil {
		t.Fatalf("failed to decode RemoveModel response: %v", err)
	}
	if len(removeRes.ActiveModels) != 0 {
		t.Fatalf("expected active_models to be empty after remove, got %v", removeRes.ActiveModels)
	}
}
