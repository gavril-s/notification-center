package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"notification-center/notifications-service/internal/domain"
)

type HistoryRepository struct {
	db *pgxpool.Pool
}

func NewHistoryRepository(db *pgxpool.Pool) *HistoryRepository {
	return &HistoryRepository{db: db}
}

func (r *HistoryRepository) Create(ctx context.Context, h *domain.NotificationHistory) error {
	query := `
		INSERT INTO notifications.notification_history (
			id, notification_id, contact_id, user_id, channel, status, sender_id,
			campaign_id, group_id, created_at, delivered_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Exec(ctx, query,
		h.ID, h.NotificationID, h.ContactID, h.UserID, h.Channel, h.Status,
		h.SenderID, h.CampaignID, h.GroupID, h.CreatedAt, h.DeliveredAt,
	)
	return err
}

func (r *HistoryRepository) GetByContactID(ctx context.Context, contactID string, page, size int) ([]*domain.NotificationHistory, int, error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM notifications.notification_history WHERE contact_id = $1`
	err := r.db.QueryRow(ctx, countQuery, contactID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count history: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, notification_id, contact_id, user_id, channel, status, sender_id,
			campaign_id, group_id, created_at, delivered_at
		FROM notifications.notification_history
		WHERE contact_id = $1
		ORDER BY created_at DESC, notification_id DESC
		LIMIT $2 OFFSET $3
	`
	offset := (page - 1) * size
	rows, err := r.db.Query(ctx, query, contactID, size, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get history: %w", err)
	}
	defer rows.Close()

	var items []*domain.NotificationHistory
	for rows.Next() {
		var h domain.NotificationHistory
		err := rows.Scan(
			&h.ID, &h.NotificationID, &h.ContactID, &h.UserID, &h.Channel, &h.Status,
			&h.SenderID, &h.CampaignID, &h.GroupID, &h.CreatedAt, &h.DeliveredAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan history: %w", err)
		}
		items = append(items, &h)
	}

	return items, total, nil
}

func (r *HistoryRepository) GetByUserID(ctx context.Context, userID string, page, size int) ([]*domain.NotificationHistory, int, error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM notifications.notification_history WHERE user_id = $1`
	err := r.db.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count history: %w", err)
	}

	// Get paginated results
	query := `
		SELECT id, notification_id, contact_id, user_id, channel, status, sender_id,
			campaign_id, group_id, created_at, delivered_at
		FROM notifications.notification_history
		WHERE user_id = $1
		ORDER BY created_at DESC, notification_id DESC
		LIMIT $2 OFFSET $3
	`
	offset := (page - 1) * size
	rows, err := r.db.Query(ctx, query, userID, size, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get history: %w", err)
	}
	defer rows.Close()

	var items []*domain.NotificationHistory
	for rows.Next() {
		var h domain.NotificationHistory
		err := rows.Scan(
			&h.ID, &h.NotificationID, &h.ContactID, &h.UserID, &h.Channel, &h.Status,
			&h.SenderID, &h.CampaignID, &h.GroupID, &h.CreatedAt, &h.DeliveredAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan history: %w", err)
		}
		items = append(items, &h)
	}

	return items, total, nil
}

func (r *HistoryRepository) GetBySenderID(ctx context.Context, senderID string, campaignID, status *string, page, size int) ([]*domain.NotificationHistory, int, error) {
	// Build query dynamically
	baseQuery := `FROM notifications.notification_history WHERE sender_id = $1`
	args := []any{senderID}
	argIdx := 2

	if campaignID != nil && *campaignID != "" {
		baseQuery += fmt.Sprintf(" AND campaign_id = $%d", argIdx)
		args = append(args, *campaignID)
		argIdx++
	}
	if status != nil && *status != "" {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *status)
		argIdx++
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count history: %w", err)
	}

	// Get paginated results
	selectQuery := `
		SELECT id, notification_id, contact_id, user_id, channel, status, sender_id,
			campaign_id, group_id, created_at, delivered_at
	` + baseQuery + ` ORDER BY created_at DESC, notification_id DESC LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)
	argIdx++
	offset := (page - 1) * size
	args = append(args, size, offset)

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get history: %w", err)
	}
	defer rows.Close()

	var items []*domain.NotificationHistory
	for rows.Next() {
		var h domain.NotificationHistory
		err := rows.Scan(
			&h.ID, &h.NotificationID, &h.ContactID, &h.UserID, &h.Channel, &h.Status,
			&h.SenderID, &h.CampaignID, &h.GroupID, &h.CreatedAt, &h.DeliveredAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan history: %w", err)
		}
		items = append(items, &h)
	}

	return items, total, nil
}
