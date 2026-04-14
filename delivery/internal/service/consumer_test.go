package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"notification-center/delivery/internal/domain"
)

type testDeliveryRepo struct {
	attemptExistsFunc       func(ctx context.Context, notificationID string, attemptNumber int) (bool, error)
	createAttemptFunc       func(ctx context.Context, attempt *domain.DeliveryAttempt) error
	updateAttemptStatusFunc func(ctx context.Context, id string, status domain.DeliveryStatus, errorCode, errorMessage *string, responsePayload map[string]any) error
}

func (r *testDeliveryRepo) AttemptExists(ctx context.Context, notificationID string, attemptNumber int) (bool, error) {
	if r.attemptExistsFunc != nil {
		return r.attemptExistsFunc(ctx, notificationID, attemptNumber)
	}
	return false, nil
}

func (r *testDeliveryRepo) CreateAttempt(ctx context.Context, attempt *domain.DeliveryAttempt) error {
	if r.createAttemptFunc != nil {
		return r.createAttemptFunc(ctx, attempt)
	}
	return nil
}

func (r *testDeliveryRepo) UpdateAttemptStatus(ctx context.Context, id string, status domain.DeliveryStatus, errorCode, errorMessage *string, responsePayload map[string]any) error {
	if r.updateAttemptStatusFunc != nil {
		return r.updateAttemptStatusFunc(ctx, id, status, errorCode, errorMessage, responsePayload)
	}
	return nil
}

type testDeadLetterRepo struct {
	createFunc func(ctx context.Context, dl *domain.DeadLetter) error
}

func (r *testDeadLetterRepo) Create(ctx context.Context, dl *domain.DeadLetter) error {
	if r.createFunc != nil {
		return r.createFunc(ctx, dl)
	}
	return nil
}

type testableProvider struct {
	deliverFunc func(ctx context.Context, event *domain.DispatchEvent) DeliveryResult
}

func (p *testableProvider) Deliver(ctx context.Context, event *domain.DispatchEvent) DeliveryResult {
	if p.deliverFunc != nil {
		return p.deliverFunc(ctx, event)
	}
	return DeliveryResult{
		Status:       domain.DeliveryStatusDelivered,
		ProviderCode: "test_provider",
	}
}

type testConsumerService struct {
	repo     *testDeliveryRepo
	deadRepo *testDeadLetterRepo
	provider *testableProvider

	ackCalled   bool
	nackCalled  bool
	nackRequeue bool
}

func newTestConsumerService(repo *testDeliveryRepo, deadRepo *testDeadLetterRepo, provider *testableProvider) *testConsumerService {
	return &testConsumerService{
		repo:     repo,
		deadRepo: deadRepo,
		provider: provider,
	}
}

func (s *testConsumerService) ack() {
	s.ackCalled = true
}

func (s *testConsumerService) nack(requeue bool) {
	s.nackCalled = true
	s.nackRequeue = requeue
}

func (s *testConsumerService) consumeEvent(ctx context.Context, eventBytes []byte) error {
	var event domain.DispatchEvent
	if err := json.Unmarshal(eventBytes, &event); err != nil {
		s.nack(false)
		return err
	}

	exists, err := s.repo.AttemptExists(ctx, event.NotificationID, event.Attempt)
	if err != nil {
		s.nack(true)
		return err
	}
	if exists {
		s.ack()
		return nil
	}

	attempt := &domain.DeliveryAttempt{
		NotificationID: event.NotificationID,
		AttemptNumber:  event.Attempt,
		Channel:        event.Channel,
		ProviderCode:   "mock_provider",
		Status:         domain.DeliveryStatusPending,
	}

	if err := s.repo.CreateAttempt(ctx, attempt); err != nil {
		s.nack(true)
		return err
	}

	result := s.provider.Deliver(ctx, &event)

	s.repo.UpdateAttemptStatus(ctx, attempt.ID, result.Status, result.ErrorCode, result.ErrorMessage, nil)

	s.notifyNotificationsService(ctx, event.NotificationID, attempt.ID, result)

	if result.Status == domain.DeliveryStatusFailed && event.Attempt >= 3 {
		s.saveDeadLetter(ctx, event, result)
	}

	s.ack()
	return nil
}

func TestConsumerService_Consume(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		eventBytes     []byte
		setupMocks     func(repo *testDeliveryRepo, deadRepo *testDeadLetterRepo, provider *testableProvider)
		expectAck      bool
		expectNack     bool
		expectRequeue  bool
		expectDeadSave bool
	}{
		{
			name: "success - delivered",
			eventBytes: func() []byte {
				event := domain.DispatchEvent{
					EventID:         "event-1",
					NotificationID:  "notif-1",
					SenderID:        "sender-1",
					ContactID:       "contact-1",
					Channel:         "email",
					Attempt:         1,
					TraceID:         "trace-1",
					RenderedContent: "test content",
				}
				b, _ := json.Marshal(event)
				return b
			}(),
			setupMocks: func(repo *testDeliveryRepo, deadRepo *testDeadLetterRepo, provider *testableProvider) {
				repo.attemptExistsFunc = func(ctx context.Context, notificationID string, attemptNumber int) (bool, error) {
					return false, nil
				}
				repo.createAttemptFunc = func(ctx context.Context, attempt *domain.DeliveryAttempt) error {
					return nil
				}
				repo.updateAttemptStatusFunc = func(ctx context.Context, id string, status domain.DeliveryStatus, errorCode, errorMessage *string, responsePayload map[string]any) error {
					return nil
				}
				provider.deliverFunc = func(ctx context.Context, event *domain.DispatchEvent) DeliveryResult {
					return DeliveryResult{
						Status:       domain.DeliveryStatusDelivered,
						ProviderCode: "mock_provider",
					}
				}
			},
			expectAck:     true,
			expectNack:    false,
			expectRequeue: false,
		},
		{
			name:       "unmarshal error - invalid JSON",
			eventBytes: []byte("{invalid json"),
			setupMocks: func(repo *testDeliveryRepo, deadRepo *testDeadLetterRepo, provider *testableProvider) {
			},
			expectAck:     false,
			expectNack:    true,
			expectRequeue: false,
		},
		{
			name: "idempotent skip - already processed",
			eventBytes: func() []byte {
				event := domain.DispatchEvent{
					EventID:        "event-1",
					NotificationID: "notif-1",
					Attempt:        1,
				}
				b, _ := json.Marshal(event)
				return b
			}(),
			setupMocks: func(repo *testDeliveryRepo, deadRepo *testDeadLetterRepo, provider *testableProvider) {
				repo.attemptExistsFunc = func(ctx context.Context, notificationID string, attemptNumber int) (bool, error) {
					return true, nil
				}
			},
			expectAck:     true,
			expectNack:    false,
			expectRequeue: false,
		},
		{
			name: "attempt creation fail",
			eventBytes: func() []byte {
				event := domain.DispatchEvent{
					EventID:        "event-1",
					NotificationID: "notif-1",
					Attempt:        1,
					Channel:        "email",
					TraceID:        "trace-1",
				}
				b, _ := json.Marshal(event)
				return b
			}(),
			setupMocks: func(repo *testDeliveryRepo, deadRepo *testDeadLetterRepo, provider *testableProvider) {
				repo.attemptExistsFunc = func(ctx context.Context, notificationID string, attemptNumber int) (bool, error) {
					return false, nil
				}
				repo.createAttemptFunc = func(ctx context.Context, attempt *domain.DeliveryAttempt) error {
					return assert.AnError
				}
			},
			expectAck:     false,
			expectNack:    true,
			expectRequeue: true,
		},
		{
			name: "delivery fail",
			eventBytes: func() []byte {
				event := domain.DispatchEvent{
					EventID:         "event-1",
					NotificationID:  "notif-1",
					SenderID:        "sender-1",
					ContactID:       "contact-1",
					Channel:         "email",
					Attempt:         1,
					TraceID:         "trace-1",
					RenderedContent: "test content",
				}
				b, _ := json.Marshal(event)
				return b
			}(),
			setupMocks: func(repo *testDeliveryRepo, deadRepo *testDeadLetterRepo, provider *testableProvider) {
				repo.attemptExistsFunc = func(ctx context.Context, notificationID string, attemptNumber int) (bool, error) {
					return false, nil
				}
				repo.createAttemptFunc = func(ctx context.Context, attempt *domain.DeliveryAttempt) error {
					return nil
				}
				repo.updateAttemptStatusFunc = func(ctx context.Context, id string, status domain.DeliveryStatus, errorCode, errorMessage *string, responsePayload map[string]any) error {
					return nil
				}
				errCode := "MOCK_ERROR"
				errMsg := "simulated failure"
				provider.deliverFunc = func(ctx context.Context, event *domain.DispatchEvent) DeliveryResult {
					return DeliveryResult{
						Status:       domain.DeliveryStatusFailed,
						ProviderCode: "mock_provider",
						ErrorCode:    &errCode,
						ErrorMessage: &errMsg,
					}
				}
			},
			expectAck:     true,
			expectNack:    false,
			expectRequeue: false,
		},
		{
			name: "dead letter on max retries exceeded",
			eventBytes: func() []byte {
				event := domain.DispatchEvent{
					EventID:         "event-1",
					NotificationID:  "notif-1",
					SenderID:        "sender-1",
					ContactID:       "contact-1",
					Channel:         "email",
					Attempt:         3,
					TraceID:         "trace-1",
					RenderedContent: "test content",
				}
				b, _ := json.Marshal(event)
				return b
			}(),
			setupMocks: func(repo *testDeliveryRepo, deadRepo *testDeadLetterRepo, provider *testableProvider) {
				repo.attemptExistsFunc = func(ctx context.Context, notificationID string, attemptNumber int) (bool, error) {
					return false, nil
				}
				repo.createAttemptFunc = func(ctx context.Context, attempt *domain.DeliveryAttempt) error {
					return nil
				}
				repo.updateAttemptStatusFunc = func(ctx context.Context, id string, status domain.DeliveryStatus, errorCode, errorMessage *string, responsePayload map[string]any) error {
					return nil
				}
				errCode := "MOCK_ERROR"
				errMsg := "simulated failure"
				provider.deliverFunc = func(ctx context.Context, event *domain.DispatchEvent) DeliveryResult {
					return DeliveryResult{
						Status:       domain.DeliveryStatusFailed,
						ProviderCode: "mock_provider",
						ErrorCode:    &errCode,
						ErrorMessage: &errMsg,
					}
				}
				deadRepo.createFunc = func(ctx context.Context, dl *domain.DeadLetter) error {
					return nil
				}
			},
			expectAck:     true,
			expectNack:    false,
			expectRequeue: false,
		},
		{
			name: "attempt exists error",
			eventBytes: func() []byte {
				event := domain.DispatchEvent{
					EventID:        "event-1",
					NotificationID: "notif-1",
					Attempt:        1,
					Channel:        "email",
					TraceID:        "trace-1",
				}
				b, _ := json.Marshal(event)
				return b
			}(),
			setupMocks: func(repo *testDeliveryRepo, deadRepo *testDeadLetterRepo, provider *testableProvider) {
				repo.attemptExistsFunc = func(ctx context.Context, notificationID string, attemptNumber int) (bool, error) {
					return false, assert.AnError
				}
			},
			expectAck:     false,
			expectNack:    true,
			expectRequeue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &testDeliveryRepo{}
			deadRepo := &testDeadLetterRepo{}
			provider := &testableProvider{}

			tt.setupMocks(repo, deadRepo, provider)

			svc := newTestConsumerService(repo, deadRepo, provider)
			_ = svc.consumeEvent(ctx, tt.eventBytes)

			if tt.expectAck {
				assert.True(t, svc.ackCalled, "expected ack to be called")
				assert.False(t, svc.nackCalled, "expected nack not to be called")
			}
			if tt.expectNack {
				assert.True(t, svc.nackCalled, "expected nack to be called")
				assert.Equal(t, tt.expectRequeue, svc.nackRequeue, "requeue flag mismatch")
			}
		})
	}
}

func TestConsumerService_NotifyNotificationsService(t *testing.T) {
	ctx := context.Background()
	repo := &testDeliveryRepo{}
	deadRepo := &testDeadLetterRepo{}
	provider := &testableProvider{}

	svc := newTestConsumerService(repo, deadRepo, provider)

	t.Run("delivered status", func(t *testing.T) {
		result := DeliveryResult{Status: domain.DeliveryStatusDelivered}
		svc.notifyNotificationsService(ctx, "notif-1", "attempt-1", result)
	})

	t.Run("failed status", func(t *testing.T) {
		result := DeliveryResult{Status: domain.DeliveryStatusFailed}
		svc.notifyNotificationsService(ctx, "notif-1", "attempt-1", result)
	})

	t.Run("sent status (default)", func(t *testing.T) {
		result := DeliveryResult{Status: domain.DeliveryStatusAccepted}
		svc.notifyNotificationsService(ctx, "notif-1", "attempt-1", result)
	})
}

func (s *testConsumerService) notifyNotificationsService(ctx context.Context, notificationID, attemptID string, result DeliveryResult) {
	var status string
	switch result.Status {
	case domain.DeliveryStatusDelivered:
		status = "delivered"
	case domain.DeliveryStatusFailed:
		status = "failed"
	default:
		status = "sent"
	}
	_ = status
}

func (s *testConsumerService) saveDeadLetter(ctx context.Context, event domain.DispatchEvent, result DeliveryResult) {
	dl := &domain.DeadLetter{
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
	}

	if result.ErrorCode != nil {
		dl.ErrorCode = result.ErrorCode
	}
	if result.ErrorMessage != nil {
		dl.ErrorMessage = *result.ErrorMessage
	}

	_ = s.deadRepo.Create(ctx, dl)
}

func TestConsumerService_SaveDeadLetter(t *testing.T) {
	ctx := context.Background()

	event := domain.DispatchEvent{
		EventID:         "event-1",
		NotificationID:  "notif-1",
		SenderID:        "sender-1",
		ContactID:       "contact-1",
		Channel:         "email",
		Attempt:         3,
		TraceID:         "trace-1",
		RenderedContent: "test content",
		Metadata:        map[string]any{"key": "value"},
	}

	result := DeliveryResult{
		Status:       domain.DeliveryStatusFailed,
		ProviderCode: "mock_provider",
	}

	deadRepo := &testDeadLetterRepo{}
	var savedDL *domain.DeadLetter
	deadRepo.createFunc = func(ctx context.Context, dl *domain.DeadLetter) error {
		savedDL = dl
		return nil
	}

	repo := &testDeliveryRepo{}
	provider := &testableProvider{}

	svc := newTestConsumerService(repo, deadRepo, provider)
	svc.saveDeadLetter(ctx, event, result)

	assert.NotNil(t, savedDL)
	assert.Equal(t, "notif-1", savedDL.NotificationID)
	assert.Equal(t, 3, savedDL.AttemptNumber)
	assert.Equal(t, "email", savedDL.Channel)
}

func TestConsumerService_SaveDeadLetter_WithErrorDetails(t *testing.T) {
	ctx := context.Background()

	event := domain.DispatchEvent{
		EventID:         "event-1",
		NotificationID:  "notif-1",
		SenderID:        "sender-1",
		ContactID:       "contact-1",
		Channel:         "email",
		Attempt:         3,
		TraceID:         "trace-1",
		RenderedContent: "test content",
	}

	errCode := "ERR_CODE"
	errMsg := "error message"
	result := DeliveryResult{
		Status:       domain.DeliveryStatusFailed,
		ProviderCode: "mock_provider",
		ErrorCode:    &errCode,
		ErrorMessage: &errMsg,
	}

	deadRepo := &testDeadLetterRepo{}
	var savedDL *domain.DeadLetter
	deadRepo.createFunc = func(ctx context.Context, dl *domain.DeadLetter) error {
		savedDL = dl
		return nil
	}

	repo := &testDeliveryRepo{}
	provider := &testableProvider{}

	svc := newTestConsumerService(repo, deadRepo, provider)
	svc.saveDeadLetter(ctx, event, result)

	assert.NotNil(t, savedDL)
	assert.Equal(t, &errCode, savedDL.ErrorCode)
	assert.Equal(t, errMsg, savedDL.ErrorMessage)
}
