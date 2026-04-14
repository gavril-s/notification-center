package mocks

import (
	"context"

	"notification-center/delivery/internal/domain"
)

type MockDeliveryRepository struct {
	AttemptExistsFunc       func(ctx context.Context, notificationID string, attemptNumber int) (bool, error)
	CreateAttemptFunc       func(ctx context.Context, attempt *domain.DeliveryAttempt) error
	UpdateAttemptStatusFunc func(ctx context.Context, id string, status domain.DeliveryStatus, errorCode, errorMessage *string, responsePayload map[string]any) error
}

func (m *MockDeliveryRepository) AttemptExists(ctx context.Context, notificationID string, attemptNumber int) (bool, error) {
	if m.AttemptExistsFunc != nil {
		return m.AttemptExistsFunc(ctx, notificationID, attemptNumber)
	}
	return false, nil
}

func (m *MockDeliveryRepository) CreateAttempt(ctx context.Context, attempt *domain.DeliveryAttempt) error {
	if m.CreateAttemptFunc != nil {
		return m.CreateAttemptFunc(ctx, attempt)
	}
	return nil
}

func (m *MockDeliveryRepository) UpdateAttemptStatus(ctx context.Context, id string, status domain.DeliveryStatus, errorCode, errorMessage *string, responsePayload map[string]any) error {
	if m.UpdateAttemptStatusFunc != nil {
		return m.UpdateAttemptStatusFunc(ctx, id, status, errorCode, errorMessage, responsePayload)
	}
	return nil
}
