package registry

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/llmrouter/backend/internal/classifier"
)

// ModelMetadata represents deterministic metadata for a registered LLM.
type ModelMetadata struct {
	ID                  string                                 `json:"id"`
	ProviderID          string                                 `json:"provider_id"`
	Name                string                                 `json:"name"`
	ContextWindow       int                                    `json:"context_window"`
	SupportedModalities []string                               `json:"supported_modalities"` // e.g. ["text", "vision"]
	CostPer1MInput      float64                                `json:"cost_per_1m_input"`
	CostPer1MOutput     float64                                `json:"cost_per_1m_output"`
	DomainScores        map[classifier.TaskDomain]float64      `json:"domain_scores"` // 0.0 - 1.0
	SupportsTools       bool                                   `json:"supports_tools"`
	RollingTTFTMs       float64                                `json:"rolling_ttft_ms"`
	P95LatencyMs        float64                                `json:"p95_latency_ms"`
	ErrorRate           float64                                `json:"error_rate"`
	LastUpdated         time.Time                              `json:"last_updated"`
}

// HasModality checks if model supports the requested modality.
func (m *ModelMetadata) HasModality(modality string) bool {
	for _, mMod := range m.SupportedModalities {
		if strings.EqualFold(mMod, modality) {
			return true
		}
	}
	return false
}

// ModelRegistry maintains the in-memory lookup cache of all registered models.
type ModelRegistry struct {
	mu     sync.RWMutex
	models map[string]*ModelMetadata
}

func NewModelRegistry() *ModelRegistry {
	r := &ModelRegistry{
		models: make(map[string]*ModelMetadata),
	}
	r.seedDefaultCatalog()
	return r
}

func (r *ModelRegistry) seedDefaultCatalog() {
	now := time.Now()
	defaults := []*ModelMetadata{
		{
			ID:                  "gemini-2.5-flash",
			ProviderID:          "gemini",
			Name:                "Gemini 2.5 Flash",
			ContextWindow:       1048576,
			SupportedModalities: []string{"text", "vision"},
			CostPer1MInput:      0.075,
			CostPer1MOutput:     0.30,
			DomainScores: map[classifier.TaskDomain]float64{
				classifier.DomainMath:              0.88,
				classifier.DomainCodeGeneration:    0.89,
				classifier.DomainMultiHopReasoning: 0.87,
				classifier.DomainCreativeWriting:   0.86,
				classifier.DomainRoutineExtraction: 0.93,
			},
			SupportsTools: true,
			RollingTTFTMs: 180,
			P95LatencyMs:  750,
			ErrorRate:     0.005,
			LastUpdated:   now,
		},
		{
			ID:                  "gemini-2.5-pro",
			ProviderID:          "gemini",
			Name:                "Gemini 2.5 Pro",
			ContextWindow:       2097152,
			SupportedModalities: []string{"text", "vision"},
			CostPer1MInput:      1.25,
			CostPer1MOutput:     5.00,
			DomainScores: map[classifier.TaskDomain]float64{
				classifier.DomainMath:              0.96,
				classifier.DomainCodeGeneration:    0.95,
				classifier.DomainMultiHopReasoning: 0.96,
				classifier.DomainCreativeWriting:   0.94,
				classifier.DomainRoutineExtraction: 0.95,
			},
			SupportsTools: true,
			RollingTTFTMs: 420,
			P95LatencyMs:  1800,
			ErrorRate:     0.008,
			LastUpdated:   now,
		},
		{
			ID:                  "gpt-4o",
			ProviderID:          "openai",
			Name:                "GPT-4o",
			ContextWindow:       128000,
			SupportedModalities: []string{"text", "vision"},
			CostPer1MInput:      2.50,
			CostPer1MOutput:     10.00,
			DomainScores: map[classifier.TaskDomain]float64{
				classifier.DomainMath:              0.94,
				classifier.DomainCodeGeneration:    0.94,
				classifier.DomainMultiHopReasoning: 0.93,
				classifier.DomainCreativeWriting:   0.95,
				classifier.DomainRoutineExtraction: 0.95,
			},
			SupportsTools: true,
			RollingTTFTMs: 380,
			P95LatencyMs:  1400,
			ErrorRate:     0.01,
			LastUpdated:   now,
		},
		{
			ID:                  "gpt-4o-mini",
			ProviderID:          "openai",
			Name:                "GPT-4o Mini",
			ContextWindow:       128000,
			SupportedModalities: []string{"text", "vision"},
			CostPer1MInput:      0.15,
			CostPer1MOutput:     0.60,
			DomainScores: map[classifier.TaskDomain]float64{
				classifier.DomainMath:              0.84,
				classifier.DomainCodeGeneration:    0.86,
				classifier.DomainMultiHopReasoning: 0.83,
				classifier.DomainCreativeWriting:   0.85,
				classifier.DomainRoutineExtraction: 0.91,
			},
			SupportsTools: true,
			RollingTTFTMs: 190,
			P95LatencyMs:  800,
			ErrorRate:     0.006,
			LastUpdated:   now,
		},
		{
			ID:                  "llama-3.3-70b-versatile",
			ProviderID:          "groq",
			Name:                "Llama 3.3 70B Versatile",
			ContextWindow:       128000,
			SupportedModalities: []string{"text"},
			CostPer1MInput:      0.59,
			CostPer1MOutput:     0.79,
			DomainScores: map[classifier.TaskDomain]float64{
				classifier.DomainMath:              0.87,
				classifier.DomainCodeGeneration:    0.88,
				classifier.DomainMultiHopReasoning: 0.88,
				classifier.DomainCreativeWriting:   0.88,
				classifier.DomainRoutineExtraction: 0.90,
			},
			SupportsTools: true,
			RollingTTFTMs: 110,
			P95LatencyMs:  450,
			ErrorRate:     0.004,
			LastUpdated:   now,
		},
		{
			ID:                  "llama-3.1-8b-instant",
			ProviderID:          "groq",
			Name:                "Llama 3.1 8B Instant",
			ContextWindow:       128000,
			SupportedModalities: []string{"text"},
			CostPer1MInput:      0.05,
			CostPer1MOutput:     0.08,
			DomainScores: map[classifier.TaskDomain]float64{
				classifier.DomainMath:              0.73,
				classifier.DomainCodeGeneration:    0.78,
				classifier.DomainMultiHopReasoning: 0.74,
				classifier.DomainCreativeWriting:   0.80,
				classifier.DomainRoutineExtraction: 0.84,
			},
			SupportsTools: true,
			RollingTTFTMs: 65,
			P95LatencyMs:  280,
			ErrorRate:     0.002,
			LastUpdated:   now,
		},
		{
			ID:                  "claude-3.5-sonnet",
			ProviderID:          "openrouter",
			Name:                "Claude 3.5 Sonnet",
			ContextWindow:       200000,
			SupportedModalities: []string{"text", "vision"},
			CostPer1MInput:      3.00,
			CostPer1MOutput:     15.00,
			DomainScores: map[classifier.TaskDomain]float64{
				classifier.DomainMath:              0.95,
				classifier.DomainCodeGeneration:    0.97,
				classifier.DomainMultiHopReasoning: 0.96,
				classifier.DomainCreativeWriting:   0.96,
				classifier.DomainRoutineExtraction: 0.95,
			},
			SupportsTools: true,
			RollingTTFTMs: 450,
			P95LatencyMs:  1900,
			ErrorRate:     0.012,
			LastUpdated:   now,
		},
	}

	for _, m := range defaults {
		r.models[strings.ToLower(m.ID)] = m
	}
}

// GetAll returns all registered models.
func (r *ModelRegistry) GetAll() []*ModelMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*ModelMetadata, 0, len(r.models))
	for _, m := range r.models {
		cpy := *m
		res = append(res, &cpy)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].ID < res[j].ID
	})
	return res
}

// Get finds a model by exact or normalized ID.
func (r *ModelRegistry) Get(modelID string) (*ModelMetadata, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normalized := strings.ToLower(strings.TrimSpace(modelID))
	m, ok := r.models[normalized]
	if !ok {
		// Fallback fuzzy search: check if key contains or is contained by modelID
		for k, v := range r.models {
			if strings.Contains(k, normalized) || strings.Contains(normalized, k) {
				return v, true
			}
		}
		return nil, false
	}
	return m, true
}

// Upsert updates or registers a model atomically.
func (r *ModelRegistry) Upsert(m *ModelMetadata) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := strings.ToLower(strings.TrimSpace(m.ID))
	m.LastUpdated = time.Now()
	r.models[key] = m
}

// UpdateOperationalStats updates live rolling TTFT, latency, and error rate.
func (r *ModelRegistry) UpdateOperationalStats(modelID string, latencyMs int64, isTTFT bool, isError bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	m, ok := r.models[strings.ToLower(strings.TrimSpace(modelID))]
	if !ok {
		return
	}

	// Exponential Moving Average (alpha = 0.1)
	const alpha = 0.1
	if isTTFT && latencyMs > 0 {
		if m.RollingTTFTMs == 0 {
			m.RollingTTFTMs = float64(latencyMs)
		} else {
			m.RollingTTFTMs = (alpha * float64(latencyMs)) + ((1 - alpha) * m.RollingTTFTMs)
		}
	} else if latencyMs > 0 {
		if m.P95LatencyMs == 0 {
			m.P95LatencyMs = float64(latencyMs)
		} else {
			m.P95LatencyMs = (alpha * float64(latencyMs)) + ((1 - alpha) * m.P95LatencyMs)
		}
	}

	errSample := 0.0
	if isError {
		errSample = 1.0
	}
	m.ErrorRate = (alpha * errSample) + ((1 - alpha) * m.ErrorRate)
	m.LastUpdated = time.Now()
}

// AtomicBatchUpdate applies verified updates from background worker.
func (r *ModelRegistry) AtomicBatchUpdate(ctx context.Context, updates []*ModelMetadata) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for _, m := range updates {
		m.LastUpdated = now
		r.models[strings.ToLower(strings.TrimSpace(m.ID))] = m
	}
}
