package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/llmrouter/backend/internal/database"
	"github.com/llmrouter/backend/internal/models"
)

type RoutingRepository struct {
	db *database.DB
}

func NewRoutingRepository(db *database.DB) *RoutingRepository {
	return &RoutingRepository{db: db}
}

func (r *RoutingRepository) GetActiveRule(ctx context.Context) (*models.RoutingRule, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, strategy, enabled, created_at, updated_at
		FROM routing_rules
		WHERE enabled = 1
		LIMIT 1
	`)

	var rule models.RoutingRule
	err := row.Scan(&rule.ID, &rule.Strategy, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return default
			return &models.RoutingRule{
				ID:        "default",
				Strategy:  models.StrategyPriority,
				Enabled:   true,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}, nil
		}
		return nil, fmt.Errorf("get active routing rule: %w", err)
	}
	return &rule, nil
}

func (r *RoutingRepository) UpdateStrategy(ctx context.Context, strategy models.RoutingStrategy) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE routing_rules
		SET strategy = ?, updated_at = ?
		WHERE id = 'default'
	`, strategy, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("update routing strategy: %w", err)
	}
	return nil
}

func (r *RoutingRepository) BatchUpdatePriorities(ctx context.Context, priorities []models.ProviderPriorityUpdate) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	stmt, err := tx.PrepareContext(ctx, `UPDATE providers SET priority = ?, updated_at = ? WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("prepare update priority stmt: %w", err)
	}
	defer stmt.Close()

	for _, p := range priorities {
		if _, err := stmt.ExecContext(ctx, p.Priority, now, p.ProviderID); err != nil {
			return fmt.Errorf("exec update priority for provider %s: %w", p.ProviderID, err)
		}
	}

	return tx.Commit()
}
