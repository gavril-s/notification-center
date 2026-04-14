-- Sources schema migrations
-- This script creates all tables in the sources schema

-- Senders table
CREATE TABLE IF NOT EXISTS sources.senders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Sender operators table (operator-to-sender assignment)
CREATE TABLE IF NOT EXISTS sources.sender_operators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES sources.senders(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'source_operator',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(sender_id, user_id)
);

-- Sender credentials table (integration credentials)
CREATE TABLE IF NOT EXISTS sources.sender_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES sources.senders(id) ON DELETE CASCADE,
    integration_key VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Templates table
CREATE TABLE IF NOT EXISTS sources.templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES sources.senders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    channel VARCHAR(50) NOT NULL CHECK (channel IN ('email', 'sms', 'telegram')),
    subject VARCHAR(500),
    body TEXT NOT NULL,
    variables JSONB NOT NULL DEFAULT '[]',
    active BOOLEAN NOT NULL DEFAULT true,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Contact groups table
CREATE TABLE IF NOT EXISTS sources.contact_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES sources.senders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Contact group members table
CREATE TABLE IF NOT EXISTS sources.contact_group_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id UUID NOT NULL REFERENCES sources.contact_groups(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(group_id, contact_id)
);

-- Recurrence rules table
CREATE TABLE IF NOT EXISTS sources.recurrence_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind VARCHAR(50) NOT NULL CHECK (kind IN ('once', 'daily', 'weekly', 'cron')),
    value TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Campaigns table
CREATE TABLE IF NOT EXISTS sources.campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES sources.senders(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES sources.contact_groups(id) ON DELETE CASCADE,
    template_id UUID NOT NULL REFERENCES sources.templates(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    channels JSONB NOT NULL DEFAULT '["email"]',
    scheduled_at TIMESTAMPTZ,
    recurrence_rule_id UUID REFERENCES sources.recurrence_rules(id) ON DELETE SET NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit log table
CREATE TABLE IF NOT EXISTS sources.audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    user_id UUID,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_sender_operators_sender_id ON sources.sender_operators(sender_id);
CREATE INDEX IF NOT EXISTS idx_sender_operators_user_id ON sources.sender_operators(user_id);
CREATE INDEX IF NOT EXISTS idx_sender_credentials_sender_id ON sources.sender_credentials(sender_id);
CREATE INDEX IF NOT EXISTS idx_sender_credentials_key ON sources.sender_credentials(integration_key);
CREATE INDEX IF NOT EXISTS idx_templates_sender_id ON sources.templates(sender_id);
CREATE INDEX IF NOT EXISTS idx_contact_groups_sender_id ON sources.contact_groups(sender_id);
CREATE INDEX IF NOT EXISTS idx_contact_group_members_group_id ON sources.contact_group_members(group_id);
CREATE INDEX IF NOT EXISTS idx_campaigns_sender_id ON sources.campaigns(sender_id);
CREATE INDEX IF NOT EXISTS idx_campaigns_scheduled_at ON sources.campaigns(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_audit_log_entity ON sources.audit_log(entity_type, entity_id);