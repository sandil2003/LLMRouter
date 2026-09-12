package ratelimit_test

import (
	"testing"
	"time"

	"github.com/llmrouter/backend/internal/ratelimit"
)

func TestTracker(t *testing.T) {
	tracker := ratelimit.NewTracker(50 * time.Millisecond)

	if tracker.IsRateLimited("gemini") {
		t.Error("expected gemini to not be rate limited initially")
	}

	tracker.MarkRateLimited("gemini", 50*time.Millisecond, "429 too many requests")

	if !tracker.IsRateLimited("gemini") {
		t.Error("expected gemini to be rate limited")
	}

	if tracker.RemainingDuration("gemini") <= 0 {
		t.Error("expected positive remaining duration")
	}

	time.Sleep(60 * time.Millisecond)

	if tracker.IsRateLimited("gemini") {
		t.Error("expected gemini rate limit to expire after sleep")
	}

	tracker.MarkRateLimited("groq", 1*time.Hour, "quota exceeded")
	if !tracker.IsRateLimited("groq") {
		t.Error("expected groq to be rate limited")
	}
	tracker.Clear("groq")
	if tracker.IsRateLimited("groq") {
		t.Error("expected groq to be unblocked after clear")
	}
}
