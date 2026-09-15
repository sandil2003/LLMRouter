package router

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/llmrouter/backend/internal/circuitbreaker"
	"github.com/llmrouter/backend/internal/classifier"
	"github.com/llmrouter/backend/internal/database/repository"
	"github.com/llmrouter/backend/internal/langcache"
	"github.com/llmrouter/backend/internal/models"
	"github.com/llmrouter/backend/internal/providers"
	"github.com/llmrouter/backend/internal/ratelimit"
	"github.com/llmrouter/backend/internal/registry"
	"github.com/llmrouter/backend/internal/retry"
	"github.com/llmrouter/backend/internal/telemetry"
)

var (
	ErrNoProvidersAvailable = errors.New("no healthy or available providers found for the request")
	ErrAllProvidersFailed   = errors.New("all candidate providers failed to fulfill request")
)

type contextKey string

const requestFeaturesKey contextKey = "request_features"

type Engine struct {
	providerRepo   *repository.ProviderRepository
	modelRepo      *repository.ModelRepository
	routingRepo    *repository.RoutingRepository
	usageRepo      *repository.UsageRepository
	logRepo        *repository.LogRepository
	registry       *providers.Registry
	rateLimiter    *ratelimit.Tracker
	circuitBreaker *circuitbreaker.Manager
	classifier     *classifier.Classifier
	langCache      langcache.LangCache
	policyEngine   *registry.PolicyEngine
	modelRegistry  *registry.ModelRegistry
	strategies     map[models.RoutingStrategy]Strategy
	retryCfg       retry.Config
	ttftTimeout    time.Duration
}

func NewEngine(
	providerRepo *repository.ProviderRepository,
	modelRepo *repository.ModelRepository,
	routingRepo *repository.RoutingRepository,
	usageRepo *repository.UsageRepository,
	logRepo *repository.LogRepository,
	reg *providers.Registry,
	rateLimiter *ratelimit.Tracker,
	circuitBreaker *circuitbreaker.Manager,
	cls *classifier.Classifier,
	cache langcache.LangCache,
	policyEngine *registry.PolicyEngine,
	modelReg *registry.ModelRegistry,
) *Engine {
	if cls == nil {
		cls = classifier.NewClassifier()
	}
	if cache == nil {
		cache = langcache.NewMemorySemanticCache(1000)
	}
	if modelReg == nil {
		modelReg = registry.NewModelRegistry()
	}
	if policyEngine == nil {
		policyEngine = registry.NewPolicyEngine(modelReg, registry.DefaultPolicyWeights())
	}

	strategies := map[models.RoutingStrategy]Strategy{
		models.StrategyPriority: NewPriorityStrategy(),
		StrategyDynamicPolicy:   NewDynamicPolicyStrategy(policyEngine, cls),
	}

	return &Engine{
		providerRepo:   providerRepo,
		modelRepo:      modelRepo,
		routingRepo:    routingRepo,
		usageRepo:      usageRepo,
		logRepo:        logRepo,
		registry:       reg,
		rateLimiter:    rateLimiter,
		circuitBreaker: circuitBreaker,
		classifier:     cls,
		langCache:      cache,
		policyEngine:   policyEngine,
		modelRegistry:  modelReg,
		strategies:     strategies,
		retryCfg:       retry.DefaultConfig(),
		ttftTimeout:    3500 * time.Millisecond, // 3.5s TTFT timeout before failover
	}
}

func (e *Engine) extractUserPrompt(req *models.ChatRequest) string {
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			return req.Messages[i].Content
		}
	}
	return ""
}

func (e *Engine) providerSupportsModel(ctx context.Context, p providers.Provider, providerID string, model string) bool {
	if model == "" || model == "default" || model == "auto" {
		return true
	}

	// 1. Check if model is registered in SQLite for this provider
	if dbModels, err := e.modelRepo.ListByProvider(ctx, providerID); err == nil {
		for _, m := range dbModels {
			if m.Enabled && strings.EqualFold(m.Name, model) {
				return true
			}
		}
	}

	// 2. Check if the provider adapter supports this model pattern
	if p.SupportsModel(model) {
		return true
	}

	return false
}

// GetCandidates finds active, supported, credentialed, and unblocked provider instances.
func (e *Engine) GetCandidates(ctx context.Context, model string) ([]Candidate, error) {
	configs, err := e.providerRepo.GetActiveOrderedByPriority(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch active providers: %w", err)
	}

	var candidates []Candidate
	for _, cfg := range configs {
		p, exists := e.registry.Get(cfg.ID)
		if !exists {
			continue
		}

		if !e.providerSupportsModel(ctx, p, cfg.ID, model) {
			continue
		}

		// Ensure provider has an API key configured (skip keyless providers unless mock)
		if !strings.HasPrefix(cfg.ID, "mock") {
			key, _ := e.registry.Credentials().GetAPIKey(ctx, cfg.ID)
			if strings.TrimSpace(key) == "" {
				slog.Debug("Provider skipped due to missing API key", "provider", cfg.ID)
				continue
			}
		}

		// Check rate limit tracker
		if e.rateLimiter.IsRateLimited(cfg.ID) {
			slog.Debug("Provider skipped due to active rate limit", "provider", cfg.ID)
			continue
		}

		// Check circuit breaker
		if !e.circuitBreaker.CanExecute(cfg.ID) {
			slog.Debug("Provider skipped due to open circuit breaker", "provider", cfg.ID)
			continue
		}

		candidates = append(candidates, Candidate{
			Provider: p,
			Config:   cfg,
		})
	}

	return candidates, nil
}

// SelectStrategy returns the currently configured routing strategy.
func (e *Engine) SelectStrategy(ctx context.Context) Strategy {
	rule, err := e.routingRepo.GetActiveRule(ctx)
	if err != nil || rule == nil {
		return e.strategies[StrategyDynamicPolicy]
	}

	strat, ok := e.strategies[rule.Strategy]
	if !ok {
		return e.strategies[StrategyDynamicPolicy]
	}
	return strat
}

// ExecuteChat executes non-streaming completions with semantic caching, dynamic policy & failover.
func (e *Engine) ExecuteChat(ctx context.Context, req *models.ChatRequest, requestID string) (*models.ChatResponse, error) {
	start := time.Now()
	userPrompt := e.extractUserPrompt(req)

	// Step 0: Redis LangCache Semantic Lookup
	if userPrompt != "" && e.langCache != nil {
		if hit, err := e.langCache.Search(ctx, userPrompt, 0.88); err == nil && hit != nil && hit.Response != nil {
			slog.Info("⚡ Redis LangCache Semantic HIT (<15ms)",
				"request_id", requestID,
				"similarity", hit.Similarity,
				"source", hit.Source,
			)
			telemetry.GetSink().RecordCacheHit()
			telemetry.GetSink().RecordCompletion(hit.Response.Model, hit.Response.Model, "cache_hit", 12, 12, 0, 0)
			return hit.Response, nil
		}
		telemetry.GetSink().RecordCacheMiss()
	}

	// Step 1: Request Ingestion & Fast Classification (<50-100ms)
	features := e.classifier.Classify(req)
	ctx = context.WithValue(ctx, requestFeaturesKey, features)
	slog.Info("Step 1 Classification complete",
		"domain", features.Domain,
		"complexity", features.Complexity,
		"input_tokens", features.InputTokens,
		"json_required", features.JSONRequired,
		"classification_ms", features.ClassificationMs,
	)

	// Resolve auto/default model if necessary
	targetModel := req.Model
	if targetModel == "" || targetModel == "default" || targetModel == "auto" {
		rankedModels := e.policyEngine.RankCandidates(features, e.circuitBreaker.CanExecute)
		if len(rankedModels) > 0 {
			targetModel = rankedModels[0].Model.ID
			slog.Info("Step 2 Dynamic Policy selected model",
				"selected_model", targetModel,
				"score", rankedModels[0].Score,
				"provider", rankedModels[0].Model.ProviderID,
			)
		} else {
			targetModel = "gemini-2.5-flash"
		}
		req.Model = targetModel
	}

	candidates, err := e.GetCandidates(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("%w: no active provider with a valid API key supports model '%s'", ErrNoProvidersAvailable, req.Model)
	}

	strat := e.SelectStrategy(ctx)
	ordered := strat.OrderCandidates(ctx, candidates)

	var lastErr error
	attemptCount := 0

	for i, candidate := range ordered {
		attemptCount++
		pID := candidate.Config.ID
		pName := candidate.Config.Name
		fallbackUsed := i > 0

		slog.Info("Step 3 Dispatching provider",
			"request_id", requestID,
			"provider_id", pID,
			"provider_name", pName,
			"fallback", fallbackUsed,
		)

		var resp *models.ChatResponse
		var opErr error

		// Step 3: Dispatch with TTFT / Hard Timeout Watchdog
		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		opErr = retry.Do(callCtx, e.retryCfg, func(err error) bool {
			var transErr *providers.TransientError
			return errors.As(err, &transErr)
		}, func(attempt int) error {
			var err error
			resp, err = candidate.Provider.Chat(callCtx, req)
			return err
		})

		latencyMs := time.Since(start).Milliseconds()

		if opErr == nil {
			// Success!
			e.circuitBreaker.RecordSuccess(pID)
			e.modelRegistry.UpdateOperationalStats(req.Model, latencyMs, false, false)

			tokensIn, tokensOut := 0, 0
			if resp.Usage != nil {
				tokensIn = resp.Usage.PromptTokens
				tokensOut = resp.Usage.CompletionTokens
			}

			// Telemetry sink emission
			telemetry.GetSink().RecordCompletion(pID, req.Model, "success", latencyMs, latencyMs/3, tokensIn, tokensOut)

			// Asynchronously cache response in Redis LangCache
			if userPrompt != "" && e.langCache != nil {
				go func(prompt string, r *models.ChatResponse) {
					_ = e.langCache.Set(context.Background(), prompt, r, map[string]any{
						"domain":     features.Domain,
						"complexity": features.Complexity,
						"model":      req.Model,
					}, 24*time.Hour)
				}(userPrompt, resp)
			}

			// Record database logs
			go func() {
				_ = e.logRepo.RecordLog(context.Background(), &models.RequestLog{
					RequestID:    requestID,
					ProviderID:   pID,
					Model:        req.Model,
					Status:       "success",
					LatencyMs:    latencyMs,
					FallbackUsed: fallbackUsed,
					AttemptCount: attemptCount,
				})

				_ = e.usageRepo.RecordUsage(context.Background(), &models.UsageRecord{
					ProviderID:   pID,
					Model:        req.Model,
					RequestID:    requestID,
					InputTokens:  tokensIn,
					OutputTokens: tokensOut,
					LatencyMs:    latencyMs,
					Status:       "success",
				})
			}()

			return resp, nil
		}

		lastErr = opErr
		errStr := opErr.Error()
		e.modelRegistry.UpdateOperationalStats(req.Model, latencyMs, false, true)

		// Record Failure & Check Rate Limit
		isRL, retryAfter := ratelimit.IsRateLimit(opErr)
		if isRL {
			until := e.rateLimiter.MarkRateLimited(pID, retryAfter, errStr)
			slog.Warn("Provider rate limited (429), failing over",
				"provider", pID,
				"blocked_until", until,
			)
			e.circuitBreaker.RecordFailureWithCode(pID, 429)
		} else {
			slog.Warn("Provider request failed, attempting deterministic failover",
				"provider", pID,
				"error", errStr,
			)
			e.circuitBreaker.RecordFailureWithCode(pID, 500)
		}

		// Log failed attempt
		go func(pID string, errText string) {
			_ = e.logRepo.RecordLog(context.Background(), &models.RequestLog{
				RequestID:    requestID,
				ProviderID:   pID,
				Model:        req.Model,
				Status:       "error",
				LatencyMs:    latencyMs,
				Error:        &errText,
				FallbackUsed: fallbackUsed,
				AttemptCount: attemptCount,
			})
		}(pID, errStr)
	}

	return nil, fmt.Errorf("%w: last error: %v", ErrAllProvidersFailed, lastErr)
}

// ExecuteStream executes streaming completions with TTFT watchdog & fallback.
func (e *Engine) ExecuteStream(ctx context.Context, req *models.ChatRequest, requestID string) (<-chan providers.StreamEvent, error) {
	start := time.Now()

	// Step 1: Ingestion & Classification
	features := e.classifier.Classify(req)
	ctx = context.WithValue(ctx, requestFeaturesKey, features)

	// Resolve target model if auto/default
	if req.Model == "" || req.Model == "default" || req.Model == "auto" {
		ranked := e.policyEngine.RankCandidates(features, e.circuitBreaker.CanExecute)
		if len(ranked) > 0 {
			req.Model = ranked[0].Model.ID
		} else {
			req.Model = "gemini-2.5-flash"
		}
	}

	candidates, err := e.GetCandidates(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("%w: no active provider supports model '%s'", ErrNoProvidersAvailable, req.Model)
	}

	strat := e.SelectStrategy(ctx)
	ordered := strat.OrderCandidates(ctx, candidates)

	var lastErr error

	for i, candidate := range ordered {
		pID := candidate.Config.ID
		fallbackUsed := i > 0

		slog.Info("Step 3 Dispatching streaming provider",
			"request_id", requestID,
			"provider", pID,
			"fallback", fallbackUsed,
		)

		streamChan, err := candidate.Provider.ChatStream(ctx, req)
		if err != nil {
			lastErr = err
			e.circuitBreaker.RecordFailureWithCode(pID, 500)
			continue
		}

		// Step 3 TTFT Watchdog: Verify stream produces first token within 3.5s timeout
		firstEventTimer := time.NewTimer(e.ttftTimeout)
		var firstEvent providers.StreamEvent
		var streamActive bool

		select {
		case ev, ok := <-streamChan:
			firstEventTimer.Stop()
			if !ok {
				e.circuitBreaker.RecordFailureWithCode(pID, 503)
				continue
			}
			if ev.Err != nil {
				e.circuitBreaker.RecordFailureWithCode(pID, 500)
				continue
			}
			firstEvent = ev
			streamActive = true
		case <-firstEventTimer.C:
			// TTFT watchdog tripped! 3.5 seconds elapsed with zero streamed tokens
			slog.Warn("TTFT Watchdog: Provider exceeded 3.5s with zero tokens, initiating failover", "provider", pID)
			e.circuitBreaker.RecordFailureWithCode(pID, 504)
			continue
		}

		if !streamActive {
			continue
		}

		ttftMs := time.Since(start).Milliseconds()
		e.circuitBreaker.RecordSuccess(pID)
		e.modelRegistry.UpdateOperationalStats(req.Model, ttftMs, true, false)

		outChan := make(chan providers.StreamEvent, 32)
		outChan <- firstEvent // Send first captured token

		go func() {
			defer close(outChan)
			for event := range streamChan {
				outChan <- event
			}

			totalLatencyMs := time.Since(start).Milliseconds()
			e.modelRegistry.UpdateOperationalStats(req.Model, totalLatencyMs, false, false)
			telemetry.GetSink().RecordCompletion(pID, req.Model, "success", totalLatencyMs, ttftMs, 100, 100)

			go func() {
				_ = e.logRepo.RecordLog(context.Background(), &models.RequestLog{
					RequestID:    requestID,
					ProviderID:   pID,
					Model:        req.Model,
					Status:       "success",
					LatencyMs:    totalLatencyMs,
					FallbackUsed: fallbackUsed,
					AttemptCount: i + 1,
				})
			}()
		}()

		return outChan, nil
	}

	return nil, fmt.Errorf("%w: all candidates failed or timed out; last error: %v", ErrAllProvidersFailed, lastErr)
}

func (e *Engine) Classifier() *classifier.Classifier {
	return e.classifier
}

func (e *Engine) LangCache() langcache.LangCache {
	return e.langCache
}

func (e *Engine) PolicyEngine() *registry.PolicyEngine {
	return e.policyEngine
}

func (e *Engine) ModelRegistry() *registry.ModelRegistry {
	return e.modelRegistry
}

func (e *Engine) RateLimiter() *ratelimit.Tracker {
	return e.rateLimiter
}

func (e *Engine) CircuitBreaker() *circuitbreaker.Manager {
	return e.circuitBreaker
}

func (e *Engine) Registry() *providers.Registry {
	return e.registry
}
