package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"notification-center/recipients/internal/domain"
)

type ContactRepository struct {
	db *pgxpool.Pool
}

func NewContactRepository(db *pgxpool.Pool) *ContactRepository {
	return &ContactRepository{db: db}
}

func (r *ContactRepository) Create(ctx context.Context, contact *domain.Contact) error {
	if contact.ID == "" {
		contact.ID = uuid.New().String()
	}

	query := `
		INSERT INTO recipients.contacts (id, user_id, channel, value, is_verified, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	`

	_, err := r.db.Exec(ctx, query,
		contact.ID, contact.UserID, contact.Channel, contact.Value, contact.IsVerified, contact.Enabled,
	)
	if err != nil {
		return fmt.Errorf("failed to create contact: %w", err)
	}

	return nil
}

func (r *ContactRepository) GetByID(ctx context.Context, id string) (*domain.Contact, error) {
	query := `
		SELECT id, user_id, channel, value, is_verified, enabled, created_at, updated_at, deleted_at
		FROM recipients.contacts
		WHERE id = $1
	`

	var contact domain.Contact
	err := r.db.QueryRow(ctx, query, id).Scan(
		&contact.ID, &contact.UserID, &contact.Channel, &contact.Value,
		&contact.IsVerified, &contact.Enabled, &contact.CreatedAt, &contact.UpdatedAt, &contact.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	return &contact, nil
}

func (r *ContactRepository) GetByUserID(ctx context.Context, userID string) ([]domain.Contact, error) {
	query := `
		SELECT id, user_id, channel, value, is_verified, enabled, created_at, updated_at, deleted_at
		FROM recipients.contacts
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}
	defer rows.Close()

	var contacts []domain.Contact
	for rows.Next() {
		var contact domain.Contact
		err := rows.Scan(
			&contact.ID, &contact.UserID, &contact.Channel, &contact.Value,
			&contact.IsVerified, &contact.Enabled, &contact.CreatedAt, &contact.UpdatedAt, &contact.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}
		contacts = append(contacts, contact)
	}

	return contacts, nil
}

func (r *ContactRepository) GetAllByUserID(ctx context.Context, userID string) ([]domain.Contact, error) {
	query := `
		SELECT id, user_id, channel, value, is_verified, enabled, created_at, updated_at, deleted_at
		FROM recipients.contacts
		WHERE user_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all contacts: %w", err)
	}
	defer rows.Close()

	var contacts []domain.Contact
	for rows.Next() {
		var contact domain.Contact
		err := rows.Scan(
			&contact.ID, &contact.UserID, &contact.Channel, &contact.Value,
			&contact.IsVerified, &contact.Enabled, &contact.CreatedAt, &contact.UpdatedAt, &contact.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}
		contacts = append(contacts, contact)
	}

	return contacts, nil
}

func (r *ContactRepository) GetByUserIDAndChannel(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
	query := `
		SELECT id, user_id, channel, value, is_verified, enabled, created_at, updated_at, deleted_at
		FROM recipients.contacts
		WHERE user_id = $1 AND channel = $2 AND value = $3 AND deleted_at IS NULL
	`

	var contact domain.Contact
	err := r.db.QueryRow(ctx, query, userID, channel, value).Scan(
		&contact.ID, &contact.UserID, &contact.Channel, &contact.Value,
		&contact.IsVerified, &contact.Enabled, &contact.CreatedAt, &contact.UpdatedAt, &contact.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get contact by channel and value: %w", err)
	}

	return &contact, nil
}

func (r *ContactRepository) Update(ctx context.Context, contact *domain.Contact) error {
	query := `
		UPDATE recipients.contacts
		SET channel = $2, value = $3, is_verified = $4, enabled = $5, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		contact.ID, contact.Channel, contact.Value, contact.IsVerified, contact.Enabled,
	)
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	return nil
}

func (r *ContactRepository) SoftDelete(ctx context.Context, id string) error {
	query := `
		UPDATE recipients.contacts
		SET deleted_at = $2, enabled = false, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to soft delete contact: %w", err)
	}

	return nil
}

func (r *ContactRepository) GetByIDs(ctx context.Context, ids []string) ([]domain.Contact, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `
		SELECT id, user_id, channel, value, is_verified, enabled, created_at, updated_at, deleted_at
		FROM recipients.contacts
		WHERE id = ANY($1)
	`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts by ids: %w", err)
	}
	defer rows.Close()

	var contacts []domain.Contact
	for rows.Next() {
		var contact domain.Contact
		err := rows.Scan(
			&contact.ID, &contact.UserID, &contact.Channel, &contact.Value,
			&contact.IsVerified, &contact.Enabled, &contact.CreatedAt, &contact.UpdatedAt, &contact.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contact: %w", err)
		}
		contacts = append(contacts, contact)
	}

	return contacts, nil
}
