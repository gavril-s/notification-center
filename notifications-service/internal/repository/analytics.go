package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"notification-center/notifications-service/internal/domain"
)

type AnalyticsRepository struct {
	db *pgxpool.Pool
}

func NewAnalyticsRepository(db *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

func (r *AnalyticsRepository) IncrementSent(ctx context.Context, senderID string, date time.Time, channel domain.Channel) error {
	query := `
		INSERT INTO notifications.sender_analytics_daily (id, sender_id, date, channel, sent_count, delivered_count, failed_count, skipped_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 1, 0, 0, 0, NOW(), NOW())
		ON CONFLICT (sender_id, date, channel)
		DO UPDATE SET sent_count = sender_analytics_daily.sent_count + 1, updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, uuid.New().String(), senderID, date, channel)
	return err
}

func (r *AnalyticsRepository) IncrementDelivered(ctx context.Context, senderID string, date time.Time, channel domain.Channel) error {
	query := `
		INSERT INTO notifications.sender_analytics_daily (id, sender_id, date, channel, sent_count, delivered_count, failed_count, skipped_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 0, 1, 0, 0, NOW(), NOW())
		ON CONFLICT (sender_id, date, channel)
		DO UPDATE SET delivered_count = sender_analytics_daily.delivered_count + 1, updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, uuid.New().String(), senderID, date, channel)
	return err
}

func (r *AnalyticsRepository) IncrementFailed(ctx context.Context, senderID string, date time.Time, channel domain.Channel) error {
	query := `
		INSERT INTO notifications.sender_analytics_daily (id, sender_id, date, channel, sent_count, delivered_count, failed_count, skipped_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 0, 0, 1, 0, NOW(), NOW())
		ON CONFLICT (sender_id, date, channel)
		DO UPDATE SET failed_count = sender_analytics_daily.failed_count + 1, updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, uuid.New().String(), senderID, date, channel)
	return err
}

func (r *AnalyticsRepository) IncrementSkipped(ctx context.Context, senderID string, date time.Time, channel domain.Channel) error {
	query := `
		INSERT INTO notifications.sender_analytics_daily (id, sender_id, date, channel, sent_count, delivered_count, failed_count, skipped_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 0, 0, 0, 1, NOW(), NOW())
		ON CONFLICT (sender_id, date, channel)
		DO UPDATE SET skipped_count = sender_analytics_daily.skipped_count + 1, updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, uuid.New().String(), senderID, date, channel)
	return err
}

func (r *AnalyticsRepository) GetBySender(ctx context.Context, senderID string, from, to time.Time) ([]*domain.SenderAnalyticsDaily, error) {
	query := `
		SELECT id, sender_id, date, channel, sent_count, delivered_count, failed_count, skipped_count, created_at, updated_at
		FROM notifications.sender_analytics_daily
		WHERE sender_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date DESC
	`
	rows, err := r.db.Query(ctx, query, senderID, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	}
	defer rows.Close()

	var items []*domain.SenderAnalyticsDaily
	for rows.Next() {
		var a domain.SenderAnalyticsDaily
		err := rows.Scan(
			&a.ID, &a.SenderID, &a.Date, &a.Channel, &a.SentCount,
			&a.DeliveredCount, &a.FailedCount, &a.SkippedCount,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan analytics: %w", err)
		}
		items = append(items, &a)
	}

	return items, nil
}
