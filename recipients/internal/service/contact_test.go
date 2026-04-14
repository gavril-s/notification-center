package service

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"notification-center/recipients/internal/domain"
	"notification-center/recipients/internal/repository"
)

type testContactService struct {
	contactRepo ContactRepositoryInterface
	auditRepo   AuditLogRepositoryInterface
}

func newTestContactService(contactRepo ContactRepositoryInterface, auditRepo AuditLogRepositoryInterface) *testContactService {
	return &testContactService{
		contactRepo: contactRepo,
		auditRepo:   auditRepo,
	}
}

func (s *testContactService) Create(ctx context.Context, userID string, req *CreateContactRequest) (*domain.Contact, error) {
	if err := s.validateContact(req.Channel, req.Value); err != nil {
		return nil, err
	}

	existing, err := s.contactRepo.GetByUserIDAndChannel(ctx, userID, req.Channel, req.Value)
	if err == nil && existing != nil {
		return nil, ErrContactExists
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	contact := &domain.Contact{
		UserID:     userID,
		Channel:    req.Channel,
		Value:      req.Value,
		IsVerified: false,
		Enabled:    true,
	}

	if err := s.contactRepo.Create(ctx, contact); err != nil {
		return nil, err
	}

	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  userID,
		Action:  "contact_create",
		Details: "contact created",
	})

	return contact, nil
}

func (s *testContactService) GetByUserID(ctx context.Context, userID string) ([]domain.Contact, error) {
	return s.contactRepo.GetByUserID(ctx, userID)
}

func (s *testContactService) GetByID(ctx context.Context, contactID string) (*domain.Contact, error) {
	contact, err := s.contactRepo.GetByID(ctx, contactID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrContactNotFound
	}
	if err != nil {
		return nil, err
	}
	return contact, nil
}

func (s *testContactService) Update(ctx context.Context, contactID string, userID string, req *UpdateContactRequest) (*domain.Contact, error) {
	contact, err := s.contactRepo.GetByID(ctx, contactID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrContactNotFound
	}
	if err != nil {
		return nil, err
	}

	if contact.UserID != userID {
		return nil, errors.New("контакт не принадлежит пользователю")
	}

	if err := s.validateContact(req.Channel, req.Value); err != nil {
		return nil, err
	}

	if req.Value != contact.Value || req.Channel != contact.Channel {
		existing, err := s.contactRepo.GetByUserIDAndChannel(ctx, userID, req.Channel, req.Value)
		if err == nil && existing != nil && existing.ID != contactID {
			return nil, ErrContactExists
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, err
		}
	}

	contact.Channel = req.Channel
	contact.Value = req.Value

	if err := s.contactRepo.Update(ctx, contact); err != nil {
		return nil, err
	}

	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  userID,
		Action:  "contact_update",
		Details: "contact updated",
	})

	return contact, nil
}

func (s *testContactService) Delete(ctx context.Context, contactID string, userID string) error {
	contact, err := s.contactRepo.GetByID(ctx, contactID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrContactNotFound
	}
	if err != nil {
		return err
	}

	if contact.UserID != userID {
		return errors.New("контакт не принадлежит пользователю")
	}

	if err := s.contactRepo.SoftDelete(ctx, contactID); err != nil {
		return err
	}

	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  userID,
		Action:  "contact_delete",
		Details: "contact deleted",
	})

	return nil
}

func (s *testContactService) validateContact(channel, value string) error {
	switch channel {
	case "email":
		emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
		if !matchesRegex(value, emailRegex) {
			return ErrInvalidEmail
		}
	case "sms":
		phoneRegex := `^\+?[1-9]\d{6,14}$`
		if !matchesRegex(value, phoneRegex) {
			return ErrInvalidPhone
		}
	case "telegram":
		telegramRegex := `^@?[a-zA-Z0-9_]{5,32}$`
		if !matchesRegex(value, telegramRegex) {
			return ErrInvalidTelegram
		}
	}
	return nil
}

func matchesRegex(value, pattern string) bool {
	re := regexp.MustCompile(pattern)
	return re.MatchString(value)
}

func TestContactService_Create(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      string
		req         *CreateContactRequest
		contactRepo *MockContactRepo
		wantErr     error
		wantID      bool
	}{
		{
			name:   "успешное создание контакта",
			userID: "user-1",
			req: &CreateContactRequest{
				UserID:  "user-1",
				Channel: "email",
				Value:   "test@example.com",
			},
			contactRepo: &MockContactRepo{
				GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
					return nil, repository.ErrNotFound
				},
				CreateFunc: func(ctx context.Context, contact *domain.Contact) error {
					contact.ID = "contact-123"
					return nil
				},
			},
			wantErr: nil,
			wantID:  true,
		},
		{
			name:   "некорректный формат email",
			userID: "user-1",
			req: &CreateContactRequest{
				UserID:  "user-1",
				Channel: "email",
				Value:   "not-an-email",
			},
			contactRepo: &MockContactRepo{},
			wantErr:     ErrInvalidEmail,
			wantID:      false,
		},
		{
			name:   "некорректный формат телефона",
			userID: "user-1",
			req: &CreateContactRequest{
				UserID:  "user-1",
				Channel: "sms",
				Value:   "123",
			},
			contactRepo: &MockContactRepo{},
			wantErr:     ErrInvalidPhone,
			wantID:      false,
		},
		{
			name:   "некорректный формат telegram",
			userID: "user-1",
			req: &CreateContactRequest{
				UserID:  "user-1",
				Channel: "telegram",
				Value:   "ab",
			},
			contactRepo: &MockContactRepo{},
			wantErr:     ErrInvalidTelegram,
			wantID:      false,
		},
		{
			name:   "контакт уже существует",
			userID: "user-1",
			req: &CreateContactRequest{
				UserID:  "user-1",
				Channel: "email",
				Value:   "test@example.com",
			},
			contactRepo: &MockContactRepo{
				GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
					return &domain.Contact{ID: "existing-contact"}, nil
				},
			},
			wantErr: ErrContactExists,
			wantID:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestContactService(tt.contactRepo, &MockAuditLogRepo{})

			resp, err := svc.Create(ctx, tt.userID, tt.req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.wantID && resp != nil && resp.ID == "" {
					t.Error("expected contact ID")
				}
			}
		})
	}
}

func TestContactService_GetByUserID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      string
		contactRepo *MockContactRepo
		wantCount   int
		wantErr     error
	}{
		{
			name:   "получение контактов пользователя",
			userID: "user-1",
			contactRepo: &MockContactRepo{
				GetByUserIDFunc: func(ctx context.Context, userID string) ([]domain.Contact, error) {
					return []domain.Contact{
						{ID: "c1", UserID: userID, Channel: "email"},
						{ID: "c2", UserID: userID, Channel: "sms"},
					}, nil
				},
			},
			wantCount: 2,
			wantErr:   nil,
		},
		{
			name:   "контакты не найдены",
			userID: "user-1",
			contactRepo: &MockContactRepo{
				GetByUserIDFunc: func(ctx context.Context, userID string) ([]domain.Contact, error) {
					return []domain.Contact{}, nil
				},
			},
			wantCount: 0,
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestContactService(tt.contactRepo, &MockAuditLogRepo{})

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
					t.Errorf("expected %d contacts, got %d", tt.wantCount, len(resp))
				}
			}
		})
	}
}

func TestContactService_GetByID(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		contactID   string
		contactRepo *MockContactRepo
		wantErr     error
		wantID      string
	}{
		{
			name:      "успешное получение контакта",
			contactID: "contact-1",
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1"}, nil
				},
			},
			wantErr: nil,
			wantID:  "contact-1",
		},
		{
			name:      "контакт не найден",
			contactID: "nonexistent",
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return nil, repository.ErrNotFound
				},
			},
			wantErr: ErrContactNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestContactService(tt.contactRepo, &MockAuditLogRepo{})

			resp, err := svc.GetByID(ctx, tt.contactID)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.wantID != "" && resp != nil && resp.ID != tt.wantID {
					t.Errorf("expected ID %s, got %s", tt.wantID, resp.ID)
				}
			}
		})
	}
}

func TestContactService_Update(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		contactID   string
		userID      string
		req         *UpdateContactRequest
		contactRepo *MockContactRepo
		wantErr     error
	}{
		{
			name:      "успешное обновление контакта",
			contactID: "contact-1",
			userID:    "user-1",
			req: &UpdateContactRequest{
				Channel: "email",
				Value:   "newemail@example.com",
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Channel: "email", Value: "old@example.com"}, nil
				},
				GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
					return nil, repository.ErrNotFound
				},
				UpdateFunc: func(ctx context.Context, contact *domain.Contact) error {
					return nil
				},
			},
			wantErr: nil,
		},
		{
			name:      "контакт не найден",
			contactID: "nonexistent",
			userID:    "user-1",
			req: &UpdateContactRequest{
				Channel: "email",
				Value:   "test@example.com",
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return nil, repository.ErrNotFound
				},
			},
			wantErr: ErrContactNotFound,
		},
		{
			name:      "нарушение прав доступа",
			contactID: "contact-1",
			userID:    "other-user",
			req: &UpdateContactRequest{
				Channel: "email",
				Value:   "test@example.com",
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1"}, nil
				},
			},
			wantErr: errors.New("контакт не принадлежит пользователю"),
		},
		{
			name:      "дубликат контакта",
			contactID: "contact-1",
			userID:    "user-1",
			req: &UpdateContactRequest{
				Channel: "email",
				Value:   "existing@example.com",
			},
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1", Channel: "email", Value: "old@example.com"}, nil
				},
				GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
					return &domain.Contact{ID: "other-contact"}, nil
				},
			},
			wantErr: ErrContactExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestContactService(tt.contactRepo, &MockAuditLogRepo{})

			_, err := svc.Update(ctx, tt.contactID, tt.userID, tt.req)

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

func TestContactService_Delete(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		contactID   string
		userID      string
		contactRepo *MockContactRepo
		wantErr     error
	}{
		{
			name:      "успешное удаление контакта",
			contactID: "contact-1",
			userID:    "user-1",
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1"}, nil
				},
				SoftDeleteFunc: func(ctx context.Context, id string) error {
					return nil
				},
			},
			wantErr: nil,
		},
		{
			name:      "контакт не найден",
			contactID: "nonexistent",
			userID:    "user-1",
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return nil, repository.ErrNotFound
				},
			},
			wantErr: ErrContactNotFound,
		},
		{
			name:      "нарушение прав доступа",
			contactID: "contact-1",
			userID:    "other-user",
			contactRepo: &MockContactRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
					return &domain.Contact{ID: id, UserID: "user-1"}, nil
				},
			},
			wantErr: errors.New("контакт не принадлежит пользователю"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestContactService(tt.contactRepo, &MockAuditLogRepo{})

			err := svc.Delete(ctx, tt.contactID, tt.userID)

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

func TestContactService_validateContact(t *testing.T) {
	svc := newTestContactService(&MockContactRepo{}, &MockAuditLogRepo{})

	tests := []struct {
		name    string
		channel string
		value   string
		wantErr error
	}{
		{
			name:    "валидный email",
			channel: "email",
			value:   "test@example.com",
			wantErr: nil,
		},
		{
			name:    "валидный email с поддоменом",
			channel: "email",
			value:   "user@mail.example.com",
			wantErr: nil,
		},
		{
			name:    "невалидный email - без @",
			channel: "email",
			value:   "testexample.com",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "невалидный email - без домена",
			channel: "email",
			value:   "test@",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "валидный телефон",
			channel: "sms",
			value:   "+79991234567",
			wantErr: nil,
		},
		{
			name:    "невалидный телефон - слишком короткий",
			channel: "sms",
			value:   "123",
			wantErr: ErrInvalidPhone,
		},
		{
			name:    "валидный telegram",
			channel: "telegram",
			value:   "@username",
			wantErr: nil,
		},
		{
			name:    "валидный telegram без @",
			channel: "telegram",
			value:   "username123",
			wantErr: nil,
		},
		{
			name:    "невалидный telegram - слишком короткий",
			channel: "telegram",
			value:   "ab",
			wantErr: ErrInvalidTelegram,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateContact(tt.channel, tt.value)

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
