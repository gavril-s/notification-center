package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"notification-center/delivery/internal/domain"
)

var ErrAttemptNotFound = errors.New("attempt not found")

type DeliveryRepository struct {
	db *pgxpool.Pool
}

func NewDeliveryRepository(db *pgxpool.Pool) *DeliveryRepository {
	return &DeliveryRepository{db: db}
}

func (r *DeliveryRepository) CreateAttempt(ctx context.Context, attempt *domain.DeliveryAttempt) error {
	query := `
		INSERT INTO delivery.delivery_attempts (
			id, notification_id, attempt_number, channel, provider_code, status,
			error_code, error_message, request_payload, response_payload,
			started_at, trace_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	requestPayload, _ := json.Marshal(attempt.RequestPayload)
	_, err := r.db.Exec(ctx, query,
		attempt.ID, attempt.NotificationID, attempt.AttemptNumber, attempt.Channel,
		attempt.ProviderCode, attempt.Status, attempt.ErrorCode, attempt.ErrorMessage,
		requestPayload, nil, attempt.StartedAt, attempt.TraceID,
	)
	return err
}

func (r *DeliveryRepository) GetByNotificationAndAttempt(ctx context.Context, notificationID string, attemptNumber int) (*domain.DeliveryAttempt, error) {
	query := `
		SELECT id, notification_id, attempt_number, channel, provider_code, status,
			error_code, error_message, request_payload, response_payload,
			started_at, completed_at, trace_id
		FROM delivery.delivery_attempts
		WHERE notification_id = $1 AND attempt_number = $2
	`
	var attempt domain.DeliveryAttempt
	var requestPayload, responsePayload []byte
	var errorCode, errorMessage, traceID *string

	err := r.db.QueryRow(ctx, query, notificationID, attemptNumber).Scan(
		&attempt.ID, &attempt.NotificationID, &attempt.AttemptNumber, &attempt.Channel,
		&attempt.ProviderCode, &attempt.Status, &errorCode, &errorMessage,
		&requestPayload, &responsePayload, &attempt.StartedAt, &attempt.CompletedAt, &traceID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAttemptNotFound
		}
		return nil, fmt.Errorf("failed to get attempt: %w", err)
	}

	attempt.ErrorCode = errorCode
	attempt.ErrorMessage = errorMessage
	attempt.TraceID = traceID

	_ = json.Unmarshal(requestPayload, &attempt.RequestPayload)
	_ = json.Unmarshal(responsePayload, &attempt.ResponsePayload)

	return &attempt, nil
}

func (r *DeliveryRepository) UpdateAttemptStatus(ctx context.Context, id string, status domain.DeliveryStatus, errorCode, errorMessage *string, responsePayload map[string]any) error {
	query := `
		UPDATE delivery.delivery_attempts
		SET status = $2, error_code = $3, error_message = $4, response_payload = $5, completed_at = NOW()
		WHERE id = $1
	`
	respPayload, _ := json.Marshal(responsePayload)
	_, err := r.db.Exec(ctx, query, id, status, errorCode, errorMessage, respPayload)
	return err
}

func (r *DeliveryRepository) AttemptExists(ctx context.Context, notificationID string, attemptNumber int) (bool, error) {
	query := `
		SELECT EXISTS(SELECT 1 FROM delivery.delivery_attempts WHERE notification_id = $1 AND attempt_number = $2)
	`
	var exists bool
	err := r.db.QueryRow(ctx, query, notificationID, attemptNumber).Scan(&exists)
	return exists, err
}
