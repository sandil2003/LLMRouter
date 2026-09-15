package telemetry

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// MetricSink collects operational metrics and exposes them in Prometheus format.
type MetricSink struct {
	mu                  sync.RWMutex
	requestsTotal       map[string]int64 // key: provider:model:status
	cacheHitsTotal      int64
	cacheMissesTotal    int64
	tokensPromptTotal   map[string]int64 // key: provider:model
	tokensOutputTotal   map[string]int64 // key: provider:model
	latenciesTotalMs    map[string]int64 // key: provider:model
	ttftTotalMs         map[string]int64 // key: provider:model
	countsByModel       map[string]int64 // key: provider:model
	circuitBreakerState map[string]int   // key: provider -> 0: closed, 1: half_open, 2: open
}

var globalSink *MetricSink
var once sync.Once

func GetSink() *MetricSink {
	once.Do(func() {
		globalSink = &MetricSink{
			requestsTotal:       make(map[string]int64),
			tokensPromptTotal:   make(map[string]int64),
			tokensOutputTotal:   make(map[string]int64),
			latenciesTotalMs:    make(map[string]int64),
			ttftTotalMs:         make(map[string]int64),
			countsByModel:       make(map[string]int64),
			circuitBreakerState: make(map[string]int),
		}
	})
	return globalSink
}

// RecordCompletion records a request completion event.
func (s *MetricSink) RecordCompletion(provider, model, status string, latencyMs, ttftMs int64, inTokens, outTokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reqKey := fmt.Sprintf("%s:%s:%s", provider, model, status)
	s.requestsTotal[reqKey]++

	modKey := fmt.Sprintf("%s:%s", provider, model)
	s.tokensPromptTotal[modKey] += int64(inTokens)
	s.tokensOutputTotal[modKey] += int64(outTokens)
	s.latenciesTotalMs[modKey] += latencyMs
	if ttftMs > 0 {
		s.ttftTotalMs[modKey] += ttftMs
	}
	s.countsByModel[modKey]++
}

// RecordCacheHit records a Redis LangCache semantic hit.
func (s *MetricSink) RecordCacheHit() {
	atomic.AddInt64(&s.cacheHitsTotal, 1)
}

// RecordCacheMiss records a cache miss.
func (s *MetricSink) RecordCacheMiss() {
	atomic.AddInt64(&s.cacheMissesTotal, 1)
}

// SetCircuitState records circuit state: 0=closed, 1=half_open, 2=open.
func (s *MetricSink) SetCircuitState(provider string, state string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	val := 0
	if state == "half_open" {
		val = 1
	} else if state == "open" {
		val = 2
	}
	s.circuitBreakerState[provider] = val
}

// Handler returns an HTTP handler exposing metrics in standard Prometheus plain text format.
func (s *MetricSink) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.RLock()
		defer s.mu.RUnlock()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		// 1. LLM Router Cache metrics
		fmt.Fprintf(w, "# HELP llmrouter_cache_hits_total Total semantic cache hits in Redis LangCache\n")
		fmt.Fprintf(w, "# TYPE llmrouter_cache_hits_total counter\n")
		fmt.Fprintf(w, "llmrouter_cache_hits_total %d\n\n", atomic.LoadInt64(&s.cacheHitsTotal))

		fmt.Fprintf(w, "# HELP llmrouter_cache_misses_total Total semantic cache misses\n")
		fmt.Fprintf(w, "# TYPE llmrouter_cache_misses_total counter\n")
		fmt.Fprintf(w, "llmrouter_cache_misses_total %d\n\n", atomic.LoadInt64(&s.cacheMissesTotal))

		// 2. Request count metrics
		fmt.Fprintf(w, "# HELP llmrouter_requests_total Total completion requests by provider, model, and status\n")
		fmt.Fprintf(w, "# TYPE llmrouter_requests_total counter\n")
		for k, cnt := range s.requestsTotal {
			fmt.Fprintf(w, "llmrouter_requests_total{key=\"%s\"} %d\n", k, cnt)
		}
		fmt.Fprintln(w)

		// 3. Tokens billed
		fmt.Fprintf(w, "# HELP llmrouter_tokens_total Total tokens processed\n")
		fmt.Fprintf(w, "# TYPE llmrouter_tokens_total counter\n")
		for k, tok := range s.tokensPromptTotal {
			fmt.Fprintf(w, "llmrouter_tokens_total{key=\"%s\",type=\"prompt\"} %d\n", k, tok)
		}
		for k, tok := range s.tokensOutputTotal {
			fmt.Fprintf(w, "llmrouter_tokens_total{key=\"%s\",type=\"completion\"} %d\n", k, tok)
		}
		fmt.Fprintln(w)

		// 4. Circuit Breaker states
		fmt.Fprintf(w, "# HELP llmrouter_circuit_breaker_state Circuit breaker state (0=closed, 1=half_open, 2=open)\n")
		fmt.Fprintf(w, "# TYPE llmrouter_circuit_breaker_state gauge\n")
		for prov, st := range s.circuitBreakerState {
			fmt.Fprintf(w, "llmrouter_circuit_breaker_state{provider=\"%s\"} %d\n", prov, st)
		}
		fmt.Fprintln(w)

		// 5. Operational Latency
		fmt.Fprintf(w, "# HELP llmrouter_avg_latency_ms Average latency in milliseconds\n")
		fmt.Fprintf(w, "# TYPE llmrouter_avg_latency_ms gauge\n")
		for modKey, totalMs := range s.latenciesTotalMs {
			cnt := s.countsByModel[modKey]
			avg := float64(0)
			if cnt > 0 {
				avg = float64(totalMs) / float64(cnt)
			}
			fmt.Fprintf(w, "llmrouter_avg_latency_ms{target=\"%s\"} %.2f\n", modKey, avg)
		}
		fmt.Fprintf(w, "\n# Timestamp: %s\n", time.Now().UTC().Format(time.RFC3339))
	}
}
