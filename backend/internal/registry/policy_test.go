package registry

import (
	"testing"

	"github.com/llmrouter/backend/internal/classifier"
)

func TestPolicyEngine_RankingAndHardExclusions(t *testing.T) {
	reg := NewModelRegistry()
	engine := NewPolicyEngine(reg, DefaultPolicyWeights())

	// 1. Hard Exclusion Test: Multimodal requirement must exclude text-only models (e.g. Llama 3.3)
	visionFeatures := &classifier.RequestFeatures{
		Domain:        classifier.DomainCreativeWriting,
		Complexity:    classifier.ComplexityLow,
		InputTokens:   500,
		Multimodal:    true,
		ToolsRequired: false,
	}

	rankedVision := engine.RankCandidates(visionFeatures, nil)
	if len(rankedVision) == 0 {
		t.Fatal("expected surviving multimodal candidates, got 0")
	}
	for _, cand := range rankedVision {
		if !cand.Model.HasModality("vision") {
			t.Errorf("model %s does not support vision but was not excluded", cand.Model.ID)
		}
	}

	// 2. Cost-Optimized Policy: Cheaper models must rank higher than expensive models
	costEngine := NewPolicyEngine(reg, PresetWeights("cost_optimized"))

	codeFeatures := &classifier.RequestFeatures{
		Domain:        classifier.DomainCodeGeneration,
		Complexity:    classifier.ComplexityMedium,
		InputTokens:   1200,
		Multimodal:    false,
		ToolsRequired: false,
	}

	costRanked := costEngine.RankCandidates(codeFeatures, nil)
	if len(costRanked) == 0 {
		t.Fatal("expected candidates for code generation")
	}

	primary := costRanked[0].Model
	if primary.ID == "gpt-4o" || primary.ID == "claude-3.5-sonnet" {
		t.Errorf("expected cost-effective primary under cost-optimized policy, got %s", primary.ID)
	}

	// 3. Quality-First Policy: Top capability models (gemini-2.5-pro, claude-3.5-sonnet, gpt-4o) should rank at top for complex math
	qualityEngine := NewPolicyEngine(reg, PresetWeights("quality_first"))

	mathFeatures := &classifier.RequestFeatures{
		Domain:        classifier.DomainMath,
		Complexity:    classifier.ComplexityHigh,
		InputTokens:   2500,
		Multimodal:    false,
		ToolsRequired: false,
	}

	qualityRanked := qualityEngine.RankCandidates(mathFeatures, nil)
	topModel := qualityRanked[0].Model
	if topModel.DomainScores[classifier.DomainMath] < 0.90 {
		t.Errorf("expected top tier math model (>=0.90), got %s with score %.2f", topModel.ID, topModel.DomainScores[classifier.DomainMath])
	}
}
