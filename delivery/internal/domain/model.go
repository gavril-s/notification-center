package domain

import (
	"time"
)

type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusAccepted  DeliveryStatus = "accepted"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
)

type DeliveryAttempt struct {
	ID              string         `json:"id"`
	NotificationID  string         `json:"notification_id"`
	AttemptNumber   int            `json:"attempt_number"`
	Channel         string         `json:"channel"`
	ProviderCode    string         `json:"provider_code"`
	Status          DeliveryStatus `json:"status"`
	ErrorCode       *string        `json:"error_code,omitempty"`
	ErrorMessage    *string        `json:"error_message,omitempty"`
	RequestPayload  map[string]any `json:"request_payload,omitempty"`
	ResponsePayload map[string]any `json:"response_payload,omitempty"`
	StartedAt       time.Time      `json:"started_at"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
	TraceID         *string        `json:"trace_id,omitempty"`
}

type DeadLetter struct {
	ID             string         `json:"id"`
	NotificationID string         `json:"notification_id"`
	AttemptNumber  int            `json:"attempt_number"`
	Channel        string         `json:"channel"`
	Payload        map[string]any `json:"payload"`
	ErrorMessage   string         `json:"error_message"`
	ErrorCode      *string        `json:"error_code,omitempty"`
	ReceivedAt     time.Time      `json:"received_at"`
	ProcessedAt    *time.Time     `json:"processed_at,omitempty"`
	TraceID        *string        `json:"trace_id,omitempty"`
}

type DispatchEvent struct {
	EventID         string         `json:"event_id"`
	NotificationID  string         `json:"notification_id"`
	SenderID        string         `json:"sender_id"`
	ContactID       string         `json:"contact_id"`
	CampaignID      *string        `json:"campaign_id,omitempty"`
	GroupID         *string        `json:"group_id,omitempty"`
	Channel         string         `json:"channel"`
	Subject         *string        `json:"subject,omitempty"`
	RenderedContent string         `json:"rendered_content"`
	UnsubscribeURL  *string        `json:"unsubscribe_url,omitempty"`
	Metadata        map[string]any `json:"metadata"`
	Attempt         int            `json:"attempt"`
	TraceID         string         `json:"trace_id"`
	CreatedAt       string         `json:"created_at"`
}
