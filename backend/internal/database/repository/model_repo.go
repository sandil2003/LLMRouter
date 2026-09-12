package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/llmrouter/backend/internal/database"
	"github.com/llmrouter/backend/internal/models"
)

type ModelRepository struct {
	db *database.DB
}

func NewModelRepository(db *database.DB) *ModelRepository {
	return &ModelRepository{db: db}
}

func (r *ModelRepository) ListByProvider(ctx context.Context, providerID string) ([]models.ModelConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, provider_id, name, enabled, created_at
		FROM models
		WHERE provider_id = ?
		ORDER BY name ASC
	`, providerID)
	if err != nil {
		return nil, fmt.Errorf("query models by provider: %w", err)
	}
	defer rows.Close()

	list := make([]models.ModelConfig, 0)
	for rows.Next() {
		var m models.ModelConfig
		if err := rows.Scan(&m.ID, &m.ProviderID, &m.Name, &m.Enabled, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan model: %w", err)
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (r *ModelRepository) ListAllActive(ctx context.Context) ([]models.ModelConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.provider_id, m.name, m.enabled, m.created_at
		FROM models m
		JOIN providers p ON m.provider_id = p.id
		WHERE m.enabled = 1 AND p.enabled = 1
		ORDER BY m.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query all active models: %w", err)
	}
	defer rows.Close()

	list := make([]models.ModelConfig, 0)
	for rows.Next() {
		var m models.ModelConfig
		if err := rows.Scan(&m.ID, &m.ProviderID, &m.Name, &m.Enabled, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan model: %w", err)
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (r *ModelRepository) Upsert(ctx context.Context, m *models.ModelConfig) error {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO models (id, provider_id, name, enabled, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			enabled = excluded.enabled
	`, m.ID, m.ProviderID, m.Name, m.Enabled, m.CreatedAt)
	if err != nil {
		return fmt.Errorf("upsert model: %w", err)
	}
	return nil
}

func (r *ModelRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM models WHERE id = ?`, id)
	return err
}
