package domain

import (
	"time"
)

type NotificationStatus string

const (
	StatusDraft               NotificationStatus = "draft"
	StatusScheduled           NotificationStatus = "scheduled"
	StatusQueued              NotificationStatus = "queued"
	StatusProcessing          NotificationStatus = "processing"
	StatusSent                NotificationStatus = "sent"
	StatusDelivered           NotificationStatus = "delivered"
	StatusFailed              NotificationStatus = "failed"
	StatusCancelled           NotificationStatus = "cancelled"
	StatusSkippedByPreference NotificationStatus = "skipped_by_preference"
)

type Channel string

const (
	ChannelEmail    Channel = "email"
	ChannelSMS      Channel = "sms"
	ChannelTelegram Channel = "telegram"
)

type Notification struct {
	ID              string             `json:"id"`
	SenderID        string             `json:"sender_id"`
	ContactID       string             `json:"contact_id"`
	CampaignID      *string            `json:"campaign_id,omitempty"`
	GroupID         *string            `json:"group_id,omitempty"`
	TemplateID      string             `json:"template_id"`
	Channel         Channel            `json:"channel"`
	Status          NotificationStatus `json:"status"`
	Subject         *string            `json:"subject,omitempty"`
	RenderedContent string             `json:"rendered_content"`
	UnsubscribeURL  *string            `json:"unsubscribe_url,omitempty"`
	IdempotencyKey  string             `json:"idempotency_key"`
	ScheduledAt     *time.Time         `json:"scheduled_at,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	SentAt          *time.Time         `json:"sent_at,omitempty"`
	DeliveredAt     *time.Time         `json:"delivered_at,omitempty"`
	FailedAt        *time.Time         `json:"failed_at,omitempty"`
	FailureReason   *string            `json:"failure_reason,omitempty"`
	AttemptCount    int                `json:"attempt_count"`
	Metadata        map[string]any     `json:"metadata"`
}

type NotificationOutbox struct {
	ID             string         `json:"id"`
	NotificationID string         `json:"notification_id"`
	EventType      string         `json:"event_type"`
	Payload        map[string]any `json:"payload"`
	Attempt        int            `json:"attempt"`
	Status         string         `json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	PublishedAt    *time.Time     `json:"published_at,omitempty"`
	ErrorMessage   *string        `json:"error_message,omitempty"`
}

type NotificationHistory struct {
	ID             string             `json:"id"`
	NotificationID string             `json:"notification_id"`
	ContactID      string             `json:"contact_id"`
	UserID         *string            `json:"user_id,omitempty"`
	Channel        Channel            `json:"channel"`
	Status         NotificationStatus `json:"status"`
	SenderID       string             `json:"sender_id"`
	CampaignID     *string            `json:"campaign_id,omitempty"`
	GroupID        *string            `json:"group_id,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	DeliveredAt    *time.Time         `json:"delivered_at,omitempty"`
}

type SenderAnalyticsDaily struct {
	ID             string    `json:"id"`
	SenderID       string    `json:"sender_id"`
	Date           time.Time `json:"date"`
	Channel        Channel   `json:"channel"`
	SentCount      int       `json:"sent_count"`
	DeliveredCount int       `json:"delivered_count"`
	FailedCount    int       `json:"failed_count"`
	SkippedCount   int       `json:"skipped_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
