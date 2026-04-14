package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"

	"notification-center/notifications-service/internal/client"
	"notification-center/notifications-service/internal/domain"
	"notification-center/notifications-service/internal/repository"
)

var (
	ErrSenderNotFound     = errors.New("sender not found")
	ErrTemplateNotFound   = errors.New("template not found")
	ErrChannelMismatch    = errors.New("channel does not match template channel")
	ErrNoContactsProvided = errors.New("no contacts provided")
	ErrMultipleChannels   = errors.New("only one channel allowed per request")
	ErrInvalidIdempotency = errors.New("invalid idempotency key")
	ErrRecipientBlocked   = errors.New("recipient blocked by preferences")
	ErrUnsubscribeScope   = errors.New("invalid unsubscribe scope")
)

type NotificationService struct {
	repo              *repository.NotificationRepository
	outboxRepo        *repository.OutboxRepository
	historyRepo       *repository.HistoryRepository
	analyticsRepo     *repository.AnalyticsRepository
	recipientsClient  *client.HTTPClient
	sourcesClient     *client.HTTPClient
	unsubscribeSecret string
}

func NewNotificationService(
	repo *repository.NotificationRepository,
	outboxRepo *repository.OutboxRepository,
	historyRepo *repository.HistoryRepository,
	analyticsRepo *repository.AnalyticsRepository,
	recipientsURL, sourcesURL, unsubscribeSecret string,
) *NotificationService {
	return &NotificationService{
		repo:              repo,
		outboxRepo:        outboxRepo,
		historyRepo:       historyRepo,
		analyticsRepo:     analyticsRepo,
		recipientsClient:  client.NewHTTPClient(recipientsURL),
		sourcesClient:     client.NewHTTPClient(sourcesURL),
		unsubscribeSecret: unsubscribeSecret,
	}
}

type SendNotificationRequest struct {
	TemplateID     string           `json:"template_id"`
	CampaignID     *string          `json:"campaign_id"`
	ContactIDs     []string         `json:"contact_ids"`
	Channels       []domain.Channel `json:"channels"`
	Variables      map[string]any   `json:"variables"`
	ScheduledAt    *time.Time       `json:"scheduled_at"`
	IdempotencyKey string           `json:"idempotency_key"`
}

func (s *NotificationService) Send(ctx context.Context, req SendNotificationRequest, integrationKey, traceID string) ([]*domain.Notification, error) {
	// 1. Validate channel count
	if len(req.Channels) != 1 {
		return nil, ErrMultipleChannels
	}
	channel := req.Channels[0]

	// 2. Validate idempotency key
	if req.IdempotencyKey == "" {
		return nil, ErrInvalidIdempotency
	}

	// 3. Validate contacts
	if len(req.ContactIDs) == 0 {
		return nil, ErrNoContactsProvided
	}

	// 4. Resolve sender from integration key
	sender, err := s.sourcesClient.ResolveCredential(ctx, integrationKey, traceID)
	if err != nil {
		return nil, ErrSenderNotFound
	}

	// 5. Get template
	tmpl, err := s.sourcesClient.GetTemplate(ctx, req.TemplateID, traceID)
	if err != nil {
		return nil, ErrTemplateNotFound
	}

	// 6. Validate channel matches template
	if tmpl.Channel != string(channel) {
		return nil, ErrChannelMismatch
	}

	// 7. Render content
	rendered := RenderTemplate(tmpl.Subject, tmpl.Body, req.Variables)

	// 8. Get campaign info for unsubscribe URL
	var groupID *string
	if req.CampaignID != nil {
		campaign, err := s.sourcesClient.GetCampaign(ctx, *req.CampaignID, traceID)
		if err == nil {
			groupID = &campaign.GroupID
		}
	}

	// 9. Create notifications
	notifications := make([]*domain.Notification, 0, len(req.ContactIDs))
	for _, contactID := range req.ContactIDs {
		notificationID := uuid.New().String()
		now := time.Now().UTC()

		// Generate unsubscribe URL
		unsubscribeURL := s.generateUnsubscribeURL(contactID, sender.ID, req.CampaignID, groupID)

		// Determine status based on scheduled_at
		status := domain.StatusQueued
		if req.ScheduledAt != nil {
			status = domain.StatusScheduled
		}

		notification := &domain.Notification{
			ID:              notificationID,
			SenderID:        sender.ID,
			ContactID:       contactID,
			CampaignID:      req.CampaignID,
			GroupID:         groupID,
			TemplateID:      req.TemplateID,
			Channel:         channel,
			Status:          status,
			Subject:         &tmpl.Subject,
			RenderedContent: rendered,
			UnsubscribeURL:  &unsubscribeURL,
			IdempotencyKey:  req.IdempotencyKey + ":" + contactID,
			ScheduledAt:     req.ScheduledAt,
			CreatedAt:       now,
			UpdatedAt:       now,
			AttemptCount:    0,
			Metadata:        req.Variables,
		}

		// Create outbox event
		outbox := &domain.NotificationOutbox{
			ID:             uuid.New().String(),
			NotificationID: notificationID,
			EventType:      "notification.dispatch.v1",
			Payload: map[string]any{
				"event_id":         uuid.New().String(),
				"notification_id":  notificationID,
				"sender_id":        sender.ID,
				"contact_id":       contactID,
				"campaign_id":      req.CampaignID,
				"group_id":         groupID,
				"channel":          channel,
				"subject":          tmpl.Subject,
				"rendered_content": rendered,
				"unsubscribe_url":  unsubscribeURL,
				"metadata":         req.Variables,
				"attempt":          1,
				"trace_id":         traceID,
				"created_at":       now.Format(time.RFC3339),
			},
			Attempt:   1,
			Status:    "pending",
			CreatedAt: now,
		}

		// Save to database
		if err := s.repo.Create(ctx, notification); err != nil {
			return nil, fmt.Errorf("failed to create notification: %w", err)
		}
		if err := s.outboxRepo.Create(ctx, outbox); err != nil {
			return nil, fmt.Errorf("failed to create outbox entry: %w", err)
		}

		// Create history entry
		history := &domain.NotificationHistory{
			ID:             uuid.New().String(),
			NotificationID: notificationID,
			ContactID:      contactID,
			Channel:        channel,
			Status:         status,
			SenderID:       sender.ID,
			CampaignID:     req.CampaignID,
			GroupID:        groupID,
			CreatedAt:      now,
		}
		if err := s.historyRepo.Create(ctx, history); err != nil {
			return nil, fmt.Errorf("failed to create history entry: %w", err)
		}

		// Update analytics
		if err := s.analyticsRepo.IncrementSent(ctx, sender.ID, now, channel); err != nil {
			return nil, fmt.Errorf("failed to update analytics: %w", err)
		}

		notifications = append(notifications, notification)
	}

	return notifications, nil
}

func (s *NotificationService) GetByID(ctx context.Context, notificationID string) (*domain.Notification, error) {
	return s.repo.GetByID(ctx, notificationID)
}

func (s *NotificationService) GetHistoryByContactID(ctx context.Context, contactID string, page, size int) ([]*domain.NotificationHistory, int, error) {
	return s.historyRepo.GetByContactID(ctx, contactID, page, size)
}

func (s *NotificationService) GetHistoryByUserID(ctx context.Context, userID string, page, size int) ([]*domain.NotificationHistory, int, error) {
	// First get user's contacts
	contacts, err := s.recipientsClient.GetUserContacts(ctx, userID, "")
	if err != nil {
		return nil, 0, err
	}

	if len(contacts.Contacts) == 0 {
		return []*domain.NotificationHistory{}, 0, nil
	}

	// Get history for each contact
	var allHistory []*domain.NotificationHistory
	total := 0

	for _, contact := range contacts.Contacts {
		items, count, err := s.historyRepo.GetByContactID(ctx, contact.ID, page, size)
		if err != nil {
			return nil, 0, err
		}
		allHistory = append(allHistory, items...)
		total += count
	}

	return allHistory, total, nil
}

func (s *NotificationService) GetHistoryBySenderID(ctx context.Context, senderID string, campaignID, status *string, page, size int) ([]*domain.NotificationHistory, int, error) {
	return s.historyRepo.GetBySenderID(ctx, senderID, campaignID, status, page, size)
}

func (s *NotificationService) GetAnalytics(ctx context.Context, senderID string, from, to time.Time) ([]*domain.SenderAnalyticsDaily, error) {
	return s.analyticsRepo.GetBySender(ctx, senderID, from, to)
}

func (s *NotificationService) UpdateDeliveryStatus(ctx context.Context, notificationID string, status domain.NotificationStatus) error {
	return s.repo.UpdateStatusWithTimestamp(ctx, notificationID, status)
}

func (s *NotificationService) generateUnsubscribeURL(contactID, senderID string, campaignID, groupID *string) string {
	// Generate signed token with available_scopes
	scopes := "sender"
	if campaignID != nil || groupID != nil {
		scopes = "sender,campaign_or_group"
	}

	// In production, this would be a proper HMAC signature
	token := fmt.Sprintf("%s|%s|%s|%v|%v", contactID, senderID, scopes, campaignID, groupID)
	return fmt.Sprintf("/unsubscribe?token=%s&available_scopes=%s", token, scopes)
}

// RenderTemplate renders a template with variables
func RenderTemplate(subject, body string, variables map[string]any) string {
	var result strings.Builder

	if subject != "" {
		tmpl, err := template.New("subject").Parse(subject)
		if err == nil {
			var buf bytes.Buffer
			tmpl.Execute(&buf, variables)
			result.WriteString(buf.String())
			result.WriteString("\n")
		}
	}

	if body != "" {
		tmpl, err := template.New("body").Parse(body)
		if err == nil {
			var buf bytes.Buffer
			tmpl.Execute(&buf, variables)
			result.WriteString(buf.String())
		}
	}

	return result.String()
}
