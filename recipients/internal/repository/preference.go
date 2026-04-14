package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"notification-center/recipients/internal/domain"
)

type PreferenceRepository struct {
	db *pgxpool.Pool
}

func NewPreferenceRepository(db *pgxpool.Pool) *PreferenceRepository {
	return &PreferenceRepository{db: db}
}

func (r *PreferenceRepository) Create(ctx context.Context, pref *domain.Preference) error {
	if pref.ID == "" {
		pref.ID = uuid.New().String()
	}

	blockedChannelsJSON, err := json.Marshal(pref.BlockedChannels)
	if err != nil {
		return fmt.Errorf("failed to marshal blocked channels: %w", err)
	}

	query := `
		INSERT INTO recipients.preferences 
			(id, contact_id, user_id, sender_id, scope_type, scope_id, enabled, quiet_from, quiet_to, blocked_channels, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
	`

	_, err = r.db.Exec(ctx, query,
		pref.ID, pref.ContactID, pref.UserID, pref.SenderID, pref.ScopeType, pref.ScopeID,
		pref.Enabled, pref.QuietFrom, pref.QuietTo, blockedChannelsJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create preference: %w", err)
	}

	return nil
}

func (r *PreferenceRepository) GetByID(ctx context.Context, id string) (*domain.Preference, error) {
	query := `
		SELECT id, contact_id, user_id, sender_id, scope_type, scope_id, enabled, quiet_from, quiet_to, blocked_channels, created_at, updated_at
		FROM recipients.preferences
		WHERE id = $1
	`

	var pref domain.Preference
	var blockedChannelsJSON []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&pref.ID, &pref.ContactID, &pref.UserID, &pref.SenderID, &pref.ScopeType, &pref.ScopeID,
		&pref.Enabled, &pref.QuietFrom, &pref.QuietTo, &blockedChannelsJSON, &pref.CreatedAt, &pref.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get preference: %w", err)
	}

	if err := json.Unmarshal(blockedChannelsJSON, &pref.BlockedChannels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal blocked channels: %w", err)
	}

	return &pref, nil
}

func (r *PreferenceRepository) GetByContactID(ctx context.Context, contactID string) ([]domain.Preference, error) {
	query := `
		SELECT id, contact_id, user_id, sender_id, scope_type, scope_id, enabled, quiet_from, quiet_to, blocked_channels, created_at, updated_at
		FROM recipients.preferences
		WHERE contact_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get preferences: %w", err)
	}
	defer rows.Close()

	var preferences []domain.Preference
	for rows.Next() {
		var pref domain.Preference
		var blockedChannelsJSON []byte
		err := rows.Scan(
			&pref.ID, &pref.ContactID, &pref.UserID, &pref.SenderID, &pref.ScopeType, &pref.ScopeID,
			&pref.Enabled, &pref.QuietFrom, &pref.QuietTo, &blockedChannelsJSON, &pref.CreatedAt, &pref.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan preference: %w", err)
		}
		if err := json.Unmarshal(blockedChannelsJSON, &pref.BlockedChannels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal blocked channels: %w", err)
		}
		preferences = append(preferences, pref)
	}

	return preferences, nil
}

func (r *PreferenceRepository) GetByContactAndScope(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
	query := `
		SELECT id, contact_id, user_id, sender_id, scope_type, scope_id, enabled, quiet_from, quiet_to, blocked_channels, created_at, updated_at
		FROM recipients.preferences
		WHERE contact_id = $1 AND scope_type = $2 AND COALESCE(sender_id, '') = COALESCE($3, '') AND COALESCE(scope_id, '') = COALESCE($4, '')
	`

	var pref domain.Preference
	var blockedChannelsJSON []byte
	err := r.db.QueryRow(ctx, query, contactID, scopeType, senderID, scopeID).Scan(
		&pref.ID, &pref.ContactID, &pref.UserID, &pref.SenderID, &pref.ScopeType, &pref.ScopeID,
		&pref.Enabled, &pref.QuietFrom, &pref.QuietTo, &blockedChannelsJSON, &pref.CreatedAt, &pref.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get preference by scope: %w", err)
	}

	if err := json.Unmarshal(blockedChannelsJSON, &pref.BlockedChannels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal blocked channels: %w", err)
	}

	return &pref, nil
}

func (r *PreferenceRepository) Update(ctx context.Context, pref *domain.Preference) error {
	blockedChannelsJSON, err := json.Marshal(pref.BlockedChannels)
	if err != nil {
		return fmt.Errorf("failed to marshal blocked channels: %w", err)
	}

	query := `
		UPDATE recipients.preferences
		SET sender_id = $2, scope_type = $3, scope_id = $4, enabled = $5, quiet_from = $6, quiet_to = $7, blocked_channels = $8, updated_at = NOW()
		WHERE id = $1
	`

	_, err = r.db.Exec(ctx, query,
		pref.ID, pref.SenderID, pref.ScopeType, pref.ScopeID, pref.Enabled, pref.QuietFrom, pref.QuietTo, blockedChannelsJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to update preference: %w", err)
	}

	return nil
}

func (r *PreferenceRepository) Upsert(ctx context.Context, pref *domain.Preference) error {
	existing, err := r.GetByContactAndScope(ctx, pref.ContactID, pref.ScopeType, pref.SenderID, pref.ScopeID)
	if err == nil {
		pref.ID = existing.ID
		return r.Update(ctx, pref)
	}
	if !errors.Is(err, ErrNotFound) {
		return err
	}
	return r.Create(ctx, pref)
}
