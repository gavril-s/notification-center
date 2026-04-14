package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/rabbitmq/amqp091-go"

	"notification-center/delivery/internal/domain"
	"notification-center/delivery/internal/repository"
)

type ConsumerService struct {
	repo     *repository.DeliveryRepository
	deadRepo *repository.DeadLetterRepository
	provider *MockProvider
}

func NewConsumerService(repo *repository.DeliveryRepository, deadRepo *repository.DeadLetterRepository, provider *MockProvider) *ConsumerService {
	return &ConsumerService{
		repo:     repo,
		deadRepo: deadRepo,
		provider: provider,
	}
}

func (s *ConsumerService) Consume(ctx context.Context, msg amqp091.Delivery) error {
	var event domain.DispatchEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("failed to unmarshal message: %v", err)
		return msg.Nack(false, false)
	}

	log.Printf("processing notification %s, attempt %d", event.NotificationID, event.Attempt)

	// Check idempotency - has this attempt already been processed?
	exists, err := s.repo.AttemptExists(ctx, event.NotificationID, event.Attempt)
	if err != nil {
		log.Printf("failed to check attempt existence: %v", err)
		return msg.Nack(false, true)
	}
	if exists {
		log.Printf("attempt already processed, skipping: notification=%s, attempt=%d", event.NotificationID, event.Attempt)
		return msg.Ack(false)
	}

	// Create delivery attempt record
	attempt := &domain.DeliveryAttempt{
		ID:             uuid.New().String(),
		NotificationID: event.NotificationID,
		AttemptNumber:  event.Attempt,
		Channel:        event.Channel,
		ProviderCode:   "mock_provider",
		Status:         domain.DeliveryStatusPending,
		StartedAt:      time.Now().UTC(),
		TraceID:        &event.TraceID,
	}

	if err := s.repo.CreateAttempt(ctx, attempt); err != nil {
		log.Printf("failed to create attempt: %v", err)
		return msg.Nack(false, true)
	}

	// Process delivery
	result := s.provider.Deliver(ctx, &event)

	// Update attempt with result
	errCode := result.ErrorCode
	errMsg := result.ErrorMessage
	s.repo.UpdateAttemptStatus(ctx, attempt.ID, result.Status, errCode, errMsg, nil)

	// Send callback to notifications service
	s.notifyNotificationsService(ctx, event.NotificationID, attempt.ID, result)

	// Handle dead letter for failures after max retries
	if result.Status == domain.DeliveryStatusFailed && event.Attempt >= 3 {
		s.saveDeadLetter(ctx, event, result)
	}

	return msg.Ack(false)
}

func (s *ConsumerService) notifyNotificationsService(ctx context.Context, notificationID, attemptID string, result DeliveryResult) {
	// Map transport status to canonical notification status
	var status string
	switch result.Status {
	case domain.DeliveryStatusDelivered:
		status = "delivered"
	case domain.DeliveryStatusFailed:
		status = "failed"
	default:
		status = "sent"
	}

	// In production, this would make an HTTP call to the notifications service
	// For now, just log the callback
	log.Printf("notifying notifications service: notification=%s, status=%s, attempt=%s", notificationID, status, attemptID)

	// TODO: Make actual HTTP call to:
	// POST /internal/notifications/{notification_id}/delivery-status
	// with payload: {attempt_id, transport_status, provider_code, error_code, occurred_at}
}

func (s *ConsumerService) saveDeadLetter(ctx context.Context, event domain.DispatchEvent, result DeliveryResult) {
	dl := &domain.DeadLetter{
		ID:             uuid.New().String(),
		NotificationID: event.NotificationID,
		AttemptNumber:  event.Attempt,
		Channel:        event.Channel,
		Payload: map[string]any{
			"event_id":         event.EventID,
			"notification_id":  event.NotificationID,
			"sender_id":        event.SenderID,
			"contact_id":       event.ContactID,
			"channel":          event.Channel,
			"subject":          event.Subject,
			"rendered_content": event.RenderedContent,
			"metadata":         event.Metadata,
		},
		ErrorMessage: "max retries exceeded",
		ReceivedAt:   time.Now().UTC(),
		TraceID:      &event.TraceID,
	}

	if result.ErrorCode != nil {
		dl.ErrorCode = result.ErrorCode
	}
	if result.ErrorMessage != nil {
		dl.ErrorMessage = *result.ErrorMessage
	}

	if err := s.deadRepo.Create(ctx, dl); err != nil {
		log.Printf("failed to save dead letter: %v", err)
	}
	log.Printf("saved to dead letter: notification=%s", event.NotificationID)
}
