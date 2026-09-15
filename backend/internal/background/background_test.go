package background

import (
	"context"
	"testing"
	"time"

	"github.com/llmrouter/backend/internal/providers"
	"github.com/llmrouter/backend/internal/providers/mock"
	"github.com/llmrouter/backend/internal/registry"
)

func TestSyncWorker_RunSync(t *testing.T) {
	modelReg := registry.NewModelRegistry()
	provReg := providers.NewRegistry(nil)

	// Register mock provider
	mockP := mock.New("openai", "OpenAI", "*")
	mockP.ResponseContent = "Ready."
	provReg.Register(mockP)

	worker := NewSyncWorker(modelReg, provReg, 1*time.Hour)

	ctx := context.Background()
	err := worker.RunSync(ctx)
	if err != nil {
		t.Fatalf("unexpected sync error: %v", err)
	}

	status := worker.GetStatus()
	if status.TotalSyncs != 1 {
		t.Errorf("expected 1 total sync, got %d", status.TotalSyncs)
	}
	if status.ModelsUpdated == 0 {
		t.Errorf("expected models updated > 0, got %d", status.ModelsUpdated)
	}

	// Verify new models (e.g. o3-mini, deepseek-r1) exist in registry now
	if _, ok := modelReg.Get("o3-mini"); !ok {
		t.Error("expected o3-mini to be added to model registry during sync")
	}
	if _, ok := modelReg.Get("deepseek-r1"); !ok {
		t.Error("expected deepseek-r1 to be added to model registry during sync")
	}
}
