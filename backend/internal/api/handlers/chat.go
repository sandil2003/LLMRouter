package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/llmrouter/backend/internal/api/middleware"
	"github.com/llmrouter/backend/internal/models"
	"github.com/llmrouter/backend/internal/providers"
	"github.com/llmrouter/backend/internal/router"
)

type ChatHandler struct {
	engine *router.Engine
}

func NewChatHandler(engine *router.Engine) *ChatHandler {
	return &ChatHandler{engine: engine}
}

func (h *ChatHandler) Completions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := middleware.GetRequestID(ctx)

	var req models.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid JSON payload", "invalid_request_error")
		return
	}

	if len(req.Messages) == 0 {
		h.writeError(w, http.StatusBadRequest, "Messages cannot be empty", "invalid_request_error")
		return
	}

	if req.Stream {
		h.handleStreaming(w, r, &req, reqID)
		return
	}

	// Non-streaming completion
	resp, err := h.engine.ExecuteChat(ctx, &req, reqID)
	if err != nil {
		h.mapEngineError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ChatHandler) handleStreaming(w http.ResponseWriter, r *http.Request, req *models.ChatRequest, reqID string) {
	ctx := r.Context()

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.writeError(w, http.StatusInternalServerError, "Streaming unsupported by server environment", "server_error")
		return
	}

	streamChan, err := h.engine.ExecuteStream(ctx, req, reqID)
	if err != nil {
		h.mapEngineError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for event := range streamChan {
		if event.Err != nil {
			errData, _ := json.Marshal(models.NewAPIError(event.Err.Error(), "stream_error", nil))
			_, _ = fmt.Fprintf(w, "data: %s\n\n", errData)
			flusher.Flush()
			return
		}

		if event.Chunk != nil {
			sseBytes, err := event.Chunk.ToSSE()
			if err == nil {
				_, _ = w.Write(sseBytes)
				flusher.Flush()
			}
		}
	}

	_, _ = w.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}

func (h *ChatHandler) writeError(w http.ResponseWriter, status int, msg, errType string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(models.NewAPIError(msg, errType, nil))
}

func (h *ChatHandler) mapEngineError(w http.ResponseWriter, err error) {
	if errors.Is(err, router.ErrNoProvidersAvailable) {
		h.writeError(w, http.StatusServiceUnavailable, err.Error(), "provider_unavailable_error")
		return
	}

	var rlErr *providers.RateLimitError
	if errors.As(err, &rlErr) {
		h.writeError(w, http.StatusTooManyRequests, rlErr.Error(), "rate_limit_error")
		return
	}

	var clientErr *providers.ClientError
	if errors.As(err, &clientErr) {
		h.writeError(w, clientErr.StatusCode, clientErr.Message, "invalid_request_error")
		return
	}

	h.writeError(w, http.StatusBadGateway, err.Error(), "gateway_error")
}
