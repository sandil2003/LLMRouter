package router_test

import (
	"context"
	"testing"
	"time"

	"github.com/llmrouter/backend/internal/circuitbreaker"
	"github.com/llmrouter/backend/internal/database"
	"github.com/llmrouter/backend/internal/database/repository"
	"github.com/llmrouter/backend/internal/models"
	"github.com/llmrouter/backend/internal/providers"
	"github.com/llmrouter/backend/internal/providers/mock"
	"github.com/llmrouter/backend/internal/ratelimit"
	"github.com/llmrouter/backend/internal/router"
)

func TestRouterFallbackOnRateLimit(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	providerRepo := repository.NewProviderRepository(db)
	modelRepo := repository.NewModelRepository(db)
	routingRepo := repository.NewRoutingRepository(db)
	usageRepo := repository.NewUsageRepository(db)
	logRepo := repository.NewLogRepository(db)

	registry := providers.NewRegistry(nil)
	rateLimiter := ratelimit.NewTracker(10 * time.Second)
	cb := circuitbreaker.NewManager(3, 10*time.Second)

	engine := router.NewEngine(
		providerRepo,
		modelRepo,
		routingRepo,
		usageRepo,
		logRepo,
		registry,
		rateLimiter,
		cb,
	)

	// Create 2 mock providers: MockA (priority 1), MockB (priority 2)
	mockA := mock.New("mock-a", "Mock A", "*")
	mockA.ShouldRateLimit = true
	mockA.RateLimitDuration = 5 * time.Second

	mockB := mock.New("mock-b", "Mock B", "*")
	mockB.ResponseContent = "Success response from Mock B"

	registry.Register(mockA)
	registry.Register(mockB)

	_ = providerRepo.Create(ctx, &models.ProviderConfig{
		ID:       "mock-a",
		Name:     "Mock A",
		Enabled:  true,
		Priority: 1,
	})
	_ = providerRepo.Create(ctx, &models.ProviderConfig{
		ID:       "mock-b",
		Name:     "Mock B",
		Enabled:  true,
		Priority: 2,
	})

	req := &models.ChatRequest{
		Model: "gpt-4o",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Hello world"},
		},
	}

	// First execution: Mock A fails with 429, router automatically falls back to Mock B!
	resp, err := engine.ExecuteChat(ctx, req, "req-fallback-1")
	if err != nil {
		t.Fatalf("expected fallback to succeed, got: %v", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "Success response from Mock B" {
		t.Fatalf("expected response from Mock B, got %+v", resp)
	}

	// Verify Mock A is marked as rate limited
	if !rateLimiter.IsRateLimited("mock-a") {
		t.Error("expected mock-a to be marked as rate limited")
	}

	// Second execution: Mock A is rate limited, so candidates list will only have Mock B!
	resp2, err := engine.ExecuteChat(ctx, req, "req-fallback-2")
	if err != nil {
		t.Fatalf("expected second request to succeed directly with Mock B, got: %v", err)
	}
	if len(resp2.Choices) == 0 || resp2.Choices[0].Message.Content != "Success response from Mock B" {
		t.Fatalf("expected response from Mock B, got %+v", resp2)
	}

	// Call counts: Mock A was called once (during first request), Mock B was called twice
	if mockA.CallCount != 1 {
		t.Errorf("expected mockA to be called 1 time, called %d", mockA.CallCount)
	}
	if mockB.CallCount != 2 {
		t.Errorf("expected mockB to be called 2 times, called %d", mockB.CallCount)
	}
}
