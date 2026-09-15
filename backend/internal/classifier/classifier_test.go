package classifier

import (
	"testing"
	"time"

	"github.com/llmrouter/backend/internal/models"
)

func TestClassifier_SpeedAndAccuracy(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name          string
		prompt        string
		wantDomain    TaskDomain
		wantJSON      bool
		minComplexity ComplexityTier
	}{
		{
			name:       "Math Calculus problem",
			prompt:     "Solve this differential equation and find the derivative: dy/dx + 2y = e^(3x). Provide step-by-step calculus proof.",
			wantDomain: DomainMath,
			wantJSON:   false,
		},
		{
			name:       "Python Code Generation",
			prompt:     "Write a Python async function to connect to a PostgreSQL database with connection pooling.\n```python\n# your code here\n```",
			wantDomain: DomainCodeGeneration,
			wantJSON:   false,
		},
		{
			name:       "JSON Schema Extraction",
			prompt:     "Extract the user profile from this text and return in JSON format with valid json_schema containing properties name and age.",
			wantDomain: DomainRoutineExtraction,
			wantJSON:   true,
		},
		{
			name:       "Creative Writing Poem",
			prompt:     "Compose a rhyming sonnet poem about artificial intelligence exploring deep space.",
			wantDomain: DomainCreativeWriting,
			wantJSON:   false,
		},
		{
			name:       "Multi-hop Reasoning",
			prompt:     "Compare and contrast the trade-offs of microservices vs monoliths. Deduce the causality and analyze counterfactual failure modes.",
			wantDomain: DomainMultiHopReasoning,
			wantJSON:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &models.ChatRequest{
				Messages: []models.ChatMessage{
					{Role: "user", Content: tc.prompt},
				},
			}

			start := time.Now()
			features := c.Classify(req)
			duration := time.Since(start)

			// Validate latency budget: must be well under 50ms (actually < 5ms)
			if duration > 50*time.Millisecond {
				t.Errorf("Classification exceeded 50ms budget: took %v", duration)
			}

			if features.Domain != tc.wantDomain {
				t.Errorf("got domain %v, want %v", features.Domain, tc.wantDomain)
			}

			if features.JSONRequired != tc.wantJSON {
				t.Errorf("got JSONRequired %v, want %v", features.JSONRequired, tc.wantJSON)
			}

			if features.InputTokens <= 0 {
				t.Errorf("expected positive input tokens, got %d", features.InputTokens)
			}
		})
	}
}
