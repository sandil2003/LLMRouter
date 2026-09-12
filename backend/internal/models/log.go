package models

import "time"

// RequestLog represents an individual request audit trail record.
type RequestLog struct {
	ID           int64     `json:"id"`
	RequestID    string    `json:"request_id"`
	ProviderID   string    `json:"provider_id"`
	Model        string    `json:"model"`
	Status       string    `json:"status"` // "success", "error", "rate_limited", "fallback"
	LatencyMs    int64     `json:"latency_ms"`
	Error        *string   `json:"error,omitempty"`
	FallbackUsed bool      `json:"fallback_used"`
	AttemptCount int       `json:"attempt_count"`
	CreatedAt    time.Time `json:"created_at"`
}

// LogFilter contains query parameters for filtering request logs.
type LogFilter struct {
	ProviderID string
	Status     string
	Limit      int
	Offset     int
}
