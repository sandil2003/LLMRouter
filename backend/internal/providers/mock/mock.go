package mock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/llmrouter/backend/internal/models"
	"github.com/llmrouter/backend/internal/providers"
)

// Provider is an in-memory mock LLM provider for robust unit and fallback testing.
type Provider struct {
	mu sync.Mutex

	id             string
	name           string
	supportedModel string

	// Simulated behaviors
	ShouldRateLimit   bool
	RateLimitDuration time.Duration
	ShouldFail        bool
	FailStatusCode    int
	ResponseContent   string
	StreamDelays      time.Duration
	CallCount         int
	LastRequest       *models.ChatRequest
}

func New(id, name, supportedModel string) *Provider {
	return &Provider{
		id:              id,
		name:            name,
		supportedModel:  supportedModel,
		ResponseContent: "Hello from " + name,
	}
}

func (m *Provider) ID() string {
	return m.id
}

func (m *Provider) Name() string {
	return m.name
}

func (m *Provider) SupportsModel(model string) bool {
	if m.supportedModel == "" || m.supportedModel == "*" {
		return true
	}
	return m.supportedModel == model
}

func (m *Provider) HealthCheck(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ShouldFail {
		return fmt.Errorf("mock health check failed for %s", m.name)
	}
	return nil
}

func (m *Provider) Chat(ctx context.Context, req *models.ChatRequest) (*models.ChatResponse, error) {
	m.mu.Lock()
	m.CallCount++
	m.LastRequest = req
	shouldRL := m.ShouldRateLimit
	rlDuration := m.RateLimitDuration
	shouldFail := m.ShouldFail
	failStatus := m.FailStatusCode
	content := m.ResponseContent
	m.mu.Unlock()

	if shouldRL {
		return nil, &providers.RateLimitError{
			ProviderID:   m.id,
			ProviderName: m.name,
			RetryAfter:   rlDuration,
			Message:      "mock quota exceeded (429)",
		}
	}

	if shouldFail {
		if failStatus == 0 {
			failStatus = 500
		}
		return nil, &providers.TransientError{
			StatusCode: failStatus,
			Message:    "mock internal server error",
		}
	}

	return &models.ChatResponse{
		ID:      fmt.Sprintf("mock-%s-%d", m.id, time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.Choice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    "assistant",
					Content: content,
				},
			},
		},
		Usage: &models.Usage{
			PromptTokens:     15,
			CompletionTokens: 25,
			TotalTokens:      40,
		},
	}, nil
}

func (m *Provider) ChatStream(ctx context.Context, req *models.ChatRequest) (<-chan providers.StreamEvent, error) {
	m.mu.Lock()
	m.CallCount++
	m.LastRequest = req
	shouldRL := m.ShouldRateLimit
	rlDuration := m.RateLimitDuration
	shouldFail := m.ShouldFail
	failStatus := m.FailStatusCode
	content := m.ResponseContent
	m.mu.Unlock()

	if shouldRL {
		return nil, &providers.RateLimitError{
			ProviderID:   m.id,
			ProviderName: m.name,
			RetryAfter:   rlDuration,
			Message:      "mock quota exceeded (429)",
		}
	}

	if shouldFail {
		if failStatus == 0 {
			failStatus = 500
		}
		return nil, &providers.TransientError{
			StatusCode: failStatus,
			Message:    "mock stream connection failed",
		}
	}

	out := make(chan providers.StreamEvent, 4)
	go func() {
		defer close(out)

		words := []string{"Hello", " ", "from", " ", m.name, "!"}
		if content != "" {
			words = []string{content}
		}

		for _, word := range words {
			select {
			case <-ctx.Done():
				out <- providers.StreamEvent{Err: ctx.Err()}
				return
			default:
				chunk := &models.ChatCompletionChunk{
					ID:      fmt.Sprintf("mock-stream-%s", m.id),
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   req.Model,
					Choices: []models.ChunkChoice{
						{
							Index: 0,
							Delta: models.ChunkDelta{
								Content: word,
							},
						},
					},
				}
				out <- providers.StreamEvent{Chunk: chunk}
			}
		}
	}()

	return out, nil
}
