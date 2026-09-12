package retry

import (
	"context"
	"math/rand"
	"time"
)

type Config struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
}

func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 200 * time.Millisecond,
		MaxDelay:     3 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// ShouldRetryFunc returns true if an error should trigger a retry attempt.
type ShouldRetryFunc func(err error) bool

// Do runs the operation with exponential backoff if shouldRetry returns true.
func Do(ctx context.Context, cfg Config, shouldRetry ShouldRetryFunc, op func(attempt int) error) error {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = 100 * time.Millisecond
	}
	if cfg.Multiplier <= 1.0 {
		cfg.Multiplier = 2.0
	}

	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		err := op(attempt)
		if err == nil {
			return nil
		}

		if attempt == cfg.MaxAttempts || !shouldRetry(err) {
			return err
		}

		sleepDuration := delay
		if cfg.Jitter {
			// Apply full jitter: random between 0 and delay
			sleepDuration = time.Duration(rand.Float64() * float64(delay))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepDuration):
		}

		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if cfg.MaxDelay > 0 && delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return nil
}
