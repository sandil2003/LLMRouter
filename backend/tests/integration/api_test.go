package integration_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/llmrouter/backend/internal/api"
	"github.com/llmrouter/backend/internal/circuitbreaker"
	"github.com/llmrouter/backend/internal/database"
	"github.com/llmrouter/backend/internal/database/repository"
	"github.com/llmrouter/backend/internal/models"
	"github.com/llmrouter/backend/internal/providers"
	"github.com/llmrouter/backend/internal/providers/mock"
	"github.com/llmrouter/backend/internal/ratelimit"
	"github.com/llmrouter/backend/internal/router"
)

type testRig struct {
	server       *httptest.Server
	db           *database.DB
	providerRepo *repository.ProviderRepository
	logRepo      *repository.LogRepository
	usageRepo    *repository.UsageRepository
	registry     *providers.Registry
	rateLimiter  *ratelimit.Tracker
	cb           *circuitbreaker.Manager
	mockA        *mock.Provider
	mockB        *mock.Provider
}

func setupTestRig(t *testing.T) *testRig {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	pRepo := repository.NewProviderRepository(db)
	mRepo := repository.NewModelRepository(db)
	rRepo := repository.NewRoutingRepository(db)
	uRepo := repository.NewUsageRepository(db)
	lRepo := repository.NewLogRepository(db)

	registry := providers.NewRegistry(nil)
	rateLimiter := ratelimit.NewTracker(10 * time.Second)
	cb := circuitbreaker.NewManager(3, 10*time.Second)

	engine := router.NewEngine(
		pRepo,
		mRepo,
		rRepo,
		uRepo,
		lRepo,
		registry,
		rateLimiter,
		cb,
	)

	// Register 2 mocks: MockA (priority 1, rate limited) and MockB (priority 2, healthy)
	mockA := mock.New("mock-a", "Mock A", "*")
	mockA.ShouldRateLimit = true
	mockA.RateLimitDuration = 5 * time.Second

	mockB := mock.New("mock-b", "Mock B", "*")
	mockB.ResponseContent = "Fallback success from Mock B"

	registry.Register(mockA)
	registry.Register(mockB)

	ctx := context.Background()
	_ = pRepo.Create(ctx, &models.ProviderConfig{
		ID:       "mock-a",
		Name:     "Mock A",
		Enabled:  true,
		Priority: 1,
	})
	_ = pRepo.Create(ctx, &models.ProviderConfig{
		ID:       "mock-b",
		Name:     "Mock B",
		Enabled:  true,
		Priority: 2,
	})

	deps := &api.ServerDeps{
		DB:           db,
		ProviderRepo: pRepo,
		ModelRepo:    mRepo,
		RoutingRepo:  rRepo,
		UsageRepo:    uRepo,
		LogRepo:      lRepo,
		Registry:     registry,
		Engine:       engine,
	}

	ts := httptest.NewServer(api.SetupRoutes(deps))
	t.Cleanup(func() {
		ts.Close()
		db.Close()
	})

	return &testRig{
		server:       ts,
		db:           db,
		providerRepo: pRepo,
		logRepo:      lRepo,
		usageRepo:    uRepo,
		registry:     registry,
		rateLimiter:  rateLimiter,
		cb:           cb,
		mockA:        mockA,
		mockB:        mockB,
	}
}

func TestHealthEndpoint(t *testing.T) {
	rig := setupTestRig(t)

	resp, err := http.Get(rig.server.URL + "/api/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	var data map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&data)
	if data["status"] != "ok" || data["database"] != "healthy" {
		t.Errorf("unexpected health body: %+v", data)
	}
}

func TestProvidersEndpoint(t *testing.T) {
	rig := setupTestRig(t)

	resp, err := http.Get(rig.server.URL + "/api/providers")
	if err != nil {
		t.Fatalf("list providers failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	var list []models.ProviderWithStatus
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode providers failed: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(list))
	}
}

func TestChatCompletionsWithFallback(t *testing.T) {
	rig := setupTestRig(t)

	reqBody := models.ChatRequest{
		Model: "gpt-4o",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Hello gateway"},
		},
	}
	bodyBytes, _ := json.Marshal(reqBody)

	resp, err := http.Post(
		rig.server.URL+"/v1/chat/completions",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		t.Fatalf("chat completion request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 OK, got %d: %s", resp.StatusCode, string(body))
	}

	var chatResp models.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		t.Fatalf("decode chat response failed: %v", err)
	}

	if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content != "Fallback success from Mock B" {
		t.Errorf("unexpected completion content: %+v", chatResp)
	}

	// Verify Mock A was hit first and is now marked rate-limited
	if !rig.rateLimiter.IsRateLimited("mock-a") {
		t.Error("expected mock-a to be marked rate-limited after 429 response")
	}

	// Wait for async log/usage recording
	time.Sleep(50 * time.Millisecond)

	logs, err := rig.logRepo.GetLogs(context.Background(), models.LogFilter{Limit: 10})
	if err != nil || len(logs) == 0 {
		t.Fatalf("expected request logs, got %v (err: %v)", logs, err)
	}

	summary, err := rig.usageRepo.GetSummary(context.Background())
	if err != nil || summary.TotalRequests == 0 {
		t.Fatalf("expected recorded usage metrics, got %+v", summary)
	}
}

func TestStreamingCompletions(t *testing.T) {
	rig := setupTestRig(t)

	reqBody := models.ChatRequest{
		Model: "gpt-4o",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Stream me"},
		},
		Stream: true,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	resp, err := http.Post(
		rig.server.URL+"/v1/chat/completions",
		"application/json",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		t.Fatalf("streaming request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 OK, got %d: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		t.Errorf("expected text/event-stream content type, got %s", contentType)
	}

	scanner := bufio.NewScanner(resp.Body)
	receivedDone := false
	chunkCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		if line == "data: [DONE]" {
			receivedDone = true
			break
		}
		if strings.HasPrefix(line, "data: ") {
			chunkCount++
		}
	}

	if !receivedDone {
		t.Error("did not receive data: [DONE] marker in SSE stream")
	}
	if chunkCount == 0 {
		t.Error("expected at least 1 streaming chunk")
	}
}

func TestProviderTestConnection(t *testing.T) {
	rig := setupTestRig(t)

	resp, err := http.Post(rig.server.URL+"/api/providers/mock-b/test", "application/json", nil)
	if err != nil {
		t.Fatalf("test connection request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	var res models.TestConnectionResult
	_ = json.NewDecoder(resp.Body).Decode(&res)
	if !res.Success {
		t.Errorf("expected connection success for healthy provider, got %+v", res)
	}
}
