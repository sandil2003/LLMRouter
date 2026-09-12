package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/llmrouter/backend/internal/database"
	"github.com/llmrouter/backend/internal/models"
)

type LogRepository struct {
	db *database.DB
}

func NewLogRepository(db *database.DB) *LogRepository {
	return &LogRepository{db: db}
}

func (r *LogRepository) RecordLog(ctx context.Context, l *models.RequestLog) error {
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now().UTC()
	}
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO request_logs (request_id, provider_id, model, status, latency_ms, error, fallback_used, attempt_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, l.RequestID, l.ProviderID, l.Model, l.Status, l.LatencyMs, l.Error, l.FallbackUsed, l.AttemptCount, l.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert request log: %w", err)
	}
	id, err := res.LastInsertId()
	if err == nil {
		l.ID = id
	}
	return nil
}

func (r *LogRepository) GetLogs(ctx context.Context, filter models.LogFilter) ([]models.RequestLog, error) {
	query := `
		SELECT id, request_id, provider_id, model, status, latency_ms, error, fallback_used, attempt_count, created_at
		FROM request_logs
	`
	var whereClauses []string
	var args []any

	if filter.ProviderID != "" {
		whereClauses = append(whereClauses, "provider_id = ?")
		args = append(args, filter.ProviderID)
	}
	if filter.Status != "" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, filter.Status)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY id DESC"

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query request logs: %w", err)
	}
	defer rows.Close()

	logs := make([]models.RequestLog, 0)
	for rows.Next() {
		var l models.RequestLog
		if err := rows.Scan(&l.ID, &l.RequestID, &l.ProviderID, &l.Model, &l.Status, &l.LatencyMs, &l.Error, &l.FallbackUsed, &l.AttemptCount, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan request log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}
