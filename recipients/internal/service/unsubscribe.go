package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"notification-center/recipients/internal/domain"
	"notification-center/recipients/internal/repository"
)

var (
	ErrUnsubscribeTokenInvalid = errors.New("недействительный токен отписки")
)

func parseRFC3339(s string) (time.Time, error) {
	if s == "" {
		return time.Now().UTC(), nil
	}
	return time.Parse(time.RFC3339, s)
}

// UnsubscribeService handles unsubscribe operations
type UnsubscribeService struct {
	unsubscribeTokenRepo *repository.UnsubscribeTokenRepository
	unsubscribeRuleRepo  *repository.UnsubscribeRuleRepository
	contactRepo          *repository.ContactRepository
	auditRepo            *repository.AuditLogRepository
}

func NewUnsubscribeService(
	unsubscribeTokenRepo *repository.UnsubscribeTokenRepository,
	unsubscribeRuleRepo *repository.UnsubscribeRuleRepository,
	contactRepo *repository.ContactRepository,
	auditRepo *repository.AuditLogRepository,
) *UnsubscribeService {
	return &UnsubscribeService{
		unsubscribeTokenRepo: unsubscribeTokenRepo,
		unsubscribeRuleRepo:  unsubscribeRuleRepo,
		contactRepo:          contactRepo,
		auditRepo:            auditRepo,
	}
}

type UnsubscribeRequest struct {
	Token         string `json:"token" binding:"required"`
	SelectedScope string `json:"selected_scope" binding:"required,oneof=sender campaign_or_group"`
}

func (s *UnsubscribeService) Unsubscribe(ctx context.Context, req *UnsubscribeRequest) error {
	// Get token info
	token, err := s.unsubscribeTokenRepo.GetByToken(ctx, req.Token)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrUnsubscribeTokenInvalid
	}
	if err != nil {
		return fmt.Errorf("failed to get unsubscribe token: %w", err)
	}

	// Verify selected_scope is allowed by token
	if req.SelectedScope == "sender" && token.ScopeType != "sender" {
		return errors.New("токен не разрешает отписку от отправителя")
	}
	if req.SelectedScope == "campaign_or_group" && token.ScopeType != "campaign" && token.ScopeType != "group" {
		return errors.New("токен не разрешает отписку от рассылки")
	}

	// Check if already unsubscribed
	exists, err := s.unsubscribeRuleRepo.ExistsByContactAndScope(ctx, token.ContactID, token.ScopeType, token.SenderID, token.ScopeID)
	if err != nil {
		return fmt.Errorf("failed to check unsubscribe rule: %w", err)
	}
	if exists {
		// Already unsubscribed, return success
		return nil
	}

	// Create unsubscribe rule
	rule := &domain.UnsubscribeRule{
		ContactID: token.ContactID,
		SenderID:  token.SenderID,
		ScopeType: token.ScopeType,
		ScopeID:   token.ScopeID,
	}

	if err := s.unsubscribeRuleRepo.Create(ctx, rule); err != nil {
		return fmt.Errorf("failed to create unsubscribe rule: %w", err)
	}

	// Create audit log
	contact, _ := s.contactRepo.GetByID(ctx, token.ContactID)
	userID := ""
	if contact != nil {
		userID = contact.UserID
	}

	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  userID,
		Action:  "unsubscribe",
		Details: fmt.Sprintf("unsubscribed from %s scope", token.ScopeType),
	})

	return nil
}

// InternalResolveService handles internal resolution for Track 4
type InternalResolveService struct {
	contactRepo         *repository.ContactRepository
	prefService         *PreferenceService
	unsubscribeRuleRepo *repository.UnsubscribeRuleRepository
}

func NewInternalResolveService(
	contactRepo *repository.ContactRepository,
	prefService *PreferenceService,
	unsubscribeRuleRepo *repository.UnsubscribeRuleRepository,
) *InternalResolveService {
	return &InternalResolveService{
		contactRepo:         contactRepo,
		prefService:         prefService,
		unsubscribeRuleRepo: unsubscribeRuleRepo,
	}
}

type ResolveContactRequest struct {
	SenderID          string   `json:"sender_id"`
	CampaignID        *string  `json:"campaign_id"`
	GroupID           *string  `json:"group_id"`
	ContactIDs        []string `json:"contact_ids"`
	RequestedChannels []string `json:"requested_channels"`
	EvaluateAt        string   `json:"evaluate_at"`
}

type ResolveContactResponse struct {
	Contacts []ResolveContact `json:"contacts"`
}

type ResolveContact struct {
	ContactID  string      `json:"contact_id"`
	UserID     string      `json:"user_id"`
	Candidates []Candidate `json:"candidates"`
}

type Candidate struct {
	Channel       string  `json:"channel"`
	Value         string  `json:"value"`
	Allowed       bool    `json:"allowed"`
	BlockedReason *string `json:"blocked_reason,omitempty"`
}

func (s *InternalResolveService) Resolve(ctx context.Context, req *ResolveContactRequest) (*ResolveContactResponse, error) {
	// Parse evaluate_at time
	evaluateAt, err := parseRFC3339(req.EvaluateAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse evaluate_at: %w", err)
	}

	// Get contacts by IDs
	contacts, err := s.contactRepo.GetByIDs(ctx, req.ContactIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	// Determine scope
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

		// If specific channels requested, only check those
		channels := req.RequestedChannels
		if len(channels) == 0 {
			channels = []string{"email", "sms", "telegram"}
		}

		for _, channel := range channels {
			allowed := true
			var blockedReason *string

			// Check unsubscribe rules
			unsubscribed, err := s.unsubscribeRuleRepo.ExistsByContactAndScope(ctx, contact.ID, scopeType, senderID, scopeID)
			if err == nil && unsubscribed {
				reason := "пользователь отписался"
				blockedReason = &reason
				allowed = false
			}

			// Check preferences if still allowed
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

func (s *InternalResolveService) GetUserContacts(ctx context.Context, userID string) ([]domain.Contact, error) {
	return s.contactRepo.GetAllByUserID(ctx, userID)
}
