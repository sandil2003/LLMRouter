package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// RedactingHandler wraps a slog.Handler and redacts known sensitive keys/values.
type RedactingHandler struct {
	slog.Handler
}

var sensitiveKeys = map[string]bool{
	"authorization": true,
	"api_key":       true,
	"apikey":        true,
	"key":           true,
	"password":      true,
	"secret":        true,
	"token":         true,
}

func (h *RedactingHandler) Handle(ctx context.Context, r slog.Record) error {
	var newAttrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		newAttrs = append(newAttrs, sanitizeAttr(a))
		return true
	})

	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	newRecord.AddAttrs(newAttrs...)
	return h.Handler.Handle(ctx, newRecord)
}

func sanitizeAttr(a slog.Attr) slog.Attr {
	keyLower := strings.ToLower(a.Key)
	if sensitiveKeys[keyLower] {
		return slog.String(a.Key, "[REDACTED]")
	}

	if a.Value.Kind() == slog.KindString {
		val := a.Value.String()
		if strings.HasPrefix(val, "Bearer ") {
			return slog.String(a.Key, "Bearer [REDACTED]")
		}
	}

	return a
}

// Setup initializes the global structured slog logger.
func Setup(levelStr string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		lvl = slog.LevelDebug
	case "WARN":
		lvl = slog.LevelWarn
	case "ERROR":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	baseHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})

	redactingHandler := &RedactingHandler{Handler: baseHandler}
	logger := slog.New(redactingHandler)
	slog.SetDefault(logger)
	return logger
}
