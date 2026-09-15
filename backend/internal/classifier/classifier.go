package classifier

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/llmrouter/backend/internal/models"
)

var (
	jsonSchemaRegex   = regexp.MustCompile(`(?i)("type"\s*:\s*"object"|"properties"\s*:|\bjson_schema\b|\bjson\s*format\b|\bvalid\s*json\b|\breturn\s*in\s*json\b)`)
	codeFenceRegex    = regexp.MustCompile("(?s)```[a-zA-Z0-9_-]*\\n?.*?```")
	toolKeywordsRegex = regexp.MustCompile(`(?i)(\bfunction_call\b|\bcall\s+tool\b|\btool_choice\b|\bexecute\s+tool\b|\bcall\s+api\b)`)

	// Domain Centroid Keyword Dictionaries with weights
	mathKeywords = map[string]float64{
		"calculate": 1.2, "equation": 1.8, "integral": 2.5, "derivative": 2.5,
		"matrix": 1.8, "polynomial": 2.0, "theorem": 2.2, "probability": 1.8,
		"algebra": 2.0, "geometry": 2.0, "trigonometry": 2.2, "arithmetic": 1.5,
		"variance": 1.8, "eigenvalue": 2.8, "vector": 1.4, "formula": 1.5,
		"logarithm": 2.0, "solve for": 1.8, "differential": 2.4, "quadrance": 2.0,
	}

	codeKeywords = map[string]float64{
		"function": 1.5, "class": 1.4, "def ": 2.0, "return ": 1.5,
		"public static": 2.5, "const ": 1.8, "import ": 1.8, "package ": 1.8,
		"algorithm": 2.0, "refactor": 2.2, "debug": 2.0, "syntax error": 2.2,
		"javascript": 2.5, "typescript": 2.5, "python": 2.5, "golang": 2.5,
		"react": 2.0, "sql": 2.2, "query": 1.6, "api endpoint": 2.0,
		"dockerfile": 2.5, "kubernetes": 2.0, "unit test": 2.2, "git": 1.6,
		"npm": 1.8, "compile": 1.8, "goroutine": 2.5, "async/await": 2.5,
	}

	reasoningKeywords = map[string]float64{
		"step-by-step": 2.2, "deduce": 2.4, "infer": 2.2, "logic puzzle": 2.8,
		"conclude": 1.8, "implication": 2.0, "fallacy": 2.5, "premise": 2.2,
		"compare and contrast": 2.2, "counterfactual": 2.8, "trade-offs": 2.0,
		"root cause": 2.2, "hypothesis": 2.0, "syllogism": 2.8, "causality": 2.4,
		"multi-hop": 2.5, "paradox": 2.4, "critique": 1.8,
	}

	creativeKeywords = map[string]float64{
		"story": 2.0, "poem": 2.5, "poetry": 2.5, "narrative": 2.2,
		"essay": 1.8, "character": 1.8, "plot": 2.0, "dialogue": 2.0,
		"metaphor": 2.4, "rhyme": 2.5, "fiction": 2.2, "creative writing": 2.8,
		"roleplay": 2.4, "fantasy": 2.0, "scifi": 2.0, "tone of voice": 2.0,
		"lyric": 2.2, "script": 1.8, "haiku": 2.8, "sonnet": 2.8,
	}

	routineKeywords = map[string]float64{
		"extract": 2.0, "summarize": 1.8, "tldr": 2.2, "bullet points": 1.8,
		"key points": 1.8, "translate": 2.2, "rephrase": 1.6, "grammar": 2.0,
		"classify": 2.0, "sentiment": 2.4, "format as": 1.8, "table": 1.6,
		"spell check": 2.0, "convert to": 1.6, "parse": 2.0, "normalize": 1.8,
	}
)

type Classifier struct{}

func NewClassifier() *Classifier {
	return &Classifier{}
}

// EstimateTokens calculates an approximate token count in sub-millisecond time.
// Uses word boundary and punctuation ratio approximation (~3.8 chars/token).
func (c *Classifier) EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	words := 0
	inWord := false
	for _, r := range text {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			words++
			inWord = true
		}
	}
	// Heuristic ratio blend: average of word count * 1.33 and char count / 3.8
	byWords := int(float64(words) * 1.33)
	byChars := len(text) / 4
	if byWords > byChars {
		return byWords
	}
	if byChars > 0 {
		return byChars
	}
	return 1
}

// Classify processes a ChatRequest and extracts RequestFeatures within under 5-10ms.
func (c *Classifier) Classify(req *models.ChatRequest) *RequestFeatures {
	start := time.Now()

	// 1. Heuristic Token Calculation & Text Aggregation
	var fullTextBuilder strings.Builder
	hasMultimodal := false

	for _, m := range req.Messages {
		fullTextBuilder.WriteString(m.Content)
		fullTextBuilder.WriteString("\n")
		// Detect multimodal markers (data:image, http://...image, base64 payload)
		if strings.Contains(m.Content, "data:image/") ||
			strings.Contains(m.Content, "image_url") ||
			strings.Contains(m.Content, ".png") ||
			strings.Contains(m.Content, ".jpg") ||
			strings.Contains(m.Content, ".jpeg") {
			hasMultimodal = true
		}
	}
	fullText := fullTextBuilder.String()
	lowerText := strings.ToLower(fullText)
	totalTokens := c.EstimateTokens(fullText)

	// 2. Structural Regex Matching
	hasJSONSchema := jsonSchemaRegex.MatchString(fullText)
	hasCodeFences := codeFenceRegex.MatchString(fullText)
	hasToolKeywords := toolKeywordsRegex.MatchString(fullText)

	// Check for explicit tools passed in metadata or request
	toolsRequired := hasToolKeywords
	if req.Metadata != nil {
		if _, ok := req.Metadata["tools"]; ok {
			toolsRequired = true
		}
	}

	// 3. Semantic Intent Domain Scoring
	domainScores := map[TaskDomain]float64{
		DomainMath:              c.scoreKeywords(lowerText, mathKeywords),
		DomainCodeGeneration:    c.scoreKeywords(lowerText, codeKeywords),
		DomainMultiHopReasoning: c.scoreKeywords(lowerText, reasoningKeywords),
		DomainCreativeWriting:   c.scoreKeywords(lowerText, creativeKeywords),
		DomainRoutineExtraction: c.scoreKeywords(lowerText, routineKeywords),
	}

	// Structural bonuses
	if hasCodeFences {
		domainScores[DomainCodeGeneration] += 3.5
	}
	if hasJSONSchema {
		domainScores[DomainRoutineExtraction] += 2.5
	}

	// Find top scoring domain
	topDomain := DomainRoutineExtraction
	maxScore := -1.0
	for d, s := range domainScores {
		if s > maxScore {
			maxScore = s
			topDomain = d
		}
	}

	// Fallback to routine extraction if no keywords strongly matched
	if maxScore <= 0.5 {
		if hasCodeFences {
			topDomain = DomainCodeGeneration
		} else {
			topDomain = DomainRoutineExtraction
		}
	}

	// 4. Complexity Tier Classification
	complexity := c.determineComplexity(topDomain, totalTokens, fullText, lowerText)

	elapsed := time.Since(start).Milliseconds()

	return &RequestFeatures{
		Domain:           topDomain,
		Complexity:       complexity,
		InputTokens:      totalTokens,
		JSONRequired:     hasJSONSchema,
		ToolsRequired:    toolsRequired,
		Multimodal:       hasMultimodal,
		LatencyBudgetMs:  100,
		ClassificationMs: elapsed,
	}
}

func (c *Classifier) scoreKeywords(text string, keywords map[string]float64) float64 {
	score := 0.0
	for kw, weight := range keywords {
		if strings.Contains(text, kw) {
			score += weight
		}
	}
	return score
}

func (c *Classifier) determineComplexity(domain TaskDomain, tokens int, text, lowerText string) ComplexityTier {
	// Signals for high complexity
	highSignals := 0

	if tokens > 1500 {
		highSignals += 2
	} else if tokens > 500 {
		highSignals++
	}

	if strings.Contains(lowerText, "step by step") ||
		strings.Contains(lowerText, "in-depth") ||
		strings.Contains(lowerText, "thoroughly analyze") ||
		strings.Contains(lowerText, "edge cases") ||
		strings.Contains(lowerText, "mathematical proof") ||
		strings.Contains(lowerText, "algorithm complexity") {
		highSignals += 2
	}

	if domain == DomainMath || domain == DomainMultiHopReasoning {
		highSignals++
	}

	if strings.Count(text, "\n") > 15 {
		highSignals++
	}

	if highSignals >= 3 {
		return ComplexityHigh
	}
	if highSignals >= 1 || tokens > 250 {
		return ComplexityMedium
	}
	return ComplexityLow
}
