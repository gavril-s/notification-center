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