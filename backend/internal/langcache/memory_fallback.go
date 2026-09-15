package langcache

import (
	"context"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/llmrouter/backend/internal/models"
)

var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true,
	"in": true, "on": true, "for": true, "to": true, "of": true,
	"can": true, "you": true, "how": true, "what": true, "does": true,
	"do": true, "explain": true, "please": true, "me": true,
}

type cacheEntry struct {
	prompt     string
	tokens     map[string]float64
	magnitude  float64
	response   *models.ChatResponse
	metadata   map[string]any
	createdAt  time.Time
	lastAccess time.Time
}

// MemorySemanticCache implements LangCache using in-memory character shingle & weighted word vector cosine similarity.
type MemorySemanticCache struct {
	mu            sync.RWMutex
	entries       []*cacheEntry
	maxEntries    int
	totalSearches int64
	hits          int64
	misses        int64
}

func NewMemorySemanticCache(maxEntries int) *MemorySemanticCache {
	if maxEntries <= 0 {
		maxEntries = 500
	}
	return &MemorySemanticCache{
		entries:    make([]*cacheEntry, 0, maxEntries),
		maxEntries: maxEntries,
	}
}

func (m *MemorySemanticCache) extractVector(text string) (map[string]float64, float64) {
	clean := strings.ToLower(strings.TrimSpace(text))
	words := strings.FieldsFunc(clean, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	counts := make(map[string]float64)

	// 1. Content words with stopword attenuation & suffix trimming
	for _, w := range words {
		if stopWords[w] {
			counts["w:"+w] += 0.2
			continue
		}
		stemmed := strings.TrimSuffix(strings.TrimSuffix(w, "s"), "ing")
		counts["w:"+stemmed] += 3.0
	}

	// 2. Character 3-grams across the clean text for morphological resilience
	runes := []rune(clean)
	if len(runes) >= 3 {
		for i := 0; i <= len(runes)-3; i++ {
			tri := string(runes[i : i+3])
			counts["c:"+tri] += 0.5
		}
	}

	sumSq := 0.0
	for _, cnt := range counts {
		sumSq += cnt * cnt
	}

	return counts, math.Sqrt(sumSq)
}

func (m *MemorySemanticCache) cosineSimilarity(v1, v2 map[string]float64, mag1, mag2 float64) float64 {
	if mag1 == 0 || mag2 == 0 {
		return 0.0
	}

	dot := 0.0
	for k, val1 := range v1 {
		if val2, exists := v2[k]; exists {
			dot += val1 * val2
		}
	}

	sim := dot / (mag1 * mag2)
	if sim > 1.0 {
		return 1.0
	}
	return sim
}

func (m *MemorySemanticCache) Search(ctx context.Context, prompt string, threshold float64) (*CacheHit, error) {
	atomic.AddInt64(&m.totalSearches, 1)

	if threshold <= 0 {
		threshold = 0.70
	}

	qTokens, qMag := m.extractVector(prompt)
	if qMag == 0 {
		atomic.AddInt64(&m.misses, 1)
		return nil, nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var bestEntry *cacheEntry
	var bestSim float64 = 0.0

	for _, entry := range m.entries {
		sim := m.cosineSimilarity(qTokens, entry.tokens, qMag, entry.magnitude)
		if sim > bestSim {
			bestSim = sim
			bestEntry = entry
		}
	}

	if bestEntry != nil && bestSim >= threshold {
		atomic.AddInt64(&m.hits, 1)
		bestEntry.lastAccess = time.Now()
		return &CacheHit{
			Response:   bestEntry.response,
			Similarity: math.Round(bestSim*100) / 100,
			Prompt:     bestEntry.prompt,
			CachedAt:   bestEntry.createdAt,
			Source:     "memory_fallback",
		}, nil
	}

	atomic.AddInt64(&m.misses, 1)
	return nil, nil
}

func (m *MemorySemanticCache) Set(ctx context.Context, prompt string, resp *models.ChatResponse, metadata map[string]any, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tokens, mag := m.extractVector(prompt)
	entry := &cacheEntry{
		prompt:     prompt,
		tokens:     tokens,
		magnitude:  mag,
		response:   resp,
		metadata:   metadata,
		createdAt:  time.Now(),
		lastAccess: time.Now(),
	}

	// LRU Eviction if capacity reached
	if len(m.entries) >= m.maxEntries {
		oldestIdx := 0
		oldestTime := m.entries[0].lastAccess
		for i := 1; i < len(m.entries); i++ {
			if m.entries[i].lastAccess.Before(oldestTime) {
				oldestTime = m.entries[i].lastAccess
				oldestIdx = i
			}
		}
		m.entries = append(m.entries[:oldestIdx], m.entries[oldestIdx+1:]...)
	}

	m.entries = append(m.entries, entry)
	return nil
}

func (m *MemorySemanticCache) GetStats(ctx context.Context) CacheStats {
	total := atomic.LoadInt64(&m.totalSearches)
	hits := atomic.LoadInt64(&m.hits)
	misses := atomic.LoadInt64(&m.misses)

	ratio := 0.0
	if total > 0 {
		ratio = float64(hits) / float64(total)
	}

	m.mu.RLock()
	count := len(m.entries)
	m.mu.RUnlock()

	return CacheStats{
		ActiveMode:    "memory_fallback",
		TotalSearches: total,
		Hits:          hits,
		Misses:        misses,
		HitRatio:      math.Round(ratio*1000) / 1000,
		CachedEntries: count,
		Connected:     true,
	}
}

func (m *MemorySemanticCache) Health(ctx context.Context) bool {
	return true
}
