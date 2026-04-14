package service

import (
	"context"
	"errors"
	"testing"

	"notification-center/recipients/internal/domain"
	"notification-center/recipients/internal/repository"
)

func TestContactScenarios_AddEmailContact(t *testing.T) {
	ctx := context.Background()

	scenarios := []struct {
		name      string
		userID    string
		email     string
		setupMock func() *MockContactRepo
		wantErr   error
		wantSaved bool
		wantValue string
	}{
		{
			name:   "успешное добавление email контакта",
			userID: "user-1",
			email:  "user@example.com",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
						return nil, repository.ErrNotFound
					},
					CreateFunc: func(ctx context.Context, contact *domain.Contact) error {
						contact.ID = "contact-email-1"
						return nil
					},
				}
			},
			wantErr:   nil,
			wantSaved: true,
			wantValue: "user@example.com",
		},
		{
			name:   "успешное добавление email с поддоменом",
			userID: "user-1",
			email:  "test.user@example.co.uk",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
						return nil, repository.ErrNotFound
					},
					CreateFunc: func(ctx context.Context, contact *domain.Contact) error {
						contact.ID = "contact-email-2"
						return nil
					},
				}
			},
			wantErr:   nil,
			wantSaved: true,
			wantValue: "test.user@example.co.uk",
		},
		{
			name:   "ошибка - некорректный формат email (без @)",
			userID: "user-1",
			email:  "userexample.com",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{}
			},
			wantErr:   ErrInvalidEmail,
			wantSaved: false,
		},
		{
			name:   "ошибка - некорректный формат email (без домена)",
			userID: "user-1",
			email:  "user@",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{}
			},
			wantErr:   ErrInvalidEmail,
			wantSaved: false,
		},
		{
			name:   "ошибка - дубликат email контакта",
			userID: "user-1",
			email:  "existing@example.com",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
						return &domain.Contact{ID: "existing-contact", UserID: userID, Channel: channel, Value: value}, nil
					},
				}
			},
			wantErr:   ErrContactExists,
			wantSaved: false,
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupMock()
			svc := newTestContactService(repo, &MockAuditLogRepo{})

			req := &CreateContactRequest{
				Channel: "email",
				Value:   tt.email,
			}

			contact, err := svc.Create(ctx, tt.userID, req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ожидалась ошибка %v, получено nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ожидалась ошибка %v, получена %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("неожиданная ошибка: %v", err)
				return
			}

			if tt.wantSaved && contact == nil {
				t.Error("ожидался созданный контакт, получено nil")
				return
			}

			if tt.wantValue != "" && contact.Value != tt.wantValue {
				t.Errorf("ожидалось значение %s, получено %s", tt.wantValue, contact.Value)
			}
		})
	}
}

func TestContactScenarios_AddPhoneContact(t *testing.T) {
	ctx := context.Background()

	scenarios := []struct {
		name      string
		userID    string
		phone     string
		setupMock func() *MockContactRepo
		wantErr   error
		wantSaved bool
		wantValue string
	}{
		{
			name:   "успешное добавление телефона с +7",
			userID: "user-1",
			phone:  "+79001234567",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
						return nil, repository.ErrNotFound
					},
					CreateFunc: func(ctx context.Context, contact *domain.Contact) error {
						contact.ID = "contact-sms-1"
						return nil
					},
				}
			},
			wantErr:   nil,
			wantSaved: true,
			wantValue: "+79001234567",
		},
		{
			name:   "успешное добавление телефона без + (8-ка)",
			userID: "user-1",
			phone:  "89001234567",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
						return nil, repository.ErrNotFound
					},
					CreateFunc: func(ctx context.Context, contact *domain.Contact) error {
						contact.ID = "contact-sms-2"
						return nil
					},
				}
			},
			wantErr:   nil,
			wantSaved: true,
			wantValue: "89001234567",
		},
		{
			name:   "ошибка - слишком короткий номер телефона",
			userID: "user-1",
			phone:  "123",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{}
			},
			wantErr:   ErrInvalidPhone,
			wantSaved: false,
		},
		{
			name:   "ошибка - номер телефона с буквами",
			userID: "user-1",
			phone:  "+7900ABCD567",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{}
			},
			wantErr:   ErrInvalidPhone,
			wantSaved: false,
		},
		{
			name:   "ошибка - дубликат телефонного контакта",
			userID: "user-1",
			phone:  "+79001234567",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
						return &domain.Contact{ID: "existing-sms", UserID: userID, Channel: channel, Value: value}, nil
					},
				}
			},
			wantErr:   ErrContactExists,
			wantSaved: false,
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupMock()
			svc := newTestContactService(repo, &MockAuditLogRepo{})

			req := &CreateContactRequest{
				Channel: "sms",
				Value:   tt.phone,
			}

			contact, err := svc.Create(ctx, tt.userID, req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ожидалась ошибка %v, получено nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ожидалась ошибка %v, получена %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("неожиданная ошибка: %v", err)
				return
			}

			if tt.wantSaved && contact == nil {
				t.Error("ожидался созданный контакт, получено nil")
				return
			}

			if tt.wantValue != "" && contact.Value != tt.wantValue {
				t.Errorf("ожидалось значение %s, получено %s", tt.wantValue, contact.Value)
			}
		})
	}
}

func TestContactScenarios_AddTelegramContact(t *testing.T) {
	ctx := context.Background()

	scenarios := []struct {
		name      string
		userID    string
		telegram  string
		setupMock func() *MockContactRepo
		wantErr   error
		wantSaved bool
		wantValue string
	}{
		{
			name:     "успешное добавление telegram с @",
			userID:   "user-1",
			telegram: "@username",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
						return nil, repository.ErrNotFound
					},
					CreateFunc: func(ctx context.Context, contact *domain.Contact) error {
						contact.ID = "contact-tg-1"
						return nil
					},
				}
			},
			wantErr:   nil,
			wantSaved: true,
			wantValue: "@username",
		},
		{
			name:     "успешное добавление telegram без @",
			userID:   "user-1",
			telegram: "username123",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
						return nil, repository.ErrNotFound
					},
					CreateFunc: func(ctx context.Context, contact *domain.Contact) error {
						contact.ID = "contact-tg-2"
						return nil
					},
				}
			},
			wantErr:   nil,
			wantSaved: true,
			wantValue: "username123",
		},
		{
			name:     "ошибка - слишком короткий username",
			userID:   "user-1",
			telegram: "ab",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{}
			},
			wantErr:   ErrInvalidTelegram,
			wantSaved: false,
		},
		{
			name:     "ошибка - username с недопустимыми символами",
			userID:   "user-1",
			telegram: "@user-name",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{}
			},
			wantErr:   ErrInvalidTelegram,
			wantSaved: false,
		},
		{
			name:     "ошибка - дубликат telegram контакта",
			userID:   "user-1",
			telegram: "@existing",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
						return &domain.Contact{ID: "existing-tg", UserID: userID, Channel: channel, Value: value}, nil
					},
				}
			},
			wantErr:   ErrContactExists,
			wantSaved: false,
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupMock()
			svc := newTestContactService(repo, &MockAuditLogRepo{})

			req := &CreateContactRequest{
				Channel: "telegram",
				Value:   tt.telegram,
			}

			contact, err := svc.Create(ctx, tt.userID, req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ожидалась ошибка %v, получено nil", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("ожидалась ошибка %v, получена %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("неожиданная ошибка: %v", err)
				return
			}

			if tt.wantSaved && contact == nil {
				t.Error("ожидался созданный контакт, получено nil")
				return
			}

			if tt.wantValue != "" && contact.Value != tt.wantValue {
				t.Errorf("ожидалось значение %s, получено %s", tt.wantValue, contact.Value)
			}
		})
	}
}

func TestContactScenarios_UpdateContact(t *testing.T) {
	ctx := context.Background()

	scenarios := []struct {
		name        string
		contactID   string
		userID      string
		newValue    string
		newChannel  string
		setupMock   func() *MockContactRepo
		wantErr     error
		wantValue   string
		wantChannel string
	}{
		{
			name:      "успешное изменение email контакта",
			contactID: "contact-1",
			userID:    "user-1",
			newValue:  "newemail@example.com",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
						return &domain.Contact{ID: id, UserID: "user-1", Channel: "email", Value: "old@example.com"}, nil
					},
					GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
						return nil, repository.ErrNotFound
					},
					UpdateFunc: func(ctx context.Context, contact *domain.Contact) error {
						return nil
					},
				}
			},
			wantErr:     nil,
			wantValue:   "newemail@example.com",
			wantChannel: "email",
		},
		{
			name:       "успешная смена канала с email на sms",
			contactID:  "contact-1",
			userID:     "user-1",
			newValue:   "+79001234567",
			newChannel: "sms",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
						return &domain.Contact{ID: id, UserID: "user-1", Channel: "email", Value: "old@example.com"}, nil
					},
					GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
						return nil, repository.ErrNotFound
					},
					UpdateFunc: func(ctx context.Context, contact *domain.Contact) error {
						return nil
					},
				}
			},
			wantErr:     nil,
			wantValue:   "+79001234567",
			wantChannel: "sms",
		},
		{
			name:      "ошибка - контакт не найден",
			contactID: "nonexistent",
			userID:    "user-1",
			newValue:  "test@example.com",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
						return nil, repository.ErrNotFound
					},
				}
			},
			wantErr: ErrContactNotFound,
		},
		{
			name:      "ошибка - контакт принадлежит другому пользователю",
			contactID: "contact-1",
			userID:    "other-user",
			newValue:  "test@example.com",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
						return &domain.Contact{ID: id, UserID: "user-1", Channel: "email", Value: "old@example.com"}, nil
					},
				}
			},
			wantErr: errors.New("контакт не принадлежит пользователю"),
		},
		{
			name:      "ошибка - обновление до уже существующего значения",
			contactID: "contact-1",
			userID:    "user-1",
			newValue:  "existing@example.com",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
						return &domain.Contact{ID: id, UserID: "user-1", Channel: "email", Value: "old@example.com"}, nil
					},
					GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
						return &domain.Contact{ID: "other-contact", UserID: userID, Channel: channel, Value: value}, nil
					},
				}
			},
			wantErr: ErrContactExists,
		},
		{
			name:      "ошибка - некорректный формат при обновлении",
			contactID: "contact-1",
			userID:    "user-1",
			newValue:  "invalid-email",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
						return &domain.Contact{ID: id, UserID: "user-1", Channel: "email", Value: "old@example.com"}, nil
					},
				}
			},
			wantErr: ErrInvalidEmail,
		},
		{
			name:      "успешное обновление без изменения значения",
			contactID: "contact-1",
			userID:    "user-1",
			newValue:  "old@example.com",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
						return &domain.Contact{ID: id, UserID: "user-1", Channel: "email", Value: "old@example.com"}, nil
					},
					UpdateFunc: func(ctx context.Context, contact *domain.Contact) error {
						return nil
					},
				}
			},
			wantErr:     nil,
			wantValue:   "old@example.com",
			wantChannel: "email",
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupMock()
			svc := newTestContactService(repo, &MockAuditLogRepo{})

			channel := tt.newChannel
			if channel == "" {
				channel = "email"
			}

			req := &UpdateContactRequest{
				Channel: channel,
				Value:   tt.newValue,
			}

			contact, err := svc.Update(ctx, tt.contactID, tt.userID, req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ожидалась ошибка %v, получено nil", tt.wantErr)
					return
				}
				if tt.wantErr.Error() != err.Error() && !errors.Is(err, tt.wantErr) {
					t.Errorf("ожидалась ошибка '%v', получена '%v'", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("неожиданная ошибка: %v", err)
				return
			}

			if tt.wantValue != "" && contact.Value != tt.wantValue {
				t.Errorf("ожидалось значение %s, получено %s", tt.wantValue, contact.Value)
			}

			if tt.wantChannel != "" && contact.Channel != tt.wantChannel {
				t.Errorf("ожидался канал %s, получен %s", tt.wantChannel, contact.Channel)
			}
		})
	}
}

func TestContactScenarios_DeleteContact(t *testing.T) {
	ctx := context.Background()

	scenarios := []struct {
		name       string
		contactID  string
		userID     string
		setupMock  func() *MockContactRepo
		wantErr    error
		wantCalled bool
	}{
		{
			name:      "успешное удаление контакта",
			contactID: "contact-1",
			userID:    "user-1",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
						return &domain.Contact{ID: id, UserID: "user-1", Channel: "email", Value: "test@example.com"}, nil
					},
					SoftDeleteFunc: func(ctx context.Context, id string) error {
						return nil
					},
				}
			},
			wantErr:    nil,
			wantCalled: true,
		},
		{
			name:      "ошибка - контакт не найден",
			contactID: "nonexistent",
			userID:    "user-1",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
						return nil, repository.ErrNotFound
					},
				}
			},
			wantErr:    ErrContactNotFound,
			wantCalled: false,
		},
		{
			name:      "ошибка - контакт принадлежит другому пользователю",
			contactID: "contact-1",
			userID:    "other-user",
			setupMock: func() *MockContactRepo {
				return &MockContactRepo{
					GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
						return &domain.Contact{ID: id, UserID: "user-1", Channel: "email", Value: "test@example.com"}, nil
					},
				}
			},
			wantErr:    errors.New("контакт не принадлежит пользователю"),
			wantCalled: false,
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupMock()
			svc := newTestContactService(repo, &MockAuditLogRepo{})

			err := svc.Delete(ctx, tt.contactID, tt.userID)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ожидалась ошибка %v, получено nil", tt.wantErr)
					return
				}
				if tt.wantErr.Error() != err.Error() {
					t.Errorf("ожидалась ошибка '%v', получена '%v'", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("неожиданная ошибка: %v", err)
				return
			}

			if tt.wantCalled && repo.SoftDeleteFunc == nil {
				t.Error("ожидался вызов SoftDelete, но функция не была вызвана")
			}
		})
	}
}

func TestContactScenarios_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("полный сценарий: создать, обновить и удалить контакт", func(t *testing.T) {
		var createdID string

		repo := &MockContactRepo{
			GetByUserIDAndChannelFunc: func(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
				return nil, repository.ErrNotFound
			},
			CreateFunc: func(ctx context.Context, contact *domain.Contact) error {
				contact.ID = "contact-new-1"
				createdID = contact.ID
				return nil
			},
			GetByIDFunc: func(ctx context.Context, id string) (*domain.Contact, error) {
				if id == createdID {
					return &domain.Contact{ID: id, UserID: "user-1", Channel: "email", Value: "test@example.com"}, nil
				}
				return nil, repository.ErrNotFound
			},
			UpdateFunc: func(ctx context.Context, contact *domain.Contact) error {
				return nil
			},
			SoftDeleteFunc: func(ctx context.Context, id string) error {
				return nil
			},
		}

		svc := newTestContactService(repo, &MockAuditLogRepo{})

		createReq := &CreateContactRequest{
			Channel: "email",
			Value:   "test@example.com",
		}

		contact, err := svc.Create(ctx, "user-1", createReq)
		if err != nil {
			t.Fatalf("создание контакта должно было пройти успешно: %v", err)
		}
		if contact == nil || contact.ID == "" {
			t.Fatal("созданный контакт должен иметь ID")
		}

		updateReq := &UpdateContactRequest{
			Channel: "email",
			Value:   "updated@example.com",
		}

		updatedContact, err := svc.Update(ctx, contact.ID, "user-1", updateReq)
		if err != nil {
			t.Fatalf("обновление контакта должно было пройти успешно: %v", err)
		}
		if updatedContact.Value != "updated@example.com" {
			t.Errorf("ожидалось значение updated@example.com, получено %s", updatedContact.Value)
		}

		err = svc.Delete(ctx, contact.ID, "user-1")
		if err != nil {
			t.Fatalf("удаление контакта должно было пройти успешно: %v", err)
		}

		deletedContact, err := svc.GetByID(ctx, contact.ID)
		if err != nil && !errors.Is(err, ErrContactNotFound) {
			t.Errorf("после удаления контакт не должен быть доступен: %v", err)
		}
		_ = deletedContact
	})
}
