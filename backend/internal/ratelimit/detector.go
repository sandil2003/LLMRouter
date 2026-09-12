package ratelimit

import (
	"errors"
	"strings"
	"time"

	"github.com/llmrouter/backend/internal/providers"
)

// IsRateLimit checks whether an error is a rate limit or quota exceeded event.
func IsRateLimit(err error) (bool, time.Duration) {
	if err == nil {
		return false, 0
	}

	var rlErr *providers.RateLimitError
	if errors.As(err, &rlErr) {
		return true, rlErr.RetryAfter
	}

	if errors.Is(err, providers.ErrQuotaExceeded) {
		return true, 0
	}

	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "quota exceeded") ||
		strings.Contains(errStr, "too many requests") {
		return true, 0
	}

	return false, 0
}
