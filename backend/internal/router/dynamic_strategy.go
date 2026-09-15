package router

import (
	"context"
	"sort"
	"strings"

	"github.com/llmrouter/backend/internal/classifier"
	"github.com/llmrouter/backend/internal/models"
	"github.com/llmrouter/backend/internal/registry"
)

const StrategyDynamicPolicy models.RoutingStrategy = "dynamic_policy"

// DynamicPolicyStrategy orders candidate providers based on the PolicyEngine utility score.
type DynamicPolicyStrategy struct {
	policyEngine *registry.PolicyEngine
	classifier   *classifier.Classifier
}

func NewDynamicPolicyStrategy(engine *registry.PolicyEngine, cls *classifier.Classifier) *DynamicPolicyStrategy {
	return &DynamicPolicyStrategy{
		policyEngine: engine,
		classifier:   cls,
	}
}

func (s *DynamicPolicyStrategy) Name() models.RoutingStrategy {
	return StrategyDynamicPolicy
}

func (s *DynamicPolicyStrategy) OrderCandidates(ctx context.Context, candidates []Candidate) []Candidate {
	if len(candidates) <= 1 {
		return candidates
	}

	// Fetch request features from context if available, or default features
	features, ok := ctx.Value("request_features").(*classifier.RequestFeatures)
	if !ok || features == nil {
		features = &classifier.RequestFeatures{
			Domain:      classifier.DomainRoutineExtraction,
			Complexity:  classifier.ComplexityMedium,
			InputTokens: 500,
		}
	}

	// Rank all models with policy engine
	ranked := s.policyEngine.RankCandidates(features, nil)
	if len(ranked) == 0 {
		return candidates
	}

	// Map candidate providers to their top-scoring model's score
	providerScores := make(map[string]float64)
	for _, cand := range candidates {
		pID := strings.ToLower(cand.Config.ID)
		bestScore := -999.0
		for _, sm := range ranked {
			if strings.EqualFold(sm.Model.ProviderID, pID) {
				if sm.Score > bestScore {
					bestScore = sm.Score
				}
			}
		}
		providerScores[pID] = bestScore
	}

	ordered := append([]Candidate{}, candidates...)
	sort.SliceStable(ordered, func(i, j int) bool {
		sI := providerScores[strings.ToLower(ordered[i].Config.ID)]
		sJ := providerScores[strings.ToLower(ordered[j].Config.ID)]
		return sI > sJ
	})

	return ordered
}
