package models

import (
	"encoding/json"
	"time"
)

// ChatMessage represents a single message in a chat conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// ChatRequest represents the standard OpenAI-compatible chat completion request.
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Choice represents a single completion choice.
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason *string     `json:"finish_reason,omitempty"`
}

// Usage represents token usage in a completion response.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse represents the standard OpenAI-compatible chat completion response.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// ChunkDelta represents the incremental delta content in a streaming chunk.
type ChunkDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// ChunkChoice represents a choice inside a streaming chunk.
type ChunkChoice struct {
	Index        int        `json:"index"`
	Delta        ChunkDelta `json:"delta"`
	FinishReason *string    `json:"finish_reason,omitempty"`
}

// ChatCompletionChunk represents an SSE chunk for streaming responses.
type ChatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
}

// APIErrorDetail holds the detailed fields of an API error.
type APIErrorDetail struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param,omitempty"`
	Code    *string `json:"code,omitempty"`
}

// APIErrorResponse is the standard OpenAI-compatible error response envelope.
type APIErrorResponse struct {
	Error APIErrorDetail `json:"error"`
}

// NewAPIError constructs a standard error response.
func NewAPIError(msg, errType string, code *string) APIErrorResponse {
	return APIErrorResponse{
		Error: APIErrorDetail{
			Message: msg,
			Type:    errType,
			Code:    code,
		},
	}
}

// Helper to marshal SSE data lines.
func (c *ChatCompletionChunk) ToSSE() ([]byte, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	res := append([]byte("data: "), data...)
	res = append(res, []byte("\n\n")...)
	return res, nil
}

// Timestamp helper
func NowTimestamp() int64 {
	return time.Now().Unix()
}
