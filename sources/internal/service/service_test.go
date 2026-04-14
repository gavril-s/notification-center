package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"notification-center/sources/internal/model"
)

type RepositoryInterface interface {
	ListSenders(ctx context.Context) ([]model.Sender, error)
	CreateSender(ctx context.Context, sender *model.Sender) error
	GetSender(ctx context.Context, id uuid.UUID) (*model.Sender, error)
	AddOperator(ctx context.Context, op *model.SenderOperator) error
	CheckOperatorPermission(ctx context.Context, senderID, userID uuid.UUID) (*model.OperatorPermission, error)
	AddCredential(ctx context.Context, cred *model.SenderCredential) error
	ResolveCredential(ctx context.Context, integrationKey string) (*model.CredentialResolution, error)
	ListTemplates(ctx context.Context, senderID *uuid.UUID) ([]model.Template, error)
	GetTemplate(ctx context.Context, id uuid.UUID) (*model.Template, error)
	CreateTemplate(ctx context.Context, tmpl *model.Template) error
	UpdateTemplate(ctx context.Context, tmpl *model.Template) error
	DeleteTemplate(ctx context.Context, id uuid.UUID) error
	ListGroups(ctx context.Context, senderID *uuid.UUID) ([]model.ContactGroup, error)
	GetGroup(ctx context.Context, id uuid.UUID) (*model.ContactGroup, error)
	CreateGroup(ctx context.Context, group *model.ContactGroup) error
	UpdateGroup(ctx context.Context, group *model.ContactGroup) error
	AddGroupMembers(ctx context.Context, groupID uuid.UUID, contactIDs []uuid.UUID) error
	GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
	ListCampaigns(ctx context.Context, senderID *uuid.UUID) ([]model.Campaign, error)
	GetCampaign(ctx context.Context, id uuid.UUID) (*model.Campaign, error)
	CreateCampaign(ctx context.Context, campaign *model.Campaign) error
	UpdateCampaign(ctx context.Context, campaign *model.Campaign) error
	GetDueCampaigns(ctx context.Context, asOf string, limit int, cursor *string) ([]model.DueCampaign, error)
	CreateAuditLog(ctx context.Context, log *model.AuditLog) error
	CreateRecurrenceRule(ctx context.Context, rrule *model.RecurrenceRule) error
}

type mockRepo struct {
	senders         []model.Sender
	senderGroups    []model.ContactGroup
	templates       []model.Template
	campaigns       []model.Campaign
	credentials     []model.SenderCredential
	operators       []model.SenderOperator
	groupMembers    map[uuid.UUID][]uuid.UUID
	recurrenceRules []model.RecurrenceRule
	dueCampaigns    []model.DueCampaign
	auditLogs       []model.AuditLog
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		groupMembers: make(map[uuid.UUID][]uuid.UUID),
	}
}

func (m *mockRepo) ListSenders(ctx context.Context) ([]model.Sender, error) {
	return m.senders, nil
}

func (m *mockRepo) CreateSender(ctx context.Context, sender *model.Sender) error {
	sender.ID = uuid.New()
	sender.CreatedAt = time.Now()
	sender.UpdatedAt = time.Now()
	m.senders = append(m.senders, *sender)
	return nil
}

func (m *mockRepo) GetSender(ctx context.Context, id uuid.UUID) (*model.Sender, error) {
	for _, s := range m.senders {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) AddOperator(ctx context.Context, op *model.SenderOperator) error {
	op.ID = uuid.New()
	op.CreatedAt = time.Now()
	m.operators = append(m.operators, *op)
	return nil
}

func (m *mockRepo) CheckOperatorPermission(ctx context.Context, senderID, userID uuid.UUID) (*model.OperatorPermission, error) {
	for _, op := range m.operators {
		if op.SenderID == senderID && op.UserID == userID {
			return &model.OperatorPermission{Allowed: true, Role: op.Role}, nil
		}
	}
	return &model.OperatorPermission{Allowed: false, Role: ""}, nil
}

func (m *mockRepo) AddCredential(ctx context.Context, cred *model.SenderCredential) error {
	cred.ID = uuid.New()
	cred.CreatedAt = time.Now()
	cred.UpdatedAt = time.Now()
	m.credentials = append(m.credentials, *cred)
	return nil
}

func (m *mockRepo) ResolveCredential(ctx context.Context, integrationKey string) (*model.CredentialResolution, error) {
	for _, c := range m.credentials {
		if c.IntegrationKey == integrationKey && c.Active {
			return &model.CredentialResolution{
				SenderID:     c.SenderID,
				CredentialID: c.ID,
				Active:       c.Active,
			}, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) ListTemplates(ctx context.Context, senderID *uuid.UUID) ([]model.Template, error) {
	var result []model.Template
	for _, t := range m.templates {
		if senderID == nil || t.SenderID == *senderID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockRepo) GetTemplate(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	for _, t := range m.templates {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) CreateTemplate(ctx context.Context, tmpl *model.Template) error {
	tmpl.ID = uuid.New()
	tmpl.CreatedAt = time.Now()
	tmpl.UpdatedAt = time.Now()
	m.templates = append(m.templates, *tmpl)
	return nil
}

func (m *mockRepo) UpdateTemplate(ctx context.Context, tmpl *model.Template) error {
	for i, t := range m.templates {
		if t.ID == tmpl.ID {
			tmpl.Version = t.Version + 1
			m.templates[i] = *tmpl
			return nil
		}
	}
	return nil
}

func (m *mockRepo) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	for i, t := range m.templates {
		if t.ID == id {
			m.templates = append(m.templates[:i], m.templates[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockRepo) ListGroups(ctx context.Context, senderID *uuid.UUID) ([]model.ContactGroup, error) {
	var result []model.ContactGroup
	for _, g := range m.senderGroups {
		if senderID == nil || g.SenderID == *senderID {
			result = append(result, g)
		}
	}
	return result, nil
}

func (m *mockRepo) GetGroup(ctx context.Context, id uuid.UUID) (*model.ContactGroup, error) {
	for _, g := range m.senderGroups {
		if g.ID == id {
			return &g, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) CreateGroup(ctx context.Context, group *model.ContactGroup) error {
	group.ID = uuid.New()
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()
	m.senderGroups = append(m.senderGroups, *group)
	return nil
}

func (m *mockRepo) UpdateGroup(ctx context.Context, group *model.ContactGroup) error {
	for i, g := range m.senderGroups {
		if g.ID == group.ID {
			m.senderGroups[i] = *group
			return nil
		}
	}
	return nil
}

func (m *mockRepo) AddGroupMembers(ctx context.Context, groupID uuid.UUID, contactIDs []uuid.UUID) error {
	if m.groupMembers == nil {
		m.groupMembers = make(map[uuid.UUID][]uuid.UUID)
	}
	existing := m.groupMembers[groupID]
	for _, cid := range contactIDs {
		found := false
		for _, e := range existing {
			if e == cid {
				found = true
				break
			}
		}
		if !found {
			existing = append(existing, cid)
		}
	}
	m.groupMembers[groupID] = existing
	return nil
}

func (m *mockRepo) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return m.groupMembers[groupID], nil
}

func (m *mockRepo) ListCampaigns(ctx context.Context, senderID *uuid.UUID) ([]model.Campaign, error) {
	var result []model.Campaign
	for _, c := range m.campaigns {
		if senderID == nil || c.SenderID == *senderID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockRepo) GetCampaign(ctx context.Context, id uuid.UUID) (*model.Campaign, error) {
	for _, c := range m.campaigns {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) CreateCampaign(ctx context.Context, campaign *model.Campaign) error {
	campaign.ID = uuid.New()
	campaign.CreatedAt = time.Now()
	campaign.UpdatedAt = time.Now()
	m.campaigns = append(m.campaigns, *campaign)
	return nil
}

func (m *mockRepo) UpdateCampaign(ctx context.Context, campaign *model.Campaign) error {
	for i, c := range m.campaigns {
		if c.ID == campaign.ID {
			m.campaigns[i] = *campaign
			return nil
		}
	}
	return nil
}

func (m *mockRepo) GetDueCampaigns(ctx context.Context, asOf string, limit int, cursor *string) ([]model.DueCampaign, error) {
	var result []model.DueCampaign
	for _, c := range m.dueCampaigns {
		result = append(result, c)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (m *mockRepo) CreateAuditLog(ctx context.Context, log *model.AuditLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()
	m.auditLogs = append(m.auditLogs, *log)
	return nil
}

func (m *mockRepo) CreateRecurrenceRule(ctx context.Context, rrule *model.RecurrenceRule) error {
	rrule.ID = uuid.New()
	rrule.CreatedAt = time.Now()
	m.recurrenceRules = append(m.recurrenceRules, *rrule)
	return nil
}

type service struct {
	repo RepositoryInterface
}

func newService(repo RepositoryInterface) *service {
	return &service{repo: repo}
}

func (s *service) ListSenders(ctx context.Context) ([]model.Sender, error) {
	return s.repo.ListSenders(ctx)
}

func (s *service) CreateSender(ctx context.Context, req model.CreateSenderRequest) (*model.Sender, error) {
	if req.Name == "" {
		return nil, ErrInvalidInput
	}

	sender := &model.Sender{
		Name:        req.Name,
		Description: req.Description,
		Active:      true,
	}

	if err := s.repo.CreateSender(ctx, sender); err != nil {
		return nil, err
	}

	s.repo.CreateAuditLog(ctx, &model.AuditLog{
		EntityType: "sender",
		EntityID:   sender.ID,
		Action:     "create",
	})

	return sender, nil
}

func (s *service) AddOperator(ctx context.Context, senderID uuid.UUID, req model.AddOperatorRequest) (*model.SenderOperator, error) {
	sender, err := s.repo.GetSender(ctx, senderID)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return op, nil
}

func (s *service) AddCredential(ctx context.Context, senderID uuid.UUID, req model.AddCredentialRequest) (*model.SenderCredential, error) {
	sender, err := s.repo.GetSender(ctx, senderID)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return cred, nil
}

func (s *service) ResolveCredential(ctx context.Context, req model.ResolveCredentialRequest) (*model.CredentialResolution, error) {
	return s.repo.ResolveCredential(ctx, req.IntegrationKey)
}

func (s *service) CheckOperatorPermission(ctx context.Context, senderID, userID uuid.UUID) (*model.OperatorPermission, error) {
	return s.repo.CheckOperatorPermission(ctx, senderID, userID)
}

func (s *service) ListTemplates(ctx context.Context, senderID *uuid.UUID) ([]model.Template, error) {
	return s.repo.ListTemplates(ctx, senderID)
}

func (s *service) GetTemplate(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	return s.repo.GetTemplate(ctx, id)
}

func (s *service) CreateTemplate(ctx context.Context, req model.CreateTemplateRequest) (*model.Template, error) {
	if req.Name == "" || req.Body == "" {
		return nil, ErrInvalidInput
	}

	sender, err := s.repo.GetSender(ctx, req.SenderID)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return tmpl, nil
}

func (s *service) UpdateTemplate(ctx context.Context, id uuid.UUID, req model.UpdateTemplateRequest) (*model.Template, error) {
	existing, err := s.repo.GetTemplate(ctx, id)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return s.repo.GetTemplate(ctx, id)
}

func (s *service) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.GetTemplate(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}

	return s.repo.DeleteTemplate(ctx, id)
}

func (s *service) ListGroups(ctx context.Context, senderID *uuid.UUID) ([]model.ContactGroup, error) {
	return s.repo.ListGroups(ctx, senderID)
}

func (s *service) GetGroup(ctx context.Context, id uuid.UUID) (*model.ContactGroup, error) {
	return s.repo.GetGroup(ctx, id)
}

func (s *service) CreateGroup(ctx context.Context, req model.CreateGroupRequest) (*model.ContactGroup, error) {
	if req.Name == "" {
		return nil, ErrInvalidInput
	}

	sender, err := s.repo.GetSender(ctx, req.SenderID)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return group, nil
}

func (s *service) UpdateGroup(ctx context.Context, id uuid.UUID, req model.UpdateGroupRequest) (*model.ContactGroup, error) {
	existing, err := s.repo.GetGroup(ctx, id)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return s.repo.GetGroup(ctx, id)
}

func (s *service) AddGroupMembers(ctx context.Context, groupID uuid.UUID, req model.AddGroupMembersRequest) error {
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return ErrNotFound
	}

	return s.repo.AddGroupMembers(ctx, groupID, req.ContactIDs)
}

func (s *service) GetGroupMembers(ctx context.Context, groupID uuid.UUID) (*model.ContactGroupMembers, error) {
	group, err := s.repo.GetGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrNotFound
	}

	contactIDs, err := s.repo.GetGroupMembers(ctx, groupID)
	if err != nil {
		return nil, err
	}

	return &model.ContactGroupMembers{
		GroupID:    groupID,
		ContactIDs: contactIDs,
	}, nil
}

func (s *service) ListCampaigns(ctx context.Context, senderID *uuid.UUID) ([]model.Campaign, error) {
	return s.repo.ListCampaigns(ctx, senderID)
}

func (s *service) GetCampaign(ctx context.Context, id uuid.UUID) (*model.CampaignDetail, error) {
	campaign, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
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

func (s *service) CreateCampaign(ctx context.Context, req model.CreateCampaignRequest) (*model.Campaign, error) {
	if req.Name == "" {
		return nil, ErrInvalidInput
	}

	sender, err := s.repo.GetSender(ctx, req.SenderID)
	if err != nil {
		return nil, err
	}
	if sender == nil {
		return nil, ErrNotFound
	}

	group, err := s.repo.GetGroup(ctx, req.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrNotFound
	}

	_, err = s.repo.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		return nil, err
	}

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
			return nil, err
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

	return campaign, nil
}

func (s *service) UpdateCampaign(ctx context.Context, id uuid.UUID, req model.UpdateCampaignRequest) (*model.CampaignDetail, error) {
	existing, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return s.GetCampaign(ctx, id)
}

func (s *service) GetDueCampaigns(ctx context.Context, asOf string, limit int, cursor *string) ([]model.DueCampaign, string, error) {
	campaigns, err := s.repo.GetDueCampaigns(ctx, asOf, limit, cursor)
	if err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(campaigns) > 0 {
		lastCampaign := campaigns[len(campaigns)-1]
		nextCursor = lastCampaign.CampaignID.String()
	}

	return campaigns, nextCursor, nil
}

func TestListSenders(t *testing.T) {
	tests := []struct {
		name    string
		senders []model.Sender
		wantLen int
		wantErr bool
	}{
		{
			name: "success",
			senders: []model.Sender{
				{ID: uuid.New(), Name: "Sender1"},
				{ID: uuid.New(), Name: "Sender2"},
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "empty list",
			senders: []model.Sender{},
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			repo.senders = tt.senders
			svc := newService(repo)

			result, err := svc.ListSenders(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("ListSenders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(result) != tt.wantLen {
				t.Errorf("ListSenders() got %v, want %v", len(result), tt.wantLen)
			}
		})
	}
}

func TestCreateSender(t *testing.T) {
	senderID := uuid.New()

	tests := []struct {
		name    string
		req     model.CreateSenderRequest
		wantErr bool
		errType error
	}{
		{
			name: "success",
			req: model.CreateSenderRequest{
				Name:        "Test Sender",
				Description: "Test Description",
			},
			wantErr: false,
		},
		{
			name: "invalid input - empty name",
			req: model.CreateSenderRequest{
				Name: "",
			},
			wantErr: true,
			errType: ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			repo.senders = []model.Sender{{ID: senderID, Name: "Existing"}}
			svc := newService(repo)

			_, err := svc.CreateSender(context.Background(), tt.req)

			if tt.wantErr {
				if err != tt.errType {
					t.Errorf("CreateSender() error = %v, wantErr %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("CreateSender() unexpected error = %v", err)
			}
		})
	}
}

func TestAddOperator(t *testing.T) {
	senderID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name      string
		senderID  uuid.UUID
		req       model.AddOperatorRequest
		setupRepo func(*mockRepo)
		wantErr   bool
		errType   error
	}{
		{
			name:     "success",
			senderID: senderID,
			req: model.AddOperatorRequest{
				UserID: userID,
				Role:   "source_operator",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
			},
			wantErr: false,
		},
		{
			name:     "sender not found",
			senderID: uuid.New(),
			req: model.AddOperatorRequest{
				UserID: userID,
				Role:   "source_operator",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: uuid.New(), Name: "Test"}}
			},
			wantErr: true,
			errType: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newService(repo)

			_, err := svc.AddOperator(context.Background(), tt.senderID, tt.req)

			if tt.wantErr {
				if err != tt.errType {
					t.Errorf("AddOperator() error = %v, wantErr %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("AddOperator() unexpected error = %v", err)
			}
		})
	}
}

func TestAddCredential(t *testing.T) {
	senderID := uuid.New()

	tests := []struct {
		name      string
		senderID  uuid.UUID
		req       model.AddCredentialRequest
		setupRepo func(*mockRepo)
		wantErr   bool
		errType   error
	}{
		{
			name:     "success",
			senderID: senderID,
			req: model.AddCredentialRequest{
				IntegrationKey: "key123",
				Name:           "Test Cred",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
			},
			wantErr: false,
		},
		{
			name:     "sender not found",
			senderID: uuid.New(),
			req: model.AddCredentialRequest{
				IntegrationKey: "key123",
				Name:           "Test Cred",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: uuid.New(), Name: "Test"}}
			},
			wantErr: true,
			errType: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newService(repo)

			_, err := svc.AddCredential(context.Background(), tt.senderID, tt.req)

			if tt.wantErr {
				if err != tt.errType {
					t.Errorf("AddCredential() error = %v, wantErr %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("AddCredential() unexpected error = %v", err)
			}
		})
	}
}

func TestResolveCredential(t *testing.T) {
	tests := []struct {
		name           string
		integrationKey string
		setupRepo      func(*mockRepo)
		wantErr        bool
		wantSenderID   bool
	}{
		{
			name:           "success",
			integrationKey: "key123",
			setupRepo: func(r *mockRepo) {
				r.credentials = []model.SenderCredential{
					{ID: uuid.New(), SenderID: uuid.New(), IntegrationKey: "key123", Active: true},
				}
			},
			wantErr:      false,
			wantSenderID: true,
		},
		{
			name:           "credential not found",
			integrationKey: "nonexistent",
			setupRepo: func(r *mockRepo) {
				r.credentials = []model.SenderCredential{}
			},
			wantErr:      false,
			wantSenderID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newService(repo)

			result, err := svc.ResolveCredential(context.Background(), model.ResolveCredentialRequest{IntegrationKey: tt.integrationKey})

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveCredential() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantSenderID && result == nil {
				t.Error("ResolveCredential() expected sender ID, got nil")
			}
			if !tt.wantSenderID && result != nil {
				t.Error("ResolveCredential() expected nil, got result")
			}
		})
	}
}

func TestCheckOperatorPermission(t *testing.T) {
	senderID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name      string
		senderID  uuid.UUID
		userID    uuid.UUID
		setupRepo func(*mockRepo)
		wantAllow bool
	}{
		{
			name:     "has permission",
			senderID: senderID,
			userID:   userID,
			setupRepo: func(r *mockRepo) {
				r.operators = []model.SenderOperator{
					{SenderID: senderID, UserID: userID, Role: "source_operator"},
				}
			},
			wantAllow: true,
		},
		{
			name:     "no permission",
			senderID: senderID,
			userID:   uuid.New(),
			setupRepo: func(r *mockRepo) {
				r.operators = []model.SenderOperator{
					{SenderID: senderID, UserID: uuid.New(), Role: "source_operator"},
				}
			},
			wantAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newService(repo)

			result, err := svc.CheckOperatorPermission(context.Background(), tt.senderID, tt.userID)

			if err != nil {
				t.Errorf("CheckOperatorPermission() unexpected error = %v", err)
				return
			}
			if result.Allowed != tt.wantAllow {
				t.Errorf("CheckOperatorPermission() = %v, want %v", result.Allowed, tt.wantAllow)
			}
		})
	}
}

func TestListTemplates(t *testing.T) {
	senderID := uuid.New()

	tests := []struct {
		name      string
		senderID  *uuid.UUID
		templates []model.Template
		wantLen   int
		wantErr   bool
	}{
		{
			name:      "success",
			senderID:  nil,
			templates: []model.Template{{ID: uuid.New()}, {ID: uuid.New()}},
			wantLen:   2,
			wantErr:   false,
		},
		{
			name:      "empty list",
			senderID:  nil,
			templates: []model.Template{},
			wantLen:   0,
			wantErr:   false,
		},
		{
			name:      "filtered by sender",
			senderID:  &senderID,
			templates: []model.Template{{ID: uuid.New(), SenderID: senderID}, {ID: uuid.New(), SenderID: uuid.New()}},
			wantLen:   1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			repo.templates = tt.templates
			svc := newService(repo)

			result, err := svc.ListTemplates(context.Background(), tt.senderID)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListTemplates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(result) != tt.wantLen {
				t.Errorf("ListTemplates() got %v, want %v", len(result), tt.wantLen)
			}
		})
	}
}

func TestGetTemplate(t *testing.T) {
	templateID := uuid.New()

	tests := []struct {
		name       string
		templateID uuid.UUID
		templates  []model.Template
		wantErr    bool
		wantNil    bool
	}{
		{
			name:       "success",
			templateID: templateID,
			templates:  []model.Template{{ID: templateID, Name: "Test"}},
			wantErr:    false,
			wantNil:    false,
		},
		{
			name:       "not found",
			templateID: uuid.New(),
			templates:  []model.Template{{ID: templateID, Name: "Test"}},
			wantErr:    false,
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			repo.templates = tt.templates
			svc := newService(repo)

			result, err := svc.GetTemplate(context.Background(), tt.templateID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantNil && result != nil {
				t.Error("GetTemplate() expected nil")
			}
			if !tt.wantNil && result == nil {
				t.Error("GetTemplate() expected non-nil")
			}
		})
	}
}

func TestCreateTemplate(t *testing.T) {
	senderID := uuid.New()

	tests := []struct {
		name      string
		req       model.CreateTemplateRequest
		setupRepo func(*mockRepo)
		wantErr   bool
		errType   error
	}{
		{
			name: "success",
			req: model.CreateTemplateRequest{
				SenderID: senderID,
				Name:     "Test Template",
				Channel:  "email",
				Body:     "Test body",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
			},
			wantErr: false,
		},
		{
			name: "invalid input - empty name",
			req: model.CreateTemplateRequest{
				SenderID: senderID,
				Name:     "",
				Channel:  "email",
				Body:     "Test body",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
			},
			wantErr: true,
			errType: ErrInvalidInput,
		},
		{
			name: "invalid input - empty body",
			req: model.CreateTemplateRequest{
				SenderID: senderID,
				Name:     "Test",
				Channel:  "email",
				Body:     "",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
			},
			wantErr: true,
			errType: ErrInvalidInput,
		},
		{
			name: "sender not found",
			req: model.CreateTemplateRequest{
				SenderID: uuid.New(),
				Name:     "Test Template",
				Channel:  "email",
				Body:     "Test body",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
			},
			wantErr: true,
			errType: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newService(repo)

			_, err := svc.CreateTemplate(context.Background(), tt.req)

			if tt.wantErr {
				if err != tt.errType {
					t.Errorf("CreateTemplate() error = %v, wantErr %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("CreateTemplate() unexpected error = %v", err)
			}
		})
	}
}

func TestUpdateTemplate(t *testing.T) {
	templateID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		req       model.UpdateTemplateRequest
		setupRepo func(*mockRepo)
		wantErr   bool
		errType   error
	}{
		{
			name: "success",
			id:   templateID,
			req: model.UpdateTemplateRequest{
				Name:    "Updated",
				Channel: "email",
				Body:    "Updated body",
				Active:  true,
			},
			setupRepo: func(r *mockRepo) {
				r.templates = []model.Template{{ID: templateID, Name: "Test"}}
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   uuid.New(),
			req: model.UpdateTemplateRequest{
				Name:    "Updated",
				Channel: "email",
				Body:    "Updated body",
				Active:  true,
			},
			setupRepo: func(r *mockRepo) {
				r.templates = []model.Template{{ID: templateID, Name: "Test"}}
			},
			wantErr: true,
			errType: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newService(repo)

			_, err := svc.UpdateTemplate(context.Background(), tt.id, tt.req)

			if tt.wantErr {
				if err != tt.errType {
					t.Errorf("UpdateTemplate() error = %v, wantErr %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("UpdateTemplate() unexpected error = %v", err)
			}
		})
	}
}

func TestDeleteTemplate(t *testing.T) {
	templateID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		setupRepo func(*mockRepo)
		wantErr   bool
		errType   error
	}{
		{
			name: "success",
			id:   templateID,
			setupRepo: func(r *mockRepo) {
				r.templates = []model.Template{{ID: templateID, Name: "Test"}}
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   uuid.New(),
			setupRepo: func(r *mockRepo) {
				r.templates = []model.Template{{ID: templateID, Name: "Test"}}
			},
			wantErr: true,
			errType: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newService(repo)

			err := svc.DeleteTemplate(context.Background(), tt.id)

			if tt.wantErr {
				if err != tt.errType {
					t.Errorf("DeleteTemplate() error = %v, wantErr %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("DeleteTemplate() unexpected error = %v", err)
			}
		})
	}
}

func TestListGroups(t *testing.T) {
	senderID := uuid.New()

	tests := []struct {
		name     string
		senderID *uuid.UUID
		groups   []model.ContactGroup
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "success",
			senderID: nil,
			groups:   []model.ContactGroup{{ID: uuid.New()}, {ID: uuid.New()}},
			wantLen:  2,
			wantErr:  false,
		},
		{
			name:     "empty list",
			senderID: nil,
			groups:   []model.ContactGroup{},
			wantLen:  0,
			wantErr:  false,
		},
		{
			name:     "filtered by sender",
			senderID: &senderID,
			groups:   []model.ContactGroup{{ID: uuid.New(), SenderID: senderID}, {ID: uuid.New(), SenderID: uuid.New()}},
			wantLen:  1,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			repo.senderGroups = tt.groups
			svc := newService(repo)

			result, err := svc.ListGroups(context.Background(), tt.senderID)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListGroups() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(result) != tt.wantLen {
				t.Errorf("ListGroups() got %v, want %v", len(result), tt.wantLen)
			}
		})
	}
}

func TestGetGroup(t *testing.T) {
	groupID := uuid.New()

	tests := []struct {
		name    string
		groupID uuid.UUID
		groups  []model.ContactGroup
		wantErr bool
		wantNil bool
	}{
		{
			name:    "success",
			groupID: groupID,
			groups:  []model.ContactGroup{{ID: groupID, Name: "Test"}},
			wantErr: false,
			wantNil: false,
		},
		{
			name:    "not found",
			groupID: uuid.New(),
			groups:  []model.ContactGroup{{ID: groupID, Name: "Test"}},
			wantErr: false,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			repo.senderGroups = tt.groups
			svc := newService(repo)

			result, err := svc.GetGroup(context.Background(), tt.groupID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantNil && result != nil {
				t.Error("GetGroup() expected nil")
			}
			if !tt.wantNil && result == nil {
				t.Error("GetGroup() expected non-nil")
			}
		})
	}
}

func TestCreateGroup(t *testing.T) {
	senderID := uuid.New()

	tests := []struct {
		name      string
		req       model.CreateGroupRequest
		setupRepo func(*mockRepo)
		wantErr   bool
		errType   error
	}{
		{
			name: "success",
			req: model.CreateGroupRequest{
				SenderID:    senderID,
				Name:        "Test Group",
				Description: "Test Description",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
			},
			wantErr: false,
		},
		{
			name: "invalid input - empty name",
			req: model.CreateGroupRequest{
				SenderID: senderID,
				Name:     "",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
			},
			wantErr: true,
			errType: ErrInvalidInput,
		},
		{
			name: "sender not found",
			req: model.CreateGroupRequest{
				SenderID: uuid.New(),
				Name:     "Test Group",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
			},
			wantErr: true,
			errType: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newService(repo)

			_, err := svc.CreateGroup(context.Background(), tt.req)

			if tt.wantErr {
				if err != tt.errType {
					t.Errorf("CreateGroup() error = %v, wantErr %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("CreateGroup() unexpected error = %v", err)
			}
		})
	}
}

func TestUpdateGroup(t *testing.T) {
	groupID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		req       model.UpdateGroupRequest
		setupRepo func(*mockRepo)
		wantErr   bool
		errType   error
	}{
		{
			name: "success",
			id:   groupID,
			req: model.UpdateGroupRequest{
				Name:        "Updated",
				Description: "Updated desc",
			},
			setupRepo: func(r *mockRepo) {
				r.senderGroups = []model.ContactGroup{{ID: groupID, Name: "Test"}}
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   uuid.New(),
			req: model.UpdateGroupRequest{
				Name:        "Updated",
				Description: "Updated desc",
			},
			setupRepo: func(r *mockRepo) {
				r.senderGroups = []model.ContactGroup{{ID: groupID, Name: "Test"}}
			},
			wantErr: true,
			errType: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newService(repo)

			_, err := svc.UpdateGroup(context.Background(), tt.id, tt.req)

			if tt.wantErr {
				if err != tt.errType {
					t.Errorf("UpdateGroup() error = %v, wantErr %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("UpdateGroup() unexpected error = %v", err)
			}
		})
	}
}

func TestAddGroupMembers(t *testing.T) {
	groupID := uuid.New()
	contactID := uuid.New()

	tests := []struct {
		name      string
		groupID   uuid.UUID
		req       model.AddGroupMembersRequest
		setupRepo func(*mockRepo)
		wantErr   bool
		errType   error
	}{
		{
			name:    "success",
			groupID: groupID,
			req: model.AddGroupMembersRequest{
				ContactIDs: []uuid.UUID{contactID},
			},
			setupRepo: func(r *mockRepo) {
				r.senderGroups = []model.ContactGroup{{ID: groupID, Name: "Test"}}
			},
			wantErr: false,
		},
		{
			name:    "group not found",
			groupID: uuid.New(),
			req: model.AddGroupMembersRequest{
				ContactIDs: []uuid.UUID{contactID},
			},
			setupRepo: func(r *mockRepo) {
				r.senderGroups = []model.ContactGroup{{ID: groupID, Name: "Test"}}
			},
			wantErr: true,
			errType: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newService(repo)

			err := svc.AddGroupMembers(context.Background(), tt.groupID, tt.req)

			if tt.wantErr {
				if err != tt.errType {
					t.Errorf("AddGroupMembers() error = %v, wantErr %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("AddGroupMembers() unexpected error = %v", err)
			}
		})
	}
}

func TestGetGroupMembers(t *testing.T) {
	groupID := uuid.New()
	contactID := uuid.New()

	tests := []struct {
		name      string
		groupID   uuid.UUID
		setupRepo func(*mockRepo)
		wantErr   bool
		errType   error
		wantLen   int
	}{
		{
			name:    "success",
			groupID: groupID,
			setupRepo: func(r *mockRepo) {
				r.senderGroups = []model.ContactGroup{{ID: groupID, Name: "Test"}}
				r.groupMembers = map[uuid.UUID][]uuid.UUID{groupID: {contactID}}
			},
			wantErr: false,
			wantLen: 1,
		},
		{
			name:    "group not found",
			groupID: uuid.New(),
			setupRepo: func(r *mockRepo) {
				r.senderGroups = []model.ContactGroup{{ID: groupID, Name: "Test"}}
			},
			wantErr: true,
			errType: ErrNotFound,
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newService(repo)

			result, err := svc.GetGroupMembers(context.Background(), tt.groupID)

			if tt.wantErr {
				if err != tt.errType {
					t.Errorf("GetGroupMembers() error = %v, wantErr %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("GetGroupMembers() unexpected error = %v", err)
			} else if len(result.ContactIDs) != tt.wantLen {
				t.Errorf("GetGroupMembers() got %v, want %v", len(result.ContactIDs), tt.wantLen)
			}
		})
	}
}

func TestListCampaigns(t *testing.T) {
	senderID := uuid.New()

	tests := []struct {
		name      string
		senderID  *uuid.UUID
		campaigns []model.Campaign
		wantLen   int
		wantErr   bool
	}{
		{
			name:      "success",
			senderID:  nil,
			campaigns: []model.Campaign{{ID: uuid.New()}, {ID: uuid.New()}},
			wantLen:   2,
			wantErr:   false,
		},
		{
			name:      "empty list",
			senderID:  nil,
			campaigns: []model.Campaign{},
			wantLen:   0,
			wantErr:   false,
		},
		{
			name:      "filtered by sender",
			senderID:  &senderID,
			campaigns: []model.Campaign{{ID: uuid.New(), SenderID: senderID}, {ID: uuid.New(), SenderID: uuid.New()}},
			wantLen:   1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			repo.campaigns = tt.campaigns
			svc := newService(repo)

			result, err := svc.ListCampaigns(context.Background(), tt.senderID)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListCampaigns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(result) != tt.wantLen {
				t.Errorf("ListCampaigns() got %v, want %v", len(result), tt.wantLen)
			}
		})
	}
}

func TestGetCampaign(t *testing.T) {
	campaignID := uuid.New()

	tests := []struct {
		name       string
		campaignID uuid.UUID
		campaigns  []model.Campaign
		wantErr    bool
		errType    error
	}{
		{
			name:       "success",
			campaignID: campaignID,
			campaigns:  []model.Campaign{{ID: campaignID, Name: "Test"}},
			wantErr:    false,
		},
		{
			name:       "not found",
			campaignID: uuid.New(),
			campaigns:  []model.Campaign{{ID: campaignID, Name: "Test"}},
			wantErr:    true,
			errType:    ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			repo.campaigns = tt.campaigns
			svc := newService(repo)

			_, err := svc.GetCampaign(context.Background(), tt.campaignID)

			if tt.wantErr {
				if err != tt.errType {
					t.Errorf("GetCampaign() error = %v, wantErr %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("GetCampaign() unexpected error = %v", err)
			}
		})
	}
}

func TestCreateCampaign(t *testing.T) {
	senderID := uuid.New()
	groupID := uuid.New()
	templateID := uuid.New()

	tests := []struct {
		name      string
		req       model.CreateCampaignRequest
		setupRepo func(*mockRepo)
		wantErr   bool
		errType   error
	}{
		{
			name: "success",
			req: model.CreateCampaignRequest{
				SenderID:   senderID,
				GroupID:    groupID,
				TemplateID: templateID,
				Name:       "Test Campaign",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
				r.senderGroups = []model.ContactGroup{{ID: groupID, Name: "Test"}}
				r.templates = []model.Template{{ID: templateID, Channel: "email"}}
			},
			wantErr: false,
		},
		{
			name: "invalid input - empty name",
			req: model.CreateCampaignRequest{
				SenderID:   senderID,
				GroupID:    groupID,
				TemplateID: templateID,
				Name:       "",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
				r.senderGroups = []model.ContactGroup{{ID: groupID, Name: "Test"}}
				r.templates = []model.Template{{ID: templateID, Channel: "email"}}
			},
			wantErr: true,
			errType: ErrInvalidInput,
		},
		{
			name: "sender not found",
			req: model.CreateCampaignRequest{
				SenderID:   uuid.New(),
				GroupID:    groupID,
				TemplateID: templateID,
				Name:       "Test Campaign",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
				r.senderGroups = []model.ContactGroup{{ID: groupID, Name: "Test"}}
				r.templates = []model.Template{{ID: templateID, Channel: "email"}}
			},
			wantErr: true,
			errType: ErrNotFound,
		},
		{
			name: "group not found",
			req: model.CreateCampaignRequest{
				SenderID:   senderID,
				GroupID:    uuid.New(),
				TemplateID: templateID,
				Name:       "Test Campaign",
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
				r.senderGroups = []model.ContactGroup{{ID: groupID, Name: "Test"}}
				r.templates = []model.Template{{ID: templateID, Channel: "email"}}
			},
			wantErr: true,
			errType: ErrNotFound,
		},
		{
			name: "with recurrence",
			req: model.CreateCampaignRequest{
				SenderID:   senderID,
				GroupID:    groupID,
				TemplateID: templateID,
				Name:       "Test Campaign",
				RecurrenceRule: &model.RecurrenceRuleRequest{
					Kind: "daily",
				},
			},
			setupRepo: func(r *mockRepo) {
				r.senders = []model.Sender{{ID: senderID, Name: "Test"}}
				r.senderGroups = []model.ContactGroup{{ID: groupID, Name: "Test"}}
				r.templates = []model.Template{{ID: templateID, Channel: "email"}}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newService(repo)

			_, err := svc.CreateCampaign(context.Background(), tt.req)

			if tt.wantErr {
				if err != tt.errType {
					t.Errorf("CreateCampaign() error = %v, wantErr %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("CreateCampaign() unexpected error = %v", err)
			}
		})
	}
}

func TestUpdateCampaign(t *testing.T) {
	campaignID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		req       model.UpdateCampaignRequest
		setupRepo func(*mockRepo)
		wantErr   bool
		errType   error
	}{
		{
			name: "success",
			id:   campaignID,
			req: model.UpdateCampaignRequest{
				Name:     "Updated",
				Channels: []string{"email"},
				Active:   true,
			},
			setupRepo: func(r *mockRepo) {
				r.campaigns = []model.Campaign{{ID: campaignID, Name: "Test"}}
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   uuid.New(),
			req: model.UpdateCampaignRequest{
				Name:     "Updated",
				Channels: []string{"email"},
				Active:   true,
			},
			setupRepo: func(r *mockRepo) {
				r.campaigns = []model.Campaign{{ID: campaignID, Name: "Test"}}
			},
			wantErr: true,
			errType: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}
			svc := newService(repo)

			_, err := svc.UpdateCampaign(context.Background(), tt.id, tt.req)

			if tt.wantErr {
				if err != tt.errType {
					t.Errorf("UpdateCampaign() error = %v, wantErr %v", err, tt.errType)
				}
			} else if err != nil {
				t.Errorf("UpdateCampaign() unexpected error = %v", err)
			}
		})
	}
}

func TestGetDueCampaigns(t *testing.T) {
	campaignID := uuid.New()
	senderID := uuid.New()
	groupID := uuid.New()
	templateID := uuid.New()

	tests := []struct {
		name         string
		limit        int
		dueCampaigns []model.DueCampaign
		wantLen      int
		wantErr      bool
	}{
		{
			name:  "success",
			limit: 10,
			dueCampaigns: []model.DueCampaign{
				{CampaignID: campaignID, SenderID: senderID, GroupID: groupID, TemplateID: templateID},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:         "empty",
			limit:        10,
			dueCampaigns: []model.DueCampaign{},
			wantLen:      0,
			wantErr:      false,
		},
		{
			name:  "pagination",
			limit: 1,
			dueCampaigns: []model.DueCampaign{
				{CampaignID: campaignID, SenderID: senderID, GroupID: groupID, TemplateID: templateID},
				{CampaignID: uuid.New(), SenderID: senderID, GroupID: groupID, TemplateID: templateID},
			},
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			repo.dueCampaigns = tt.dueCampaigns
			svc := newService(repo)

			result, cursor, err := svc.GetDueCampaigns(context.Background(), "2024-01-01", tt.limit, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDueCampaigns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(result) != tt.wantLen {
				t.Errorf("GetDueCampaigns() got %v, want %v", len(result), tt.wantLen)
			}
			if tt.wantLen > 0 && cursor == "" {
				t.Error("GetDueCampaigns() expected cursor for non-empty result")
			}
		})
	}
}
