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