package router

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/llmrouter/backend/internal/circuitbreaker"
	"github.com/llmrouter/backend/internal/database/repository"
	"github.com/llmrouter/backend/internal/models"
	"github.com/llmrouter/backend/internal/providers"
	"github.com/llmrouter/backend/internal/ratelimit"
	"github.com/llmrouter/backend/internal/retry"
)

var (
	ErrNoProvidersAvailable = errors.New("no healthy or available providers found for the request")
	ErrAllProvidersFailed   = errors.New("all candidate providers failed to fulfill request")
)

type Engine struct {
	providerRepo   *repository.ProviderRepository
	modelRepo      *repository.ModelRepository
	routingRepo    *repository.RoutingRepository
	usageRepo      *repository.UsageRepository
	logRepo        *repository.LogRepository
	registry       *providers.Registry
	rateLimiter    *ratelimit.Tracker
	circuitBreaker *circuitbreaker.Manager
	strategies     map[models.RoutingStrategy]Strategy
	retryCfg       retry.Config
}

func NewEngine(
	providerRepo *repository.ProviderRepository,
	modelRepo *repository.ModelRepository,
	routingRepo *repository.RoutingRepository,
	usageRepo *repository.UsageRepository,
	logRepo *repository.LogRepository,
	registry *providers.Registry,
	rateLimiter *ratelimit.Tracker,
	circuitBreaker *circuitbreaker.Manager,
) *Engine {
	strategies := map[models.RoutingStrategy]Strategy{
		models.StrategyPriority: NewPriorityStrategy(),
	}

	return &Engine{
		providerRepo:   providerRepo,
		modelRepo:      modelRepo,
		routingRepo:    routingRepo,
		usageRepo:      usageRepo,
		logRepo:        logRepo,
		registry:       registry,
		rateLimiter:    rateLimiter,
		circuitBreaker: circuitBreaker,
		strategies:     strategies,
		retryCfg:       retry.DefaultConfig(),
	}
}

// GetCandidates finds active, supported, and unblocked provider instances.
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

		if !p.SupportsModel(model) {
			continue
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
		return e.strategies[models.StrategyPriority]
	}

	strat, ok := e.strategies[rule.Strategy]
	if !ok {
		return e.strategies[models.StrategyPriority]
	}
	return strat
}

// ExecuteChat executes non-streaming chat completions with transparent fallback & retry.
func (e *Engine) ExecuteChat(ctx context.Context, req *models.ChatRequest, requestID string) (*models.ChatResponse, error) {
	start := time.Now()

	candidates, err := e.GetCandidates(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, ErrNoProvidersAvailable
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

		slog.Info("Attempting provider",
			"request_id", requestID,
			"provider_id", pID,
			"provider_name", pName,
			"fallback", fallbackUsed,
			"priority", candidate.Config.Priority,
		)

		var resp *models.ChatResponse
		var opErr error

		// Retry logic for transient 5xx failures
		opErr = retry.Do(ctx, e.retryCfg, func(err error) bool {
			var transErr *providers.TransientError
			return errors.As(err, &transErr)
		}, func(attempt int) error {
			var err error
			resp, err = candidate.Provider.Chat(ctx, req)
			return err
		})

		latencyMs := time.Since(start).Milliseconds()

		if opErr == nil {
			// Success!
			e.circuitBreaker.RecordSuccess(pID)

			// Record log
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

				tokensIn, tokensOut := 0, 0
				if resp.Usage != nil {
					tokensIn = resp.Usage.PromptTokens
					tokensOut = resp.Usage.CompletionTokens
				}
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

		// Classify error and record status
		isRL, retryAfter := ratelimit.IsRateLimit(opErr)
		if isRL {
			until := e.rateLimiter.MarkRateLimited(pID, retryAfter, errStr)
			slog.Warn("Provider rate limited, initiating fallback",
				"provider", pID,
				"retry_after", retryAfter,
				"blocked_until", until,
			)
			e.circuitBreaker.RecordFailure(pID)
		} else {
			slog.Warn("Provider request failed",
				"provider", pID,
				"error", errStr,
			)
			e.circuitBreaker.RecordFailure(pID)
		}

		// Record failed attempt log
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

		// If this is a bad client request (4xx not rate-limited), don't fallback across providers
		var clientErr *providers.ClientError
		if errors.As(opErr, &clientErr) && !strings.Contains(errStr, "model") {
			return nil, opErr
		}
	}

	return nil, fmt.Errorf("%w: last error: %v", ErrAllProvidersFailed, lastErr)
}

// ExecuteStream executes streaming chat completions with fallback on connection failure.
func (e *Engine) ExecuteStream(ctx context.Context, req *models.ChatRequest, requestID string) (<-chan providers.StreamEvent, error) {
	start := time.Now()

	candidates, err := e.GetCandidates(ctx, req.Model)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, ErrNoProvidersAvailable
	}

	strat := e.SelectStrategy(ctx)
	ordered := strat.OrderCandidates(ctx, candidates)

	var lastErr error

	for i, candidate := range ordered {
		pID := candidate.Config.ID
		fallbackUsed := i > 0

		slog.Info("Attempting streaming provider",
			"request_id", requestID,
			"provider", pID,
			"fallback", fallbackUsed,
		)

		streamChan, err := candidate.Provider.ChatStream(ctx, req)
		if err != nil {
			lastErr = err
			errStr := err.Error()

			isRL, retryAfter := ratelimit.IsRateLimit(err)
			if isRL {
				e.rateLimiter.MarkRateLimited(pID, retryAfter, errStr)
			}
			e.circuitBreaker.RecordFailure(pID)

			latencyMs := time.Since(start).Milliseconds()
			go func(pID, errText string) {
				_ = e.logRepo.RecordLog(context.Background(), &models.RequestLog{
					RequestID:    requestID,
					ProviderID:   pID,
					Model:        req.Model,
					Status:       "error",
					LatencyMs:    latencyMs,
					Error:        &errText,
					FallbackUsed: fallbackUsed,
					AttemptCount: i + 1,
				})
			}(pID, errStr)

			continue // Fallback to next provider candidate!
		}

		// Connected successfully!
		e.circuitBreaker.RecordSuccess(pID)

		outChan := make(chan providers.StreamEvent, 16)
		go func() {
			defer close(outChan)
			for event := range streamChan {
				outChan <- event
			}

			latencyMs := time.Since(start).Milliseconds()
			go func() {
				_ = e.logRepo.RecordLog(context.Background(), &models.RequestLog{
					RequestID:    requestID,
					ProviderID:   pID,
					Model:        req.Model,
					Status:       "success",
					LatencyMs:    latencyMs,
					FallbackUsed: fallbackUsed,
					AttemptCount: i + 1,
				})
			}()
		}()

		return outChan, nil
	}

	return nil, fmt.Errorf("%w: last error: %v", ErrAllProvidersFailed, lastErr)
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
