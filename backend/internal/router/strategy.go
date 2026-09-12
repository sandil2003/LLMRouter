package router

import (
	"context"

	"github.com/llmrouter/backend/internal/models"
	"github.com/llmrouter/backend/internal/providers"
)

// Candidate represents a provider eligible for routing.
type Candidate struct {
	Provider providers.Provider
	Config   models.ProviderConfig
}

// Strategy determines how candidate providers are ordered/selected.
type Strategy interface {
	Name() models.RoutingStrategy
	OrderCandidates(ctx context.Context, candidates []Candidate) []Candidate
}
