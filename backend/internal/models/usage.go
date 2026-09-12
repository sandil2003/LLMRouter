package models

import "time"

// UsageRecord stores token and latency metrics for completions.
type UsageRecord struct {
	ID           int64     `json:"id"`
	ProviderID   string    `json:"provider_id"`
	Model        string    `json:"model"`
	RequestID    string    `json:"request_id"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	LatencyMs    int64     `json:"latency_ms"`
	Status       string    `json:"status"` // "success", "error", "rate_limited"
	CreatedAt    time.Time `json:"created_at"`
}

// UsageSummary aggregates statistics for the dashboard/analytics page.
type UsageSummary struct {
	TotalRequests      int64            `json:"total_requests"`
	SuccessfulRequests int64            `json:"successful_requests"`
	FailedRequests     int64            `json:"failed_requests"`
	TotalInputTokens   int64            `json:"total_input_tokens"`
	TotalOutputTokens  int64            `json:"total_output_tokens"`
	AverageLatencyMs   float64          `json:"average_latency_ms"`
	ByProvider         map[string]int64 `json:"by_provider"`
	ByModel            map[string]int64 `json:"by_model"`
}
