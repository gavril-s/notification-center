package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"notification-center/notifications-service/internal/client"
	"notification-center/notifications-service/internal/domain"
	"notification-center/notifications-service/internal/repository"
)

var ErrNotificationNotFound = repository.ErrNotificationNotFound

func TestSendNotification_Behavior(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setupMock  func() *scenarioMocks
		req        SendNotificationRequest
		assertions func(*testing.T, *scenarioMocks, []*domain.Notification, error)
	}{
		{
			name: "creates notification in database for single contact",
			setupMock: func() *scenarioMocks {
				m := newScenarioMocks()
				m.sourcesClient.sender = &testSender
				m.sourcesClient.template = &testTemplate
				return m
			},
			req: SendNotificationRequest{
				TemplateID:     "template-1",
				ContactIDs:     []string{"contact-1"},
				Channels:       []domain.Channel{domain.ChannelEmail},
				Variables:      map[string]any{"name": "John"},
				IdempotencyKey: "idem-1",
			},
			assertions: func(t *testing.T, m *scenarioMocks, notifications []*domain.Notification, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(notifications) != 1 {
					t.Fatalf("expected 1 notification, got %d", len(notifications))
				}
				if m.notificationRepo.created["contact-1"] == nil {
					t.Error("expected notification to be created in repository")
				}
			},
		},
		{
			name: "creates outbox entry for message queue dispatch",
			setupMock: func() *scenarioMocks {
				m := newScenarioMocks()
				m.sourcesClient.sender = &testSender
				m.sourcesClient.template = &testTemplate
				return m
			},
			req: SendNotificationRequest{
				TemplateID:     "template-1",
				ContactIDs:     []string{"contact-1"},
				Channels:       []domain.Channel{domain.ChannelEmail},
				Variables:      map[string]any{},
				IdempotencyKey: "idem-2",
			},
			assertions: func(t *testing.T, m *scenarioMocks, notifications []*domain.Notification, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(m.outboxRepo.entries) != 1 {
					t.Fatalf("expected 1 outbox entry, got %d", len(m.outboxRepo.entries))
				}
				outbox := m.outboxRepo.entries[0]
				if outbox.EventType != "notification.dispatch.v1" {
					t.Errorf("expected event type notification.dispatch.v1, got %s", outbox.EventType)
				}
			},
		},
		{
			name: "creates history entry for audit trail",
			setupMock: func() *scenarioMocks {
				m := newScenarioMocks()
				m.sourcesClient.sender = &testSender
				m.sourcesClient.template = &testTemplate
				return m
			},
			req: SendNotificationRequest{
				TemplateID:     "template-1",
				ContactIDs:     []string{"contact-1"},
				Channels:       []domain.Channel{domain.ChannelEmail},
				Variables:      map[string]any{},
				IdempotencyKey: "idem-3",
			},
			assertions: func(t *testing.T, m *scenarioMocks, notifications []*domain.Notification, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(m.historyRepo.entries) != 1 {
					t.Fatalf("expected 1 history entry, got %d", len(m.historyRepo.entries))
				}
				history := m.historyRepo.entries[0]
				if history.ContactID != "contact-1" {
					t.Errorf("expected contact-1, got %s", history.ContactID)
				}
			},
		},
		{
			name: "increments sent analytics counter",
			setupMock: func() *scenarioMocks {
				m := newScenarioMocks()
				m.sourcesClient.sender = &testSender
				m.sourcesClient.template = &testTemplate
				return m
			},
			req: SendNotificationRequest{
				TemplateID:     "template-1",
				ContactIDs:     []string{"contact-1"},
				Channels:       []domain.Channel{domain.ChannelEmail},
				Variables:      map[string]any{},
				IdempotencyKey: "idem-4",
			},
			assertions: func(t *testing.T, m *scenarioMocks, notifications []*domain.Notification, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !m.analyticsRepo.sentIncremented {
					t.Error("expected analytics sent counter to be incremented")
				}
			},
		},
		{
			name: "creates multiple notifications for multiple contacts",
			setupMock: func() *scenarioMocks {
				m := newScenarioMocks()
				m.sourcesClient.sender = &testSender
				m.sourcesClient.template = &testTemplate
				return m
			},
			req: SendNotificationRequest{
				TemplateID:     "template-1",
				ContactIDs:     []string{"contact-1", "contact-2", "contact-3"},
				Channels:       []domain.Channel{domain.ChannelEmail},
				Variables:      map[string]any{},
				IdempotencyKey: "idem-5",
			},
			assertions: func(t *testing.T, m *scenarioMocks, notifications []*domain.Notification, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(notifications) != 3 {
					t.Fatalf("expected 3 notifications, got %d", len(notifications))
				}
				if len(m.notificationRepo.created) != 3 {
					t.Errorf("expected 3 notifications in repo, got %d", len(m.notificationRepo.created))
				}
				if len(m.outboxRepo.entries) != 3 {
					t.Errorf("expected 3 outbox entries, got %d", len(m.outboxRepo.entries))
				}
				if len(m.historyRepo.entries) != 3 {
					t.Errorf("expected 3 history entries, got %d", len(m.historyRepo.entries))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := tt.setupMock()
			svc := newTestableServiceFromMocks(mocks)

			result, err := svc.Send(ctx, tt.req, "integration-key", "trace-1")

			tt.assertions(t, mocks, result, err)
		})
	}
}

func TestScheduledNotification_Behavior(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		setupMock  func() *scenarioMocks
		req        SendNotificationRequest
		wantStatus domain.NotificationStatus
	}{
		{
			name: "has scheduled status when scheduled_at is in the future",
			setupMock: func() *scenarioMocks {
				m := newScenarioMocks()
				m.sourcesClient.sender = &testSender
				m.sourcesClient.template = &testTemplate
				return m
			},
			req: SendNotificationRequest{
				TemplateID:     "template-1",
				ContactIDs:     []string{"contact-1"},
				Channels:       []domain.Channel{domain.ChannelEmail},
				Variables:      map[string]any{},
				ScheduledAt:    timePointer(time.Now().Add(1 * time.Hour)),
				IdempotencyKey: "sched-1",
			},
			wantStatus: domain.StatusScheduled,
		},
		{
			name: "has queued status when scheduled_at is nil",
			setupMock: func() *scenarioMocks {
				m := newScenarioMocks()
				m.sourcesClient.sender = &testSender
				m.sourcesClient.template = &testTemplate
				return m
			},
			req: SendNotificationRequest{
				TemplateID:     "template-1",
				ContactIDs:     []string{"contact-1"},
				Channels:       []domain.Channel{domain.ChannelEmail},
				Variables:      map[string]any{},
				IdempotencyKey: "queued-1",
			},
			wantStatus: domain.StatusQueued,
		},
		{
			name: "has scheduled status when scheduled_at is far in future",
			setupMock: func() *scenarioMocks {
				m := newScenarioMocks()
				m.sourcesClient.sender = &testSender
				m.sourcesClient.template = &testTemplate
				return m
			},
			req: SendNotificationRequest{
				TemplateID:     "template-1",
				ContactIDs:     []string{"contact-1"},
				Channels:       []domain.Channel{domain.ChannelEmail},
				Variables:      map[string]any{},
				ScheduledAt:    timePointer(time.Now().Add(24 * time.Hour)),
				IdempotencyKey: "sched-2",
			},
			wantStatus: domain.StatusScheduled,
		},
		{
			name: "history entry also has scheduled status",
			setupMock: func() *scenarioMocks {
				m := newScenarioMocks()
				m.sourcesClient.sender = &testSender
				m.sourcesClient.template = &testTemplate
				return m
			},
			req: SendNotificationRequest{
				TemplateID:     "template-1",
				ContactIDs:     []string{"contact-1"},
				Channels:       []domain.Channel{domain.ChannelEmail},
				Variables:      map[string]any{},
				ScheduledAt:    timePointer(time.Now().Add(30 * time.Minute)),
				IdempotencyKey: "sched-3",
			},
			wantStatus: domain.StatusScheduled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := tt.setupMock()
			svc := newTestableServiceFromMocks(mocks)

			result, err := svc.Send(ctx, tt.req, "integration-key", "trace-1")

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) == 0 {
				t.Fatal("expected at least one notification")
			}
			if result[0].Status != tt.wantStatus {
				t.Errorf("expected status %s, got %s", tt.wantStatus, result[0].Status)
			}

			if len(mocks.historyRepo.entries) > 0 {
				if mocks.historyRepo.entries[0].Status != tt.wantStatus {
					t.Errorf("expected history status %s, got %s", tt.wantStatus, mocks.historyRepo.entries[0].Status)
				}
			}
		})
	}
}

func TestRenderTemplate_Behavior(t *testing.T) {
	tests := []struct {
		name           string
		subject        string
		body           string
		variables      map[string]any
		expectedResult string
	}{
		{
			name:           "substitutes name variable in subject",
			subject:        "Hello {{.Name}}",
			body:           "",
			variables:      map[string]any{"Name": "Alice"},
			expectedResult: "Hello Alice\n",
		},
		{
			name:           "substitutes order number in body",
			subject:        "",
			body:           "Your order #{{.OrderID}} is confirmed",
			variables:      map[string]any{"OrderID": "12345"},
			expectedResult: "Your order #12345 is confirmed",
		},
		{
			name:           "substitutes multiple variables in subject and body",
			subject:        "Welcome {{.Name}}",
			body:           "Hello {{.Name}}, your balance is {{.Balance}}",
			variables:      map[string]any{"Name": "Bob", "Balance": "100.50"},
			expectedResult: "Welcome Bob\nHello Bob, your balance is 100.50",
		},
		{
			name:           "outputs no value when variable is missing",
			subject:        "Hello {{.Name}}",
			body:           "Body: {{.Missing}}",
			variables:      map[string]any{},
			expectedResult: "Hello <no value>\nBody: <no value>",
		},
		{
			name:           "handles extra variables gracefully without error",
			subject:        "Hello {{.Name}}",
			body:           "Test",
			variables:      map[string]any{"Name": "John", "Extra": "ignored"},
			expectedResult: "Hello John\nTest",
		},
		{
			name:           "handles numeric variables",
			subject:        "Order #{{.OrderID}}",
			body:           "Total: ${{.Total}}",
			variables:      map[string]any{"OrderID": 42, "Total": 99.99},
			expectedResult: "Order #42\nTotal: $99.99",
		},
		{
			name:           "returns empty string when both subject and body are empty",
			subject:        "",
			body:           "",
			variables:      map[string]any{"Name": "Any"},
			expectedResult: "",
		},
		{
			name:           "preserves newlines in template correctly",
			subject:        "Line 1\nLine 2",
			body:           "Body line",
			variables:      map[string]any{},
			expectedResult: "Line 1\nLine 2\nBody line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderTemplate(tt.subject, tt.body, tt.variables)

			if result != tt.expectedResult {
				t.Errorf("expected %q, got %q", tt.expectedResult, result)
			}
		})
	}
}

func TestUnsubscribeURL_Behavior(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func() *scenarioMocks
		contactID   string
		senderID    string
		campaignID  *string
		groupID     *string
		checkToken  func(*testing.T, string)
		checkScopes func(*testing.T, string)
	}{
		{
			name: "URL contains contact_id",
			setupMock: func() *scenarioMocks {
				return newScenarioMocks()
			},
			contactID:  "contact-123",
			senderID:   "sender-456",
			campaignID: nil,
			groupID:    nil,
			checkToken: func(t *testing.T, url string) {
				if !strings.Contains(url, "contact-123") {
					t.Error("expected URL to contain contact_id")
				}
			},
			checkScopes: func(t *testing.T, url string) {
				if !strings.Contains(url, "sender") || strings.Contains(url, "campaign_or_group") {
					t.Error("expected scope to be 'sender' only")
				}
			},
		},
		{
			name: "URL contains sender_id",
			setupMock: func() *scenarioMocks {
				return newScenarioMocks()
			},
			contactID:  "contact-123",
			senderID:   "sender-456",
			campaignID: nil,
			groupID:    nil,
			checkToken: func(t *testing.T, url string) {
				if !strings.Contains(url, "sender-456") {
					t.Error("expected URL to contain sender_id")
				}
			},
			checkScopes: func(t *testing.T, url string) {},
		},
		{
			name: "has sender scope when no campaign or group",
			setupMock: func() *scenarioMocks {
				return newScenarioMocks()
			},
			contactID:  "contact-1",
			senderID:   "sender-1",
			campaignID: nil,
			groupID:    nil,
			checkToken: func(t *testing.T, url string) {},
			checkScopes: func(t *testing.T, url string) {
				if !strings.Contains(url, "available_scopes=sender") {
					t.Errorf("expected available_scopes=sender, got %s", url)
				}
			},
		},
		{
			name: "has sender and campaign_or_group scope when campaign exists",
			setupMock: func() *scenarioMocks {
				m := newScenarioMocks()
				m.sourcesClient.sender = &testSender
				m.sourcesClient.campaign = &testCampaign
				return m
			},
			contactID:  "contact-1",
			senderID:   "sender-1",
			campaignID: stringPointer("campaign-1"),
			groupID:    nil,
			checkToken: func(t *testing.T, url string) {
				if !strings.Contains(url, "campaign-1") {
					t.Error("expected URL to contain campaign_id when provided")
				}
			},
			checkScopes: func(t *testing.T, url string) {
				if !strings.Contains(url, "available_scopes=sender,campaign_or_group") {
					t.Errorf("expected available_scopes=sender,campaign_or_group, got %s", url)
				}
			},
		},
		{
			name: "has sender and campaign_or_group scope when group exists",
			setupMock: func() *scenarioMocks {
				m := newScenarioMocks()
				m.sourcesClient.sender = &testSender
				return m
			},
			contactID:  "contact-1",
			senderID:   "sender-1",
			campaignID: nil,
			groupID:    stringPointer("group-1"),
			checkToken: func(t *testing.T, url string) {},
			checkScopes: func(t *testing.T, url string) {
				if !strings.Contains(url, "available_scopes=sender,campaign_or_group") {
					t.Errorf("expected available_scopes=sender,campaign_or_group, got %s", url)
				}
			},
		},
		{
			name: "has sender and campaign_or_group scope when both campaign and group exist",
			setupMock: func() *scenarioMocks {
				m := newScenarioMocks()
				m.sourcesClient.sender = &testSender
				m.sourcesClient.campaign = &testCampaign
				return m
			},
			contactID:  "contact-1",
			senderID:   "sender-1",
			campaignID: stringPointer("campaign-1"),
			groupID:    stringPointer("group-1"),
			checkToken: func(t *testing.T, url string) {},
			checkScopes: func(t *testing.T, url string) {
				if !strings.Contains(url, "available_scopes=sender,campaign_or_group") {
					t.Errorf("expected available_scopes=sender,campaign_or_group, got %s", url)
				}
			},
		},
		{
			name: "URL starts with /unsubscribe path",
			setupMock: func() *scenarioMocks {
				return newScenarioMocks()
			},
			contactID:  "contact-1",
			senderID:   "sender-1",
			campaignID: nil,
			groupID:    nil,
			checkToken: func(t *testing.T, url string) {
				if !strings.HasPrefix(url, "/unsubscribe?") {
					t.Errorf("expected URL to start with /unsubscribe?, got %s", url)
				}
			},
			checkScopes: func(t *testing.T, url string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := tt.setupMock()
			svc := newTestableServiceFromMocks(mocks)

			url := svc.generateUnsubscribeURL(tt.contactID, tt.senderID, tt.campaignID, tt.groupID)

			tt.checkToken(t, url)
			tt.checkScopes(t, url)
		})
	}
}

type scenarioMocks struct {
	notificationRepo *mockNotificationRepoWithCapture
	outboxRepo       *mockOutboxRepoWithCapture
	historyRepo      *mockHistoryRepoWithCapture
	analyticsRepo    *mockAnalyticsRepoWithCapture
	sourcesClient    *mockSourcesClientWithCapture
	recipientsClient *mockRecipientsClient
}

type mockNotificationRepoWithCapture struct {
	created map[string]*domain.Notification
}

func (m *mockNotificationRepoWithCapture) Create(ctx context.Context, n *domain.Notification) error {
	if m.created == nil {
		m.created = make(map[string]*domain.Notification)
	}
	m.created[n.ContactID] = n
	return nil
}

func (m *mockNotificationRepoWithCapture) GetByID(ctx context.Context, id string) (*domain.Notification, error) {
	if n, ok := m.created[id]; ok {
		return n, nil
	}
	return nil, ErrNotificationNotFound
}

func (m *mockNotificationRepoWithCapture) UpdateStatusWithTimestamp(ctx context.Context, id string, status domain.NotificationStatus) error {
	return nil
}

type mockOutboxRepoWithCapture struct {
	entries []*domain.NotificationOutbox
}

func (m *mockOutboxRepoWithCapture) Create(ctx context.Context, outbox *domain.NotificationOutbox) error {
	m.entries = append(m.entries, outbox)
	return nil
}

type mockHistoryRepoWithCapture struct {
	entries []*domain.NotificationHistory
}

func (m *mockHistoryRepoWithCapture) Create(ctx context.Context, h *domain.NotificationHistory) error {
	m.entries = append(m.entries, h)
	return nil
}

func (m *mockHistoryRepoWithCapture) GetByContactID(ctx context.Context, contactID string, page, size int) ([]*domain.NotificationHistory, int, error) {
	return m.entries, len(m.entries), nil
}

func (m *mockHistoryRepoWithCapture) GetBySenderID(ctx context.Context, senderID string, campaignID, status *string, page, size int) ([]*domain.NotificationHistory, int, error) {
	return m.entries, len(m.entries), nil
}

type mockAnalyticsRepoWithCapture struct {
	sentIncremented bool
}

func (m *mockAnalyticsRepoWithCapture) IncrementSent(ctx context.Context, senderID string, date time.Time, channel domain.Channel) error {
	m.sentIncremented = true
	return nil
}

func (m *mockAnalyticsRepoWithCapture) GetBySender(ctx context.Context, senderID string, from, to time.Time) ([]*domain.SenderAnalyticsDaily, error) {
	return nil, nil
}

type mockSourcesClientWithCapture struct {
	sender   *testSenderType
	template *testTemplateType
	campaign *testCampaignType
}

type testSenderType struct {
	ID   string
	Name string
}

type testTemplateType struct {
	ID      string
	Channel string
	Subject string
	Body    string
}

type testCampaignType struct {
	ID      string
	GroupID string
}

func (m *mockSourcesClientWithCapture) ResolveCredential(ctx context.Context, integrationKey, traceID string) (*client.Sender, error) {
	if m.sender != nil {
		return &client.Sender{ID: m.sender.ID, Name: m.sender.Name}, nil
	}
	return &client.Sender{ID: "sender-1", Name: "Test Sender"}, nil
}

func (m *mockSourcesClientWithCapture) GetTemplate(ctx context.Context, templateID, traceID string) (*client.Template, error) {
	if m.template != nil {
		return &client.Template{
			ID:      m.template.ID,
			Channel: m.template.Channel,
			Subject: m.template.Subject,
			Body:    m.template.Body,
		}, nil
	}
	return &client.Template{
		ID:      templateID,
		Channel: "email",
		Subject: "Test Subject",
		Body:    "Test Body",
	}, nil
}

func (m *mockSourcesClientWithCapture) GetCampaign(ctx context.Context, campaignID, traceID string) (*client.Campaign, error) {
	if m.campaign != nil {
		return &client.Campaign{ID: m.campaign.ID, GroupID: m.campaign.GroupID}, nil
	}
	return &client.Campaign{ID: campaignID, GroupID: "group-1"}, nil
}

var (
	testSender   = testSenderType{ID: "sender-1", Name: "Test Sender"}
	testTemplate = testTemplateType{ID: "template-1", Channel: "email", Subject: "Test Subject", Body: "Test Body"}
	testCampaign = testCampaignType{ID: "campaign-1", GroupID: "group-1"}
)

func newScenarioMocks() *scenarioMocks {
	return &scenarioMocks{
		notificationRepo: &mockNotificationRepoWithCapture{},
		outboxRepo:       &mockOutboxRepoWithCapture{},
		historyRepo:      &mockHistoryRepoWithCapture{},
		analyticsRepo:    &mockAnalyticsRepoWithCapture{},
		sourcesClient:    &mockSourcesClientWithCapture{},
		recipientsClient: &mockRecipientsClient{},
	}
}

func newTestableServiceFromMocks(m *scenarioMocks) *testableNotificationService {
	return &testableNotificationService{
		repo:              m.notificationRepo,
		outboxRepo:        m.outboxRepo,
		historyRepo:       m.historyRepo,
		analyticsRepo:     m.analyticsRepo,
		sourcesClient:     m.sourcesClient,
		recipientsClient:  m.recipientsClient,
		unsubscribeSecret: "test-secret",
	}
}

func timePointer(t time.Time) *time.Time {
	return &t
}

func stringPointer(s string) *string {
	return &s
}
