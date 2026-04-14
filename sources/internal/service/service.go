package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"notification-center/sources/internal/model"
	"notification-center/sources/internal/repository"
)

var (
	ErrNotFound        = errors.New("ресурс не найден")
	ErrAlreadyExists   = errors.New("ресурс уже существует")
	ErrInvalidInput    = errors.New("некорректные входные данные")
	ErrChannelMismatch = errors.New("канал кампании должен соответствовать каналу шаблона")
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// Sender operations
func (s *Service) ListSenders(ctx context.Context) ([]model.Sender, error) {
	return s.repo.ListSenders(ctx)
}

func (s *Service) CreateSender(ctx context.Context, req model.CreateSenderRequest) (*model.Sender, error) {
	if req.Name == "" {
		return nil, ErrInvalidInput
	}

	sender := &model.Sender{
		Name:        req.Name,
		Description: req.Description,
		Active:      true,
	}

	if err := s.repo.CreateSender(ctx, sender); err != nil {
		return nil, fmt.Errorf("failed to create sender: %w", err)
	}

	// Audit log
	s.repo.CreateAuditLog(ctx, &model.AuditLog{
		EntityType: "sender",
		EntityID:   sender.ID,
		Action:     "create",
		Details:    jsonRaw(map[string]string{"name": sender.Name}),
	})

	return sender, nil
}

func (s *Service) AddOperator(ctx context.Context, senderID uuid.UUID, req model.AddOperatorRequest) (*model.SenderOperator, error) {
	// Verify sender exists
	sender, err := s.repo.GetSender(ctx, senderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender: %w", err)
	}
	if sender == nil {
		return nil, ErrNotFound
	}

	op := &model.SenderOperator{
		SenderID: senderID,
		UserID:   req.UserID,
		Role:     req.Role,
	}

	if err := s.repo.AddOperator(ctx, op); err != nil {
		return nil, fmt.Errorf("failed to add operator: %w", err)
	}

	// Audit log
	s.repo.CreateAuditLog(ctx, &model.AuditLog{
		EntityType: "sender_operator",
		EntityID:   op.ID,
		Action:     "create",
		UserID:     &req.UserID,
		Details:    jsonRaw(map[string]string{"sender_id": senderID.String(), "role": req.Role}),
	})

	return op, nil
}

func (s *Service) AddCredential(ctx context.Context, senderID uuid.UUID, req model.AddCredentialRequest) (*model.SenderCredential, error) {
	// Verify sender exists
	sender, err := s.repo.GetSender(ctx, senderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender: %w", err)
	}
	if sender == nil {
		return nil, ErrNotFound
	}

	cred := &model.SenderCredential{
		SenderID:       senderID,
		IntegrationKey: req.IntegrationKey,
		Name:           req.Name,
		Active:         true,
	}

	if err := s.repo.AddCredential(ctx, cred); err != nil {
		return nil, fmt.Errorf("failed to add credential: %w", err)
	}

	// Audit log
	s.repo.CreateAuditLog(ctx, &model.AuditLog{
		EntityType: "sender_credential",
		EntityID:   cred.ID,
		Action:     "create",
		Details:    jsonRaw(map[string]string{"sender_id": senderID.String(), "name": req.Name}),
	})

	return cred, nil
}

func (s *Service) ResolveCredential(ctx context.Context, req model.ResolveCredentialRequest) (*model.CredentialResolution, error) {
	return s.repo.ResolveCredential(ctx, req.IntegrationKey)
}

func (s *Service) CheckOperatorPermission(ctx context.Context, senderID, userID uuid.UUID) (*model.OperatorPermission, error) {
	return s.repo.CheckOperatorPermission(ctx, senderID, userID)
}

// Template operations
func (s *Service) ListTemplates(ctx context.Context, senderID *uuid.UUID) ([]model.Template, error) {
	return s.repo.ListTemplates(ctx, senderID)
}

func (s *Service) GetTemplate(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	return s.repo.GetTemplate(ctx, id)
}

func (s *Service) CreateTemplate(ctx context.Context, req model.CreateTemplateRequest) (*model.Template, error) {
	if req.Name == "" || req.Body == "" {
		return nil, ErrInvalidInput
	}

	// Verify sender exists
	sender, err := s.repo.GetSender(ctx, req.SenderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender: %w", err)
	}
	if sender == nil {
		return nil, ErrNotFound
	}

	tmpl := &model.Template{
		SenderID:  req.SenderID,
		Name:      req.Name,
		Channel:   req.Channel,
		Subject:   req.Subject,
		Body:      req.Body,
		Variables: req.Variables,
		Active:    true,
		Version:   1,
	}

	if err := s.repo.CreateTemplate(ctx, tmpl); err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// Audit log
	s.repo.CreateAuditLog(ctx, &model.AuditLog{
		EntityType: "template",
		EntityID:   tmpl.ID,
		Action:     "create",
		Details:    jsonRaw(map[string]string{"name": tmpl.Name, "channel": tmpl.Channel}),
	})

	return tmpl, nil
}

func (s *Service) UpdateTemplate(ctx context.Context, id uuid.UUID, req model.UpdateTemplateRequest) (*model.Template, error) {
	// Verify template exists
	existing, err := s.repo.GetTemplate(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	if existing == nil {
		return nil, ErrNotFound
	}

	tmpl := &model.Template{
		ID:        id,
		Name:      req.Name,
		Channel:   req.Channel,
		Subject:   req.Subject,
		Body:      req.Body,
		Variables: req.Variables,
		Active:    req.Active,
	}

	if err := s.repo.UpdateTemplate(ctx, tmpl); err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	// Audit log
	s.repo.CreateAuditLog(ctx, &model.AuditLog{
		EntityType: "template",
		EntityID:   id,
		Action:     "update",
		Details:    jsonRaw(map[string]string{"name": tmpl.Name}),
	})

	return s.repo.GetTemplate(ctx, id)
}

func (s *Service) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	// Verify template exists
	existing, err := s.repo.GetTemplate(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}
	if existing == nil {
		return ErrNotFound
	}

	if err := s.repo.DeleteTemplate(ctx, id); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	// Audit log
	s.repo.CreateAuditLog(ctx, &model.AuditLog{
		EntityType: "template",
		EntityID:   id,
		Action:     "delete",
	})

	return nil
}

// Contact Group operations
func (s *Service) ListGroups(ctx context.Context, senderID *uuid.UUID) ([]model.ContactGroup, error) {
	return s.repo.ListGroups(ctx, senderID)
}

func (s *Service) GetGroup(ctx context.Context, id uuid.UUID) (*model.ContactGroup, error) {
	return s.repo.GetGroup(ctx, id)
}

func (s *Service) CreateGroup(ctx context.Context, req model.CreateGroupRequest) (*model.ContactGroup, error) {
	if req.Name == "" {
		return nil, ErrInvalidInput
	}

	// Verify sender exists
	sender, err := s.repo.GetSender(ctx, req.SenderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender: %w", err)
	}
	if sender == nil {
		return nil, ErrNotFound
	}

	group := &model.ContactGroup{
		SenderID:    req.SenderID,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.repo.CreateGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	// Audit log
	s.repo.CreateAuditLog(ctx, &model.AuditLog{
		EntityType: "contact_group",
		EntityID:   group.ID,
		Action:     "create",
		Details:    jsonRaw(map[string]string{"name": group.Name}),
	})

	return group, nil
}

func (s *Service) UpdateGroup(ctx context.Context, id uuid.UUID, req model.UpdateGroupRequest) (*model.ContactGroup, error) {
	// Verify group exists
	existing, err := s.repo.GetGroup(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if existing == nil {
		return nil, ErrNotFound
	}

	group := &model.ContactGroup{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.repo.UpdateGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	// Audit log
	s.repo.CreateAuditLog(ctx, &model.AuditLog{
		EntityType: "contact_group",
		EntityID:   id,
		Action:     "update",
		Details:    jsonRaw(map[string]string{"name": group.Name}),
	})

	return s.repo.GetGroup(ctx, id)
}

func (s *Service) AddGroupMembers(ctx context.Context, groupID uuid.UUID, req model.AddGroupMembersRequest) error {
	// Verify group exists
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return ErrNotFound
	}

	if err := s.repo.AddGroupMembers(ctx, groupID, req.ContactIDs); err != nil {
		return fmt.Errorf("failed to add group members: %w", err)
	}

	// Audit log
	s.repo.CreateAuditLog(ctx, &model.AuditLog{
		EntityType: "contact_group_members",
		EntityID:   groupID,
		Action:     "add_members",
		Details:    jsonRaw(map[string]any{"count": len(req.ContactIDs)}),
	})

	return nil
}

func (s *Service) GetGroupMembers(ctx context.Context, groupID uuid.UUID) (*model.ContactGroupMembers, error) {
	// Verify group exists
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return nil, ErrNotFound
	}

	contactIDs, err := s.repo.GetGroupMembers(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group members: %w", err)
	}

	return &model.ContactGroupMembers{
		GroupID:    groupID,
		ContactIDs: contactIDs,
	}, nil
}

// Campaign operations
func (s *Service) ListCampaigns(ctx context.Context, senderID *uuid.UUID) ([]model.Campaign, error) {
	return s.repo.ListCampaigns(ctx, senderID)
}

func (s *Service) GetCampaign(ctx context.Context, id uuid.UUID) (*model.CampaignDetail, error) {
	campaign, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}
	if campaign == nil {
		return nil, ErrNotFound
	}

	return &model.CampaignDetail{
		CampaignID:  campaign.ID,
		SenderID:    campaign.SenderID,
		GroupID:     campaign.GroupID,
		TemplateID:  campaign.TemplateID,
		Channels:    campaign.Channels,
		ScheduledAt: campaign.ScheduledAt,
		Active:      campaign.Active,
	}, nil
}

func (s *Service) CreateCampaign(ctx context.Context, req model.CreateCampaignRequest) (*model.Campaign, error) {
	if req.Name == "" {
		return nil, ErrInvalidInput
	}

	// Verify sender exists
	sender, err := s.repo.GetSender(ctx, req.SenderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sender: %w", err)
	}
	if sender == nil {
		return nil, ErrNotFound
	}

	// Verify group exists
	group, err := s.repo.GetGroup(ctx, req.GroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return nil, ErrNotFound
	}

	// Verify template exists
	_, err = s.repo.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Create recurrence rule if provided
	var recurrenceRuleID *uuid.UUID
	if req.RecurrenceRule != nil {
		rrule := &model.RecurrenceRule{
			Kind:  req.RecurrenceRule.Kind,
			Value: "",
		}
		if req.RecurrenceRule.Value != nil {
			rrule.Value = *req.RecurrenceRule.Value
		}
		if err := s.repo.CreateRecurrenceRule(ctx, rrule); err != nil {
			return nil, fmt.Errorf("failed to create recurrence rule: %w", err)
		}
		recurrenceRuleID = &rrule.ID
	}

	campaign := &model.Campaign{
		SenderID:         req.SenderID,
		GroupID:          req.GroupID,
		TemplateID:       req.TemplateID,
		Name:             req.Name,
		Channels:         req.Channels,
		ScheduledAt:      req.ScheduledAt,
		RecurrenceRuleID: recurrenceRuleID,
		Active:           true,
	}

	if err := s.repo.CreateCampaign(ctx, campaign); err != nil {
		return nil, err
	}

	// Audit log
	s.repo.CreateAuditLog(ctx, &model.AuditLog{
		EntityType: "campaign",
		EntityID:   campaign.ID,
		Action:     "create",
		Details:    jsonRaw(map[string]string{"name": campaign.Name}),
	})

	return campaign, nil
}

func (s *Service) UpdateCampaign(ctx context.Context, id uuid.UUID, req model.UpdateCampaignRequest) (*model.CampaignDetail, error) {
	// Verify campaign exists
	existing, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign: %w", err)
	}
	if existing == nil {
		return nil, ErrNotFound
	}

	campaign := &model.Campaign{
		ID:          id,
		Name:        req.Name,
		Channels:    req.Channels,
		ScheduledAt: req.ScheduledAt,
		Active:      req.Active,
	}

	if err := s.repo.UpdateCampaign(ctx, campaign); err != nil {
		return nil, fmt.Errorf("failed to update campaign: %w", err)
	}

	// Audit log
	s.repo.CreateAuditLog(ctx, &model.AuditLog{
		EntityType: "campaign",
		EntityID:   id,
		Action:     "update",
		Details:    jsonRaw(map[string]string{"name": req.Name}),
	})

	return s.GetCampaign(ctx, id)
}

func (s *Service) GetDueCampaigns(ctx context.Context, asOf string, limit int, cursor *string) ([]model.DueCampaign, string, error) {
	campaigns, err := s.repo.GetDueCampaigns(ctx, asOf, limit, cursor)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get due campaigns: %w", err)
	}

	var nextCursor string
	if len(campaigns) > 0 {
		lastCampaign := campaigns[len(campaigns)-1]
		nextCursor = lastCampaign.CampaignID.String()
	}

	return campaigns, nextCursor, nil
}

func jsonRaw(v any) *json.RawMessage {
	data, _ := json.Marshal(v)
	raw := json.RawMessage(data)
	return &raw
}
