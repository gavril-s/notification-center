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
	ErrPreferenceNotFound = errors.New("настройка не найдена")
)

// PreferenceService handles preference operations
type PreferenceService struct {
	prefRepo    *repository.PreferenceRepository
	contactRepo *repository.ContactRepository
	auditRepo   *repository.AuditLogRepository
}

func NewPreferenceService(
	prefRepo *repository.PreferenceRepository,
	contactRepo *repository.ContactRepository,
	auditRepo *repository.AuditLogRepository,
) *PreferenceService {
	return &PreferenceService{
		prefRepo:    prefRepo,
		contactRepo: contactRepo,
		auditRepo:   auditRepo,
	}
}

type UpdatePreferenceRequest struct {
	ContactID       string   `json:"contact_id" binding:"required"`
	SenderID        *string  `json:"sender_id"`
	ScopeType       string   `json:"scope_type" binding:"required,oneof=global sender campaign group"`
	ScopeID         *string  `json:"scope_id"`
	Enabled         bool     `json:"enabled"`
	QuietFrom       *string  `json:"quiet_from"`
	QuietTo         *string  `json:"quiet_to"`
	BlockedChannels []string `json:"blocked_channels"`
}

func (s *PreferenceService) GetByContactID(ctx context.Context, contactID string) ([]domain.Preference, error) {
	// Verify contact exists
	_, err := s.contactRepo.GetByID(ctx, contactID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrContactNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	return s.prefRepo.GetByContactID(ctx, contactID)
}

func (s *PreferenceService) GetByUserID(ctx context.Context, userID string) ([]domain.Preference, error) {
	// Get all contacts for user
	contacts, err := s.contactRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}

	var allPrefs []domain.Preference
	for _, contact := range contacts {
		prefs, err := s.prefRepo.GetByContactID(ctx, contact.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get preferences: %w", err)
		}
		allPrefs = append(allPrefs, prefs...)
	}

	return allPrefs, nil
}

func (s *PreferenceService) Update(ctx context.Context, userID string, req *UpdatePreferenceRequest) (*domain.Preference, error) {
	// Verify contact belongs to user
	contact, err := s.contactRepo.GetByID(ctx, req.ContactID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrContactNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	if contact.UserID != userID {
		return nil, errors.New("контакт не принадлежит пользователю")
	}

	// Validate quiet hours if provided
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
		return nil, fmt.Errorf("failed to update preference: %w", err)
	}

	// Create audit log
	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  userID,
		Action:  "preference_update",
		Details: fmt.Sprintf("preference updated for contact: %s", req.ContactID),
	})

	return pref, nil
}

func (s *PreferenceService) validateQuietHours(from, to *string) error {
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

// EvaluatePreference evaluates if a notification should be sent based on preferences
func (s *PreferenceService) EvaluatePreference(ctx context.Context, contactID string, senderID *string, scopeType string, scopeID *string, channel string, evaluateAt time.Time) (bool, *string, error) {
	// Get contact
	contact, err := s.contactRepo.GetByID(ctx, contactID)
	if errors.Is(err, repository.ErrNotFound) {
		return false, nil, ErrContactNotFound
	}
	if err != nil {
		return false, nil, fmt.Errorf("failed to get contact: %w", err)
	}

	// Check if contact is enabled
	if !contact.Enabled {
		reason := "контакт отключен"
		return false, &reason, nil
	}

	// Check global preference (scope_type = global)
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

	// Check sender preference if sender_id provided
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

	// Check scope-specific preference (campaign or group)
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

func (s *PreferenceService) isChannelBlocked(blockedChannels []string, channel string) bool {
	for _, c := range blockedChannels {
		if c == channel {
			return true
		}
	}
	return false
}

func (s *PreferenceService) isInQuietHours(from, to *string, evaluateAt time.Time) bool {
	if from == nil || to == nil {
		return false
	}

	currentTime := evaluateAt.Format("15:04")
	return currentTime >= *from && currentTime <= *to
}
