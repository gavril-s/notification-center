package service

import (
	"context"
	"testing"
	"time"

	"notification-center/recipients/internal/domain"
	"notification-center/recipients/internal/repository"
)

type testPreferenceScenario struct {
	name        string
	contactID   string
	senderID    *string
	scopeType   string
	scopeID     *string
	channel     string
	evaluateAt  time.Time
	contactRepo *MockContactRepo
	prefRepo    *MockPreferenceRepo
	wantAllowed bool
	wantReason  string
}

func TestPreferenceScenarios_QuietHours(t *testing.T) {
	ctx := context.Background()

	tests := []testPreferenceScenario{
		{
			name:       "тихие часы ночью 22:00-08:00 - ночное время внутри",
			contactID:  "contact-1",
			channel:    "email",
			evaluateAt: time.Date(2024, 1, 1, 23, 30, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						from := "22:00"
						to := "08:00"
						return &domain.Preference{
							ContactID: contactID,
							Enabled:   true,
							QuietFrom: &from,
							QuietTo:   &to,
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: false,
			wantReason:  "тихие часы",
		},
		{
			name:       "тихие часы ночью 22:00-08:00 - ранее утро внутри",
			contactID:  "contact-1",
			channel:    "sms",
			evaluateAt: time.Date(2024, 1, 1, 6, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						from := "22:00"
						to := "08:00"
						return &domain.Preference{
							ContactID: contactID,
							Enabled:   true,
							QuietFrom: &from,
							QuietTo:   &to,
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: false,
			wantReason:  "тихие часы",
		},
		{
			name:       "тихие часы ночью 22:00-08:00 - дневное время вне диапазона",
			contactID:  "contact-1",
			channel:    "email",
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						from := "22:00"
						to := "08:00"
						return &domain.Preference{
							ContactID: contactID,
							Enabled:   true,
							QuietFrom: &from,
							QuietTo:   &to,
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: true,
		},
		{
			name:       "тихие часы дневные 10:00-18:00 - дневное время внутри",
			contactID:  "contact-1",
			channel:    "telegram",
			evaluateAt: time.Date(2024, 1, 1, 15, 30, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						from := "10:00"
						to := "18:00"
						return &domain.Preference{
							ContactID: contactID,
							Enabled:   true,
							QuietFrom: &from,
							QuietTo:   &to,
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: false,
			wantReason:  "тихие часы",
		},
		{
			name:       "тихие часы не заданы - уведомление разрешено",
			contactID:  "contact-1",
			channel:    "email",
			evaluateAt: time.Date(2024, 1, 1, 23, 30, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{
							ContactID:       contactID,
							Enabled:         true,
							BlockedChannels: []string{},
							QuietFrom:       nil,
							QuietTo:         nil,
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestPreferenceService(tt.prefRepo, tt.contactRepo, &MockAuditLogRepo{})

			allowed, reason, err := svc.EvaluatePreference(ctx, tt.contactID, tt.senderID, tt.scopeType, tt.scopeID, tt.channel, tt.evaluateAt)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tt.wantAllowed {
				t.Errorf("expected allowed=%v, got %v", tt.wantAllowed, allowed)
			}
			if tt.wantAllowed {
				if reason != nil {
					t.Errorf("expected no reason, got %v", *reason)
				}
			} else {
				if reason == nil {
					t.Errorf("expected reason %q, got nil", tt.wantReason)
				} else if *reason != tt.wantReason && *reason != "тихие часы" {
					t.Errorf("expected reason %q, got %q", tt.wantReason, *reason)
				}
			}
		})
	}
}

func TestPreferenceScenarios_ChannelBlock(t *testing.T) {
	ctx := context.Background()

	tests := []testPreferenceScenario{
		{
			name:       "канал sms заблокирован - попытка отправить sms",
			contactID:  "contact-1",
			channel:    "sms",
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{
							ContactID:       contactID,
							Enabled:         true,
							BlockedChannels: []string{"sms"},
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: false,
			wantReason:  "канал заблокирован",
		},
		{
			name:       "канал sms заблокирован - отправка email разрешена",
			contactID:  "contact-1",
			channel:    "email",
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{
							ContactID:       contactID,
							Enabled:         true,
							BlockedChannels: []string{"sms"},
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: true,
		},
		{
			name:       "несколько каналов заблокированы - sms и telegram",
			contactID:  "contact-1",
			channel:    "telegram",
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{
							ContactID:       contactID,
							Enabled:         true,
							BlockedChannels: []string{"sms", "telegram"},
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: false,
			wantReason:  "канал заблокирован",
		},
		{
			name:       "канал не заблокирован - пустой список блокировок",
			contactID:  "contact-1",
			channel:    "email",
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{
							ContactID:       contactID,
							Enabled:         true,
							BlockedChannels: []string{},
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestPreferenceService(tt.prefRepo, tt.contactRepo, &MockAuditLogRepo{})

			allowed, reason, err := svc.EvaluatePreference(ctx, tt.contactID, tt.senderID, tt.scopeType, tt.scopeID, tt.channel, tt.evaluateAt)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tt.wantAllowed {
				t.Errorf("expected allowed=%v, got %v", tt.wantAllowed, allowed)
			}
			if tt.wantAllowed {
				if reason != nil {
					t.Errorf("expected no reason, got %v", *reason)
				}
			} else {
				if reason == nil {
					t.Errorf("expected reason, got nil")
				}
			}
		})
	}
}

func TestPreferenceScenarios_DisableSender(t *testing.T) {
	ctx := context.Background()
	senderID := "sender-newsletter"

	tests := []testPreferenceScenario{
		{
			name:       "отправитель отключен - уведомление заблокировано",
			contactID:  "contact-1",
			senderID:   &senderID,
			channel:    "email",
			scopeType:  "sender",
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{
							ContactID: contactID,
							Enabled:   true,
						}, nil
					}
					if scopeType == "sender" && senderID != nil && *senderID == "sender-newsletter" {
						return &domain.Preference{
							ContactID: contactID,
							SenderID:  senderID,
							ScopeType: "sender",
							Enabled:   false,
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: false,
			wantReason:  "от отправителя отключены",
		},
		{
			name:       "отправитель включен - уведомление разрешено",
			contactID:  "contact-1",
			senderID:   &senderID,
			channel:    "email",
			scopeType:  "sender",
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{
							ContactID: contactID,
							Enabled:   true,
						}, nil
					}
					if scopeType == "sender" && senderID != nil && *senderID == "sender-newsletter" {
						return &domain.Preference{
							ContactID: contactID,
							SenderID:  senderID,
							ScopeType: "sender",
							Enabled:   true,
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: true,
		},
		{
			name:       "без указания отправителя - используются глобальные настройки",
			contactID:  "contact-1",
			senderID:   nil,
			channel:    "email",
			scopeType:  "global",
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{
							ContactID: contactID,
							Enabled:   true,
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: true,
		},
		{
			name:       "отправитель отключен на уровне sender - другой отправитель разрешен",
			contactID:  "contact-1",
			senderID:   strPtr("sender-promo"),
			channel:    "email",
			scopeType:  "sender",
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{
							ContactID: contactID,
							Enabled:   true,
						}, nil
					}
					if scopeType == "sender" && senderID != nil && *senderID == "sender-newsletter" {
						return &domain.Preference{
							ContactID: contactID,
							SenderID:  senderID,
							ScopeType: "sender",
							Enabled:   false,
						}, nil
					}
					if scopeType == "sender" && senderID != nil && *senderID == "sender-promo" {
						return &domain.Preference{
							ContactID: contactID,
							SenderID:  senderID,
							ScopeType: "sender",
							Enabled:   true,
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestPreferenceService(tt.prefRepo, tt.contactRepo, &MockAuditLogRepo{})

			allowed, reason, err := svc.EvaluatePreference(ctx, tt.contactID, tt.senderID, tt.scopeType, tt.scopeID, tt.channel, tt.evaluateAt)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tt.wantAllowed {
				t.Errorf("expected allowed=%v, got %v", tt.wantAllowed, allowed)
			}
			if tt.wantAllowed {
				if reason != nil {
					t.Errorf("expected no reason, got %v", *reason)
				}
			} else {
				if reason == nil {
					t.Errorf("expected reason, got nil")
				}
			}
		})
	}
}

func TestPreferenceScenarios_PriorityOrder(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		contactID   string
		senderID    *string
		scopeType   string
		scopeID     *string
		channel     string
		evaluateAt  time.Time
		contactRepo *MockContactRepo
		prefRepo    *MockPreferenceRepo
		wantAllowed bool
		wantReason  string
	}{
		{
			name:       "глобальные настройки отключены - проверка sender не нужна",
			contactID:  "contact-1",
			senderID:   strPtr("sender-1"),
			channel:    "email",
			scopeType:  "sender",
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{
							ContactID: contactID,
							Enabled:   false,
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: false,
			wantReason:  "глобальные уведомления отключены",
		},
		{
			name:       "sender настройки переопределяют глобальные - канал заблокирован у sender",
			contactID:  "contact-1",
			senderID:   strPtr("sender-1"),
			channel:    "sms",
			scopeType:  "sender",
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{
							ContactID:       contactID,
							Enabled:         true,
							BlockedChannels: []string{},
						}, nil
					}
					if scopeType == "sender" {
						return &domain.Preference{
							ContactID:       contactID,
							SenderID:        senderID,
							ScopeType:       "sender",
							Enabled:         true,
							BlockedChannels: []string{"sms"},
						}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: false,
			wantReason:  "канал заблокирован в настройках отправителя",
		},
		{
			name:       "контакт отключен - проверка настроек не выполняется",
			contactID:  "contact-1",
			senderID:   strPtr("sender-1"),
			channel:    "email",
			scopeType:  "global",
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: false}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					t.Fatal("preference repository should not be called when contact is disabled")
					return nil, nil
				},
			},
			wantAllowed: false,
			wantReason:  "контакт отключен",
		},
		{
			name:       "нет настроек - уведомление разрешено по умолчанию",
			contactID:  "contact-1",
			channel:    "email",
			scopeType:  "global",
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestPreferenceService(tt.prefRepo, tt.contactRepo, &MockAuditLogRepo{})

			allowed, reason, err := svc.EvaluatePreference(ctx, tt.contactID, tt.senderID, tt.scopeType, tt.scopeID, tt.channel, tt.evaluateAt)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if allowed != tt.wantAllowed {
				t.Errorf("expected allowed=%v, got %v", tt.wantAllowed, allowed)
			}
			if tt.wantAllowed {
				if reason != nil {
					t.Errorf("expected no reason, got %v", *reason)
				}
			} else {
				if reason == nil {
					t.Errorf("expected reason, got nil")
				}
			}
		})
	}
}
