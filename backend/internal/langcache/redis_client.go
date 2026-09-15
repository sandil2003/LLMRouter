package langcache

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/llmrouter/backend/internal/models"
)

// RedisLangCacheClient interacts with Redis LangCache REST Service.
type RedisLangCacheClient struct {
	endpoint      string
	cacheID       string
	apiKey        string
	httpClient    *http.Client
	fallback      *MemorySemanticCache
	totalSearches int64
	hits          int64
	misses        int64
	isAvailable   int32 // 1 if available, 0 if in fallback
}

type Config struct {
	Endpoint string
	CacheID  string
	APIKey   string
	RedisURL string
}

func NewLangCache(cfg Config) LangCache {
	fallback := NewMemorySemanticCache(1000)

	if cfg.Endpoint == "" && cfg.RedisURL == "" {
		slog.Info("No Redis or LangCache endpoint configured; utilizing embedded in-memory semantic cache.")
		return fallback
	}

	client := &RedisLangCacheClient{
		endpoint:   cfg.Endpoint,
		cacheID:    cfg.CacheID,
		apiKey:     cfg.APIKey,
		httpClient: &http.Client{Timeout: 3 * time.Second},
		fallback:   fallback,
	}

	// Probe health on startup
	if client.Health(context.Background()) {
		atomic.StoreInt32(&client.isAvailable, 1)
		slog.Info("Connected to Redis LangCache service successfully", "endpoint", cfg.Endpoint)
	} else {
		atomic.StoreInt32(&client.isAvailable, 0)
		slog.Warn("Redis LangCache endpoint unreachable; will use embedded in-memory semantic fallback")
	}

	return client
}

type searchRequest struct {
	Prompt    string  `json:"prompt"`
	Threshold float64 `json:"threshold"`
}

type searchResponse struct {
	Hit        bool                 `json:"hit"`
	Similarity float64              `json:"similarity"`
	Prompt     string               `json:"prompt"`
	Response   *models.ChatResponse `json:"response"`
}

func (r *RedisLangCacheClient) Search(ctx context.Context, prompt string, threshold float64) (*CacheHit, error) {
	atomic.AddInt64(&r.totalSearches, 1)

	// If marked unavailable or missing endpoint, route to local memory fallback
	if atomic.LoadInt32(&r.isAvailable) == 0 || r.endpoint == "" {
		return r.fallback.Search(ctx, prompt, threshold)
	}

	url := fmt.Sprintf("%s/v1/caches/%s/search", r.endpoint, r.cacheID)
	reqBody, _ := json.Marshal(searchRequest{
		Prompt:    prompt,
		Threshold: threshold,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return r.fallback.Search(ctx, prompt, threshold)
	}

	req.Header.Set("Content-Type", "application/json")
	if r.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.apiKey)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		atomic.StoreInt32(&r.isAvailable, 0)
		slog.Warn("Redis LangCache request failed, falling back to in-memory semantic cache", "err", err)
		return r.fallback.Search(ctx, prompt, threshold)
	}
	defer resp.Body.Close()

	var searchResp searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return r.fallback.Search(ctx, prompt, threshold)
	}

	if searchResp.Hit && searchResp.Response != nil {
		atomic.AddInt64(&r.hits, 1)
		return &CacheHit{
			Response:   searchResp.Response,
			Similarity: searchResp.Similarity,
			Prompt:     searchResp.Prompt,
			CachedAt:   time.Now(),
			Source:     "redis_langcache",
		}, nil
	}

	atomic.AddInt64(&r.misses, 1)
	return nil, nil
}

type setRequest struct {
	Prompt   string               `json:"prompt"`
	Response *models.ChatResponse `json:"response"`
	Metadata map[string]any       `json:"metadata,omitempty"`
	TTL      int64                `json:"ttl_seconds,omitempty"`
}

func (r *RedisLangCacheClient) Set(ctx context.Context, prompt string, resp *models.ChatResponse, metadata map[string]any, ttl time.Duration) error {
	// Always seed local memory fallback as well for ultra-fast local hits
	_ = r.fallback.Set(ctx, prompt, resp, metadata, ttl)

	if atomic.LoadInt32(&r.isAvailable) == 0 || r.endpoint == "" {
		return nil
	}

	url := fmt.Sprintf("%s/v1/caches/%s/entries", r.endpoint, r.cacheID)
	reqBody, _ := json.Marshal(setRequest{
		Prompt:   prompt,
		Response: resp,
		Metadata: metadata,
		TTL:      int64(ttl.Seconds()),
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if r.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.apiKey)
	}

	httpResp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()
	_, _ = io.Copy(io.Discard, httpResp.Body)

	return nil
}

func (r *RedisLangCacheClient) GetStats(ctx context.Context) CacheStats {
	isConn := atomic.LoadInt32(&r.isAvailable) == 1
	mode := "redis_langcache"
	if !isConn {
		mode = "memory_fallback"
	}

	total := atomic.LoadInt64(&r.totalSearches)
	hits := atomic.LoadInt64(&r.hits)
	misses := atomic.LoadInt64(&r.misses)

	memStats := r.fallback.GetStats(ctx)
	if !isConn {
		return memStats
	}

	ratio := 0.0
	if total > 0 {
		ratio = float64(hits) / float64(total)
	}

	return CacheStats{
		ActiveMode:    mode,
		TotalSearches: total,
		Hits:          hits,
		Misses:        misses,
		HitRatio:      ratio,
		CachedEntries: memStats.CachedEntries,
		Connected:     isConn,
	}
}

func (r *RedisLangCacheClient) Health(ctx context.Context) bool {
	if r.endpoint == "" {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.endpoint+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 400
}
