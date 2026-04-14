package domain

import (
	"time"
)

// User represents a user account in the system
type User struct {
	ID           string    `json:"user_id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Contact represents a user's contact (email, phone, telegram)
type Contact struct {
	ID         string     `json:"contact_id"`
	UserID     string     `json:"user_id"`
	Channel    string     `json:"channel"` // email, sms, telegram
	Value      string     `json:"value"`
	IsVerified bool       `json:"is_verified"`
	Enabled    bool       `json:"enabled"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

// Preference represents user preferences for a contact
type Preference struct {
	ID              string    `json:"preference_id"`
	ContactID       string    `json:"contact_id"`
	UserID          string    `json:"user_id"`
	SenderID        *string   `json:"sender_id,omitempty"`
	ScopeType       string    `json:"scope_type"` // global, sender, campaign, group
	ScopeID         *string   `json:"scope_id,omitempty"`
	Enabled         bool      `json:"enabled"`
	QuietFrom       *string   `json:"quiet_from,omitempty"`
	QuietTo         *string   `json:"quiet_to,omitempty"`
	BlockedChannels []string  `json:"blocked_channels"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UnsubscribeRule represents an unsubscribe rule
type UnsubscribeRule struct {
	ID        string    `json:"unsubscribe_rule_id"`
	ContactID string    `json:"contact_id"`
	UserID    string    `json:"user_id"`
	SenderID  *string   `json:"sender_id,omitempty"`
	ScopeType string    `json:"scope_type"` // sender, campaign, group
	ScopeID   *string   `json:"scope_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// RefreshToken represents a JWT refresh token
type RefreshToken struct {
	ID        string    `json:"token_id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// GuestClaimToken represents a token for claiming guest contacts
type GuestClaimToken struct {
	ID        string     `json:"token_id"`
	Token     string     `json:"token"`
	ContactID string     `json:"contact_id"`
	UserID    *string    `json:"user_id,omitempty"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// UnsubscribeToken represents an unsubscribe token
type UnsubscribeToken struct {
	ID        string    `json:"token_id"`
	Token     string    `json:"token"`
	ContactID string    `json:"contact_id"`
	SenderID  *string   `json:"sender_id,omitempty"`
	ScopeType string    `json:"scope_type"` // sender, campaign, group
	ScopeID   *string   `json:"scope_id,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        string    `json:"audit_log_id"`
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}
