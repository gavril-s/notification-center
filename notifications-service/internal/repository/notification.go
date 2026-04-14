package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"notification-center/notifications-service/internal/domain"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrIdempotencyConflict  = errors.New("notification with this idempotency key already exists")
)

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	query := `
		INSERT INTO notifications.notifications (
			id, sender_id, contact_id, campaign_id, group_id, template_id, channel,
			status, subject, rendered_content, unsubscribe_url, idempotency_key,
			scheduled_at, created_at, updated_at, attempt_count, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	metadata, _ := json.Marshal(n.Metadata)
	_, err := r.db.Exec(ctx, query,
		n.ID, n.SenderID, n.ContactID, n.CampaignID, n.GroupID, n.TemplateID, n.Channel,
		n.Status, n.Subject, n.RenderedContent, n.UnsubscribeURL, n.IdempotencyKey,
		n.ScheduledAt, n.CreatedAt, n.UpdatedAt, n.AttemptCount, metadata,
	)
	return err
}

func (r *NotificationRepository) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	query := `
		SELECT id, sender_id, contact_id, campaign_id, group_id, template_id, channel,
			status, subject, rendered_content, unsubscribe_url, idempotency_key,
			scheduled_at, created_at, updated_at, sent_at, delivered_at, failed_at,
			failure_reason, attempt_count, metadata
		FROM notifications.notifications
		WHERE id = $1
	`
	var n domain.Notification
	var metadata []byte
	var campaignID, groupID sql.NullString
	var scheduledAt, sentAt, deliveredAt, failedAt sql.NullTime
	var subject, unsubscribeURL, failureReason sql.NullString

	err := r.db.QueryRow(ctx, query, id).Scan(
		&n.ID, &n.SenderID, &n.ContactID, &campaignID, &groupID, &n.TemplateID, &n.Channel,
		&n.Status, &subject, &n.RenderedContent, &unsubscribeURL, &n.IdempotencyKey,
		&scheduledAt, &n.CreatedAt, &n.UpdatedAt, &sentAt, &deliveredAt, &failedAt,
		&failureReason, &n.AttemptCount, &metadata,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotificationNotFound
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	// Set nullable fields
	if campaignID.Valid {
		n.CampaignID = &campaignID.String
	}
	if groupID.Valid {
		n.GroupID = &groupID.String
	}
	if subject.Valid {
		n.Subject = &subject.String
	}
	if unsubscribeURL.Valid {
		n.UnsubscribeURL = &unsubscribeURL.String
	}
	if scheduledAt.Valid {
		n.ScheduledAt = &scheduledAt.Time
	}
	if sentAt.Valid {
		n.SentAt = &sentAt.Time
	}
	if deliveredAt.Valid {
		n.DeliveredAt = &deliveredAt.Time
	}
	if failedAt.Valid {
		n.FailedAt = &failedAt.Time
	}
	if failureReason.Valid {
		n.FailureReason = &failureReason.String
	}

	_ = json.Unmarshal(metadata, &n.Metadata)
	return &n, nil
}

func (r *NotificationRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Notification, error) {
	query := `
		SELECT id, sender_id, contact_id, campaign_id, group_id, template_id, channel,
			status, subject, rendered_content, unsubscribe_url, idempotency_key,
			scheduled_at, created_at, updated_at, sent_at, delivered_at, failed_at,
			failure_reason, attempt_count, metadata
		FROM notifications.notifications
		WHERE idempotency_key = $1
	`
	var n domain.Notification
	var metadata []byte
	var campaignID, groupID sql.NullString
	var scheduledAt, sentAt, deliveredAt, failedAt sql.NullTime
	var subject, unsubscribeURL, failureReason sql.NullString

	err := r.db.QueryRow(ctx, query, key).Scan(
		&n.ID, &n.SenderID, &n.ContactID, &campaignID, &groupID, &n.TemplateID, &n.Channel,
		&n.Status, &subject, &n.RenderedContent, &unsubscribeURL, &n.IdempotencyKey,
		&scheduledAt, &n.CreatedAt, &n.UpdatedAt, &sentAt, &deliveredAt, &failedAt,
		&failureReason, &n.AttemptCount, &metadata,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotificationNotFound
		}
		return nil, fmt.Errorf("failed to get notification by idempotency key: %w", err)
	}

	if campaignID.Valid {
		n.CampaignID = &campaignID.String
	}
	if groupID.Valid {
		n.GroupID = &groupID.String
	}
	if subject.Valid {
		n.Subject = &subject.String
	}
	if unsubscribeURL.Valid {
		n.UnsubscribeURL = &unsubscribeURL.String
	}
	if scheduledAt.Valid {
		n.ScheduledAt = &scheduledAt.Time
	}
	if sentAt.Valid {
		n.SentAt = &sentAt.Time
	}
	if deliveredAt.Valid {
		n.DeliveredAt = &deliveredAt.Time
	}
	if failedAt.Valid {
		n.FailedAt = &failedAt.Time
	}
	if failureReason.Valid {
		n.FailureReason = &failureReason.String
	}

	_ = json.Unmarshal(metadata, &n.Metadata)
	return &n, nil
}

func (r *NotificationRepository) UpdateStatus(ctx context.Context, id string, status domain.NotificationStatus) error {
	query := `
		UPDATE notifications.notifications
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, status)
	return err
}

func (r *NotificationRepository) UpdateStatusWithTimestamp(ctx context.Context, id string, status domain.NotificationStatus) error {
	query := `
		UPDATE notifications.notifications
		SET status = $2, updated_at = NOW()
	`
	switch status {
	case domain.StatusSent:
		query += ", sent_at = NOW()"
	case domain.StatusDelivered:
		query += ", delivered_at = NOW()"
	case domain.StatusFailed:
		query += ", failed_at = NOW()"
	}
	query += " WHERE id = $1"

	_, err := r.db.Exec(ctx, query, id, status)
	return err
}

func (r *NotificationRepository) IncrementAttemptCount(ctx context.Context, id string) error {
	query := `
		UPDATE notifications.notifications
		SET attempt_count = attempt_count + 1, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
