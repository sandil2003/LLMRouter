package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

func New(id, apiKey, baseURL string) *Provider {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	name := "Gemini"
	if id == "" {
		id = "gemini"
	}

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

func (p *Provider) SupportsModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "gemini")
}

func (p *Provider) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/models?key=%s", p.baseURL, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
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

// Internal Gemini Request/Response structures
type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
}

type geminiRequest struct {
	Contents          []geminiContent         `json:"contents"`
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiCandidate struct {
	Content struct {
		Parts []geminiPart `json:"parts"`
		Role  string       `json:"role"`
	} `json:"content"`
	FinishReason string `json:"finishReason"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type geminiResponse struct {
	Candidates    []geminiCandidate    `json:"candidates"`
	UsageMetadata *geminiUsageMetadata `json:"usageMetadata,omitempty"`
}

func (p *Provider) toGeminiRequest(req *models.ChatRequest) *geminiRequest {
	gReq := &geminiRequest{
		Contents: make([]geminiContent, 0, len(req.Messages)),
	}

	for _, m := range req.Messages {
		role := m.Role
		if role == "system" {
			gReq.SystemInstruction = &geminiContent{
				Role:  "system",
				Parts: []geminiPart{{Text: m.Content}},
			}
			continue
		}
		if role == "assistant" {
			role = "model"
		} else {
			role = "user"
		}
		gReq.Contents = append(gReq.Contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	if req.Temperature != nil || req.TopP != nil || req.MaxTokens != nil {
		gReq.GenerationConfig = &geminiGenerationConfig{
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			MaxOutputTokens: req.MaxTokens,
		}
	}

	return gReq
}

func (p *Provider) Chat(ctx context.Context, req *models.ChatRequest) (*models.ChatResponse, error) {
	model := req.Model
	if !strings.HasPrefix(model, "models/") && !strings.Contains(model, "gemini") {
		model = "gemini-1.5-flash"
	}

	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, model, p.apiKey)
	gReq := p.toGeminiRequest(req)
	bodyBytes, err := json.Marshal(gReq)
	if err != nil {
		return nil, fmt.Errorf("marshal gemini request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", providers.ErrProviderDown, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, providers.ParseError(p.id, p.name, resp)
	}
	defer resp.Body.Close()

	var gResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&gResp); err != nil {
		return nil, fmt.Errorf("decode gemini response: %w", err)
	}

	var contentBuilder strings.Builder
	finishReason := "stop"
	if len(gResp.Candidates) > 0 {
		cand := gResp.Candidates[0]
		for _, part := range cand.Content.Parts {
			contentBuilder.WriteString(part.Text)
		}
		if cand.FinishReason != "" {
			finishReason = strings.ToLower(cand.FinishReason)
		}
	}

	var usage *models.Usage
	if gResp.UsageMetadata != nil {
		usage = &models.Usage{
			PromptTokens:     gResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: gResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      gResp.UsageMetadata.TotalTokenCount,
		}
	}

	return &models.ChatResponse{
		ID:      fmt.Sprintf("gemini-%d", time.Now().UnixNano()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []models.Choice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    "assistant",
					Content: contentBuilder.String(),
				},
				FinishReason: &finishReason,
			},
		},
		Usage: usage,
	}, nil
}

func (p *Provider) ChatStream(ctx context.Context, req *models.ChatRequest) (<-chan providers.StreamEvent, error) {
	model := req.Model
	if !strings.HasPrefix(model, "models/") && !strings.Contains(model, "gemini") {
		model = "gemini-1.5-flash"
	}

	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", p.baseURL, model, p.apiKey)
	gReq := p.toGeminiRequest(req)
	bodyBytes, err := json.Marshal(gReq)
	if err != nil {
		return nil, fmt.Errorf("marshal gemini request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create stream request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", providers.ErrProviderDown, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, providers.ParseError(p.id, p.name, resp)
	}

	outChan := make(chan providers.StreamEvent, 16)
	go func() {
		defer close(outChan)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			select {
			case <-ctx.Done():
				outChan <- providers.StreamEvent{Err: ctx.Err()}
				return
			default:
			}

			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					outChan <- providers.StreamEvent{Err: err}
				}
				return
			}

			line = bytes.TrimSpace(line)
			if !bytes.HasPrefix(line, []byte("data: ")) {
				continue
			}

			data := bytes.TrimPrefix(line, []byte("data: "))
			var gResp geminiResponse
			if err := json.Unmarshal(data, &gResp); err != nil {
				continue
			}

			if len(gResp.Candidates) > 0 {
				var text string
				for _, part := range gResp.Candidates[0].Content.Parts {
					text += part.Text
				}

				if text != "" {
					chunk := &models.ChatCompletionChunk{
						ID:      fmt.Sprintf("gemini-chunk-%d", time.Now().UnixNano()),
						Object:  "chat.completion.chunk",
						Created: time.Now().Unix(),
						Model:   req.Model,
						Choices: []models.ChunkChoice{
							{
								Index: 0,
								Delta: models.ChunkDelta{
									Content: text,
								},
							},
						},
					}
					outChan <- providers.StreamEvent{Chunk: chunk}
				}
			}
		}
	}()

	return outChan, nil
}
