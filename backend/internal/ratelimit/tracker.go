package ratelimit

import (
	"sync"
	"time"
)

type ProviderRateLimit struct {
	BlockedUntil time.Time
	Reason       string
}

// Tracker maintains in-memory rate-limit cooldown windows for providers.
type Tracker struct {
	mu              sync.RWMutex
	blocked         map[string]ProviderRateLimit
	defaultCooldown time.Duration
}

func NewTracker(defaultCooldown time.Duration) *Tracker {
	if defaultCooldown <= 0 {
		defaultCooldown = 60 * time.Second
	}
	return &Tracker{
		blocked:         make(map[string]ProviderRateLimit),
		defaultCooldown: defaultCooldown,
	}
}

// MarkRateLimited sets a cooldown period for a provider.
func (t *Tracker) MarkRateLimited(providerID string, retryAfter time.Duration, reason string) time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()

	cooldown := retryAfter
	if cooldown <= 0 {
		cooldown = t.defaultCooldown
	}

	until := time.Now().Add(cooldown)
	t.blocked[providerID] = ProviderRateLimit{
		BlockedUntil: until,
		Reason:       reason,
	}
	return until
}

// IsRateLimited checks if a provider is currently blocked.
func (t *Tracker) IsRateLimited(providerID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	info, exists := t.blocked[providerID]
	if !exists {
		return false
	}

	return time.Now().Before(info.BlockedUntil)
}

// RemainingDuration returns remaining cooldown duration, or 0 if not blocked.
func (t *Tracker) RemainingDuration(providerID string) time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()

	info, exists := t.blocked[providerID]
	if !exists {
		return 0
	}

	remaining := time.Until(info.BlockedUntil)
	if remaining <= 0 {
		return 0
	}
	return remaining
}

// Clear explicitly removes a rate limit block (e.g. after successful test).
func (t *Tracker) Clear(providerID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.blocked, providerID)
}
