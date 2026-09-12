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

type breakerInfo struct {
	state              State
	consecutiveFailures int
	openedAt           time.Time
}

// Manager maintains circuit breakers per provider.
type Manager struct {
	mu               sync.RWMutex
	breakers         map[string]*breakerInfo
	failureThreshold int
	openTimeout      time.Duration
}

func NewManager(failureThreshold int, openTimeout time.Duration) *Manager {
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	if openTimeout <= 0 {
		openTimeout = 30 * time.Second
	}

	return &Manager{
		breakers:         make(map[string]*breakerInfo),
		failureThreshold: failureThreshold,
		openTimeout:      openTimeout,
	}
}

func (m *Manager) getOrCreate(providerID string) *breakerInfo {
	b, exists := m.breakers[providerID]
	if !exists {
		b = &breakerInfo{
			state: StateClosed,
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
			// Transition to Half-Open to allow single probe request
			b.state = StateHalfOpen
			return true
		}
		return false

	case StateHalfOpen:
		// In half-open, allow trial request
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
}

// RecordFailure records a failure; trips circuit to Open if threshold is reached.
func (m *Manager) RecordFailure(providerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	b := m.getOrCreate(providerID)
	b.consecutiveFailures++

	if b.state == StateHalfOpen || b.consecutiveFailures >= m.failureThreshold {
		b.state = StateOpen
		b.openedAt = time.Now()
	}
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
}
