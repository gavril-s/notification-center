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