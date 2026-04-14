package service

import (
	"context"

	"notification-center/recipients/internal/domain"
	"notification-center/recipients/internal/repository"
)

type UserRepositoryInterface interface {
	GetByLogin(ctx context.Context, login string) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
}

type RefreshTokenRepositoryInterface interface {
	GetByToken(ctx context.Context, token string) (*domain.RefreshToken, error)
	Create(ctx context.Context, token *domain.RefreshToken) error
	Delete(ctx context.Context, token string) error
}

type AuditLogRepositoryInterface interface {
	Create(ctx context.Context, log *domain.AuditLog) error
}

type ContactRepositoryInterface interface {
	Create(ctx context.Context, contact *domain.Contact) error
	GetByID(ctx context.Context, id string) (*domain.Contact, error)
	GetByUserID(ctx context.Context, userID string) ([]domain.Contact, error)
	GetAllByUserID(ctx context.Context, userID string) ([]domain.Contact, error)
	GetByUserIDAndChannel(ctx context.Context, userID, channel, value string) (*domain.Contact, error)
	Update(ctx context.Context, contact *domain.Contact) error
	SoftDelete(ctx context.Context, id string) error
	GetByIDs(ctx context.Context, ids []string) ([]domain.Contact, error)
}

type PreferenceRepositoryInterface interface {
	Create(ctx context.Context, pref *domain.Preference) error
	GetByID(ctx context.Context, id string) (*domain.Preference, error)
	GetByContactID(ctx context.Context, contactID string) ([]domain.Preference, error)
	GetByContactAndScope(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error)
	Update(ctx context.Context, pref *domain.Preference) error
	Upsert(ctx context.Context, pref *domain.Preference) error
}

type UnsubscribeTokenRepositoryInterface interface {
	GetByToken(ctx context.Context, token string) (*domain.UnsubscribeToken, error)
}

type UnsubscribeRuleRepositoryInterface interface {
	Create(ctx context.Context, rule *domain.UnsubscribeRule) error
	GetByContactID(ctx context.Context, contactID string) ([]domain.UnsubscribeRule, error)
	ExistsByContactAndScope(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (bool, error)
	Delete(ctx context.Context, id string) error
	DeleteByContactAndScope(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) error
}

type MockUserRepo struct {
	GetByLoginFunc func(ctx context.Context, login string) (*domain.User, error)
	GetByIDFunc    func(ctx context.Context, id string) (*domain.User, error)
	CreateFunc     func(ctx context.Context, user *domain.User) error
}

func (m *MockUserRepo) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	if m.GetByLoginFunc != nil {
		return m.GetByLoginFunc(ctx, login)
	}
	return nil, repository.ErrNotFound
}

func (m *MockUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, repository.ErrNotFound
}

func (m *MockUserRepo) Create(ctx context.Context, user *domain.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, user)
	}
	return nil
}

type MockRefreshTokenRepo struct {
	GetByTokenFunc func(ctx context.Context, token string) (*domain.RefreshToken, error)
	CreateFunc     func(ctx context.Context, token *domain.RefreshToken) error
	DeleteFunc     func(ctx context.Context, token string) error
}

func (m *MockRefreshTokenRepo) GetByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	if m.GetByTokenFunc != nil {
		return m.GetByTokenFunc(ctx, token)
	}
	return nil, repository.ErrNotFound
}

func (m *MockRefreshTokenRepo) Create(ctx context.Context, token *domain.RefreshToken) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, token)
	}
	return nil
}

func (m *MockRefreshTokenRepo) Delete(ctx context.Context, token string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, token)
	}
	return nil
}

type MockAuditLogRepo struct {
	CreateFunc func(ctx context.Context, log *domain.AuditLog) error
}

func (m *MockAuditLogRepo) Create(ctx context.Context, log *domain.AuditLog) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, log)
	}
	return nil
}

type MockContactRepo struct {
	CreateFunc                func(ctx context.Context, contact *domain.Contact) error
	GetByIDFunc               func(ctx context.Context, id string) (*domain.Contact, error)
	GetByUserIDFunc           func(ctx context.Context, userID string) ([]domain.Contact, error)
	GetAllByUserIDFunc        func(ctx context.Context, userID string) ([]domain.Contact, error)
	GetByUserIDAndChannelFunc func(ctx context.Context, userID, channel, value string) (*domain.Contact, error)
	UpdateFunc                func(ctx context.Context, contact *domain.Contact) error
	SoftDeleteFunc            func(ctx context.Context, id string) error
	GetByIDsFunc              func(ctx context.Context, ids []string) ([]domain.Contact, error)
}

func (m *MockContactRepo) Create(ctx context.Context, contact *domain.Contact) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, contact)
	}
	return nil
}

func (m *MockContactRepo) GetByID(ctx context.Context, id string) (*domain.Contact, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, repository.ErrNotFound
}

func (m *MockContactRepo) GetByUserID(ctx context.Context, userID string) ([]domain.Contact, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockContactRepo) GetAllByUserID(ctx context.Context, userID string) ([]domain.Contact, error) {
	if m.GetAllByUserIDFunc != nil {
		return m.GetAllByUserIDFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockContactRepo) GetByUserIDAndChannel(ctx context.Context, userID, channel, value string) (*domain.Contact, error) {
	if m.GetByUserIDAndChannelFunc != nil {
		return m.GetByUserIDAndChannelFunc(ctx, userID, channel, value)
	}
	return nil, repository.ErrNotFound
}

func (m *MockContactRepo) Update(ctx context.Context, contact *domain.Contact) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, contact)
	}
	return nil
}

func (m *MockContactRepo) SoftDelete(ctx context.Context, id string) error {
	if m.SoftDeleteFunc != nil {
		return m.SoftDeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockContactRepo) GetByIDs(ctx context.Context, ids []string) ([]domain.Contact, error) {
	if m.GetByIDsFunc != nil {
		return m.GetByIDsFunc(ctx, ids)
	}
	return nil, nil
}

type MockPreferenceRepo struct {
	CreateFunc               func(ctx context.Context, pref *domain.Preference) error
	GetByIDFunc              func(ctx context.Context, id string) (*domain.Preference, error)
	GetByContactIDFunc       func(ctx context.Context, contactID string) ([]domain.Preference, error)
	GetByContactAndScopeFunc func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error)
	UpdateFunc               func(ctx context.Context, pref *domain.Preference) error
	UpsertFunc               func(ctx context.Context, pref *domain.Preference) error
}

func (m *MockPreferenceRepo) Create(ctx context.Context, pref *domain.Preference) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, pref)
	}
	return nil
}

func (m *MockPreferenceRepo) GetByID(ctx context.Context, id string) (*domain.Preference, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, repository.ErrNotFound
}

func (m *MockPreferenceRepo) GetByContactID(ctx context.Context, contactID string) ([]domain.Preference, error) {
	if m.GetByContactIDFunc != nil {
		return m.GetByContactIDFunc(ctx, contactID)
	}
	return nil, nil
}

func (m *MockPreferenceRepo) GetByContactAndScope(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (*domain.Preference, error) {
	if m.GetByContactAndScopeFunc != nil {
		return m.GetByContactAndScopeFunc(ctx, contactID, scopeType, senderID, scopeID)
	}
	return nil, repository.ErrNotFound
}

func (m *MockPreferenceRepo) Update(ctx context.Context, pref *domain.Preference) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, pref)
	}
	return nil
}

func (m *MockPreferenceRepo) Upsert(ctx context.Context, pref *domain.Preference) error {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, pref)
	}
	return nil
}

type MockUnsubscribeTokenRepo struct {
	GetByTokenFunc func(ctx context.Context, token string) (*domain.UnsubscribeToken, error)
}

func (m *MockUnsubscribeTokenRepo) GetByToken(ctx context.Context, token string) (*domain.UnsubscribeToken, error) {
	if m.GetByTokenFunc != nil {
		return m.GetByTokenFunc(ctx, token)
	}
	return nil, repository.ErrNotFound
}

type MockUnsubscribeRuleRepo struct {
	CreateFunc                  func(ctx context.Context, rule *domain.UnsubscribeRule) error
	GetByContactIDFunc          func(ctx context.Context, contactID string) ([]domain.UnsubscribeRule, error)
	ExistsByContactAndScopeFunc func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (bool, error)
	DeleteFunc                  func(ctx context.Context, id string) error
	DeleteByContactAndScopeFunc func(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) error
}

func (m *MockUnsubscribeRuleRepo) Create(ctx context.Context, rule *domain.UnsubscribeRule) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, rule)
	}
	return nil
}

func (m *MockUnsubscribeRuleRepo) GetByContactID(ctx context.Context, contactID string) ([]domain.UnsubscribeRule, error) {
	if m.GetByContactIDFunc != nil {
		return m.GetByContactIDFunc(ctx, contactID)
	}
	return nil, nil
}

func (m *MockUnsubscribeRuleRepo) ExistsByContactAndScope(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (bool, error) {
	if m.ExistsByContactAndScopeFunc != nil {
		return m.ExistsByContactAndScopeFunc(ctx, contactID, scopeType, senderID, scopeID)
	}
	return false, nil
}

func (m *MockUnsubscribeRuleRepo) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *MockUnsubscribeRuleRepo) DeleteByContactAndScope(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) error {
	if m.DeleteByContactAndScopeFunc != nil {
		return m.DeleteByContactAndScopeFunc(ctx, contactID, scopeType, senderID, scopeID)
	}
	return nil
}
