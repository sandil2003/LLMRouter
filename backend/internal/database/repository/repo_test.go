package repository_test

import (
	"context"
	"testing"

	"github.com/llmrouter/backend/internal/database"
	"github.com/llmrouter/backend/internal/database/repository"
	"github.com/llmrouter/backend/internal/models"
)

func setupTestDB(t *testing.T) *database.DB {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestProviderRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewProviderRepository(db)
	ctx := context.Background()

	p := &models.ProviderConfig{
		ID:       "gemini",
		Name:     "Gemini",
		Enabled:  true,
		Priority: 1,
		BaseURL:  "https://generativelanguage.googleapis.com",
	}

	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("create provider failed: %v", err)
	}

	fetched, err := repo.GetByID(ctx, "gemini")
	if err != nil {
		t.Fatalf("get provider failed: %v", err)
	}
	if fetched.Name != "Gemini" || fetched.Priority != 1 {
		t.Errorf("unexpected provider data: %+v", fetched)
	}

	all, err := repo.GetAll(ctx)
	if err != nil || len(all) != 1 {
		t.Fatalf("expected 1 provider, got %d (err: %v)", len(all), err)
	}

	// Update priority
	if err := repo.UpdatePriority(ctx, "gemini", 5); err != nil {
		t.Fatalf("update priority failed: %v", err)
	}
	updated, _ := repo.GetByID(ctx, "gemini")
	if updated.Priority != 5 {
		t.Errorf("expected priority 5, got %d", updated.Priority)
	}

	// Delete
	if err := repo.Delete(ctx, "gemini"); err != nil {
		t.Fatalf("delete provider failed: %v", err)
	}
	_, err = repo.GetByID(ctx, "gemini")
	if err != repository.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUsageAndLogRepositories(t *testing.T) {
	db := setupTestDB(t)
	usageRepo := repository.NewUsageRepository(db)
	logRepo := repository.NewLogRepository(db)
	ctx := context.Background()

	u := &models.UsageRecord{
		ProviderID:   "groq",
		Model:        "llama3-70b",
		RequestID:    "req-1",
		InputTokens:  100,
		OutputTokens: 200,
		LatencyMs:    150,
		Status:       "success",
	}
	if err := usageRepo.RecordUsage(ctx, u); err != nil {
		t.Fatalf("record usage failed: %v", err)
	}

	summary, err := usageRepo.GetSummary(ctx)
	if err != nil {
		t.Fatalf("get summary failed: %v", err)
	}
	if summary.TotalRequests != 1 || summary.TotalInputTokens != 100 || summary.TotalOutputTokens != 200 {
		t.Errorf("unexpected summary: %+v", summary)
	}

	l := &models.RequestLog{
		RequestID:    "req-1",
		ProviderID:   "groq",
		Model:        "llama3-70b",
		Status:       "success",
		LatencyMs:    150,
		FallbackUsed: false,
		AttemptCount: 1,
	}
	if err := logRepo.RecordLog(ctx, l); err != nil {
		t.Fatalf("record log failed: %v", err)
	}

	logs, err := logRepo.GetLogs(ctx, models.LogFilter{Limit: 10})
	if err != nil || len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d (err: %v)", len(logs), err)
	}
}
