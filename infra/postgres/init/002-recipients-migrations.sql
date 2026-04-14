-- Recipients schema tables
-- Run after 001-create-schemas.sql

-- Users table
CREATE TABLE IF NOT EXISTS recipients.users (
    id UUID PRIMARY KEY,
    login VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_login ON recipients.users(login);

-- Refresh tokens table
CREATE TABLE IF NOT EXISTS recipients.refresh_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES recipients.users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token ON recipients.refresh_tokens(token);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON recipients.refresh_tokens(user_id);

-- Contacts table
CREATE TABLE IF NOT EXISTS recipients.contacts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES recipients.users(id) ON DELETE CASCADE,
    channel VARCHAR(50) NOT NULL,
    value TEXT NOT NULL,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(user_id, channel, value)
);

CREATE INDEX IF NOT EXISTS idx_contacts_user_id ON recipients.contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_deleted_at ON recipients.contacts(deleted_at);

-- Preferences table
CREATE TABLE IF NOT EXISTS recipients.preferences (
    id UUID PRIMARY KEY,
    contact_id UUID NOT NULL REFERENCES recipients.contacts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES recipients.users(id) ON DELETE CASCADE,
    sender_id UUID,
    scope_type VARCHAR(50) NOT NULL,
    scope_id UUID,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    quiet_from VARCHAR(5),
    quiet_to VARCHAR(5),
    blocked_channels JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_preferences_unique 
    ON recipients.preferences(contact_id, scope_type, COALESCE(sender_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(scope_id, '00000000-0000-0000-0000-000000000000'::uuid));

CREATE INDEX IF NOT EXISTS idx_preferences_contact_id ON recipients.preferences(contact_id);
CREATE INDEX IF NOT EXISTS idx_preferences_user_id ON recipients.preferences(user_id);

-- Unsubscribe rules table
CREATE TABLE IF NOT EXISTS recipients.unsubscribe_rules (
    id UUID PRIMARY KEY,
    contact_id UUID NOT NULL REFERENCES recipients.contacts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES recipients.users(id) ON DELETE CASCADE,
    sender_id UUID,
    scope_type VARCHAR(50) NOT NULL,
    scope_id UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_unsubscribe_rules_unique 
    ON recipients.unsubscribe_rules(contact_id, scope_type, COALESCE(sender_id, '00000000-0000-0000-0000-000000000000'::uuid), COALESCE(scope_id, '00000000-0000-0000-0000-000000000000'::uuid));

CREATE INDEX IF NOT EXISTS idx_unsubscribe_rules_contact_id ON recipients.unsubscribe_rules(contact_id);

-- Guest claim tokens table
CREATE TABLE IF NOT EXISTS recipients.guest_claim_tokens (
    id UUID PRIMARY KEY,
    token VARCHAR(255) NOT NULL UNIQUE,
    contact_id UUID NOT NULL REFERENCES recipients.contacts(id) ON DELETE CASCADE,
    user_id UUID REFERENCES recipients.users(id) ON DELETE SET NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_guest_claim_tokens_token ON recipients.guest_claim_tokens(token);

-- Unsubscribe tokens table
CREATE TABLE IF NOT EXISTS recipients.unsubscribe_tokens (
    id UUID PRIMARY KEY,
    token VARCHAR(255) NOT NULL UNIQUE,
    contact_id UUID NOT NULL REFERENCES recipients.contacts(id) ON DELETE CASCADE,
    sender_id UUID,
    scope_type VARCHAR(50) NOT NULL,
    scope_id UUID,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_unsubscribe_tokens_token ON recipients.unsubscribe_tokens(token);

-- Audit log table
CREATE TABLE IF NOT EXISTS recipients.audit_log (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES recipients.users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    details TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON recipients.audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON recipients.audit_log(created_at);

-- Schema migrations table
CREATE TABLE IF NOT EXISTS recipients.schema_migrations (
    version_id bigint not null primary key,
    is_applied boolean not null default false,
    timestamp timestamptz not null default now()
);

-- Insert default admin user (password: admin123, hash is bcrypt of "admin123")
INSERT INTO recipients.users (id, login, password_hash, is_admin, created_at, updated_at)
VALUES ('00000000-0000-0000-0000-000000000001', 'admin', '$2y$10$.Anx3M4A811kK9PygwYpseAmDH/rk1mUcwGAvvuPyX3hsyrUhLzh6', true, NOW(), NOW())
ON CONFLICT (login) DO NOTHING;
