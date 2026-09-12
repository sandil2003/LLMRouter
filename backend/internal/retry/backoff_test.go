package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/llmrouter/backend/internal/retry"
)

func TestRetryBackoff(t *testing.T) {
	cfg := retry.Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     50 * time.Millisecond,
		Multiplier:   2.0,
		Jitter:       false,
	}

	attempts := 0
	errDummy := errors.New("temporary error")

	err := retry.Do(context.Background(), cfg, func(err error) bool {
		return errors.Is(err, errDummy)
	}, func(attempt int) error {
		attempts = attempt
		if attempt < 3 {
			return errDummy
		}
		return nil
	})

	if err != nil {
		t.Fatalf("expected success on attempt 3, got error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}
