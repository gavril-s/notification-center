package service

import (
	"context"
	"math/rand"
	"time"

	"notification-center/delivery/internal/domain"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

type DeliveryResult struct {
	Status        domain.DeliveryStatus
	ProviderCode  string
	ErrorCode     *string
	ErrorMessage  *string
	ResponseDelay time.Duration
}

func (p *MockProvider) Deliver(ctx context.Context, event *domain.DispatchEvent) DeliveryResult {
	// Simulate processing delay
	delay := time.Millisecond * time.Duration(50+rand.Intn(100))
	select {
	case <-ctx.Done():
		return DeliveryResult{
			Status:       domain.DeliveryStatusFailed,
			ProviderCode: "mock_provider",
		}
	case <-time.After(delay):
	}

	// Simulate delivery outcome (90% success, 10% failure for demo)
	successRate := 0.9
	if rand.Float64() < successRate {
		return DeliveryResult{
			Status:       domain.DeliveryStatusDelivered,
			ProviderCode: "mock_provider",
		}
	}

	errCode := "MOCK_ERROR"
	errMsg := "simulated delivery failure"
	return DeliveryResult{
		Status:       domain.DeliveryStatusFailed,
		ProviderCode: "mock_provider",
		ErrorCode:    &errCode,
		ErrorMessage: &errMsg,
	}
}
