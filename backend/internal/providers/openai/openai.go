package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/llmrouter/backend/internal/models"
	"github.com/llmrouter/backend/internal/providers"
)

type Provider struct {
	*providers.BaseHTTPClient
	id      string
	name    string
	apiKey  string
	baseURL string
}

func New(id, name, apiKey, baseURL string) *Provider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Provider{
		BaseHTTPClient: providers.NewBaseHTTPClient(60 * time.Second),
		id:             id,
		name:           name,
		apiKey:         apiKey,
		baseURL:        baseURL,
	}
}

func (p *Provider) ID() string {
	return p.id
}

func (p *Provider) Name() string {
	return p.name
}

func (p *Provider) SetAPIKey(key string) {
	p.apiKey = key
}

func (p *Provider) SupportsModel(model string) bool {
	// If needed, check model prefix or supported models list
	return true
}

func (p *Provider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", providers.ErrProviderDown, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return providers.ParseError(p.id, p.name, resp)
	}
	return nil
}

func (p *Provider) Chat(ctx context.Context, req *models.ChatRequest) (*models.ChatResponse, error) {
	reqCopy := *req
	reqCopy.Stream = false

	bodyBytes, err := json.Marshal(reqCopy)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", providers.ErrProviderDown, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, providers.ParseError(p.id, p.name, resp)
	}
	defer resp.Body.Close()

	var chatResp models.ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode chat response: %w", err)
	}

	return &chatResp, nil
}

func (p *Provider) ChatStream(ctx context.Context, req *models.ChatRequest) (<-chan providers.StreamEvent, error) {
	reqCopy := *req
	reqCopy.Stream = true

	bodyBytes, err := json.Marshal(reqCopy)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", providers.ErrProviderDown, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, providers.ParseError(p.id, p.name, resp)
	}

	outChan := make(chan providers.StreamEvent, 16)
	go providers.StreamSSE(ctx, resp.Body, outChan)

	return outChan, nil
}
