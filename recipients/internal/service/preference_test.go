package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"notification-center/recipients/internal/domain"
	"notification-center/recipients/internal/repository"
)

type testPreferenceService struct {
	prefRepo    PreferenceRepositoryInterface
	contactRepo ContactRepositoryInterface
	auditRepo   AuditLogRepositoryInterface
}

func newTestPreferenceService(
	prefRepo PreferenceRepositoryInterface,
	contactRepo ContactRepositoryInterface,
	auditRepo AuditLogRepositoryInterface,
) *testPreferenceService {
	return &testPreferenceService{
		prefRepo:    prefRepo,
		contactRepo: contactRepo,
		auditRepo:   auditRepo,
	}
}

func (s *testPreferenceService) GetByContactID(ctx context.Context, contactID string) ([]domain.Preference, error) {
	_, err := s.contactRepo.GetByID(ctx, contactID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrContactNotFound
	}
	if err != nil {
		return nil, err
	}

	return s.prefRepo.GetByContactID(ctx, contactID)
}

func (s *testPreferenceService) GetByUserID(ctx context.Context, userID string) ([]domain.Preference, error) {
	contacts, err := s.contactRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var allPrefs []domain.Preference
	for _, contact := range contacts {
		prefs, err := s.prefRepo.GetByContactID(ctx, contact.ID)
		if err != nil {
			return nil, err
		}
		allPrefs = append(allPrefs, prefs...)
	}

	return allPrefs, nil
}

func (s *testPreferenceService) Update(ctx context.Context, userID string, req *UpdatePreferenceRequest) (*domain.Preference, error) {
	contact, err := s.contactRepo.GetByID(ctx, req.ContactID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrContactNotFound
	}
	if err != nil {
		return nil, err
	}

	if contact.UserID != userID {
		return nil, errors.New("контакт не принадлежит пользователю")
	}

	if req.QuietFrom != nil || req.QuietTo != nil {
		if err := s.validateQuietHours(req.QuietFrom, req.QuietTo); err != nil {
			return nil, err
		}
	}

	pref := &domain.Preference{
		ContactID:       req.ContactID,
		UserID:          userID,
		SenderID:        req.SenderID,
		ScopeType:       req.ScopeType,
		ScopeID:         req.ScopeID,
		Enabled:         req.Enabled,
		QuietFrom:       req.QuietFrom,
		QuietTo:         req.QuietTo,
		BlockedChannels: req.BlockedChannels,
	}

	if err := s.prefRepo.Upsert(ctx, pref); err != nil {
		return nil, err
	}

	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  userID,
		Action:  "preference_update",
		Details: "preference updated",
	})

	return pref, nil
}

func (s *testPreferenceService) validateQuietHours(from, to *string) error {
	if from != nil {
		if _, err := time.Parse("15:04", *from); err != nil {
			return errors.New("некорректный формат времени quiet_from (ожидается HH:MM)")
		}
	}
	if to != nil {
		if _, err := time.Parse("15:04", *to); err != nil {
			return errors.New("некорректный формат времени quiet_to (ожидается HH:MM)")
		}
	}
	return nil
}

func (s *testPreferenceService) EvaluatePreference(ctx context.Context, contactID string, senderID *string, scopeType string, scopeID *string, channel string, evaluateAt time.Time) (bool, *string, error) {
	contact, err := s.contactRepo.GetByID(ctx, contactID)
	if errors.Is(err, repository.ErrNotFound) {
		return false, nil, ErrContactNotFound
	}
	if err != nil {
		return false, nil, err
	}

	if !contact.Enabled {
		reason := "контакт отключен"
		return false, &reason, nil
	}

	globalPref, err := s.prefRepo.GetByContactAndScope(ctx, contactID, "global", nil, nil)
	if err == nil && globalPref != nil {
		if !globalPref.Enabled {
			reason := "глобальные уведомления отключены"
			return false, &reason, nil
		}
		if s.isChannelBlocked(globalPref.BlockedChannels, channel) {
			reason := "канал заблокирован в глобальных настройках"
			return false, &reason, nil
		}
		if s.isInQuietHours(globalPref.QuietFrom, globalPref.QuietTo, evaluateAt) {
			reason := "тихие часы"
			return false, &reason, nil
		}
	}

	if senderID != nil {
		senderPref, err := s.prefRepo.GetByContactAndScope(ctx, contactID, "sender", senderID, nil)
		if err == nil && senderPref != nil {
			if !senderPref.Enabled {
				reason := "уведомления от отправителя отключены"
				return false, &reason, nil
			}
			if s.isChannelBlocked(senderPref.BlockedChannels, channel) {
				reason := "канал заблокирован в настройках отправителя"
				return false, &reason, nil
			}
			if s.isInQuietHours(senderPref.QuietFrom, senderPref.QuietTo, evaluateAt) {
				reason := "тихие часы для отправителя"
				return false, &reason, nil
			}
		}
	}

	if scopeID != nil {
		scopePref, err := s.prefRepo.GetByContactAndScope(ctx, contactID, scopeType, senderID, scopeID)
		if err == nil && scopePref != nil {
			if !scopePref.Enabled {
				reason := "уведомления для рассылки отключены"
				return false, &reason, nil
			}
			if s.isChannelBlocked(scopePref.BlockedChannels, channel) {
				reason := "канал заблокирован в настройках рассылки"
				return false, &reason, nil
			}
			if s.isInQuietHours(scopePref.QuietFrom, scopePref.QuietTo, evaluateAt) {
				reason := "тихие часы для рассылки"
				return false, &reason, nil
			}
		}
	}

	return true, nil, nil
}

func (s *testPreferenceService) isChannelBlocked(blockedChannels []string, channel string) bool {
	for _, c := range blockedChannels {
		if c == channel {
			return true
		}
	}
	return false
}

func (s *testPreferenceService) isInQuietHours(from, to *string, evaluateAt time.Time) bool {
	if from == nil || to == nil {
		return false
	}

	currentTime := evaluateAt.Format("15:04")

	// Handle overnight quiet hours (e.g., 22:00-08:00)
	if *from > *to {
		return currentTime >= *from || currentTime <= *to
	}

	// Normal span (e.g., 09:00-18:00)
	return currentTime >= *from && currentTime <= *to
}

func TestPreferenceService_GetByContactID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		contactID   string
		contactRepo *MockContactRepo
		prefRepo    *MockPreferenceRepo
		wantErr     error
		wantCount   int
	}{
		{
			name:      "успешное получение настроек контакта",
			contactID: "contact-1",
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1"}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactIDFunc: func(ctx context.Context, contactID string) ([]domain.Preference, error) {
					return []domain.Preference{{ID: "pref-1", ContactID: contactID}}, nil
				},
			},
			wantErr:   nil,
			wantCount: 1,
		},
		{
			name:      "контакт не найден",
			contactID: "nonexistent",
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return nil, repository.ErrNotFound
				},
			},
			prefRepo:  &MockPreferenceRepo{},
			wantErr:   ErrContactNotFound,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestPreferenceService(tt.prefRepo, tt.contactRepo, &MockAuditLogRepo{})

			resp, err := svc.GetByContactID(ctx, tt.contactID)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(resp) != tt.wantCount {
					t.Errorf("expected %d preferences, got %d", tt.wantCount, len(resp))
				}
			}
		})
	}
}

func TestPreferenceService_GetByUserID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      string
		contactRepo *MockContactRepo
		prefRepo    *MockPreferenceRepo
		wantErr     error
		wantCount   int
	}{
		{
			name:   "успешное получение настроек пользователя",
			userID: "user-1",
			contactRepo: &MockContactRepo{
				GetByUserIDFunc: func(ctx context.Context, userID string) ([]domain.Contact, error) {
					return []domain.Contact{{ID: "c1", UserID: userID}}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactIDFunc: func(ctx context.Context, contactID string) ([]domain.Preference, error) {
					return []domain.Preference{{ID: "pref-1", ContactID: contactID}}, nil
				},
			},
			wantErr:   nil,
			wantCount: 1,
		},
		{
			name:   "ошибка при получении контактов",
			userID: "user-1",
			contactRepo: &MockContactRepo{
				GetByUserIDFunc: func(ctx context.Context, userID string) ([]domain.Contact, error) {
					return nil, anError
				},
			},
			prefRepo:  &MockPreferenceRepo{},
			wantErr:   anError,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestPreferenceService(tt.prefRepo, tt.contactRepo, &MockAuditLogRepo{})

			resp, err := svc.GetByUserID(ctx, tt.userID)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(resp) != tt.wantCount {
					t.Errorf("expected %d preferences, got %d", tt.wantCount, len(resp))
				}
			}
		})
	}
}

func TestPreferenceService_Update(t *testing.T) {
	ctx := context.Background()

	from := "22:00"
	to := "08:00"

	tests := []struct {
		name        string
		userID      string
		req         *UpdatePreferenceRequest
		contactRepo *MockContactRepo
		prefRepo    *MockPreferenceRepo
		wantErr     error
	}{
		{
			name:   "успешное обновление настройки",
			userID: "user-1",
			req: &UpdatePreferenceRequest{
				ContactID: "contact-1",
				ScopeType: "global",
				Enabled:   true,
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1"}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				UpsertFunc: func(ctx context.Context, pref *domain.Preference) error {
					return nil
				},
			},
			wantErr: nil,
		},
		{
			name:   "контакт не найден",
			userID: "user-1",
			req: &UpdatePreferenceRequest{
				ContactID: "nonexistent",
				ScopeType: "global",
				Enabled:   true,
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return nil, repository.ErrNotFound
				},
			},
			prefRepo: &MockPreferenceRepo{},
			wantErr:  ErrContactNotFound,
		},
		{
			name:   "нарушение прав доступа",
			userID: "other-user",
			req: &UpdatePreferenceRequest{
				ContactID: "contact-1",
				ScopeType: "global",
				Enabled:   true,
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1"}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{},
			wantErr:  errors.New("контакт не принадлежит пользователю"),
		},
		{
			name:   "некорректный формат времени quiet_from",
			userID: "user-1",
			req: &UpdatePreferenceRequest{
				ContactID: "contact-1",
				ScopeType: "global",
				Enabled:   true,
				QuietFrom: strPtr("not-a-time"),
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1"}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{},
			wantErr:  errors.New("некорректный формат времени quiet_from (ожидается HH:MM)"),
		},
		{
			name:   "некорректный формат времени quiet_to",
			userID: "user-1",
			req: &UpdatePreferenceRequest{
				ContactID: "contact-1",
				ScopeType: "global",
				Enabled:   true,
				QuietTo:   strPtr("not-a-time"),
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1"}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{},
			wantErr:  errors.New("некорректный формат времени quiet_to (ожидается HH:MM)"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestPreferenceService(tt.prefRepo, tt.contactRepo, &MockAuditLogRepo{})

			_, err := svc.Update(ctx, tt.userID, tt.req)

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

	_ = from
	_ = to
}

func strPtr(s string) *string {
	return &s
}

func TestPreferenceService_validateQuietHours(t *testing.T) {
	svc := newTestPreferenceService(&MockPreferenceRepo{}, &MockContactRepo{}, &MockAuditLogRepo{})

	tests := []struct {
		name    string
		from    *string
		to      *string
		wantErr error
	}{
		{
			name:    "валидное время без тихих часов",
			from:    nil,
			to:      nil,
			wantErr: nil,
		},
		{
			name:    "валидное время - начало",
			from:    strPtr("22:00"),
			to:      nil,
			wantErr: nil,
		},
		{
			name:    "валидное время - конец",
			from:    nil,
			to:      strPtr("08:00"),
			wantErr: nil,
		},
		{
			name:    "валидное время - оба",
			from:    strPtr("22:00"),
			to:      strPtr("08:00"),
			wantErr: nil,
		},
		{
			name:    "невалидное время - начало",
			from:    strPtr("25:00"),
			to:      strPtr("08:00"),
			wantErr: errors.New("некорректный формат времени quiet_from (ожидается HH:MM)"),
		},
		{
			name:    "невалидное время - конец",
			from:    strPtr("22:00"),
			to:      strPtr("30:00"),
			wantErr: errors.New("некорректный формат времени quiet_to (ожидается HH:MM)"),
		},
		{
			name:    "невалидное время - неверный формат",
			from:    strPtr("wrong"),
			to:      strPtr("08:00"),
			wantErr: errors.New("некорректный формат времени quiet_from (ожидается HH:MM)"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateQuietHours(tt.from, tt.to)

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

func TestPreferenceService_EvaluatePreference(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

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
		wantReason  *string
		wantErr     error
	}{
		{
			name:       "успешная оценка - отключенный контакт",
			contactID:  "contact-1",
			channel:    "email",
			evaluateAt: now,
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: false}, nil
				},
			},
			prefRepo:    &MockPreferenceRepo{},
			wantAllowed: false,
			wantReason:  strPtr("контакт отключен"),
		},
		{
			name:       "успешная оценка - глобальные уведомления отключены",
			contactID:  "contact-1",
			channel:    "email",
			evaluateAt: now,
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{Enabled: false}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: false,
			wantReason:  strPtr("глобальные уведомления отключены"),
		},
		{
			name:       "успешная оценка - канал заблокирован",
			contactID:  "contact-1",
			channel:    "sms",
			evaluateAt: now,
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{Enabled: true, BlockedChannels: []string{"sms"}}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: false,
			wantReason:  strPtr("канал заблокирован в глобальных настройках"),
		},
		{
			name:       "успешная оценка - тихие часы",
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
						from := "10:00"
						to := "18:00"
						return &domain.Preference{Enabled: true, QuietFrom: &from, QuietTo: &to}, nil
					}
					return nil, repository.ErrNotFound
				},
			},
			wantAllowed: false,
			wantReason:  strPtr("тихие часы"),
		},
		{
			name:       "успешная оценка - уведомление разрешено",
			contactID:  "contact-1",
			channel:    "email",
			evaluateAt: now,
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Enabled: true}, nil
				},
			},
			prefRepo: &MockPreferenceRepo{
				GetByContactAndScopeFunc: func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
					if scopeType == "global" {
						return &domain.Preference{Enabled: true}, nil
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

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if allowed != tt.wantAllowed {
					t.Errorf("expected allowed=%v, got %v", tt.wantAllowed, allowed)
				}
				if tt.wantReason != nil && reason == nil {
					t.Errorf("expected reason %v, got nil", *tt.wantReason)
				}
				if tt.wantReason == nil && reason != nil {
					t.Errorf("expected no reason, got %v", *reason)
				}
			}
		})
	}
}

func TestPreferenceService_isChannelBlocked(t *testing.T) {
	svc := newTestPreferenceService(&MockPreferenceRepo{}, &MockContactRepo{}, &MockAuditLogRepo{})

	tests := []struct {
		name            string
		blockedChannels []string
		channel         string
		wantBlocked     bool
	}{
		{
			name:            "канал заблокирован",
			blockedChannels: []string{"sms", "telegram"},
			channel:         "sms",
			wantBlocked:     true,
		},
		{
			name:            "канал не заблокирован",
			blockedChannels: []string{"sms", "telegram"},
			channel:         "email",
			wantBlocked:     false,
		},
		{
			name:            "пустой список блокировок",
			blockedChannels: []string{},
			channel:         "email",
			wantBlocked:     false,
		},
		{
			name:            "nil список блокировок",
			blockedChannels: nil,
			channel:         "email",
			wantBlocked:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked := svc.isChannelBlocked(tt.blockedChannels, tt.channel)

			if blocked != tt.wantBlocked {
				t.Errorf("expected blocked=%v, got %v", tt.wantBlocked, blocked)
			}
		})
	}
}

func TestPreferenceService_isInQuietHours(t *testing.T) {
	svc := newTestPreferenceService(&MockPreferenceRepo{}, &MockContactRepo{}, &MockAuditLogRepo{})

	from := "22:00"
	to := "08:00"
	dayFrom := "10:00"
	dayTo := "18:00"

	tests := []struct {
		name       string
		from       *string
		to         *string
		evaluateAt time.Time
		wantResult bool
	}{
		{
			name:       "внутри дневных часов",
			from:       &dayFrom,
			to:         &dayTo,
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			wantResult: true,
		},
		{
			name:       "вне дневных часов",
			from:       &dayFrom,
			to:         &dayTo,
			evaluateAt: time.Date(2024, 1, 1, 8, 0, 0, 0, time.UTC),
			wantResult: false,
		},
		{
			name:       "nil from",
			from:       nil,
			to:         &dayTo,
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			wantResult: false,
		},
		{
			name:       "nil to",
			from:       &dayFrom,
			to:         nil,
			evaluateAt: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.isInQuietHours(tt.from, tt.to, tt.evaluateAt)

			if result != tt.wantResult {
				t.Errorf("expected %v, got %v", tt.wantResult, result)
			}
		})
	}

	_ = from
	_ = to
}
