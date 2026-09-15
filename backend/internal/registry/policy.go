package registry

import (
	"math"
	"sort"

	"github.com/llmrouter/backend/internal/classifier"
)

// PolicyWeights represents user-configured utility function weights.
type PolicyWeights struct {
	Capability float64 `json:"capability"` // w1 (0.0 - 1.0)
	Cost       float64 `json:"cost"`       // w2 (0.0 - 1.0)
	Latency    float64 `json:"latency"`    // w3 (0.0 - 1.0)
	Preset     string  `json:"preset"`     // "balanced", "quality_first", "cost_optimized", "latency_optimized"
}

// DefaultPolicyWeights returns the standard balanced preset.
func DefaultPolicyWeights() PolicyWeights {
	return PolicyWeights{
		Capability: 0.5,
		Cost:       0.3,
		Latency:    0.2,
		Preset:     "balanced",
	}
}

// PresetWeights returns standard weight profiles.
func PresetWeights(name string) PolicyWeights {
	switch name {
	case "quality_first":
		return PolicyWeights{Capability: 0.90, Cost: 0.05, Latency: 0.05, Preset: "quality_first"}
	case "cost_optimized":
		return PolicyWeights{Capability: 0.20, Cost: 0.70, Latency: 0.10, Preset: "cost_optimized"}
	case "latency_optimized":
		return PolicyWeights{Capability: 0.20, Cost: 0.10, Latency: 0.70, Preset: "latency_optimized"}
	default:
		return DefaultPolicyWeights()
	}
}

// ScoredModel represents a ranked candidate model produced by the policy engine.
type ScoredModel struct {
	Model       *ModelMetadata `json:"model"`
	Score       float64        `json:"score"`
	Capability  float64        `json:"capability"`
	CostNorm    float64        `json:"cost_norm"`
	LatencyNorm float64        `json:"latency_norm"`
}

// PolicyEngine evaluates requests against the ModelRegistry using hard constraints & utility scoring.
type PolicyEngine struct {
	registry *ModelRegistry
	weights  PolicyWeights
}

func NewPolicyEngine(registry *ModelRegistry, weights PolicyWeights) *PolicyEngine {
	if weights.Capability == 0 && weights.Cost == 0 && weights.Latency == 0 {
		weights = DefaultPolicyWeights()
	}
	return &PolicyEngine{
		registry: registry,
		weights:  weights,
	}
}

func (p *PolicyEngine) SetWeights(w PolicyWeights) {
	p.weights = w
}

func (p *PolicyEngine) GetWeights() PolicyWeights {
	return p.weights
}

// RankCandidates filters and ranks all registered models given the request features.
func (p *PolicyEngine) RankCandidates(
	features *classifier.RequestFeatures,
	isProviderHealthy func(providerID string) bool,
) []ScoredModel {
	allModels := p.registry.GetAll()
	if len(allModels) == 0 {
		return nil
	}

	var surviving []*ModelMetadata

	// 1. Hard Exclusion Filtering
	for _, m := range allModels {
		// Context window constraint
		if features.InputTokens > 0 && m.ContextWindow < features.InputTokens {
			continue
		}

		// Multimodal constraint
		if features.Multimodal && !m.HasModality("vision") {
			continue
		}

		// Tools / Function calling constraint
		if features.ToolsRequired && !m.SupportsTools {
			continue
		}

		// Provider operational health & circuit breaker check
		if isProviderHealthy != nil && !isProviderHealthy(m.ProviderID) {
			continue
		}

		surviving = append(surviving, m)
	}

	if len(surviving) == 0 {
		return nil
	}

	// Determine normalizers across surviving candidate models
	maxCost := 0.01
	maxLatency := 100.0
	for _, m := range surviving {
		avgCost := (m.CostPer1MInput + m.CostPer1MOutput) / 2.0
		if avgCost > maxCost {
			maxCost = avgCost
		}
		effLatency := m.P95LatencyMs
		if effLatency <= 0 {
			effLatency = m.RollingTTFTMs * 3.0
		}
		if effLatency > maxLatency {
			maxLatency = effLatency
		}
	}

	w1 := p.weights.Capability
	w2 := p.weights.Cost
	w3 := p.weights.Latency

	var scored []ScoredModel
	for _, m := range surviving {
		capScore := m.DomainScores[features.Domain]
		if capScore <= 0 {
			capScore = 0.70
		}

		// Normalize cost [0..1]
		avgCost := (m.CostPer1MInput + m.CostPer1MOutput) / 2.0
		costNorm := avgCost / maxCost

		// Normalize latency [0..1]
		effLatency := m.P95LatencyMs
		if effLatency <= 0 {
			effLatency = m.RollingTTFTMs * 3.0
		}
		latencyNorm := effLatency / maxLatency

		// Dynamic utility formula: Score = w1*Capability - w2*Cost - w3*Latency
		utility := (w1 * capScore) - (w2 * costNorm) - (w3 * latencyNorm)

		scored = append(scored, ScoredModel{
			Model:       m,
			Score:       math.Round(utility*1000) / 1000,
			Capability:  math.Round(capScore*100) / 100,
			CostNorm:    math.Round(costNorm*100) / 100,
			LatencyNorm: math.Round(latencyNorm*100) / 100,
		})
	}

	// Sort descending by utility Score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	return scored
}
