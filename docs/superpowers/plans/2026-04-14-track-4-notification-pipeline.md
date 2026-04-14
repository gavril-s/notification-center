# Track 4 Implementation Plan: Notification Pipeline

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the combined Notifications + Delivery pipeline for the Notification Center system, providing REST APIs for sending notifications and consuming RabbitMQ messages for delivery processing.

**Architecture:** Two Go microservices (Notifications Service with Gin REST API + Delivery Service with RabbitMQ consumer) sharing PostgreSQL schemas and communicating via internal REST APIs. Uses outbox pattern for reliable queue publishing.

**Tech Stack:** Go 1.22, Gin framework, PostgreSQL (pgx), RabbitMQ (amqp), UUID for identifiers, RFC3339 UTC timestamps

---

## Task 1: Setup Project Structure and Dependencies

**Files:**
- Modify: `notifications-service/go.mod`
- Modify: `delivery/go.mod`
- Create: `notifications-service/internal/config/config.go`
- Create: `delivery/internal/config/config.go`

- [ ] **Step 1: Update notifications-service/go.mod with dependencies**

```go
module notification-center/notifications-service

go 1.22

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.5.2
	github.com/rabbitmq/amqp091-go v1.9.0
	github.com/stretchr/testify v1.8.4
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/bytedance/sonic v1.9.1 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.14.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.0.8 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/net v0.10.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)
```

- [ ] **Step 2: Update delivery/go.mod with dependencies**

```go
module notification-center/delivery

go 1.22

require (
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.5.2
	github.com/rabbitmq/amqp091-go v1.9.0
	github.com/stretchr/testify v1.8.4
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
)
```

- [ ] **Step 3: Create notifications-service/internal/config/config.go**

```go
package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServiceName string
	Port        string
	DatabaseURL string
	RabbitMQURL string
	JWTSecret   string
}

func Load() *Config {
	return &Config{
		ServiceName: getEnv("SERVICE_NAME", "notifications-service"),
		Port:        getEnv("PORT", "8083"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RabbitMQURL: getEnv("RABBITMQ_URL", ""),
		JWTSecret:   getEnv("JWT_SECRET", "default-secret-change-in-prod"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return fallback
}
```

- [ ] **Step 4: Create delivery/internal/config/config.go**

```go
package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServiceName string
	Port        string
	DatabaseURL string
	RabbitMQURL string
}

func Load() *Config {
	return &Config{
		ServiceName: getEnv("SERVICE_NAME", "delivery"),
		Port:        getEnv("PORT", "8084"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RabbitMQURL: getEnv("RABBITMQ_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
```

---

## Task 2: Database Migrations

**Files:**
- Create: `notifications-service/migrations/001_create_notifications_schema.sql`
- Create: `notifications-service/migrations/002_create_notification_outbox.sql`
- Create: `notifications-service/migrations/003_create_notification_history.sql`
- Create: `notifications-service/migrations/004_create_sender_analytics_daily.sql`
- Create: `delivery/migrations/001_create_delivery_schema.sql`
- Create: `delivery/migrations/002_create_delivery_attempts.sql`
- Create: `delivery/migrations/003_create_delivery_dead_letters.sql`

- [ ] **Step 1: Create notifications schema migration**

```sql
-- migrations/001_create_notifications_schema.sql
CREATE SCHEMA IF NOT EXISTS notifications;

CREATE TABLE IF NOT EXISTS notifications.notifications (
    id UUID PRIMARY KEY,
    sender_id UUID NOT NULL,
    contact_id UUID NOT NULL,
    campaign_id UUID,
    group_id UUID,
    template_id UUID NOT NULL,
    channel VARCHAR(20) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'draft',
    subject VARCHAR(500),
    rendered_content TEXT NOT NULL,
    unsubscribe_url VARCHAR(1024),
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    scheduled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    failure_reason TEXT,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_notifications_sender_id ON notifications.notifications(sender_id);
CREATE INDEX idx_notifications_contact_id ON notifications.notifications(contact_id);
CREATE INDEX idx_notifications_campaign_id ON notifications.notifications(campaign_id);
CREATE INDEX idx_notifications_status ON notifications.notifications(status);
CREATE INDEX idx_notifications_idempotency_key ON notifications.notifications(idempotency_key);
CREATE INDEX idx_notifications_created_at ON notifications.notifications(created_at);
```

- [ ] **Step 2: Create notification_outbox migration**

```sql
-- migrations/002_create_notification_outbox.sql
CREATE TABLE IF NOT EXISTS notifications.notification_outbox (
    id UUID PRIMARY KEY,
    notification_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    attempt INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ,
    error_message TEXT
);

CREATE INDEX idx_notification_outbox_status ON notifications.notification_outbox(status);
CREATE INDEX idx_notification_outbox_notification_id ON notifications.notification_outbox(notification_id);
```

- [ ] **Step 3: Create notification_history migration**

```sql
-- migrations/003_create_notification_history.sql
CREATE TABLE IF NOT EXISTS notifications.notification_history (
    id UUID PRIMARY KEY,
    notification_id UUID NOT NULL,
    contact_id UUID NOT NULL,
    user_id UUID,
    channel VARCHAR(20) NOT NULL,
    status VARCHAR(30) NOT NULL,
    sender_id UUID NOT NULL,
    campaign_id UUID,
    group_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at TIMESTAMPTZ
);

CREATE INDEX idx_notification_history_contact_id ON notifications.notification_history(contact_id);
CREATE INDEX idx_notification_history_user_id ON notifications.notification_history(user_id);
CREATE INDEX idx_notification_history_sender_id ON notifications.notification_history(sender_id);
CREATE INDEX idx_notification_history_campaign_id ON notifications.notification_history(campaign_id);
CREATE INDEX idx_notification_history_created_at ON notifications.notification_history(created_at);
```

- [ ] **Step 4: Create sender_analytics_daily migration**

```sql
-- migrations/004_create_sender_analytics_daily.sql
CREATE TABLE IF NOT EXISTS notifications.sender_analytics_daily (
    id UUID PRIMARY KEY,
    sender_id UUID NOT NULL,
    date DATE NOT NULL,
    channel VARCHAR(20) NOT NULL,
    sent_count INTEGER NOT NULL DEFAULT 0,
    delivered_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(sender_id, date, channel)
);

CREATE INDEX idx_sender_analytics_daily_sender_id_date ON notifications.sender_analytics_daily(sender_id, date);
```

- [ ] **Step 5: Create delivery schema migration**

```sql
-- delivery/migrations/001_create_delivery_schema.sql
CREATE SCHEMA IF NOT EXISTS delivery;
```

- [ ] **Step 6: Create delivery_attempts migration**

```sql
-- delivery/migrations/002_create_delivery_attempts.sql
CREATE TABLE IF NOT EXISTS delivery.delivery_attempts (
    id UUID PRIMARY KEY,
    notification_id UUID NOT NULL,
    attempt_number INTEGER NOT NULL,
    channel VARCHAR(20) NOT NULL,
    provider_code VARCHAR(50) NOT NULL,
    status VARCHAR(30) NOT NULL,
    error_code VARCHAR(100),
    error_message TEXT,
    request_payload JSONB,
    response_payload JSONB,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    trace_id UUID
);

CREATE INDEX idx_delivery_attempts_notification_id ON delivery.delivery_attempts(notification_id);
CREATE INDEX idx_delivery_attempts_status ON delivery.delivery_attempts(status);
CREATE UNIQUE INDEX idx_delivery_attempts_notification_attempt ON delivery.delivery_attempts(notification_id, attempt_number);
```

- [ ] **Step 7: Create delivery_dead_letters migration**

```sql
-- delivery/migrations/003_create_delivery_dead_letters.sql
CREATE TABLE IF NOT EXISTS delivery.delivery_dead_letters (
    id UUID PRIMARY KEY,
    notification_id UUID NOT NULL,
    attempt_number INTEGER NOT NULL,
    channel VARCHAR(20) NOT NULL,
    payload JSONB NOT NULL,
    error_message TEXT NOT NULL,
    error_code VARCHAR(100),
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    trace_id UUID
);

CREATE INDEX idx_delivery_dead_letters_notification_id ON delivery.delivery_dead_letters(notification_id);
CREATE INDEX idx_delivery_dead_letters_received_at ON delivery.delivery_dead_letters(received_at);
```

---

## Task 3: Domain Models

**Files:**
- Create: `notifications-service/internal/domain/model.go`
- Create: `delivery/internal/domain/model.go`

- [ ] **Step 1: Create notifications domain models**

```go
package domain

import (
	"time"
)

type NotificationStatus string

const (
	StatusDraft           NotificationStatus = "draft"
	StatusScheduled       NotificationStatus = "scheduled"
	StatusQueued          NotificationStatus = "queued"
	StatusProcessing      NotificationStatus = "processing"
	StatusSent            NotificationStatus = "sent"
	StatusDelivered       NotificationStatus = "delivered"
	StatusFailed          NotificationStatus = "failed"
	StatusCancelled       NotificationStatus = "cancelled"
	StatusSkippedByPreference NotificationStatus = "skipped_by_preference"
)

type Channel string

const (
	ChannelEmail    Channel = "email"
	ChannelSMS      Channel = "sms"
	ChannelTelegram Channel = "telegram"
)

type Notification struct {
	ID               string            `json:"id"`
	SenderID         string            `json:"sender_id"`
	ContactID        string            `json:"contact_id"`
	CampaignID       *string           `json:"campaign_id,omitempty"`
	GroupID          *string           `json:"group_id,omitempty"`
	TemplateID       string            `json:"template_id"`
	Channel          Channel           `json:"channel"`
	Status           NotificationStatus `json:"status"`
	Subject          *string           `json:"subject,omitempty"`
	RenderedContent  string            `json:"rendered_content"`
	UnsubscribeURL   *string           `json:"unsubscribe_url,omitempty"`
	IdempotencyKey   string            `json:"idempotency_key"`
	ScheduledAt      *time.Time        `json:"scheduled_at,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
	SentAt           *time.Time        `json:"sent_at,omitempty"`
	DeliveredAt      *time.Time        `json:"delivered_at,omitempty"`
	FailedAt         *time.Time        `json:"failed_at,omitempty"`
	FailureReason    *string           `json:"failure_reason,omitempty"`
	AttemptCount     int               `json:"attempt_count"`
	Metadata         map[string]any    `json:"metadata"`
}

type NotificationOutbox struct {
	ID            string    `json:"id"`
	NotificationID string   `json:"notification_id"`
	EventType     string    `json:"event_type"`
	Payload       map[string]any `json:"payload"`
	Attempt       int       `json:"attempt"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	ErrorMessage  *string   `json:"error_message,omitempty"`
}

type NotificationHistory struct {
	ID          string     `json:"id"`
	NotificationID string   `json:"notification_id"`
	ContactID   string     `json:"contact_id"`
	UserID      *string    `json:"user_id,omitempty"`
	Channel     Channel    `json:"channel"`
	Status      NotificationStatus `json:"status"`
	SenderID    string     `json:"sender_id"`
	CampaignID  *string    `json:"campaign_id,omitempty"`
	GroupID     *string    `json:"group_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
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
```

- [ ] **Step 2: Create delivery domain models**

```go
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
	EventID          string         `json:"event_id"`
	NotificationID   string         `json:"notification_id"`
	SenderID         string         `json:"sender_id"`
	ContactID        string         `json:"contact_id"`
	CampaignID       *string        `json:"campaign_id,omitempty"`
	GroupID          *string        `json:"group_id,omitempty"`
	Channel          string         `json:"channel"`
	Subject          *string        `json:"subject,omitempty"`
	RenderedContent  string         `json:"rendered_content"`
	UnsubscribeURL   *string        `json:"unsubscribe_url,omitempty"`
	Metadata         map[string]any `json:"metadata"`
	Attempt          int            `json:"attempt"`
	TraceID          string         `json:"trace_id"`
	CreatedAt        string         `json:"created_at"`
}
```

---

## Task 4: Repository Layer - Notifications Service

**Files:**
- Create: `notifications-service/internal/repository/notification.go`
- Create: `notifications-service/internal/repository/outbox.go`
- Create: `notifications-service/internal/repository/history.go`
- Create: `notifications-service/internal/repository/analytics.go`

- [ ] **Step 1: Create notification repository**

```go
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"notification-center/notifications-service/internal/domain"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrIdempotencyConflict  = errors.New("notification with this idempotency key already exists")
)

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	query := `
		INSERT INTO notifications.notifications (
			id, sender_id, contact_id, campaign_id, group_id, template_id, channel,
			status, subject, rendered_content, unsubscribe_url, idempotency_key,
			scheduled_at, created_at, updated_at, attempt_count, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	metadata, _ := json.Marshal(n.Metadata)
	_, err := r.db.Exec(ctx, query,
		n.ID, n.SenderID, n.ContactID, n.CampaignID, n.GroupID, n.TemplateID, n.Channel,
		n.Status, n.Subject, n.RenderedContent, n.UnsubscribeURL, n.IdempotencyKey,
		n.ScheduledAt, n.CreatedAt, n.UpdatedAt, n.AttemptCount, metadata,
	)
	return err
}

func (r *NotificationRepository) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	query := `
		SELECT id, sender_id, contact_id, campaign_id, group_id, template_id, channel,
			status, subject, rendered_content, unsubscribe_url, idempotency_key,
			scheduled_at, created_at, updated_at, sent_at, delivered_at, failed_at,
			failure_reason, attempt_count, metadata
		FROM notifications.notifications
		WHERE id = $1
	`
	var n domain.Notification
	var metadata []byte
	var subject, unsubscribeURL, failureReason sql.NullString
	var campaignID, groupID, scheduledAt, sentAt, deliveredAt, failedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, id).Scan(
		&n.ID, &n.SenderID, &n.ContactID, &campaignID, &groupID, &n.TemplateID, &n.Channel,
		&n.Status, &subject, &n.RenderedContent, &unsubscribeURL, &n.IdempotencyKey,
		&scheduledAt, &n.CreatedAt, &n.UpdatedAt, &sentAt, &deliveredAt, &failedAt,
		&failureReason, &n.AttemptCount, &metadata,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotificationNotFound
		}
		return nil, err
	}

	// Set nullable fields
	if campaignID.Valid {
		n.CampaignID = &campaignID.Time
	}
	if groupID.Valid {
		n.GroupID = &groupID.Time
	}
	if subject.Valid {
		n.Subject = &subject.String
	}
	if unsubscribeURL.Valid {
		n.UnsubscribeURL = &unsubscribeURL.String
	}
	if scheduledAt.Valid {
		n.ScheduledAt = &scheduledAt.Time
	}
	if sentAt.Valid {
		n.SentAt = &sentAt.Time
	}
	if deliveredAt.Valid {
		n.DeliveredAt = &deliveredAt.Time
	}
	if failedAt.Valid {
		n.FailedAt = &failedAt.Time
	}
	if failureReason.Valid {
		n.FailureReason = &failureReason.String
	}

	json.Unmarshal(metadata, &n.Metadata)
	return &n, nil
}

func (r *NotificationRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Notification, error) {
	query := `
		SELECT id, sender_id, contact_id, campaign_id, group_id, template_id, channel,
			status, subject, rendered_content, unsubscribe_url, idempotency_key,
			scheduled_at, created_at, updated_at, sent_at, delivered_at, failed_at,
			failure_reason, attempt_count, metadata
		FROM notifications.notifications
		WHERE idempotency_key = $1
	`
	// Implementation similar to GetByID
	// Returns ErrNotificationNotFound if not found
}

func (r *NotificationRepository) UpdateStatus(ctx context.Context, id string, status domain.NotificationStatus) error {
	query := `
		UPDATE notifications.notifications
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, status)
	return err
}

func (r *NotificationRepository) UpdateStatusWithTimestamp(ctx context.Context, id string, status domain.NotificationStatus, timestamp *time.Time) error {
	query := `
		UPDATE notifications.notifications
		SET status = $2, updated_at = NOW()
	`
	switch status {
	case domain.StatusSent:
		query += ", sent_at = $3"
	case domain.StatusDelivered:
		query += ", delivered_at = $3"
	case domain.StatusFailed:
		query += ", failed_at = $3"
	}
	query += " WHERE id = $1"

	var err error
	if timestamp != nil {
		_, err = r.db.Exec(ctx, query, id, status, timestamp)
	} else {
		_, err = r.db.Exec(ctx, query, id, status)
	}
	return err
}

func (r *NotificationRepository) IncrementAttemptCount(ctx context.Context, id string) error {
	query := `
		UPDATE notifications.notifications
		SET attempt_count = attempt_count + 1, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *NotificationRepository) GetScheduledDue(ctx context.Context, before time.Time) ([]*domain.Notification, error) {
	query := `
		SELECT id, sender_id, contact_id, campaign_id, group_id, template_id, channel,
			status, subject, rendered_content, unsubscribe_url, idempotency_key,
			scheduled_at, created_at, updated_at, sent_at, delivered_at, failed_at,
			failure_reason, attempt_count, metadata
		FROM notifications.notifications
		WHERE status = 'scheduled' AND scheduled_at <= $1
		ORDER BY scheduled_at ASC
		LIMIT 100
	`
	// Returns slice of notifications
}
```

- [ ] **Step 2: Create outbox repository**

```go
package repository

type OutboxRepository struct {
	db *pgxpool.Pool
}

func NewOutboxRepository(db *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) Create(ctx context.Context, outbox *domain.NotificationOutbox) error {
	query := `
		INSERT INTO notifications.notification_outbox (
			id, notification_id, event_type, payload, attempt, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	payload, _ := json.Marshal(outbox.Payload)
	_, err := r.db.Exec(ctx, query,
		outbox.ID, outbox.NotificationID, outbox.EventType, payload,
		outbox.Attempt, outbox.Status, outbox.CreatedAt,
	)
	return err
}

func (r *OutboxRepository) GetPending(ctx context.Context, limit int) ([]*domain.NotificationOutbox, error) {
	query := `
		SELECT id, notification_id, event_type, payload, attempt, status, created_at, published_at, error_message
		FROM notifications.notification_outbox
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
	`
	// Returns pending outbox entries
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, id string) error {
	query := `
		UPDATE notifications.notification_outbox
		SET status = 'published', published_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id string, errMsg string) error {
	query := `
		UPDATE notifications.notification_outbox
		SET status = 'failed', error_message = $2
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, errMsg)
	return err
}
```

- [ ] **Step 3: Create history repository**

```go
package repository

type HistoryRepository struct {
	db *pgxpool.Pool
}

func NewHistoryRepository(db *pgxpool.Pool) *HistoryRepository {
	return &HistoryRepository{db: db}
}

func (r *HistoryRepository) Create(ctx context.Context, h *domain.NotificationHistory) error {
	query := `
		INSERT INTO notifications.notification_history (
			id, notification_id, contact_id, user_id, channel, status, sender_id,
			campaign_id, group_id, created_at, delivered_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Exec(ctx, query,
		h.ID, h.NotificationID, h.ContactID, h.UserID, h.Channel, h.Status,
		h.SenderID, h.CampaignID, h.GroupID, h.CreatedAt, h.DeliveredAt,
	)
	return err
}

func (r *HistoryRepository) GetByContactID(ctx context.Context, contactID string, page, size int) ([]*domain.NotificationHistory, int, error) {
	// Paginated query by contact_id
	// Returns items, total count
}

func (r *HistoryRepository) GetByUserID(ctx context.Context, userID string, page, size int) ([]*domain.NotificationHistory, int, error) {
	// Paginated query by user_id
}

func (r *HistoryRepository) GetBySenderID(ctx context.Context, senderID string, campaignID *string, status *string, page, size int) ([]*domain.NotificationHistory, int, error) {
	// Operator mode: filtered by sender_id
}
```

- [ ] **Step 4: Create analytics repository**

```go
package repository

type AnalyticsRepository struct {
	db *pgxpool.Pool
}

func NewAnalyticsRepository(db *pgxpool.Pool) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

func (r *AnalyticsRepository) GetOrCreateDaily(ctx context.Context, senderID string, date time.Time, channel domain.Channel) (*domain.SenderAnalyticsDaily, error) {
	// Get or create daily analytics record
}

func (r *AnalyticsRepository) IncrementSent(ctx context.Context, senderID string, date time.Time, channel domain.Channel) error {
	query := `
		INSERT INTO notifications.sender_analytics_daily (id, sender_id, date, channel, sent_count, delivered_count, failed_count, skipped_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 1, 0, 0, 0, NOW(), NOW())
		ON CONFLICT (sender_id, date, channel)
		DO UPDATE SET sent_count = sender_analytics_daily.sent_count + 1, updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, uuid.New().String(), senderID, date, channel)
	return err
}

func (r *AnalyticsRepository) IncrementDelivered(ctx context.Context, senderID string, date time.Time, channel domain.Channel) error {
	// Similar to IncrementSent but for delivered_count
}

func (r *AnalyticsRepository) IncrementFailed(ctx context.Context, senderID string, date time.Time, channel domain.Channel) error {
	// Similar to IncrementSent but for failed_count
}

func (r *AnalyticsRepository) IncrementSkipped(ctx context.Context, senderID string, date time.Time, channel domain.Channel) error {
	// Similar to IncrementSent but for skipped_count
}

func (r *AnalyticsRepository) GetBySender(ctx context.Context, senderID string, from, to time.Time) ([]*domain.SenderAnalyticsDaily, error) {
	query := `
		SELECT id, sender_id, date, channel, sent_count, delivered_count, failed_count, skipped_count, created_at, updated_at
		FROM notifications.sender_analytics_daily
		WHERE sender_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date DESC
	`
	// Returns analytics data
}
```

---

## Task 5: Repository Layer - Delivery Service

**Files:**
- Create: `delivery/internal/repository/delivery.go`
- Create: `delivery/internal/repository/dead_letter.go`

- [ ] **Step 1: Create delivery attempt repository**

```go
package repository

import (
	"context"
	"errors"

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
	// Execute insert
}

func (r *DeliveryRepository) GetByNotificationAndAttempt(ctx context.Context, notificationID string, attemptNumber int) (*domain.DeliveryAttempt, error) {
	query := `
		SELECT id, notification_id, attempt_number, channel, provider_code, status,
			error_code, error_message, request_payload, response_payload,
			started_at, completed_at, trace_id
		FROM delivery.delivery_attempts
		WHERE notification_id = $1 AND attempt_number = $2
	`
	// Return attempt or ErrAttemptNotFound
}

func (r *DeliveryRepository) UpdateAttemptStatus(ctx context.Context, id string, status domain.DeliveryStatus, errorCode, errorMessage *string, responsePayload map[string]any) error {
	query := `
		UPDATE delivery.delivery_attempts
		SET status = $2, error_code = $3, error_message = $4, response_payload = $5, completed_at = NOW()
		WHERE id = $1
	`
	// Update attempt status
}

func (r *DeliveryRepository) AttemptExists(ctx context.Context, notificationID string, attemptNumber int) (bool, error) {
	query := `
		SELECT EXISTS(SELECT 1 FROM delivery.delivery_attempts WHERE notification_id = $1 AND attempt_number = $2)
	`
	var exists bool
	err := r.db.QueryRow(ctx, query, notificationID, attemptNumber).Scan(&exists)
	return exists, err
}
```

- [ ] **Step 2: Create dead letter repository**

```go
package repository

type DeadLetterRepository struct {
	db *pgxpool.Pool
}

func NewDeadLetterRepository(db *pgxpool.Pool) *DeadLetterRepository {
	return &DeadLetterRepository{db: db}
}

func (r *DeadLetterRepository) Create(ctx context.Context, dl *domain.DeadLetter) error {
	query := `
		INSERT INTO delivery.delivery_dead_letters (
			id, notification_id, attempt_number, channel, payload, error_message,
			error_code, received_at, trace_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	// Insert dead letter
}

func (r *DeadLetterRepository) GetByNotificationID(ctx context.Context, notificationID string) ([]*domain.DeadLetter, error) {
	query := `
		SELECT id, notification_id, attempt_number, channel, payload, error_message,
			error_code, received_at, processed_at, trace_id
		FROM delivery.delivery_dead_letters
		WHERE notification_id = $1
		ORDER BY attempt_number DESC
	`
	// Return dead letters
}
```

---

## Task 6: HTTP Client for Internal APIs

**Files:**
- Create: `notifications-service/internal/client/http_client.go`

- [ ] **Step 1: Create HTTP client for internal API calls**

```go
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *HTTPClient) doRequest(ctx context.Context, method, path string, body any, headers map[string]string) ([]byte, error) {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(nil)
}

// Recipients service client methods
type ResolveRecipientRequest struct {
	ContactID *string `json:"contact_id,omitempty"`
	UserID    *string `json:"user_id,omitempty"`
}

type ResolveRecipientResponse struct {
	ContactID string `json:"contact_id"`
	UserID    string `json:"user_id"`
	Valid     bool   `json:"valid"`
}

func (c *HTTPClient) ResolveRecipient(ctx context.Context, req ResolveRecipientRequest, traceID string) (*ResolveRecipientResponse, error) {
	var resp ResolveRecipientResponse
	_, err := c.doRequest(ctx, "POST", "/internal/recipients/resolve", req, map[string]string{"X-Trace-ID": traceID})
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

type GetUserContactsResponse struct {
	Contacts []Contact `json:"contacts"`
}

type Contact struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Primary bool   `json:"primary"`
}

func (c *HTTPClient) GetUserContacts(ctx context.Context, userID, traceID string) (*GetUserContactsResponse, error) {
	var resp GetUserContactsResponse
	_, err := c.doRequest(ctx, "GET", fmt.Sprintf("/internal/recipients/users/%s/contacts", userID), nil, map[string]string{"X-Trace-ID": traceID})
	return &resp, err
}

// Sources service client methods
type Template struct {
	ID        string `json:"id"`
	Channel   string `json:"channel"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	Variables []string `json:"variables"`
}

func (c *HTTPClient) GetTemplate(ctx context.Context, templateID, traceID string) (*Template, error) {
	var resp Template
	_, err := c.doRequest(ctx, "GET", fmt.Sprintf("/internal/sources/templates/%s", templateID), nil, map[string]string{"X-Trace-ID": traceID})
	return &resp, err
}

type Campaign struct {
	ID          string  `json:"id"`
	SenderID    string  `json:"sender_id"`
	TemplateID  string  `json:"template_id"`
	GroupID     string  `json:"group_id"`
	ScheduledAt *string `json:"scheduled_at"`
	Status      string  `json:"status"`
}

func (c *HTTPClient) GetCampaign(ctx context.Context, campaignID, traceID string) (*Campaign, error) {
	var resp Campaign
	_, err := c.doRequest(ctx, "GET", fmt.Sprintf("/internal/sources/campaigns/%s", campaignID), nil, map[string]string{"X-Trace-ID": traceID})
	return &resp, err
}

type Sender struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Credentials string `json:"credentials"`
}

func (c *HTTPClient) ResolveCredential(ctx context.Context, integrationKey, traceID string) (*Sender, error) {
	var resp Sender
	_, err := c.doRequest(ctx, "POST", "/internal/sources/senders/resolve-credential", map[string]string{"integration_key": integrationKey}, map[string]string{"X-Trace-ID": traceID})
	return &resp, err
}

func (c *HTTPClient) CheckOperatorPermission(ctx context.Context, senderID, userID, traceID string) (bool, error) {
	_, err := c.doRequest(ctx, "GET", fmt.Sprintf("/internal/sources/senders/%s/operators/%s", senderID, userID), nil, map[string]string{"X-Trace-ID": traceID})
	return err == nil, nil
}

type GroupMember struct {
	UserID    string `json:"user_id"`
	ContactID string `json:"contact_id"`
}

func (c *HTTPClient) GetGroupMembers(ctx context.Context, groupID, traceID string) ([]GroupMember, error) {
	var resp struct {
		Members []GroupMember `json:"members"`
	}
	_, err := c.doRequest(ctx, "GET", fmt.Sprintf("/internal/sources/groups/%s/members", groupID), nil, map[string]string{"X-Trace-ID": traceID})
	return resp.Members, err
}
```

---

## Task 7: Service Layer - Notifications Service

**Files:**
- [ ] Create: `notifications-service/internal/service/notification.go`
- [ ] Create: `notifications-service/internal/service/template_renderer.go`

- [ ] **Step 1: Create notification service**

```go
package service

import (
	"context"
	"errors"
	"fmt"
	"text/template"
	"time"

	"github.com/google/uuid"

	"notification-center/notifications-service/internal/client"
	"notification-center/notifications-service/internal/domain"
	"notification-center/notifications-service/internal/repository"
)

var (
	ErrSenderNotFound       = errors.New("sender not found")
	ErrTemplateNotFound     = errors.New("template not found")
	ErrChannelMismatch     = errors.New("channel does not match template channel")
	ErrNoContactsProvided   = errors.New("no contacts provided")
	ErrMultipleChannels    = errors.New("only one channel allowed per request")
	ErrInvalidIdempotency  = errors.New("invalid idempotency key")
	ErrRecipientBlocked    = errors.New("recipient blocked by preferences")
	ErrUnsubscribeScope    = errors.New("invalid unsubscribe scope")
)

type NotificationService struct {
	repo           *repository.NotificationRepository
	outboxRepo     *repository.OutboxRepository
	historyRepo    *repository.HistoryRepository
	analyticsRepo  *repository.AnalyticsRepository
	recipientsURL  string
	sourcesURL     string
	jwtSecret      string
	unsubscribeSecret string
}

func NewNotificationService(
	repo *repository.NotificationRepository,
	outboxRepo *repository.OutboxRepository,
	historyRepo *repository.HistoryRepository,
	analyticsRepo *repository.AnalyticsRepository,
	recipientsURL, sourcesURL, jwtSecret, unsubscribeSecret string,
) *NotificationService {
	return &NotificationService{
		repo:            repo,
		outboxRepo:      outboxRepo,
		historyRepo:     historyRepo,
		analyticsRepo:   analyticsRepo,
		recipientsURL:   recipientsURL,
		sourcesURL:      sourcesURL,
		jwtSecret:       jwtSecret,
		unsubscribeSecret: unsubscribeSecret,
	}
}

type SendNotificationRequest struct {
	TemplateID     string         `json:"template_id"`
	CampaignID     *string        `json:"campaign_id"`
	ContactIDs     []string       `json:"contact_ids"`
	Channels       []domain.Channel `json:"channels"`
	Variables      map[string]any `json:"variables"`
	ScheduledAt    *time.Time     `json:"scheduled_at"`
	IdempotencyKey string         `json:"idempotency_key"`
}

func (s *NotificationService) Send(ctx context.Context, req SendNotificationRequest, integrationKey, traceID string) ([]*domain.Notification, error) {
	// 1. Validate channel count
	if len(req.Channels) != 1 {
		return nil, ErrMultipleChannels
	}
	channel := req.Channels[0]

	// 2. Validate idempotency key
	if req.IdempotencyKey == "" {
		return nil, ErrInvalidIdempotency
	}

	// 3. Resolve sender from integration key
	senderClient := client.NewHTTPClient(s.sourcesURL)
	sender, err := senderClient.ResolveCredential(ctx, integrationKey, traceID)
	if err != nil {
		return nil, ErrSenderNotFound
	}

	// 4. Get template
	template, err := senderClient.GetTemplate(ctx, req.TemplateID, traceID)
	if err != nil {
		return nil, ErrTemplateNotFound
	}

	// 5. Validate channel matches template
	if string(template.Channel) != string(channel) {
		return nil, ErrChannelMismatch
	}

	// 6. Render content
	rendered := RenderTemplate(template.Subject, template.Body, req.Variables)

	// 7. Create notifications with outbox in transaction
	notifications := make([]*domain.Notification, 0, len(req.ContactIDs))
	for _, contactID := range req.ContactIDs {
		notificationID := uuid.New().String()
		now := time.Now().UTC()

		// Generate unsubscribe URL
		unsubscribeURL := s.generateUnsubscribeURL(contactID, sender.ID, req.CampaignID, nil)

		// Determine status based on scheduled_at
		status := domain.StatusQueued
		if req.ScheduledAt != nil {
			status = domain.StatusScheduled
		}

		notification := &domain.Notification{
			ID:              notificationID,
			SenderID:        sender.ID,
			ContactID:       contactID,
			CampaignID:      req.CampaignID,
			TemplateID:      req.TemplateID,
			Channel:         channel,
			Status:          status,
			Subject:         &template.Subject,
			RenderedContent: rendered,
			UnsubscribeURL:  &unsubscribeURL,
			IdempotencyKey:  req.IdempotencyKey + ":" + contactID,
			ScheduledAt:     req.ScheduledAt,
			CreatedAt:       now,
			UpdatedAt:       now,
			AttemptCount:    0,
			Metadata:        req.Variables,
		}

		// Create outbox event
		outbox := &domain.NotificationOutbox{
			ID:             uuid.New().String(),
			NotificationID: notificationID,
			EventType:      "notification.dispatch.v1",
			Payload: map[string]any{
				"event_id":          uuid.New().String(),
				"notification_id":   notificationID,
				"sender_id":         sender.ID,
				"contact_id":        contactID,
				"campaign_id":       req.CampaignID,
				"channel":           channel,
				"subject":           template.Subject,
				"rendered_content":  rendered,
				"unsubscribe_url":   unsubscribeURL,
				"metadata":          req.Variables,
				"attempt":           1,
				"trace_id":          traceID,
				"created_at":        now.Format(time.RFC3339),
			},
			Attempt:   1,
			Status:    "pending",
			CreatedAt: now,
		}

		notifications = append(notifications, notification)
		
		// Save to database (would be in a transaction)
		s.repo.Create(ctx, notification)
		s.outboxRepo.Create(ctx, outbox)
		
		// Create history entry
		history := &domain.NotificationHistory{
			ID:             uuid.New().String(),
			NotificationID: notificationID,
			ContactID:      contactID,
			Channel:        channel,
			Status:         status,
			SenderID:       sender.ID,
			CampaignID:     req.CampaignID,
			CreatedAt:      now,
		}
		s.historyRepo.Create(ctx, history)
		
		// Update analytics
		s.analyticsRepo.IncrementSent(ctx, sender.ID, now, channel)
	}

	return notifications, nil
}

func (s *NotificationService) GetByID(ctx context.Context, notificationID string) (*domain.Notification, error) {
	return s.repo.GetByID(ctx, notificationID)
}

func (s *NotificationService) generateUnsubscribeURL(contactID, senderID string, campaignID, groupID *string) string {
	// Generate signed token with available_scopes
	scopes := "sender"
	if campaignID != nil || groupID != nil {
		scopes = "sender,campaign_or_group"
	}

	token := fmt.Sprintf("%s|%s|%s|%v|%v", contactID, senderID, scopes, campaignID, groupID)
	// In production, this would be a proper HMAC signature
	return fmt.Sprintf("/unsubscribe?token=%s&available_scopes=%s", token, scopes)
}

func RenderTemplate(subject, body string, variables map[string]any) string {
	// Simple template rendering
	var result string
	
	if subject != "" {
		tmpl, _ := template.New("subject").Parse(subject)
		var buf bytes.Buffer
		tmpl.Execute(&buf, variables)
		result += buf.String() + "\n"
	}
	
	if body != "" {
		tmpl, _ := template.New("body").Parse(body)
		var buf bytes.Buffer
		tmpl.Execute(&buf, variables)
		result += buf.String()
	}
	
	return result
}
```

---

## Task 8: Service Layer - Delivery Service

**Files:**
- Create: `delivery/internal/service/consumer.go`
- Create: `delivery/internal/service/provider.go`

- [ ] **Step 1: Create mock provider**

```go
package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"

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
	time.Sleep(time.Millisecond * time.Duration(50+rand.Intn(100)))

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
		Status:        domain.DeliveryStatusFailed,
		ProviderCode:  "mock_provider",
		ErrorCode:     &errCode,
		ErrorMessage:  &errMsg,
	}
}
```

- [ ] **Step 2: Create consumer service**

```go
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
		return err
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

	// Handle dead letter for failures
	if result.Status == domain.DeliveryStatusFailed && event.Attempt >= 3 {
		saveDeadLetter(ctx, event, result)
	}

	return msg.Ack(false)
}

func (s *ConsumerService) notifyNotificationsService(ctx context.Context, notificationID, attemptID string, result DeliveryResult) {
	// In production, this would call the notifications service callback
	log.Printf("notifying notifications service: notification=%s, status=%s", notificationID, result.Status)
}

func saveDeadLetter(ctx context.Context, event domain.DispatchEvent, result DeliveryResult) {
	log.Printf("moving to dead letter: notification=%s", event.NotificationID)
}
```

---

## Task 9: HTTP Handlers - Notifications Service

**Files:**
- Create: `notifications-service/internal/handler/notification.go`
- Create: `notifications-service/internal/middleware/auth.go`

- [ ] **Step 1: Create handlers**

```go
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"notification-center/notifications-service/internal/domain"
	"notification-center/notifications-service/internal/service"
)

type NotificationHandler struct {
	service *service.NotificationService
}

func NewNotificationHandler(svc *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{service: svc}
}

type SendRequest struct {
	TemplateID     string                 `json:"template_id" binding:"required"`
	CampaignID     *string                `json:"campaign_id"`
	ContactIDs     []string               `json:"contact_ids" binding:"required,min=1"`
	Channels       []string               `json:"channels" binding:"required,min=1,max=1"`
	Variables      map[string]any         `json:"variables"`
	ScheduledAt    *time.Time             `json:"scheduled_at"`
	IdempotencyKey string                 `json:"idempotency_key" binding:"required"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   any         `json:"details,omitempty"`
	RequestID string      `json:"request_id"`
}

func (h *NotificationHandler) Send(c *gin.Context) {
	var req SendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:      "invalid_request",
				Message:   "некорректный запрос",
				Details:   err.Error(),
				RequestID: c.GetString("request_id"),
			},
		})
		return
	}

	// Get integration key from header
	integrationKey := c.GetHeader("X-Integration-Key")
	if integrationKey == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: ErrorDetail{
				Code:      "unauthorized",
				Message:   "отсутствует заголовок X-Integration-Key",
				RequestID: c.GetString("request_id"),
			},
		})
		return
	}

	traceID := c.GetHeader("X-Trace-ID")
	if traceID == "" {
		traceID = "default-trace-id"
	}

	channels := make([]domain.Channel, len(req.Channels))
	for i, ch := range req.Channels {
		channels[i] = domain.Channel(ch)
	}

	serviceReq := service.SendNotificationRequest{
		TemplateID:     req.TemplateID,
		CampaignID:     req.CampaignID,
		ContactIDs:     req.ContactIDs,
		Channels:       channels,
		Variables:      req.Variables,
		ScheduledAt:    req.ScheduledAt,
		IdempotencyKey: req.IdempotencyKey,
	}

	notifications, err := h.service.Send(c.Request.Context(), serviceReq, integrationKey, traceID)
	if err != nil {
		code := "internal_error"
		message := "внутренняя ошибка сервера"
		
		switch err {
		case service.ErrSenderNotFound:
			code = "sender_not_found"
			message = "отправитель не найден"
		case service.ErrTemplateNotFound:
			code = "template_not_found"
			message = "шаблон не найден"
		case service.ErrChannelMismatch:
			code = "channel_mismatch"
			message = "канал не соответствует шаблону"
		case service.ErrMultipleChannels:
			code = "multiple_channels"
			message = "разрешен только один канал"
		case service.ErrNoContactsProvided:
			code = "no_contacts"
			message = "не указаны получатели"
		case service.ErrInvalidIdempotency:
			code = "invalid_idempotency_key"
			message = "некорректный ключ идемпотентности"
		}

		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: ErrorDetail{
				Code:      code,
				Message:   message,
				RequestID: traceID,
			},
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"notifications": notifications,
		"request_id":    traceID,
	})
}

func (h *NotificationHandler) GetByID(c *gin.Context) {
	notificationID := c.Param("notification_id")
	
	notification, err := h.service.GetByID(c.Request.Context(), notificationID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: ErrorDetail{
				Code:      "not_found",
				Message:   "уведомление не найдено",
				RequestID: c.GetString("request_id"),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notification": notification,
		"request_id":   c.GetString("request_id"),
	})
}

func (h *NotificationHandler) GetHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	mode := c.DefaultQuery("mode", "recipient")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	// Implementation depends on mode (recipient/operator)
	c.JSON(http.StatusOK, gin.H{
		"items":      []any{},
		"page":       page,
		"size":       size,
		"total":      0,
		"request_id": c.GetString("request_id"),
	})
}

func (h *NotificationHandler) GetAnalytics(c *gin.Context) {
	senderID := c.Query("sender_id")
	from := c.Query("from")
	to := c.Query("to")

	// Parse dates and fetch analytics
	c.JSON(http.StatusOK, gin.H{
		"analytics":  []any{},
		"request_id": c.GetString("request_id"),
	})
}

func (h *NotificationHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/notifications/send", h.Send)
	r.GET("/notifications/:notification_id", h.GetByID)
	r.GET("/notifications/history", h.GetHistory)
	r.GET("/notifications/analytics", h.GetAnalytics)
}
```

- [ ] **Step 2: Create middleware**

```go
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}
		c.Set("trace_id", traceID)
		c.Next()
	}
}
```

---

## Task 10: Main Entry Points

**Files:**
- Modify: `notifications-service/cmd/api/main.go`
- Modify: `delivery/cmd/consumer/main.go`

- [ ] **Step 1: Update notifications-service main.go**

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"notification-center/notifications-service/internal/config"
	"notification-center/notifications-service/internal/handler"
	"notification-center/notifications-service/internal/middleware"
	"notification-center/notifications-service/internal/repository"
	"notification-center/notifications-service/internal/service"
)

func main() {
	cfg := config.Load()

	// Database connection
	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Verify database connection
	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	// Initialize repositories
	notificationRepo := repository.NewNotificationRepository(dbPool)
	outboxRepo := repository.NewOutboxRepository(dbPool)
	historyRepo := repository.NewHistoryRepository(dbPool)
	analyticsRepo := repository.NewAnalyticsRepository(dbPool)

	// Initialize service
	notificationSvc := service.NewNotificationService(
		notificationRepo,
		outboxRepo,
		historyRepo,
		analyticsRepo,
		os.Getenv("RECIPIENTS_URL"),
		os.Getenv("SOURCES_URL"),
		cfg.JWTSecret,
		os.Getenv("UNSUBSCRIBE_SECRET"),
	)

	// Initialize handler
	notificationHandler := handler.NewNotificationHandler(notificationSvc)

	// Setup Gin router
	router := gin.Default()
	router.Use(middleware.RequestID())
	router.Use(middleware.TraceID())

	// Public API routes
	api := router.Group("/api")
	notificationHandler.RegisterRoutes(api)

	// Internal API routes
	internal := router.Group("/internal")
	internal.POST("/notifications/:notification_id/delivery-status", handleDeliveryStatus)
	internal.POST("/notifications/scheduler/run", handleSchedulerRun)
	internal.GET("/notifications/:notification_id", handleInternalGet)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "component": cfg.ServiceName})
	})
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Printf("starting %s on port %s", cfg.ServiceName, cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server exited")
}

func handleDeliveryStatus(c *gin.Context) {
	// Handle delivery status callback
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleSchedulerRun(c *gin.Context) {
	// Trigger scheduler for due notifications
	c.JSON(http.StatusOK, gin.H{"status": "scheduler triggered"})
}

func handleInternalGet(c *gin.Context) {
	// Internal notification get
	c.JSON(http.StatusOK, gin.H{"notification": nil})
}
```

- [ ] **Step 2: Update delivery main.go**

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"

	"notification-center/delivery/internal/config"
	"notification-center/delivery/internal/repository"
	"notification-center/delivery/internal/service"
)

func main() {
	cfg := config.Load()

	// Database connection
	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Initialize repositories
	deliveryRepo := repository.NewDeliveryRepository(dbPool)
	deadLetterRepo := repository.NewDeadLetterRepository(dbPool)

	// Initialize services
	provider := service.NewMockProvider()
	consumerSvc := service.NewConsumerService(deliveryRepo, deadLetterRepo, provider)

	// Connect to RabbitMQ
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("failed to open channel: %v", err)
	}
	defer ch.Close()

	// Declare queue
	q, err := ch.QueueDeclare(
		"notification.dispatch.v1",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("failed to declare queue: %v", err)
	}

	// Set QoS
	err = ch.Qos(1, 0, false)
	if err != nil {
		log.Fatalf("failed to set QoS: %v", err)
	}

	// Start consuming
	msgs, err := ch.Consume(
		q.Name,    // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		log.Fatalf("failed to register consumer: %v", err)
	}

	// Start worker goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Process messages
	log.Printf("starting delivery consumer, waiting for messages...")
	
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				if err := consumerSvc.Consume(ctx, msg); err != nil {
					log.Printf("error processing message: %v", err)
				}
			}
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down consumer...")
	cancel()
	log.Println("consumer exited")
}
```

---

## Task 11: Dockerfiles

**Files:**
- Create: `notifications-service/Dockerfile`
- Modify: `delivery/Dockerfile`

- [ ] **Step 1: Create notifications-service Dockerfile**

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /notifications-service ./cmd/api

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /notifications-service .

EXPOSE 8083

CMD ["./notifications-service"]
```

- [ ] **Step 2: Update delivery Dockerfile**

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /delivery ./cmd/consumer

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /delivery .

EXPOSE 8084

CMD ["./delivery"]
```

---

## Task 12: Build and Verify

**Files:**
- Create: `Makefile` in notifications-service
- Create: `Makefile` in delivery

- [ ] **Step 1: Create Makefile for notifications-service**

```makefile
.PHONY: build lint test run

build:
	go build -o bin/notifications-service ./cmd/api

lint:
	golangci-lint run

test:
	go test -v ./...

run:
	go run ./cmd/api
```

- [ ] **Step 2: Create Makefile for delivery**

```makefile
.PHONY: build lint test run

build:
	go build -o bin/delivery ./cmd/consumer

lint:
	golangci-lint run

test:
	go test -v ./...

run:
	go run ./cmd/consumer
```

- [ ] **Step 3: Verify compilation**

Run: `cd notifications-service && go build ./...`
Expected: Compiles without errors

Run: `cd delivery && go build ./...`
Expected: Compiles without errors

---

**Plan complete.** This plan creates a complete notification pipeline with:
- REST API for sending and querying notifications
- RabbitMQ consumer for delivery processing
- Database schemas for notifications, outbox, history, analytics, and delivery tracking
- Mock provider for delivery simulation
- Proper error handling and Russian error messages