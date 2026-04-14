package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"notification-center/delivery/internal/domain"
)

func TestMockProvider_Deliver(t *testing.T) {
	provider := NewMockProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	event := &domain.DispatchEvent{
		EventID:         "event-1",
		NotificationID:  "notif-1",
		Channel:         "email",
		RenderedContent: "test content",
	}

	result := provider.Deliver(ctx, event)

	assert.Equal(t, "mock_provider", result.ProviderCode)
	assert.True(t, result.Status == domain.DeliveryStatusDelivered || result.Status == domain.DeliveryStatusFailed)

	if result.Status == domain.DeliveryStatusFailed {
		assert.NotNil(t, result.ErrorCode)
		assert.NotNil(t, result.ErrorMessage)
	} else {
		assert.Nil(t, result.ErrorCode)
		assert.Nil(t, result.ErrorMessage)
	}
}

func TestMockProvider_Deliver_ContextCancelled(t *testing.T) {
	provider := NewMockProvider()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := provider.Deliver(ctx, &domain.DispatchEvent{
		EventID:         "event-1",
		NotificationID:  "notif-1",
		Channel:         "email",
		RenderedContent: "test content",
	})

	assert.Equal(t, domain.DeliveryStatusFailed, result.Status)
	assert.Equal(t, "mock_provider", result.ProviderCode)
}

func TestMockProvider_Deliver_MultipleAttempts(t *testing.T) {
	provider := NewMockProvider()
	ctx := context.Background()

	delivered := 0
	failed := 0

	for i := 0; i < 20; i++ {
		result := provider.Deliver(ctx, &domain.DispatchEvent{
			EventID:         "event-1",
			NotificationID:  "notif-1",
			Channel:         "email",
			RenderedContent: "test content",
		})

		if result.Status == domain.DeliveryStatusDelivered {
			delivered++
		} else if result.Status == domain.DeliveryStatusFailed {
			failed++
		}
	}

	assert.Equal(t, 20, delivered+failed)
	assert.True(t, delivered > 0, "expected at least one delivery success")
	assert.True(t, failed > 0, "expected at least one delivery failure")
}

func TestMockProvider_Deliver_ResponseDelay(t *testing.T) {
	provider := NewMockProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	result := provider.Deliver(ctx, &domain.DispatchEvent{
		EventID:         "event-1",
		NotificationID:  "notif-1",
		Channel:         "email",
		RenderedContent: "test content",
	})
	elapsed := time.Since(start)

	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(50))
	assert.LessOrEqual(t, elapsed.Milliseconds(), int64(200))
	assert.Equal(t, "mock_provider", result.ProviderCode)
}
