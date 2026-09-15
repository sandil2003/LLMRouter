package classifier

// TaskDomain represents the high-level domain classification of a prompt.
type TaskDomain string

const (
	DomainMath              TaskDomain = "math"
	DomainMultiHopReasoning TaskDomain = "multi_hop_reasoning"
	DomainCreativeWriting   TaskDomain = "creative_writing"
	DomainCodeGeneration    TaskDomain = "code_generation"
	DomainRoutineExtraction TaskDomain = "routine_extraction"
)

// ComplexityTier represents the reasoning complexity demand of a prompt.
type ComplexityTier string

const (
	ComplexityLow    ComplexityTier = "low"    // Simple factoid Q&A, short conversion
	ComplexityMedium ComplexityTier = "medium" // Structured synthesis, multi-constraint formatting
	ComplexityHigh   ComplexityTier = "high"   // Deep mathematical proof, algorithmic design, long multi-step reasoning
)

// RequestFeatures represents the extracted metadata and classification output payload.
type RequestFeatures struct {
	Domain           TaskDomain     `json:"domain"`
	Complexity       ComplexityTier `json:"complexity"`
	InputTokens      int            `json:"input_tokens"`
	JSONRequired     bool           `json:"json_required"`
	ToolsRequired    bool           `json:"tools_required"`
	Multimodal       bool           `json:"multimodal"`
	LatencyBudgetMs  int            `json:"latency_budget_ms"`
	ClassificationMs int64          `json:"classification_ms"`
}
