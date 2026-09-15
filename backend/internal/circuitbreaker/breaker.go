package circuitbreaker

import (
	"sync"
	"time"
)

type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half_open"
)

type windowEvent struct {
	timestamp  time.Time
	isFailure  bool
	statusCode int
}

type breakerInfo struct {
	state               State
	consecutiveFailures int
	window              []windowEvent
	openedAt            time.Time
}

// Manager maintains sliding-window circuit breakers per provider.
type Manager struct {
	mu               sync.RWMutex
	breakers         map[string]*breakerInfo
	failureThreshold int
	windowDuration   time.Duration
	openTimeout      time.Duration
}

func NewManager(failureThreshold int, openTimeout time.Duration) *Manager {
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	if openTimeout <= 0 {
		openTimeout = 60 * time.Second // 60s cooldown as specified
	}

	return &Manager{
		breakers:         make(map[string]*breakerInfo),
		failureThreshold: failureThreshold,
		windowDuration:   60 * time.Second,
		openTimeout:      openTimeout,
	}
}

func (m *Manager) getOrCreate(providerID string) *breakerInfo {
	b, exists := m.breakers[providerID]
	if !exists {
		b = &breakerInfo{
			state:  StateClosed,
			window: make([]windowEvent, 0, 32),
		}
		m.breakers[providerID] = b
	}
	return b
}

// CanExecute returns true if the provider circuit breaker allows a request.
func (m *Manager) CanExecute(providerID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	b := m.getOrCreate(providerID)
	now := time.Now()

	switch b.state {
	case StateClosed:
		return true

	case StateOpen:
		if now.Sub(b.openedAt) >= m.openTimeout {
			// Cooldown expired; transition to Half-Open to allow trial probe
			b.state = StateHalfOpen
			return true
		}
		return false

	case StateHalfOpen:
		// In half-open, allow single trial probe
		return true

	default:
		return true
	}
}

// RecordSuccess transitions Half-Open or Closed to healthy Closed state.
func (m *Manager) RecordSuccess(providerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	b := m.getOrCreate(providerID)
	b.consecutiveFailures = 0
	b.state = StateClosed
	m.pruneAndAppend(b, windowEvent{timestamp: time.Now(), isFailure: false})
}

// RecordFailure records a generic failure.
func (m *Manager) RecordFailure(providerID string) {
	m.RecordFailureWithCode(providerID, 500)
}

// RecordFailureWithCode records a failure with HTTP status code (e.g. 429, 503).
func (m *Manager) RecordFailureWithCode(providerID string, statusCode int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	b := m.getOrCreate(providerID)
	b.consecutiveFailures++
	now := time.Now()

	m.pruneAndAppend(b, windowEvent{
		timestamp:  now,
		isFailure:  true,
		statusCode: statusCode,
	})

	// Count sliding-window rate limit / overload errors in last 60s
	rateLimitOrOverloadFailures := 0
	for _, ev := range b.window {
		if ev.isFailure && (ev.statusCode == 429 || ev.statusCode == 503 || ev.statusCode == 504 || ev.statusCode == 500) {
			rateLimitOrOverloadFailures++
		}
	}

	// Trip circuit breaker if consecutive failures exceed threshold or sliding window threshold hit
	if b.state == StateHalfOpen ||
		b.consecutiveFailures >= m.failureThreshold ||
		rateLimitOrOverloadFailures >= m.failureThreshold {
		b.state = StateOpen
		b.openedAt = now
	}
}

func (m *Manager) pruneAndAppend(b *breakerInfo, ev windowEvent) {
	cutoff := ev.timestamp.Add(-m.windowDuration)
	filtered := b.window[:0]
	for _, e := range b.window {
		if e.timestamp.After(cutoff) {
			filtered = append(filtered, e)
		}
	}
	b.window = append(filtered, ev)
}

// GetStatus returns the current state and consecutive error count.
func (m *Manager) GetStatus(providerID string) (State, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	b, exists := m.breakers[providerID]
	if !exists {
		return StateClosed, 0
	}
	return b.state, b.consecutiveFailures
}

// Reset explicitly clears failures and closes the circuit.
func (m *Manager) Reset(providerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	b := m.getOrCreate(providerID)
	b.consecutiveFailures = 0
	b.state = StateClosed
	b.window = b.window[:0]
}
