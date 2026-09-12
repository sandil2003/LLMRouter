package models

import "time"

type RoutingStrategy string

const (
	StrategyPriority   RoutingStrategy = "priority"
	StrategyRoundRobin RoutingStrategy = "round_robin"
)

// RoutingRule represents configured routing behavior stored in DB.
type RoutingRule struct {
	ID        string          `json:"id"`
	Strategy  RoutingStrategy `json:"strategy"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ProviderPriorityUpdate represents batch priority updates from UI.
type ProviderPriorityUpdate struct {
	ProviderID string `json:"provider_id"`
	Priority   int    `json:"priority"`
}

// RoutingUpdateRequest defines the payload for updating routing configuration.
type RoutingUpdateRequest struct {
	Strategy   *RoutingStrategy         `json:"strategy,omitempty"`
	Priorities []ProviderPriorityUpdate `json:"priorities,omitempty"`
}
