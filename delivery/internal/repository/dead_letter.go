package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"notification-center/delivery/internal/domain"
)

type DeadLetterRepository struct {
	db *pgxpool.Pool
}

func NewDeadLetterRepository(db *pgxpool.Pool) *DeadLetterRepository {
	return &DeadLetterRepository{db: db}
}

func (r *DeadLetterRepository) Create(ctx context.Context, dl *domain.DeadLetter) error {
	query := `
		INSERT INTO delivery.delivery_dead_letters (
			id, notification_id, attempt_number, channel, payload, error_message,
			error_code, received_at, trace_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	payload, _ := json.Marshal(dl.Payload)
	_, err := r.db.Exec(ctx, query,
		dl.ID, dl.NotificationID, dl.AttemptNumber, dl.Channel, payload,
		dl.ErrorMessage, dl.ErrorCode, dl.ReceivedAt, dl.TraceID,
	)
	return err
}

func (r *DeadLetterRepository) GetByNotificationID(ctx context.Context, notificationID string) ([]*domain.DeadLetter, error) {
	query := `
		SELECT id, notification_id, attempt_number, channel, payload, error_message,
			error_code, received_at, processed_at, trace_id
		FROM delivery.delivery_dead_letters
		WHERE notification_id = $1
		ORDER BY attempt_number DESC
	`
	rows, err := r.db.Query(ctx, query, notificationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get dead letters: %w", err)
	}
	defer rows.Close()

	var items []*domain.DeadLetter
	for rows.Next() {
		var dl domain.DeadLetter
		var payload []byte
		var errorCode, traceID *string

		err := rows.Scan(
			&dl.ID, &dl.NotificationID, &dl.AttemptNumber, &dl.Channel, &payload,
			&dl.ErrorMessage, &errorCode, &dl.ReceivedAt, &dl.ProcessedAt, &traceID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dead letter: %w", err)
		}

		dl.ErrorCode = errorCode
		dl.TraceID = traceID

		_ = json.Unmarshal(payload, &dl.Payload)
		items = append(items, &dl)
	}

	return items, nil
}
