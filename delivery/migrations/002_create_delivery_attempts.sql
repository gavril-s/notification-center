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