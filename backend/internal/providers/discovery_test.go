package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDiscoverModelsFallback(t *testing.T) {
	ctx := context.Background()

	// Test fallback for Gemini
	resGemini := DiscoverModels(ctx, "gemini-1", "Gemini", "", "", []string{"gemini-2.5-flash"})
	if len(resGemini.Models) == 0 {
		t.Fatalf("expected discovered models for gemini, got 0")
	}

	foundActive := false
	for _, m := range resGemini.Models {
		if m.ID == "gemini-2.5-flash" {
			if !m.IsActive {
				t.Errorf("expected gemini-2.5-flash to be marked active")
			}
			foundActive = true
			break
		}
	}
	if !foundActive {
		t.Errorf("expected gemini-2.5-flash to be present in discovered models")
	}

	// Test fallback for Groq
	resGroq := DiscoverModels(ctx, "groq-1", "Groq", "", "", nil)
	if len(resGroq.Models) == 0 {
		t.Fatalf("expected discovered models for groq, got 0")
	}

	// Test fallback for OpenAI
	resOpenAI := DiscoverModels(ctx, "openai-1", "OpenAI", "", "", nil)
	if len(resOpenAI.Models) == 0 {
		t.Fatalf("expected discovered models for openai, got 0")
	}
}

func TestDiscoverModelsLiveOpenAICompatible(t *testing.T) {
	// Create mock HTTP server simulating an OpenAI-compatible /models endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "custom-llm-1", "name": "Custom LLM 1", "context_length": 65536},
				{"id": "custom-llm-2", "name": "Custom LLM 2", "context_length": 131072},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ctx := context.Background()
	res := DiscoverModels(ctx, "custom-1", "Custom", server.URL, "test-key", []string{"custom-llm-2"})

	if res.Source != "live_api" {
		t.Fatalf("expected source live_api, got %s", res.Source)
	}
	if len(res.Models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(res.Models))
	}

	// Active model should be sorted first
	if res.Models[0].ID != "custom-llm-2" || !res.Models[0].IsActive {
		t.Errorf("expected active model custom-llm-2 to be first and marked active")
	}
}
