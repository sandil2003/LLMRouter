package models

import "time"

// ProviderHealthState represents the health status of a provider.
type ProviderHealthState string

const (
	HealthStateHealthy     ProviderHealthState = "healthy"
	HealthStateDegraded    ProviderHealthState = "degraded"
	HealthStateRateLimited ProviderHealthState = "rate_limited"
	HealthStateDown        ProviderHealthState = "down"
)

// ProviderConfig represents a provider record in SQLite.
type ProviderConfig struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	Priority  int       `json:"priority"`
	BaseURL   string    `json:"base_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProviderWithStatus combines persistent config with live operational health metrics.
type ProviderWithStatus struct {
	ProviderConfig
	HealthState       ProviderHealthState `json:"health_state"`
	RateLimited       bool                `json:"rate_limited"`
	RateLimitResetIn  int64               `json:"rate_limit_reset_in_seconds,omitempty"`
	CircuitState      string              `json:"circuit_state"`
	ConsecutiveErrors int                 `json:"consecutive_errors"`
	ActiveModels      []string            `json:"active_models"`
	HasAPIKey         bool                `json:"has_api_key"`
}

// ModelConfig represents an LLM model offered by a provider.
type ModelConfig struct {
	ID         string    `json:"id"`
	ProviderID string    `json:"provider_id"`
	Name       string    `json:"name"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
}

// TestConnectionResult holds the result of /api/providers/:id/test.
type TestConnectionResult struct {
	Success   bool   `json:"success"`
	LatencyMs int64  `json:"latency_ms"`
	Message   string `json:"message"`
}
