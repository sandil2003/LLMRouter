package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/llmrouter/backend/internal/models"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				reqID := GetRequestID(r.Context())
				slog.Error("Unhandled panic recovered",
					"request_id", reqID,
					"panic", rec,
					"stack", string(debug.Stack()),
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				code := "internal_error"
				_ = json.NewEncoder(w).Encode(models.NewAPIError(
					fmt.Sprintf("Internal gateway error: %v", rec),
					"server_error",
					&code,
				))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
