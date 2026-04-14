package mocks

import (
	"context"

	"notification-center/delivery/internal/domain"
	"notification-center/delivery/internal/service"
)

type MockProvider struct {
	DeliverFunc func(ctx context.Context, event *domain.DispatchEvent) service.DeliveryResult
}

func (m *MockProvider) Deliver(ctx context.Context, event *domain.DispatchEvent) service.DeliveryResult {
	if m.DeliverFunc != nil {
		return m.DeliverFunc(ctx, event)
	}
	return service.DeliveryResult{
		Status:       domain.DeliveryStatusDelivered,
		ProviderCode: "mock_provider",
	}
}
