package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/llmrouter/backend/internal/database"
	"github.com/llmrouter/backend/internal/models"
)

type UsageRepository struct {
	db *database.DB
}

func NewUsageRepository(db *database.DB) *UsageRepository {
	return &UsageRepository{db: db}
}

func (r *UsageRepository) RecordUsage(ctx context.Context, u *models.UsageRecord) error {
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now().UTC()
	}
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO usage (provider_id, model, request_id, input_tokens, output_tokens, latency_ms, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, u.ProviderID, u.Model, u.RequestID, u.InputTokens, u.OutputTokens, u.LatencyMs, u.Status, u.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert usage: %w", err)
	}
	id, err := res.LastInsertId()
	if err == nil {
		u.ID = id
	}
	return nil
}

func (r *UsageRepository) GetSummary(ctx context.Context) (*models.UsageSummary, error) {
	summary := &models.UsageSummary{
		ByProvider: make(map[string]int64),
		ByModel:    make(map[string]int64),
	}

	row := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status != 'success' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(AVG(latency_ms), 0.0)
		FROM usage
	`)

	var avgLat sql.NullFloat64
	err := row.Scan(
		&summary.TotalRequests,
		&summary.SuccessfulRequests,
		&summary.FailedRequests,
		&summary.TotalInputTokens,
		&summary.TotalOutputTokens,
		&avgLat,
	)
	if err != nil {
		return nil, fmt.Errorf("get usage summary aggregates: %w", err)
	}
	if avgLat.Valid {
		summary.AverageLatencyMs = avgLat.Float64
	}

	// By provider
	pRows, err := r.db.QueryContext(ctx, `SELECT provider_id, COUNT(*) FROM usage GROUP BY provider_id`)
	if err == nil {
		defer pRows.Close()
		for pRows.Next() {
			var pid string
			var count int64
			if err := pRows.Scan(&pid, &count); err == nil {
				summary.ByProvider[pid] = count
			}
		}
	}

	// By model
	mRows, err := r.db.QueryContext(ctx, `SELECT model, COUNT(*) FROM usage GROUP BY model`)
	if err == nil {
		defer mRows.Close()
		for mRows.Next() {
			var mName string
			var count int64
			if err := mRows.Scan(&mName, &count); err == nil {
				summary.ByModel[mName] = count
			}
		}
	}

	return summary, nil
}
