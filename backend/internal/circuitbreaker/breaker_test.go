package circuitbreaker_test

import (
	"testing"
	"time"

	"github.com/llmrouter/backend/internal/circuitbreaker"
)

func TestCircuitBreakerTransitions(t *testing.T) {
	cb := circuitbreaker.NewManager(2, 50*time.Millisecond)

	// Initially closed
	if !cb.CanExecute("mock-1") {
		t.Error("expected circuit to allow execution initially")
	}

	state, fails := cb.GetStatus("mock-1")
	if state != circuitbreaker.StateClosed || fails != 0 {
		t.Errorf("unexpected status: %s, %d", state, fails)
	}

	// 1 failure - still closed
	cb.RecordFailure("mock-1")
	if !cb.CanExecute("mock-1") {
		t.Error("expected circuit to remain closed after 1 failure")
	}

	// 2nd failure - should trip to Open
	cb.RecordFailure("mock-1")
	if cb.CanExecute("mock-1") {
		t.Error("expected circuit to be Open and block execution")
	}

	state, fails = cb.GetStatus("mock-1")
	if state != circuitbreaker.StateOpen {
		t.Errorf("expected StateOpen, got %s", state)
	}

	// Wait for open timeout -> transitions to HalfOpen
	time.Sleep(60 * time.Millisecond)
	if !cb.CanExecute("mock-1") {
		t.Error("expected circuit to allow probe execution after timeout (Half-Open)")
	}

	// Success closes circuit
	cb.RecordSuccess("mock-1")
	state, fails = cb.GetStatus("mock-1")
	if state != circuitbreaker.StateClosed || fails != 0 {
		t.Errorf("expected StateClosed with 0 failures after success, got %s, %d", state, fails)
	}
}
