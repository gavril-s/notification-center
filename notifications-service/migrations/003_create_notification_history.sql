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