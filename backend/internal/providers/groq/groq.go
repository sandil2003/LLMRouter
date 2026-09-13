package groq

import (
	"strings"

	"github.com/llmrouter/backend/internal/providers/openai"
)

type Provider struct {
	*openai.Provider
}

func New(id, apiKey, baseURL string) *Provider {
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}
	name := "Groq"
	if id == "" {
		id = "groq"
	}
	return &Provider{
		Provider: openai.New(id, name, apiKey, baseURL),
	}
}

func (p *Provider) SupportsModel(model string) bool {
	m := strings.ToLower(model)
	return strings.Contains(m, "llama") || strings.Contains(m, "mixtral") || strings.Contains(m, "gemma") || strings.Contains(m, "whisper") || strings.Contains(m, "groq")
}
