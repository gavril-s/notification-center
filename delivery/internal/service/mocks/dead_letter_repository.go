package mocks

import (
	"context"

	"notification-center/delivery/internal/domain"
)

type MockDeadLetterRepository struct {
	CreateFunc func(ctx context.Context, dl *domain.DeadLetter) error
}

func (m *MockDeadLetterRepository) Create(ctx context.Context, dl *domain.DeadLetter) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, dl)
	}
	return nil
}
