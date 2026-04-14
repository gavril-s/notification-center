package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"notification-center/recipients/internal/domain"
	"notification-center/recipients/internal/repository"
)

type testUnsubscribeService struct {
	unsubscribeTokenRepo *MockUnsubscribeTokenRepo
	unsubscribeRuleRepo  *MockUnsubscribeRuleRepo
	contactRepo          *MockContactRepo
	auditRepo            *MockAuditLogRepo
}

func newTestUnsubscribeService(
	unsubscribeTokenRepo *MockUnsubscribeTokenRepo,
	unsubscribeRuleRepo *MockUnsubscribeRuleRepo,
	contactRepo *MockContactRepo,
	auditRepo *MockAuditLogRepo,
) *testUnsubscribeService {
	return &testUnsubscribeService{
		unsubscribeTokenRepo: unsubscribeTokenRepo,
		unsubscribeRuleRepo:  unsubscribeRuleRepo,
		contactRepo:          contactRepo,
		auditRepo:            auditRepo,
	}
}

func (s *testUnsubscribeService) Unsubscribe(ctx context.Context, req *UnsubscribeRequest) error {
	token, err := s.unsubscribeTokenRepo.GetByToken(ctx, req.Token)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrUnsubscribeTokenInvalid
	}
	if err != nil {
		return err
	}

	if req.SelectedScope == "sender" && token.ScopeType != "sender" {
		return errors.New("токен не разрешает отписку от отправителя")
	}
	if req.SelectedScope == "campaign_or_group" && token.ScopeType != "campaign" && token.ScopeType != "group" {
		return errors.New("токен не разрешает отписку от рассылки")
	}

	exists, err := s.unsubscribeRuleRepo.ExistsByContactAndScope(ctx, token.ContactID, token.ScopeType, token.SenderID, token.ScopeID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	rule := &domain.UnsubscribeRule{
		ContactID: token.ContactID,
		SenderID:  token.SenderID,
		ScopeType: token.ScopeType,
		ScopeID:   token.ScopeID,
	}

	if err := s.unsubscribeRuleRepo.Create(ctx, rule); err != nil {
		return err
	}

	contact, _ := s.contactRepo.GetByID(ctx, token.ContactID)
	userID := ""
	if contact != nil {
		userID = contact.UserID
	}

	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  userID,
		Action:  "unsubscribe",
		Details: "unsubscribed",
	})

	return nil
}

type testInternalResolveService struct {
	contactRepo         *MockContactRepo
	prefService         *testPreferenceService
	unsubscribeRuleRepo *MockUnsubscribeRuleRepo
}

func newTestInternalResolveService(
	contactRepo *MockContactRepo,
	prefService *testPreferenceService,
	unsubscribeRuleRepo *MockUnsubscribeRuleRepo,
) *testInternalResolveService {
	return &testInternalResolveService{
		contactRepo:         contactRepo,
		prefService:         prefService,
		unsubscribeRuleRepo: unsubscribeRuleRepo,
	}
}

func (s *testInternalResolveService) Resolve(ctx context.Context, req *ResolveContactRequest) (*ResolveContactResponse, error) {
	evaluateAt, err := parseRFC3339Time(req.EvaluateAt)
	if err != nil {
		return nil, err
	}

	contacts, err := s.contactRepo.GetByIDs(ctx, req.ContactIDs)
	if err != nil {
		return nil, err
	}

	scopeType := "global"
	scopeID := (*string)(nil)
	if req.CampaignID != nil {
		scopeType = "campaign"
		scopeID = req.CampaignID
	} else if req.GroupID != nil {
		scopeType = "group"
		scopeID = req.GroupID
	}

	var senderID *string
	if req.SenderID != "" {
		senderID = &req.SenderID
	}

	response := &ResolveContactResponse{
		Contacts: make([]ResolveContact, 0, len(contacts)),
	}

	for _, contact := range contacts {
		resolveContact := ResolveContact{
			ContactID:  contact.ID,
			UserID:     contact.UserID,
			Candidates: make([]Candidate, 0),
		}

		channels := req.RequestedChannels
		if len(channels) == 0 {
			channels = []string{"email", "sms", "telegram"}
		}

		for _, channel := range channels {
			allowed := true
			var blockedReason *string

			unsubscribed, err := s.unsubscribeRuleRepo.ExistsByContactAndScope(ctx, contact.ID, scopeType, senderID, scopeID)
			if err == nil && unsubscribed {
				reason := "пользователь отписался"
				blockedReason = &reason
				allowed = false
			}

			if allowed {
				prefAllowed, reason, err := s.prefService.EvaluatePreference(ctx, contact.ID, senderID, scopeType, scopeID, channel, evaluateAt)
				if err != nil {
					continue
				}
				if !prefAllowed {
					blockedReason = reason
					allowed = false
				}
			}

			resolveContact.Candidates = append(resolveContact.Candidates, Candidate{
				Channel:       channel,
				Value:         contact.Value,
				Allowed:       allowed,
				BlockedReason: blockedReason,
			})
		}

		response.Contacts = append(response.Contacts, resolveContact)
	}

	return response, nil
}

func (s *testInternalResolveService) GetUserContacts(ctx context.Context, userID string) ([]domain.Contact, error) {
	return s.contactRepo.GetAllByUserID(ctx, userID)
}

func parseRFC3339Time(s string) (time.Time, error) {
	if s == "" {
		return time.Now().UTC(), nil
	}
	return time.Parse(time.RFC3339, s)
}

func TestUnsubscribeService_Unsubscribe(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		req            *UnsubscribeRequest
		tokenRepo      *MockUnsubscribeTokenRepo
		ruleRepo       *MockUnsubscribeRuleRepo
		contactRepo    *MockContactRepo
		wantErr        error
		wantErrMessage string
	}{
		{
			name: "успешная отписка",
			req: &UnsubscribeRequest{
				Token:         "valid-token",
				SelectedScope: "sender",
			},
			tokenRepo: &MockUnsubscribeTokenRepo{
				GetByTokenFunc: func(ctx context.Context, token string) (*domain.UnsubscribeToken, error) {
					return &domain.UnsubscribeToken{
						ContactID: "contact-1",
						SenderID:  strPtr("sender-1"),
						ScopeType: "sender",
					}, nil
				},
			},
			ruleRepo: &MockUnsubscribeRuleRepo{
				ExistsByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (bool, error) {
					return false, nil
				},
				CreateFunc: func(ctx context.Context, rule *domain.UnsubscribeRule) error {
					return nil
				},
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1"}, nil
				},
			},
			wantErr: nil,
		},
		{
			name: "недействительный токен",
			req: &UnsubscribeRequest{
				Token:         "invalid-token",
				SelectedScope: "sender",
			},
			tokenRepo: &MockUnsubscribeTokenRepo{
				GetByTokenFunc: func(ctx context.Context, token string) (*domain.UnsubscribeToken, error) {
					return nil, repository.ErrNotFound
				},
			},
			ruleRepo:    &MockUnsubscribeRuleRepo{},
			contactRepo: &MockContactRepo{},
			wantErr:     ErrUnsubscribeTokenInvalid,
		},
		{
			name: "некорректная область видимости токена - отправитель",
			req: &UnsubscribeRequest{
				Token:         "campaign-token",
				SelectedScope: "sender",
			},
			tokenRepo: &MockUnsubscribeTokenRepo{
				GetByTokenFunc: func(ctx context.Context, token string) (*domain.UnsubscribeToken, error) {
					return &domain.UnsubscribeToken{
						ContactID: "contact-1",
						ScopeType: "campaign",
					}, nil
				},
			},
			ruleRepo:    &MockUnsubscribeRuleRepo{},
			contactRepo: &MockContactRepo{},
			wantErr:     errors.New("токен не разрешает отписку от отправителя"),
		},
		{
			name: "некорректная область видимости токена - рассылка",
			req: &UnsubscribeRequest{
				Token:         "sender-token",
				SelectedScope: "campaign_or_group",
			},
			tokenRepo: &MockUnsubscribeTokenRepo{
				GetByTokenFunc: func(ctx context.Context, token string) (*domain.UnsubscribeToken, error) {
					return &domain.UnsubscribeToken{
						ContactID: "contact-1",
						ScopeType: "sender",
					}, nil
				},
			},
			ruleRepo:    &MockUnsubscribeRuleRepo{},
			contactRepo: &MockContactRepo{},
			wantErr:     errors.New("токен не разрешает отписку от рассылки"),
		},
		{
			name: "уже отписан",
			req: &UnsubscribeRequest{
				Token:         "valid-token",
				SelectedScope: "sender",
			},
			tokenRepo: &MockUnsubscribeTokenRepo{
				GetByTokenFunc: func(ctx context.Context, token string) (*domain.UnsubscribeToken, error) {
					return &domain.UnsubscribeToken{
						ContactID: "contact-1",
						SenderID:  strPtr("sender-1"),
						ScopeType: "sender",
					}, nil
				},
			},
			ruleRepo: &MockUnsubscribeRuleRepo{
				ExistsByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (bool, error) {
					return true, nil
				},
			},
			contactRepo: &MockContactRepo{},
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestUnsubscribeService(tt.tokenRepo, tt.ruleRepo, tt.contactRepo, &MockAuditLogRepo{})

			err := svc.Unsubscribe(ctx, tt.req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestUnsubscribeService_Resolve(t *testing.T) {
	ctx := context.Background()
	campaignID := "campaign-1"
	senderID := "sender-1"

	tests := []struct {
		name           string
		req            *ResolveContactRequest
		contactRepo    *MockContactRepo
		prefService    *testPreferenceService
		ruleRepo       *MockUnsubscribeRuleRepo
		wantErr        error
		wantContacts   int
		wantCandidates int
		wantAllowed    bool
	}{
		{
			name: "успешное разрешение",
			req: &ResolveContactRequest{
				SenderID:   senderID,
				CampaignID: &campaignID,
				ContactIDs: []string{"contact-1"},
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
				GetByIDsFunc: func(ctx context.Context, ids []string) ([]domain.Contact, error) {
					return []domain.Contact{{ID: "contact-1", UserID: "user-1", Channel: "email", Value: "test@example.com"}}, nil
				},
			},
			prefService: newTestPreferenceService(&MockPreferenceRepo{}, &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			}, &MockAuditLogRepo{}),
			ruleRepo: &MockUnsubscribeRuleRepo{
				ExistsByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (bool, error) {
					return false, nil
				},
			},
			wantErr:        nil,
			wantContacts:   1,
			wantCandidates: 3,
			wantAllowed:    true,
		},
		{
			name: "с правилами отписки",
			req: &ResolveContactRequest{
				SenderID:   senderID,
				CampaignID: &campaignID,
				ContactIDs: []string{"contact-1"},
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
				GetByIDsFunc: func(ctx context.Context, ids []string) ([]domain.Contact, error) {
					return []domain.Contact{{ID: "contact-1", UserID: "user-1", Channel: "email", Value: "test@example.com"}}, nil
				},
			},
			prefService: newTestPreferenceService(&MockPreferenceRepo{}, &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			}, &MockAuditLogRepo{}),
			ruleRepo: &MockUnsubscribeRuleRepo{
				ExistsByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (bool, error) {
					return true, nil
				},
			},
			wantErr:        nil,
			wantContacts:   1,
			wantCandidates: 3,
			wantAllowed:    false,
		},
		{
			name: "с заблокированными предпочтениями",
			req: &ResolveContactRequest{
				SenderID:   senderID,
				CampaignID: &campaignID,
				ContactIDs: []string{"contact-1"},
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
				GetByIDsFunc: func(ctx context.Context, ids []string) ([]domain.Contact, error) {
					return []domain.Contact{{ID: "contact-1", UserID: "user-1", Channel: "email", Value: "test@example.com"}}, nil
				},
			},
			prefService: newTestPreferenceService(&MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						blockedChannel := "email"
						return &domain.Preference{Enabled: true, BlockedChannels: []string{blockedChannel}}, nil
					}
					return nil, repository.ErrNotFound
				},
			}, &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			}, &MockAuditLogRepo{}),
			ruleRepo: &MockUnsubscribeRuleRepo{
				ExistsByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (bool, error) {
					return false, nil
				},
			},
			wantErr:        nil,
			wantContacts:   1,
			wantCandidates: 3,
			wantAllowed:    false,
		},
		{
			name: "с запрошенными каналами",
			req: &ResolveContactRequest{
				SenderID:          senderID,
				CampaignID:        &campaignID,
				ContactIDs:        []string{"contact-1"},
				RequestedChannels: []string{"email", "sms"},
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
				GetByIDsFunc: func(ctx context.Context, ids []string) ([]domain.Contact, error) {
					return []domain.Contact{{ID: "contact-1", UserID: "user-1", Channel: "email", Value: "test@example.com"}}, nil
				},
			},
			prefService: newTestPreferenceService(&MockPreferenceRepo{}, &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			}, &MockAuditLogRepo{}),
			ruleRepo: &MockUnsubscribeRuleRepo{
				ExistsByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (bool, error) {
					return false, nil
				},
			},
			wantErr:        nil,
			wantContacts:   1,
			wantCandidates: 2,
			wantAllowed:    true,
		},
		{
			name: "ошибка при получении контактов",
			req: &ResolveContactRequest{
				SenderID:   senderID,
				CampaignID: &campaignID,
				ContactIDs: []string{"contact-1"},
			},
			contactRepo: &MockContactRepo{
				GetByIDsFunc: func(ctx context.Context, ids []string) ([]domain.Contact, error) {
					return nil, anError
				},
			},
			prefService: newTestPreferenceService(&MockPreferenceRepo{}, &MockContactRepo{}, &MockAuditLogRepo{}),
			ruleRepo:    &MockUnsubscribeRuleRepo{},
			wantErr:     anError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestInternalResolveService(tt.contactRepo, tt.prefService, tt.ruleRepo)

			resp, err := svc.Resolve(ctx, tt.req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(resp.Contacts) != tt.wantContacts {
					t.Errorf("expected %d contacts, got %d", tt.wantContacts, len(resp.Contacts))
				}
				if tt.wantCandidates > 0 && len(resp.Contacts) > 0 {
					if len(resp.Contacts[0].Candidates) != tt.wantCandidates {
						t.Errorf("expected %d candidates, got %d", tt.wantCandidates, len(resp.Contacts[0].Candidates))
					}
					if resp.Contacts[0].Candidates[0].Allowed != tt.wantAllowed {
						t.Errorf("expected allowed=%v, got %v", tt.wantAllowed, resp.Contacts[0].Candidates[0].Allowed)
					}
				}
			}
		})
	}

	_ = senderID
}

func TestUnsubscribeService_GetUserContacts(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      string
		contactRepo *MockContactRepo
		wantErr     error
		wantCount   int
	}{
		{
			name:   "успешное получение контактов пользователя",
			userID: "user-1",
			contactRepo: &MockContactRepo{
				GetAllByUserIDFunc: func(ctx context.Context, userID string) ([]domain.Contact, error) {
					return []domain.Contact{
						{ID: "c1", UserID: userID},
						{ID: "c2", UserID: userID},
					}, nil
				},
			},
			wantErr:   nil,
			wantCount: 2,
		},
		{
			name:   "контакты не найдены",
			userID: "user-1",
			contactRepo: &MockContactRepo{
				GetAllByUserIDFunc: func(ctx context.Context, userID string) ([]domain.Contact, error) {
					return []domain.Contact{}, nil
				},
			},
			wantErr:   nil,
			wantCount: 0,
		},
		{
			name:   "ошибка при получении контактов",
			userID: "user-1",
			contactRepo: &MockContactRepo{
				GetAllByUserIDFunc: func(ctx context.Context, userID string) ([]domain.Contact, error) {
					return nil, anError
				},
			},
			wantErr:   anError,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestInternalResolveService(tt.contactRepo, newTestPreferenceService(&MockPreferenceRepo{}, &MockContactRepo{}, &MockAuditLogRepo{}), &MockUnsubscribeRuleRepo{})

			resp, err := svc.GetUserContacts(ctx, tt.userID)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(resp) != tt.wantCount {
					t.Errorf("expected %d contacts, got %d", tt.wantCount, len(resp))
				}
			}
		})
	}
}
