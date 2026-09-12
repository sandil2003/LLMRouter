package providers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/llmrouter/backend/internal/models"
)

// Standard typed errors across all providers
var (
	ErrProviderDown      = errors.New("provider service is down or unreachable")
	ErrQuotaExceeded     = errors.New("provider quota exceeded")
	ErrUnauthorized      = errors.New("invalid or missing API credentials")
	ErrModelNotSupported = errors.New("requested model is not supported by this provider")
)

// RateLimitError signals an explicit 429 or quota throttling event.
type RateLimitError struct {
	ProviderID   string
	ProviderName string
	RetryAfter   time.Duration
	Message      string
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limit reached on %s (retry after %v): %s", e.ProviderName, e.RetryAfter, e.Message)
	}
	return fmt.Sprintf("rate limit reached on %s: %s", e.ProviderName, e.Message)
}

// TransientError indicates a 5xx server error that is eligible for retry.
type TransientError struct {
	StatusCode int
	Message    string
}

func (e *TransientError) Error() string {
	return fmt.Sprintf("transient error (%d): %s", e.StatusCode, e.Message)
}

// ClientError indicates a 4xx error (bad request, etc.) that should not be retried.
type ClientError struct {
	StatusCode int
	Message    string
}

func (e *ClientError) Error() string {
	return fmt.Sprintf("client error (%d): %s", e.StatusCode, e.Message)
}

// StreamEvent represents either a streamed chunk or a terminal error.
type StreamEvent struct {
	Chunk *models.ChatCompletionChunk
	Err   error
}

// Provider is the primary abstraction every LLM backend implements.
type Provider interface {
	ID() string
	Name() string
	Chat(ctx context.Context, req *models.ChatRequest) (*models.ChatResponse, error)
	ChatStream(ctx context.Context, req *models.ChatRequest) (<-chan StreamEvent, error)
	HealthCheck(ctx context.Context) error
	SupportsModel(model string) bool
}
