package langcache

import (
	"context"
	"testing"
	"time"

	"github.com/llmrouter/backend/internal/models"
)

func TestMemorySemanticCache_HitAndMiss(t *testing.T) {
	cache := NewMemorySemanticCache(100)
	ctx := context.Background()

	mockResp := &models.ChatResponse{
		ID:    "resp-123",
		Model: "gemini-2.5-flash",
		Choices: []models.Choice{
			{
				Message: models.ChatMessage{
					Role:    "assistant",
					Content: "Semantic caching stores responses based on intent!",
				},
			},
		},
	}

	prompt := "How does semantic caching work in LLMs?"
	err := cache.Set(ctx, prompt, mockResp, nil, 1*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error setting cache: %v", err)
	}

	// 1. Exact match test
	hit, err := cache.Search(ctx, prompt, 0.85)
	if err != nil {
		t.Fatalf("unexpected error during search: %v", err)
	}
	if hit == nil {
		t.Fatal("expected cache hit for exact prompt, got nil")
	}
	if hit.Response.ID != "resp-123" {
		t.Errorf("got response id %s, want resp-123", hit.Response.ID)
	}

	// 2. High semantic similarity test
	similarPrompt := "Can you explain how semantic caching works for LLMs?"
	hit2, err := cache.Search(ctx, similarPrompt, 0.70)
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}
	if hit2 == nil {
		t.Fatal("expected cache hit for semantically similar prompt, got nil")
	}
	if hit2.Similarity < 0.70 {
		t.Errorf("similarity %.2f lower than expected threshold 0.70", hit2.Similarity)
	}

	// 3. Different topic miss test
	differentPrompt := "Write a recipe for chocolate chip cookies"
	hit3, err := cache.Search(ctx, differentPrompt, 0.75)
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}
	if hit3 != nil {
		t.Errorf("expected cache miss for different topic, got hit with similarity %.2f", hit3.Similarity)
	}

	// 4. Verify telemetry stats
	stats := cache.GetStats(ctx)
	if stats.Hits != 2 {
		t.Errorf("expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
}
