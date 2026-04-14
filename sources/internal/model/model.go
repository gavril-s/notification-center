package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Sender represents a sender entity
type Sender struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SenderOperator represents an operator assignment to a sender
type SenderOperator struct {
	ID        uuid.UUID `json:"id"`
	SenderID  uuid.UUID `json:"sender_id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"` // "source_operator" or "admin"
	CreatedAt time.Time `json:"created_at"`
}

// OperatorPermission represents operator permission check result
type OperatorPermission struct {
	Allowed bool   `json:"allowed"`
	Role    string `json:"role,omitempty"`
}

// SenderCredential represents integration credentials
type SenderCredential struct {
	ID             uuid.UUID `json:"id"`
	SenderID       uuid.UUID `json:"sender_id"`
	IntegrationKey string    `json:"integration_key"`
	Name           string    `json:"name"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CredentialResolution represents credential resolution result
type CredentialResolution struct {
	SenderID     uuid.UUID `json:"sender_id"`
	CredentialID uuid.UUID `json:"credential_id"`
	Active       bool      `json:"active"`
}

// Template represents a message template
type Template struct {
	ID        uuid.UUID `json:"id"`
	SenderID  uuid.UUID `json:"sender_id"`
	Name      string    `json:"name"`
	Channel   string    `json:"channel"` // "email", "sms", "telegram"
	Subject   *string   `json:"subject,omitempty"`
	Body      string    `json:"body"`
	Variables []string  `json:"variables"`
	Active    bool      `json:"active"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ContactGroup represents a recipient group
type ContactGroup struct {
	ID          uuid.UUID `json:"id"`
	SenderID    uuid.UUID `json:"sender_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ContactGroupMembers represents group membership response
type ContactGroupMembers struct {
	GroupID    uuid.UUID   `json:"group_id"`
	ContactIDs []uuid.UUID `json:"contact_ids"`
}

// RecurrenceRule represents a recurrence rule
type RecurrenceRule struct {
	ID        uuid.UUID `json:"id"`
	Kind      string    `json:"kind"` // "once", "daily", "weekly", "cron"
	Value     string    `json:"value,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Campaign represents a notification campaign
type Campaign struct {
	ID               uuid.UUID  `json:"id"`
	SenderID         uuid.UUID  `json:"sender_id"`
	GroupID          uuid.UUID  `json:"group_id"`
	TemplateID       uuid.UUID  `json:"template_id"`
	Name             string     `json:"name"`
	Channels         []string   `json:"channels"`
	ScheduledAt      *time.Time `json:"scheduled_at,omitempty"`
	RecurrenceRuleID *uuid.UUID `json:"recurrence_rule_id,omitempty"`
	Active           bool       `json:"active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// CampaignDetail represents detailed campaign info for internal API
type CampaignDetail struct {
	CampaignID  uuid.UUID  `json:"campaign_id"`
	SenderID    uuid.UUID  `json:"sender_id"`
	GroupID     uuid.UUID  `json:"group_id"`
	TemplateID  uuid.UUID  `json:"template_id"`
	Channels    []string   `json:"channels"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	Active      bool       `json:"active"`
}

// DueCampaign represents a due campaign for scheduling
type DueCampaign struct {
	OccurrenceID string    `json:"occurrence_id"`
	CampaignID   uuid.UUID `json:"campaign_id"`
	SenderID     uuid.UUID `json:"sender_id"`
	GroupID      uuid.UUID `json:"group_id"`
	TemplateID   uuid.UUID `json:"template_id"`
	Channels     []string  `json:"channels"`
	ScheduledAt  time.Time `json:"scheduled_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID         uuid.UUID        `json:"id"`
	EntityType string           `json:"entity_type"`
	EntityID   uuid.UUID        `json:"entity_id"`
	Action     string           `json:"action"`
	UserID     *uuid.UUID       `json:"user_id,omitempty"`
	Details    *json.RawMessage `json:"details,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
}

// Request/Response DTOs

type CreateSenderRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type AddOperatorRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Role   string    `json:"role" binding:"required,oneof=source_operator admin"`
}

type AddCredentialRequest struct {
	IntegrationKey string `json:"integration_key" binding:"required"`
	Name           string `json:"name" binding:"required"`
}

type CreateTemplateRequest struct {
	SenderID  uuid.UUID `json:"sender_id" binding:"required"`
	Name      string    `json:"name" binding:"required"`
	Channel   string    `json:"channel" binding:"required,oneof=email sms telegram"`
	Subject   *string   `json:"subject"`
	Body      string    `json:"body" binding:"required"`
	Variables []string  `json:"variables"`
}

type UpdateTemplateRequest struct {
	Name      string   `json:"name" binding:"required"`
	Channel   string   `json:"channel" binding:"required,oneof=email sms telegram"`
	Subject   *string  `json:"subject"`
	Body      string   `json:"body" binding:"required"`
	Variables []string `json:"variables"`
	Active    bool     `json:"active"`
}

type CreateGroupRequest struct {
	SenderID    uuid.UUID `json:"sender_id" binding:"required"`
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
}

type UpdateGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type AddGroupMembersRequest struct {
	ContactIDs []uuid.UUID `json:"contact_ids" binding:"required"`
}

type CreateCampaignRequest struct {
	SenderID       uuid.UUID              `json:"sender_id" binding:"required"`
	GroupID        uuid.UUID              `json:"group_id" binding:"required"`
	TemplateID     uuid.UUID              `json:"template_id" binding:"required"`
	Name           string                 `json:"name" binding:"required"`
	Channels       []string               `json:"channels"`
	ScheduledAt    *time.Time             `json:"scheduled_at"`
	RecurrenceRule *RecurrenceRuleRequest `json:"recurrence_rule"`
}

type RecurrenceRuleRequest struct {
	Kind  string  `json:"kind" binding:"required,oneof=once daily weekly cron"`
	Value *string `json:"value"`
}

type UpdateCampaignRequest struct {
	Name        string     `json:"name" binding:"required"`
	Channels    []string   `json:"channels" binding:"required"`
	ScheduledAt *time.Time `json:"scheduled_at"`
	Active      bool       `json:"active"`
}

type ResolveCredentialRequest struct {
	IntegrationKey string `json:"integration_key" binding:"required"`
}

type ListResponse struct {
	Items  interface{} `json:"items"`
	Total  int         `json:"total"`
	Cursor string      `json:"cursor,omitempty"`
}
