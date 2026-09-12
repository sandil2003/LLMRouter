package openrouter

import (
	"github.com/llmrouter/backend/internal/providers/openai"
)

type Provider struct {
	*openai.Provider
}

func New(id, apiKey, baseURL string) *Provider {
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	name := "OpenRouter"
	if id == "" {
		id = "openrouter"
	}
	return &Provider{
		Provider: openai.New(id, name, apiKey, baseURL),
	}
}
