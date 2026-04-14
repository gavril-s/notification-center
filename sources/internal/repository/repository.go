package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"notification-center/sources/internal/model"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Sender operations
func (r *Repository) ListSenders(ctx context.Context) ([]model.Sender, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, active, created_at, updated_at
		FROM sources.senders
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list senders: %w", err)
	}
	defer rows.Close()

	var senders []model.Sender
	for rows.Next() {
		var s model.Sender
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Active, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan sender: %w", err)
		}
		senders = append(senders, s)
	}
	return senders, nil
}

func (r *Repository) CreateSender(ctx context.Context, sender *model.Sender) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO sources.senders (name, description, active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, sender.Name, sender.Description, sender.Active).Scan(&sender.ID, &sender.CreatedAt, &sender.UpdatedAt)
}

func (r *Repository) GetSender(ctx context.Context, id uuid.UUID) (*model.Sender, error) {
	var s model.Sender
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, active, created_at, updated_at
		FROM sources.senders WHERE id = $1
	`, id).Scan(&s.ID, &s.Name, &s.Description, &s.Active, &s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sender: %w", err)
	}
	return &s, nil
}

// Sender Operator operations
func (r *Repository) AddOperator(ctx context.Context, op *model.SenderOperator) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO sources.sender_operators (sender_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (sender_id, user_id) DO UPDATE SET role = EXCLUDED.role
		RETURNING id, created_at
	`, op.SenderID, op.UserID, op.Role).Scan(&op.ID, &op.CreatedAt)
}

func (r *Repository) CheckOperatorPermission(ctx context.Context, senderID, userID uuid.UUID) (*model.OperatorPermission, error) {
	var op model.SenderOperator
	err := r.pool.QueryRow(ctx, `
		SELECT id, sender_id, user_id, role, created_at
		FROM sources.sender_operators
		WHERE sender_id = $1 AND user_id = $2
	`, senderID, userID).Scan(&op.ID, &op.SenderID, &op.UserID, &op.Role, &op.CreatedAt)
	if err == pgx.ErrNoRows {
		return &model.OperatorPermission{Allowed: false, Role: ""}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check operator permission: %w", err)
	}
	return &model.OperatorPermission{Allowed: true, Role: op.Role}, nil
}

// Sender Credential operations
func (r *Repository) AddCredential(ctx context.Context, cred *model.SenderCredential) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO sources.sender_credentials (sender_id, integration_key, name, active)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`, cred.SenderID, cred.IntegrationKey, cred.Name, cred.Active).Scan(&cred.ID, &cred.CreatedAt, &cred.UpdatedAt)
}

func (r *Repository) ResolveCredential(ctx context.Context, integrationKey string) (*model.CredentialResolution, error) {
	var cred model.SenderCredential
	err := r.pool.QueryRow(ctx, `
		SELECT id, sender_id, integration_key, name, active, created_at, updated_at
		FROM sources.sender_credentials
		WHERE integration_key = $1 AND active = true
	`, integrationKey).Scan(&cred.ID, &cred.SenderID, &cred.IntegrationKey, &cred.Name, &cred.Active, &cred.CreatedAt, &cred.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to resolve credential: %w", err)
	}
	return &model.CredentialResolution{
		SenderID:     cred.SenderID,
		CredentialID: cred.ID,
		Active:       cred.Active,
	}, nil
}

// Template operations
func (r *Repository) ListTemplates(ctx context.Context, senderID *uuid.UUID) ([]model.Template, error) {
	query := `
		SELECT id, sender_id, name, channel, subject, body, variables, active, version, created_at, updated_at
		FROM sources.templates
	`
	args := []interface{}{}
	if senderID != nil {
		query += " WHERE sender_id = $1"
		args = append(args, *senderID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []model.Template
	for rows.Next() {
		var t model.Template
		var variables []byte
		if err := rows.Scan(&t.ID, &t.SenderID, &t.Name, &t.Channel, &t.Subject, &t.Body, &variables, &t.Active, &t.Version, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		json.Unmarshal(variables, &t.Variables)
		templates = append(templates, t)
	}
	return templates, nil
}

func (r *Repository) GetTemplate(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	var t model.Template
	var variables []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, sender_id, name, channel, subject, body, variables, active, version, created_at, updated_at
		FROM sources.templates WHERE id = $1
	`, id).Scan(&t.ID, &t.SenderID, &t.Name, &t.Channel, &t.Subject, &t.Body, &variables, &t.Active, &t.Version, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	json.Unmarshal(variables, &t.Variables)
	return &t, nil
}

func (r *Repository) CreateTemplate(ctx context.Context, tmpl *model.Template) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO sources.templates (sender_id, name, channel, subject, body, variables, active, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, tmpl.SenderID, tmpl.Name, tmpl.Channel, tmpl.Subject, tmpl.Body, tmpl.Variables, tmpl.Active, tmpl.Version).Scan(&tmpl.ID, &tmpl.CreatedAt, &tmpl.UpdatedAt)
}

func (r *Repository) UpdateTemplate(ctx context.Context, tmpl *model.Template) error {
	return r.pool.QueryRow(ctx, `
		UPDATE sources.templates
		SET name = $1, channel = $2, subject = $3, body = $4, variables = $5, active = $6, version = version + 1, updated_at = NOW()
		WHERE id = $7
		RETURNING version, updated_at
	`, tmpl.Name, tmpl.Channel, tmpl.Subject, tmpl.Body, tmpl.Variables, tmpl.Active, tmpl.ID).Scan(&tmpl.Version, &tmpl.UpdatedAt)
}

func (r *Repository) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM sources.templates WHERE id = $1", id)
	return err
}

// Contact Group operations
func (r *Repository) ListGroups(ctx context.Context, senderID *uuid.UUID) ([]model.ContactGroup, error) {
	query := `
		SELECT id, sender_id, name, description, created_at, updated_at
		FROM sources.contact_groups
	`
	args := []interface{}{}
	if senderID != nil {
		query += " WHERE sender_id = $1"
		args = append(args, *senderID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	defer rows.Close()

	var groups []model.ContactGroup
	for rows.Next() {
		var g model.ContactGroup
		if err := rows.Scan(&g.ID, &g.SenderID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (r *Repository) GetGroup(ctx context.Context, id uuid.UUID) (*model.ContactGroup, error) {
	var g model.ContactGroup
	err := r.pool.QueryRow(ctx, `
		SELECT id, sender_id, name, description, created_at, updated_at
		FROM sources.contact_groups WHERE id = $1
	`, id).Scan(&g.ID, &g.SenderID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	return &g, nil
}

func (r *Repository) CreateGroup(ctx context.Context, group *model.ContactGroup) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO sources.contact_groups (sender_id, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`, group.SenderID, group.Name, group.Description).Scan(&group.ID, &group.CreatedAt, &group.UpdatedAt)
}

func (r *Repository) UpdateGroup(ctx context.Context, group *model.ContactGroup) error {
	return r.pool.QueryRow(ctx, `
		UPDATE sources.contact_groups
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING updated_at
	`, group.Name, group.Description, group.ID).Scan(&group.UpdatedAt)
}

// Contact Group Members operations
func (r *Repository) AddGroupMembers(ctx context.Context, groupID uuid.UUID, contactIDs []uuid.UUID) error {
	if len(contactIDs) == 0 {
		return nil
	}
	for _, contactID := range contactIDs {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO sources.contact_group_members (group_id, contact_id)
			VALUES ($1, $2)
			ON CONFLICT (group_id, contact_id) DO NOTHING
		`, groupID, contactID)
		if err != nil {
			return fmt.Errorf("failed to add group member: %w", err)
		}
	}
	return nil
}

func (r *Repository) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT contact_id FROM sources.contact_group_members WHERE group_id = $1
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}
	defer rows.Close()

	var contactIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan contact id: %w", err)
		}
		contactIDs = append(contactIDs, id)
	}
	return contactIDs, nil
}

// Campaign operations
func (r *Repository) ListCampaigns(ctx context.Context, senderID *uuid.UUID) ([]model.Campaign, error) {
	query := `
		SELECT c.id, c.sender_id, c.group_id, c.template_id, c.name, c.channels, c.scheduled_at, c.recurrence_rule_id, c.active, c.created_at, c.updated_at
		FROM sources.campaigns c
	`
	args := []interface{}{}
	if senderID != nil {
		query += " WHERE c.sender_id = $1"
		args = append(args, *senderID)
	}
	query += " ORDER BY c.created_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list campaigns: %w", err)
	}
	defer rows.Close()

	var campaigns []model.Campaign
	for rows.Next() {
		var c model.Campaign
		var channels []byte
		if err := rows.Scan(&c.ID, &c.SenderID, &c.GroupID, &c.TemplateID, &c.Name, &channels, &c.ScheduledAt, &c.RecurrenceRuleID, &c.Active, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan campaign: %w", err)
		}
		json.Unmarshal(channels, &c.Channels)
		campaigns = append(campaigns, c)
	}
	return campaigns, nil
}

func (r *Repository) GetCampaign(ctx context.Context, id uuid.UUID) (*model.Campaign, error) {
	var c model.Campaign
	var channels []byte
	err := r.pool.QueryRow(ctx, `
		SELECT c.id, c.sender_id, c.group_id, c.template_id, c.name, c.channels, c.scheduled_at, c.recurrence_rule_id, c.active, c.created_at, c.updated_at
		FROM sources.campaigns c WHERE c.id = $1
	`, id).Scan(&c.ID, &c.SenderID, &c.GroupID, &c.TemplateID, &c.Name, &channels, &c.ScheduledAt, &c.RecurrenceRuleID, &c.Active, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}
	json.Unmarshal(channels, &c.Channels)
	return &c, nil
}

func (r *Repository) CreateCampaign(ctx context.Context, campaign *model.Campaign) error {
	// Get template to verify channel
	tmpl, err := r.GetTemplate(ctx, campaign.TemplateID)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}
	if tmpl == nil {
		return fmt.Errorf("template not found")
	}

	// Verify channel matches template
	if len(campaign.Channels) == 0 {
		campaign.Channels = []string{tmpl.Channel}
	}
	if len(campaign.Channels) > 0 && campaign.Channels[0] != tmpl.Channel {
		return fmt.Errorf("канал кампании должен соответствовать каналу шаблона")
	}

	return r.pool.QueryRow(ctx, `
		INSERT INTO sources.campaigns (sender_id, group_id, template_id, name, channels, scheduled_at, recurrence_rule_id, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, campaign.SenderID, campaign.GroupID, campaign.TemplateID, campaign.Name, campaign.Channels, campaign.ScheduledAt, campaign.RecurrenceRuleID, campaign.Active).Scan(&campaign.ID, &campaign.CreatedAt, &campaign.UpdatedAt)
}

func (r *Repository) UpdateCampaign(ctx context.Context, campaign *model.Campaign) error {
	return r.pool.QueryRow(ctx, `
		UPDATE sources.campaigns
		SET name = $1, channels = $2, scheduled_at = $3, active = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`, campaign.Name, campaign.Channels, campaign.ScheduledAt, campaign.Active, campaign.ID).Scan(&campaign.UpdatedAt)
}

// Due campaigns query
func (r *Repository) GetDueCampaigns(ctx context.Context, asOf string, limit int, cursor *string) ([]model.DueCampaign, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT 
			c.id,
			c.sender_id,
			c.group_id,
			c.template_id,
			c.channels,
			COALESCE(c.scheduled_at, c.created_at) as scheduled_at
		FROM sources.campaigns c
		WHERE c.active = true 
			AND (c.scheduled_at IS NULL OR c.scheduled_at <= $1)
			AND ($2::text IS NULL OR c.id > $2::uuid)
		ORDER BY c.id
		LIMIT $3
	`, asOf, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get due campaigns: %w", err)
	}
	defer rows.Close()

	var campaigns []model.DueCampaign
	for rows.Next() {
		var c model.DueCampaign
		var channels []byte
		if err := rows.Scan(&c.CampaignID, &c.SenderID, &c.GroupID, &c.TemplateID, &channels, &c.ScheduledAt); err != nil {
			return nil, fmt.Errorf("failed to scan due campaign: %w", err)
		}
		json.Unmarshal(channels, &c.Channels)
		// Generate occurrence_id as campaign_id:scheduled_at
		c.OccurrenceID = fmt.Sprintf("%s:%s", c.CampaignID.String(), c.ScheduledAt.Format("2006-01-02T15:04:05Z07:00"))
		campaigns = append(campaigns, c)
	}
	return campaigns, nil
}

// Audit log operations
func (r *Repository) CreateAuditLog(ctx context.Context, log *model.AuditLog) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO sources.audit_log (entity_type, entity_id, action, user_id, details)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, log.EntityType, log.EntityID, log.Action, log.UserID, log.Details).Scan(&log.ID, &log.CreatedAt)
}

// Recurrence rule operations
func (r *Repository) GetRecurrenceRule(ctx context.Context, id uuid.UUID) (*model.RecurrenceRule, error) {
	var rrule model.RecurrenceRule
	err := r.pool.QueryRow(ctx, `
		SELECT id, kind, value, created_at FROM sources.recurrence_rules WHERE id = $1
	`, id).Scan(&rrule.ID, &rrule.Kind, &rrule.Value, &rrule.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get recurrence rule: %w", err)
	}
	return &rrule, nil
}

func (r *Repository) CreateRecurrenceRule(ctx context.Context, rrule *model.RecurrenceRule) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO sources.recurrence_rules (kind, value)
		VALUES ($1, $2)
		RETURNING id, created_at
	`, rrule.Kind, rrule.Value).Scan(&rrule.ID, &rrule.CreatedAt)
}
