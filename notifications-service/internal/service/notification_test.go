package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"notification-center/notifications-service/internal/client"
	"notification-center/notifications-service/internal/domain"
	"notification-center/notifications-service/internal/repository"
)

type notificationRepository interface {
	Create(ctx context.Context, n *domain.Notification) error
	GetByID(ctx context.Context, id string) (*domain.Notification, error)
	UpdateStatusWithTimestamp(ctx context.Context, id string, status domain.NotificationStatus) error
}

type outboxRepository interface {
	Create(ctx context.Context, outbox *domain.NotificationOutbox) error
}

type historyRepository interface {
	Create(ctx context.Context, h *domain.NotificationHistory) error
	GetByContactID(ctx context.Context, contactID string, page, size int) ([]*domain.NotificationHistory, int, error)
	GetBySenderID(ctx context.Context, senderID string, campaignID, status *string, page, size int) ([]*domain.NotificationHistory, int, error)
}

type analyticsRepository interface {
	IncrementSent(ctx context.Context, senderID string, date time.Time, channel domain.Channel) error
	GetBySender(ctx context.Context, senderID string, from, to time.Time) ([]*domain.SenderAnalyticsDaily, error)
}

type sourcesClient interface {
	ResolveCredential(ctx context.Context, integrationKey, traceID string) (*client.Sender, error)
	GetTemplate(ctx context.Context, templateID, traceID string) (*client.Template, error)
	GetCampaign(ctx context.Context, campaignID, traceID string) (*client.Campaign, error)
}

type recipientsClient interface {
	GetUserContacts(ctx context.Context, userID, traceID string) (*client.GetUserContactsResponse, error)
}

type mockNotificationRepo struct {
	notifications map[string]*domain.Notification
}

func (m *mockNotificationRepo) Create(ctx context.Context, n *domain.Notification) error {
	if m.notifications == nil {
		m.notifications = make(map[string]*domain.Notification)
	}
	m.notifications[n.ID] = n
	return nil
}

func (m *mockNotificationRepo) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	if m.notifications == nil {
		return nil, repository.ErrNotificationNotFound
	}
	n, ok := m.notifications[id]
	if !ok {
		return nil, repository.ErrNotificationNotFound
	}
	return n, nil
}

func (m *mockNotificationRepo) UpdateStatusWithTimestamp(ctx context.Context, id string, status domain.NotificationStatus) error {
	if m.notifications != nil {
		if n, ok := m.notifications[id]; ok {
			n.Status = status
		}
	}
	return nil
}

type mockOutboxRepo struct {
	outboxes []*domain.NotificationOutbox
}

func (m *mockOutboxRepo) Create(ctx context.Context, outbox *domain.NotificationOutbox) error {
	m.outboxes = append(m.outboxes, outbox)
	return nil
}

type mockHistoryRepo struct {
	histories  []*domain.NotificationHistory
	totalCount int
}

func (m *mockHistoryRepo) Create(ctx context.Context, h *domain.NotificationHistory) error {
	return nil
}

func (m *mockHistoryRepo) GetByContactID(ctx context.Context, contactID string, page, size int) ([]*domain.NotificationHistory, int, error) {
	return m.histories, m.totalCount, nil
}

func (m *mockHistoryRepo) GetBySenderID(ctx context.Context, senderID string, campaignID, status *string, page, size int) ([]*domain.NotificationHistory, int, error) {
	return m.histories, m.totalCount, nil
}

type mockAnalyticsRepo struct {
	analytics []*domain.SenderAnalyticsDaily
}

func (m *mockAnalyticsRepo) IncrementSent(ctx context.Context, senderID string, date time.Time, channel domain.Channel) error {
	return nil
}

func (m *mockAnalyticsRepo) GetBySender(ctx context.Context, senderID string, from, to time.Time) ([]*domain.SenderAnalyticsDaily, error) {
	return m.analytics, nil
}

type mockSourcesClient struct {
	sender         *client.Sender
	template       *client.Template
	campaign       *client.Campaign
	resolveErr     error
	getTemplateErr error
	getCampaignErr error
}

func (m *mockSourcesClient) ResolveCredential(ctx context.Context, integrationKey, traceID string) (*client.Sender, error) {
	if m.resolveErr != nil {
		return nil, m.resolveErr
	}
	if m.sender != nil {
		return m.sender, nil
	}
	return &client.Sender{ID: "sender-1", Name: "Test Sender"}, nil
}

func (m *mockSourcesClient) GetTemplate(ctx context.Context, templateID, traceID string) (*client.Template, error) {
	if m.getTemplateErr != nil {
		return nil, m.getTemplateErr
	}
	if m.template != nil {
		return m.template, nil
	}
	return &client.Template{
		ID:      templateID,
		Channel: "email",
		Subject: "Test Subject",
		Body:    "Test Body",
	}, nil
}

func (m *mockSourcesClient) GetCampaign(ctx context.Context, campaignID, traceID string) (*client.Campaign, error) {
	if m.getCampaignErr != nil {
		return nil, m.getCampaignErr
	}
	if m.campaign != nil {
		return m.campaign, nil
	}
	return &client.Campaign{ID: campaignID, GroupID: "group-1"}, nil
}

type mockRecipientsClient struct {
	contacts *client.GetUserContactsResponse
	err      error
}

func (m *mockRecipientsClient) GetUserContacts(ctx context.Context, userID, traceID string) (*client.GetUserContactsResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.contacts != nil {
		return m.contacts, nil
	}
	return &client.GetUserContactsResponse{
		Contacts: []client.Contact{
			{ID: "contact-1", Type: "email", Value: "test@example.com", Primary: true},
		},
	}, nil
}

type testableNotificationService struct {
	repo              notificationRepository
	outboxRepo        outboxRepository
	historyRepo       historyRepository
	analyticsRepo     analyticsRepository
	sourcesClient     sourcesClient
	recipientsClient  recipientsClient
	unsubscribeSecret string
}

func (s *testableNotificationService) Send(ctx context.Context, req SendNotificationRequest, integrationKey, traceID string) ([]*domain.Notification, error) {
	if len(req.Channels) != 1 {
		return nil, ErrMultipleChannels
	}
	if req.IdempotencyKey == "" {
		return nil, ErrInvalidIdempotency
	}
	if len(req.ContactIDs) == 0 {
		return nil, ErrNoContactsProvided
	}

	channel := req.Channels[0]

	sender, err := s.sourcesClient.ResolveCredential(ctx, integrationKey, traceID)
	if err != nil {
		return nil, ErrSenderNotFound
	}

	tmpl, err := s.sourcesClient.GetTemplate(ctx, req.TemplateID, traceID)
	if err != nil {
		return nil, ErrTemplateNotFound
	}

	if tmpl.Channel != string(channel) {
		return nil, ErrChannelMismatch
	}

	rendered := RenderTemplate(tmpl.Subject, tmpl.Body, req.Variables)

	var groupID *string
	if req.CampaignID != nil {
		campaign, err := s.sourcesClient.GetCampaign(ctx, *req.CampaignID, traceID)
		if err == nil {
			groupID = &campaign.GroupID
		}
	}

	notifications := make([]*domain.Notification, 0, len(req.ContactIDs))
	for _, contactID := range req.ContactIDs {
		notificationID := "notif-" + contactID
		now := time.Now().UTC()

		unsubscribeURL := s.generateUnsubscribeURL(contactID, sender.ID, req.CampaignID, groupID)

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

		outbox := &domain.NotificationOutbox{
			ID:             "outbox-" + notificationID,
			NotificationID: notificationID,
			EventType:      "notification.dispatch.v1",
			Payload:        map[string]any{},
			Attempt:        1,
			Status:         "pending",
			CreatedAt:      now,
		}

		if err := s.repo.Create(ctx, notification); err != nil {
			return nil, err
		}
		if err := s.outboxRepo.Create(ctx, outbox); err != nil {
			return nil, err
		}

		history := &domain.NotificationHistory{
			ID:             "history-" + notificationID,
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
			return nil, err
		}

		if err := s.analyticsRepo.IncrementSent(ctx, sender.ID, now, channel); err != nil {
			return nil, err
		}

		notifications = append(notifications, notification)
	}

	return notifications, nil
}

func (s *testableNotificationService) GetByID(ctx context.Context, notificationID string) (*domain.Notification, error) {
	return s.repo.GetByID(ctx, notificationID)
}

func (s *testableNotificationService) GetHistoryByContactID(ctx context.Context, contactID string, page, size int) ([]*domain.NotificationHistory, int, error) {
	return s.historyRepo.GetByContactID(ctx, contactID, page, size)
}

func (s *testableNotificationService) GetHistoryByUserID(ctx context.Context, userID string, page, size int) ([]*domain.NotificationHistory, int, error) {
	contacts, err := s.recipientsClient.GetUserContacts(ctx, userID, "")
	if err != nil {
		return nil, 0, err
	}

	if len(contacts.Contacts) == 0 {
		return []*domain.NotificationHistory{}, 0, nil
	}

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

func (s *testableNotificationService) GetHistoryBySenderID(ctx context.Context, senderID string, campaignID, status *string, page, size int) ([]*domain.NotificationHistory, int, error) {
	return s.historyRepo.GetBySenderID(ctx, senderID, campaignID, status, page, size)
}

func (s *testableNotificationService) GetAnalytics(ctx context.Context, senderID string, from, to time.Time) ([]*domain.SenderAnalyticsDaily, error) {
	return s.analyticsRepo.GetBySender(ctx, senderID, from, to)
}

func (s *testableNotificationService) UpdateDeliveryStatus(ctx context.Context, notificationID string, status domain.NotificationStatus) error {
	return s.repo.UpdateStatusWithTimestamp(ctx, notificationID, status)
}

func (s *testableNotificationService) generateUnsubscribeURL(contactID, senderID string, campaignID, groupID *string) string {
	scopes := "sender"
	if campaignID != nil || groupID != nil {
		scopes = "sender,campaign_or_group"
	}

	token := contactID + "|" + senderID + "|" + scopes + "|" + toString(campaignID) + "|" + toString(groupID)
	return "/unsubscribe?token=" + token + "&available_scopes=" + scopes
}

func toString(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func TestSend_Success(t *testing.T) {
	ctx := context.Background()
	repo := &mockNotificationRepo{}
	outboxRepo := &mockOutboxRepo{}
	historyRepo := &mockHistoryRepo{}
	analyticsRepo := &mockAnalyticsRepo{}
	sourcesClient := &mockSourcesClient{}
	recipientsClient := &mockRecipientsClient{}

	svc := &testableNotificationService{
		repo:              repo,
		outboxRepo:        outboxRepo,
		historyRepo:       historyRepo,
		analyticsRepo:     analyticsRepo,
		sourcesClient:     sourcesClient,
		recipientsClient:  recipientsClient,
		unsubscribeSecret: "test-secret",
	}

	req := SendNotificationRequest{
		TemplateID:     "template-1",
		ContactIDs:     []string{"contact-1"},
		Channels:       []domain.Channel{domain.ChannelEmail},
		Variables:      map[string]any{"name": "John"},
		IdempotencyKey: "idem-1",
	}

	notifications, err := svc.Send(ctx, req, "integration-key", "trace-1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifications))
	}
	if notifications[0].TemplateID != "template-1" {
		t.Errorf("expected template ID template-1, got %s", notifications[0].TemplateID)
	}
}

func TestSend_MultipleChannelsError(t *testing.T) {
	ctx := context.Background()
	svc := &testableNotificationService{}

	req := SendNotificationRequest{
		TemplateID:     "template-1",
		ContactIDs:     []string{"contact-1"},
		Channels:       []domain.Channel{domain.ChannelEmail, domain.ChannelSMS},
		IdempotencyKey: "idem-1",
	}

	_, err := svc.Send(ctx, req, "integration-key", "trace-1")

	if !errors.Is(err, ErrMultipleChannels) {
		t.Errorf("expected ErrMultipleChannels, got %v", err)
	}
}

func TestSend_NoIdempotencyKey(t *testing.T) {
	ctx := context.Background()
	svc := &testableNotificationService{}

	req := SendNotificationRequest{
		TemplateID: "template-1",
		ContactIDs: []string{"contact-1"},
		Channels:   []domain.Channel{domain.ChannelEmail},
	}

	_, err := svc.Send(ctx, req, "integration-key", "trace-1")

	if !errors.Is(err, ErrInvalidIdempotency) {
		t.Errorf("expected ErrInvalidIdempotency, got %v", err)
	}
}

func TestSend_NoContacts(t *testing.T) {
	ctx := context.Background()
	svc := &testableNotificationService{}

	req := SendNotificationRequest{
		TemplateID:     "template-1",
		ContactIDs:     []string{},
		Channels:       []domain.Channel{domain.ChannelEmail},
		IdempotencyKey: "idem-1",
	}

	_, err := svc.Send(ctx, req, "integration-key", "trace-1")

	if !errors.Is(err, ErrNoContactsProvided) {
		t.Errorf("expected ErrNoContactsProvided, got %v", err)
	}
}

func TestSend_SenderNotFound(t *testing.T) {
	ctx := context.Background()
	sourcesClient := &mockSourcesClient{
		resolveErr: errors.New("sender not found"),
	}
	svc := &testableNotificationService{
		sourcesClient: sourcesClient,
	}

	req := SendNotificationRequest{
		TemplateID:     "template-1",
		ContactIDs:     []string{"contact-1"},
		Channels:       []domain.Channel{domain.ChannelEmail},
		IdempotencyKey: "idem-1",
	}

	_, err := svc.Send(ctx, req, "integration-key", "trace-1")

	if !errors.Is(err, ErrSenderNotFound) {
		t.Errorf("expected ErrSenderNotFound, got %v", err)
	}
}

func TestSend_TemplateNotFound(t *testing.T) {
	ctx := context.Background()
	sourcesClient := &mockSourcesClient{
		getTemplateErr: errors.New("template not found"),
	}
	svc := &testableNotificationService{
		sourcesClient: sourcesClient,
	}

	req := SendNotificationRequest{
		TemplateID:     "template-1",
		ContactIDs:     []string{"contact-1"},
		Channels:       []domain.Channel{domain.ChannelEmail},
		IdempotencyKey: "idem-1",
	}

	_, err := svc.Send(ctx, req, "integration-key", "trace-1")

	if !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("expected ErrTemplateNotFound, got %v", err)
	}
}

func TestSend_ChannelMismatch(t *testing.T) {
	ctx := context.Background()
	sourcesClient := &mockSourcesClient{
		template: &client.Template{
			ID:      "template-1",
			Channel: "sms",
			Subject: "Test",
			Body:    "Test",
		},
	}
	svc := &testableNotificationService{
		sourcesClient: sourcesClient,
	}

	req := SendNotificationRequest{
		TemplateID:     "template-1",
		ContactIDs:     []string{"contact-1"},
		Channels:       []domain.Channel{domain.ChannelEmail},
		IdempotencyKey: "idem-1",
	}

	_, err := svc.Send(ctx, req, "integration-key", "trace-1")

	if !errors.Is(err, ErrChannelMismatch) {
		t.Errorf("expected ErrChannelMismatch, got %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	ctx := context.Background()
	notification := &domain.Notification{
		ID:         "notif-1",
		TemplateID: "template-1",
		Status:     domain.StatusQueued,
	}
	repo := &mockNotificationRepo{
		notifications: map[string]*domain.Notification{
			"notif-1": notification,
		},
	}
	svc := &testableNotificationService{
		repo: repo,
	}

	result, err := svc.GetByID(ctx, "notif-1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ID != "notif-1" {
		t.Errorf("expected ID notif-1, got %s", result.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := &mockNotificationRepo{}
	svc := &testableNotificationService{
		repo: repo,
	}

	_, err := svc.GetByID(ctx, "notif-1")

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestGetHistoryByContactID_Success(t *testing.T) {
	ctx := context.Background()
	histories := []*domain.NotificationHistory{
		{ID: "history-1", ContactID: "contact-1"},
		{ID: "history-2", ContactID: "contact-1"},
	}
	historyRepo := &mockHistoryRepo{
		histories:  histories,
		totalCount: 2,
	}
	svc := &testableNotificationService{
		historyRepo: historyRepo,
	}

	result, total, err := svc.GetHistoryByContactID(ctx, "contact-1", 1, 10)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 histories, got %d", len(result))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestGetHistoryByContactID_Pagination(t *testing.T) {
	ctx := context.Background()
	histories := []*domain.NotificationHistory{
		{ID: "history-1", ContactID: "contact-1"},
	}
	historyRepo := &mockHistoryRepo{
		histories:  histories,
		totalCount: 100,
	}
	svc := &testableNotificationService{
		historyRepo: historyRepo,
	}

	result, total, err := svc.GetHistoryByContactID(ctx, "contact-1", 2, 10)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 history, got %d", len(result))
	}
	if total != 100 {
		t.Errorf("expected total 100, got %d", total)
	}
}

func TestGetHistoryByUserID_Success(t *testing.T) {
	ctx := context.Background()
	recipientsClient := &mockRecipientsClient{
		contacts: &client.GetUserContactsResponse{
			Contacts: []client.Contact{
				{ID: "contact-1", Type: "email"},
				{ID: "contact-2", Type: "telegram"},
			},
		},
	}
	historyRepo := &mockHistoryRepo{
		histories: []*domain.NotificationHistory{
			{ID: "history-1", ContactID: "contact-1"},
			{ID: "history-2", ContactID: "contact-2"},
		},
		totalCount: 2,
	}
	svc := &testableNotificationService{
		recipientsClient: recipientsClient,
		historyRepo:      historyRepo,
	}

	result, total, err := svc.GetHistoryByUserID(ctx, "user-1", 1, 10)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 4 {
		t.Errorf("expected 4 histories (2 per contact), got %d", len(result))
	}
	if total != 4 {
		t.Errorf("expected total 4, got %d", total)
	}
}

func TestGetHistoryByUserID_NoContacts(t *testing.T) {
	ctx := context.Background()
	recipientsClient := &mockRecipientsClient{
		contacts: &client.GetUserContactsResponse{
			Contacts: []client.Contact{},
		},
	}
	svc := &testableNotificationService{
		recipientsClient: recipientsClient,
	}

	result, total, err := svc.GetHistoryByUserID(ctx, "user-1", 1, 10)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 histories, got %d", len(result))
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
}

func TestGetHistoryBySenderID_Success(t *testing.T) {
	ctx := context.Background()
	histories := []*domain.NotificationHistory{
		{ID: "history-1", SenderID: "sender-1"},
		{ID: "history-2", SenderID: "sender-1"},
	}
	historyRepo := &mockHistoryRepo{
		histories:  histories,
		totalCount: 2,
	}
	svc := &testableNotificationService{
		historyRepo: historyRepo,
	}

	campaignID := "campaign-1"
	status := string(domain.StatusSent)
	result, total, err := svc.GetHistoryBySenderID(ctx, "sender-1", &campaignID, &status, 1, 10)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 histories, got %d", len(result))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestGetHistoryBySenderID_WithFilters(t *testing.T) {
	ctx := context.Background()
	historyRepo := &mockHistoryRepo{
		histories:  []*domain.NotificationHistory{{ID: "history-1"}},
		totalCount: 1,
	}
	svc := &testableNotificationService{
		historyRepo: historyRepo,
	}

	campaignID := "campaign-1"
	status := "sent"
	_, _, err := svc.GetHistoryBySenderID(ctx, "sender-1", &campaignID, &status, 1, 10)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestGetAnalytics_Success(t *testing.T) {
	ctx := context.Background()
	analytics := []*domain.SenderAnalyticsDaily{
		{ID: "analytics-1", SenderID: "sender-1", SentCount: 10},
		{ID: "analytics-2", SenderID: "sender-1", SentCount: 5},
	}
	analyticsRepo := &mockAnalyticsRepo{
		analytics: analytics,
	}
	svc := &testableNotificationService{
		analyticsRepo: analyticsRepo,
	}

	from := time.Now().AddDate(0, 0, -7)
	to := time.Now()
	result, err := svc.GetAnalytics(ctx, "sender-1", from, to)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 analytics, got %d", len(result))
	}
}

func TestGetAnalytics_DateRange(t *testing.T) {
	ctx := context.Background()
	analyticsRepo := &mockAnalyticsRepo{
		analytics: []*domain.SenderAnalyticsDaily{},
	}
	svc := &testableNotificationService{
		analyticsRepo: analyticsRepo,
	}

	from := time.Now().AddDate(0, 0, -30)
	to := time.Now()
	_, err := svc.GetAnalytics(ctx, "sender-1", from, to)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUpdateDeliveryStatus(t *testing.T) {
	ctx := context.Background()
	repo := &mockNotificationRepo{
		notifications: map[string]*domain.Notification{
			"notif-1": {ID: "notif-1", Status: domain.StatusQueued},
		},
	}
	svc := &testableNotificationService{
		repo: repo,
	}

	err := svc.UpdateDeliveryStatus(ctx, "notif-1", domain.StatusDelivered)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.notifications["notif-1"].Status != domain.StatusDelivered {
		t.Errorf("expected status delivered, got %s", repo.notifications["notif-1"].Status)
	}
}

func TestGenerateUnsubscribeURL(t *testing.T) {
	svc := &testableNotificationService{unsubscribeSecret: "test-secret"}

	url := svc.generateUnsubscribeURL("contact-1", "sender-1", nil, nil)

	if url == "" {
		t.Error("expected non-empty URL")
	}
	if len(url) < 10 {
		t.Error("URL too short")
	}
}

func TestGenerateUnsubscribeURL_WithCampaign(t *testing.T) {
	svc := &testableNotificationService{unsubscribeSecret: "test-secret"}

	campaignID := "campaign-1"
	groupID := "group-1"
	url := svc.generateUnsubscribeURL("contact-1", "sender-1", &campaignID, &groupID)

	if url == "" {
		t.Error("expected non-empty URL")
	}
	if len(url) < 10 {
		t.Error("URL too short")
	}
}

func TestRenderTemplate_WithSubjectAndBody(t *testing.T) {
	result := RenderTemplate("Hello {{.name}}", "Your code is: {{.code}}", map[string]any{"name": "John", "code": "12345"})

	if result == "" {
		t.Error("expected non-empty result")
	}
	expected := "Hello John\nYour code is: 12345"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderTemplate_WithVariables(t *testing.T) {
	result := RenderTemplate("Welcome {{.user}}", "Your order #{{.order}} is confirmed", map[string]any{"user": "Alice", "order": "98765"})

	if result == "" {
		t.Error("expected non-empty result")
	}
	expected := "Welcome Alice\nYour order #98765 is confirmed"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderTemplate_WithMissingVariables(t *testing.T) {
	result := RenderTemplate("Hello {{.name}}", "Body: {{.missing}}", map[string]any{})

	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestRenderTemplate_EmptyTemplate(t *testing.T) {
	result := RenderTemplate("", "", map[string]any{"key": "value"})

	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}
