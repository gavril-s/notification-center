package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"notification-center/notifications-service/internal/domain"
)

type OutboxRepository struct {
	db *pgxpool.Pool
}

func NewOutboxRepository(db *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) Create(ctx context.Context, outbox *domain.NotificationOutbox) error {
	query := `
		INSERT INTO notifications.notification_outbox (
			id, notification_id, event_type, payload, attempt, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	payload, _ := json.Marshal(outbox.Payload)
	_, err := r.db.Exec(ctx, query,
		outbox.ID, outbox.NotificationID, outbox.EventType, payload,
		outbox.Attempt, outbox.Status, outbox.CreatedAt,
	)
	return err
}

func (r *OutboxRepository) GetPending(ctx context.Context, limit int) ([]*domain.NotificationOutbox, error) {
	query := `
		SELECT id, notification_id, event_type, payload, attempt, status, created_at, published_at, error_message
		FROM notifications.notification_outbox
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
	`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending outbox entries: %w", err)
	}
	defer rows.Close()

	var outboxes []*domain.NotificationOutbox
	for rows.Next() {
		var o domain.NotificationOutbox
		var payload []byte
		var publishedAt, errorMessage *string

		err := rows.Scan(&o.ID, &o.NotificationID, &o.EventType, &payload, &o.Attempt, &o.Status, &o.CreatedAt, &publishedAt, &errorMessage)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox entry: %w", err)
		}

		_ = json.Unmarshal(payload, &o.Payload)
		if publishedAt != nil {
			o.PublishedAt = nil // Would need proper time parsing
		}
		if errorMessage != nil {
			o.ErrorMessage = errorMessage
		}
		outboxes = append(outboxes, &o)
	}

	return outboxes, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, id string) error {
	query := `
		UPDATE notifications.notification_outbox
		SET status = 'published', published_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id string, errMsg string) error {
	query := `
		UPDATE notifications.notification_outbox
		SET status = 'failed', error_message = $2
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, errMsg)
	return err
}
