package langcache

import (
	"context"
	"time"

	"github.com/llmrouter/backend/internal/models"
)

// CacheHit represents a successful match from Redis LangCache.
type CacheHit struct {
	Response   *models.ChatResponse `json:"response"`
	Similarity float64              `json:"similarity"`
	Prompt     string               `json:"prompt"`
	CachedAt   time.Time            `json:"cached_at"`
	Source     string               `json:"source"` // "redis_langcache" or "memory_fallback"
}

// CacheStats provides operational telemetry for the semantic cache layer.
type CacheStats struct {
	ActiveMode    string  `json:"active_mode"` // "redis_langcache" or "memory_fallback"
	TotalSearches int64   `json:"total_searches"`
	Hits          int64   `json:"hits"`
	Misses        int64   `json:"misses"`
	HitRatio      float64 `json:"hit_ratio"`
	CachedEntries int     `json:"cached_entries"`
	Connected     bool    `json:"connected"`
}

// LangCache defines the semantic caching contract.
type LangCache interface {
	Search(ctx context.Context, prompt string, threshold float64) (*CacheHit, error)
	Set(ctx context.Context, prompt string, resp *models.ChatResponse, metadata map[string]any, ttl time.Duration) error
	GetStats(ctx context.Context) CacheStats
	Health(ctx context.Context) bool
}
