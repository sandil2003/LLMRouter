package router

import (
	"context"
	"sort"

	"github.com/llmrouter/backend/internal/models"
)

// PriorityStrategy orders providers by ascending priority (1 first, then 2, etc.).
type PriorityStrategy struct{}

func NewPriorityStrategy() *PriorityStrategy {
	return &PriorityStrategy{}
}

func (s *PriorityStrategy) Name() models.RoutingStrategy {
	return models.StrategyPriority
}

func (s *PriorityStrategy) OrderCandidates(ctx context.Context, candidates []Candidate) []Candidate {
	ordered := make([]Candidate, len(candidates))
	copy(ordered, candidates)

	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Config.Priority != ordered[j].Config.Priority {
			return ordered[i].Config.Priority < ordered[j].Config.Priority
		}
		return ordered[i].Config.Name < ordered[j].Config.Name
	})

	return ordered
}
