package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/llmrouter/backend/internal/models"
)

// BaseHTTPClient handles common request execution, headers, and SSE streaming.
type BaseHTTPClient struct {
	Client *http.Client
}

func NewBaseHTTPClient(timeout time.Duration) *BaseHTTPClient {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &BaseHTTPClient{
		Client: &http.Client{Timeout: timeout},
	}
}

// ParseError inspects an HTTP response and creates an appropriate typed error.
func ParseError(providerID, providerName string, resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	bodyStr := strings.TrimSpace(string(body))

	if resp.StatusCode == http.StatusTooManyRequests || strings.Contains(strings.ToLower(bodyStr), "quota exceeded") || strings.Contains(strings.ToLower(bodyStr), "rate limit") {
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return &RateLimitError{
			ProviderID:   providerID,
			ProviderName: providerName,
			RetryAfter:   retryAfter,
			Message:      fmt.Sprintf("%s (status %d)", bodyStr, resp.StatusCode),
		}
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("%w: %s", ErrUnauthorized, bodyStr)
	}

	if resp.StatusCode >= 500 {
		return &TransientError{
			StatusCode: resp.StatusCode,
			Message:    bodyStr,
		}
	}

	if resp.StatusCode >= 400 {
		return &ClientError{
			StatusCode: resp.StatusCode,
			Message:    bodyStr,
		}
	}

	return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, bodyStr)
}

func parseRetryAfter(headerVal string) time.Duration {
	if headerVal == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(headerVal); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(headerVal); err == nil {
		diff := time.Until(t)
		if diff > 0 {
			return diff
		}
	}
	return 0
}

// StreamSSE reads lines from an SSE response body and emits events on outChan.
func StreamSSE(ctx context.Context, body io.ReadCloser, outChan chan<- StreamEvent) {
	defer close(outChan)
	defer body.Close()

	reader := bufio.NewReader(body)
	for {
		select {
		case <-ctx.Done():
			outChan <- StreamEvent{Err: ctx.Err()}
			return
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				outChan <- StreamEvent{Err: err}
			}
			return
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		if bytes.HasPrefix(line, []byte("data: ")) {
			payload := bytes.TrimPrefix(line, []byte("data: "))
			if string(payload) == "[DONE]" {
				return
			}

			var chunk models.ChatCompletionChunk
			if err := json.Unmarshal(payload, &chunk); err != nil {
				continue // Skip non-JSON or commentary SSE lines
			}
			outChan <- StreamEvent{Chunk: &chunk}
		}
	}
}
