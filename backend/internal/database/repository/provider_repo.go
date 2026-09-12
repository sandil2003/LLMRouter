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

var ErrNotFound = errors.New("record not found")

type ProviderRepository struct {
	db *database.DB
}

func NewProviderRepository(db *database.DB) *ProviderRepository {
	return &ProviderRepository{db: db}
}

func (r *ProviderRepository) GetAll(ctx context.Context) ([]models.ProviderConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, enabled, priority, base_url, created_at, updated_at
		FROM providers
		ORDER BY priority ASC, name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query providers: %w", err)
	}
	defer rows.Close()

	var list []models.ProviderConfig
	for rows.Next() {
		var p models.ProviderConfig
		if err := rows.Scan(&p.ID, &p.Name, &p.Enabled, &p.Priority, &p.BaseURL, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *ProviderRepository) GetActiveOrderedByPriority(ctx context.Context) ([]models.ProviderConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, enabled, priority, base_url, created_at, updated_at
		FROM providers
		WHERE enabled = 1
		ORDER BY priority ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query active providers: %w", err)
	}
	defer rows.Close()

	var list []models.ProviderConfig
	for rows.Next() {
		var p models.ProviderConfig
		if err := rows.Scan(&p.ID, &p.Name, &p.Enabled, &p.Priority, &p.BaseURL, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan active provider: %w", err)
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *ProviderRepository) GetByID(ctx context.Context, id string) (*models.ProviderConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, enabled, priority, base_url, created_at, updated_at
		FROM providers
		WHERE id = ?
	`, id)

	var p models.ProviderConfig
	err := row.Scan(&p.ID, &p.Name, &p.Enabled, &p.Priority, &p.BaseURL, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get provider by id: %w", err)
	}
	return &p, nil
}

func (r *ProviderRepository) Create(ctx context.Context, p *models.ProviderConfig) error {
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO providers (id, name, enabled, priority, base_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Name, p.Enabled, p.Priority, p.BaseURL, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert provider: %w", err)
	}
	return nil
}

func (r *ProviderRepository) Update(ctx context.Context, p *models.ProviderConfig) error {
	p.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		UPDATE providers
		SET name = ?, enabled = ?, priority = ?, base_url = ?, updated_at = ?
		WHERE id = ?
	`, p.Name, p.Enabled, p.Priority, p.BaseURL, p.UpdatedAt, p.ID)
	if err != nil {
		return fmt.Errorf("update provider: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ProviderRepository) UpdatePriority(ctx context.Context, id string, priority int) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE providers
		SET priority = ?, updated_at = ?
		WHERE id = ?
	`, priority, time.Now().UTC(), id)
	return err
}

func (r *ProviderRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM providers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete provider: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
