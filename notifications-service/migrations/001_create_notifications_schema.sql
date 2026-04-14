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