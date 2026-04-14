package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"notification-center/recipients/internal/domain"
	"notification-center/recipients/internal/repository"
)

var (
	ErrInvalidEmail    = errors.New("некорректный формат email")
	ErrInvalidPhone    = errors.New("некорректный формат телефона")
	ErrInvalidTelegram = errors.New("некорректный формат telegram")
)

// ContactService handles contact operations
type ContactService struct {
	contactRepo *repository.ContactRepository
	auditRepo   *repository.AuditLogRepository
}

func NewContactService(contactRepo *repository.ContactRepository, auditRepo *repository.AuditLogRepository) *ContactService {
	return &ContactService{
		contactRepo: contactRepo,
		auditRepo:   auditRepo,
	}
}

type CreateContactRequest struct {
	UserID  string `json:"user_id" binding:"required"`
	Channel string `json:"channel" binding:"required,oneof=email sms telegram"`
	Value   string `json:"value" binding:"required"`
}

type UpdateContactRequest struct {
	Channel string `json:"channel" binding:"required,oneof=email sms telegram"`
	Value   string `json:"value" binding:"required"`
}

func (s *ContactService) Create(ctx context.Context, userID string, req *CreateContactRequest) (*domain.Contact, error) {
	// Validate contact value
	if err := s.validateContact(req.Channel, req.Value); err != nil {
		return nil, err
	}

	// Check for duplicate contact
	existing, err := s.contactRepo.GetByUserIDAndChannel(ctx, userID, req.Channel, req.Value)
	if err == nil && existing != nil {
		return nil, ErrContactExists
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("failed to check duplicate contact: %w", err)
	}

	contact := &domain.Contact{
		UserID:     userID,
		Channel:    req.Channel,
		Value:      req.Value,
		IsVerified: false,
		Enabled:    true,
	}

	if err := s.contactRepo.Create(ctx, contact); err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}

	// Create audit log
	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  userID,
		Action:  "contact_create",
		Details: fmt.Sprintf("contact created: %s %s", contact.Channel, contact.Value),
	})

	return contact, nil
}

func (s *ContactService) GetByUserID(ctx context.Context, userID string) ([]domain.Contact, error) {
	return s.contactRepo.GetByUserID(ctx, userID)
}

func (s *ContactService) GetByID(ctx context.Context, contactID string) (*domain.Contact, error) {
	contact, err := s.contactRepo.GetByID(ctx, contactID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrContactNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}
	return contact, nil
}

func (s *ContactService) Update(ctx context.Context, contactID string, userID string, req *UpdateContactRequest) (*domain.Contact, error) {
	// Get existing contact
	contact, err := s.contactRepo.GetByID(ctx, contactID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrContactNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	// Verify ownership
	if contact.UserID != userID {
		return nil, errors.New("контакт не принадлежит пользователю")
	}

	// Validate new value
	if err := s.validateContact(req.Channel, req.Value); err != nil {
		return nil, err
	}

	// Check for duplicate if changing value/channel
	if req.Value != contact.Value || req.Channel != contact.Channel {
		existing, err := s.contactRepo.GetByUserIDAndChannel(ctx, userID, req.Channel, req.Value)
		if err == nil && existing != nil && existing.ID != contactID {
			return nil, ErrContactExists
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("failed to check duplicate contact: %w", err)
		}
	}

	contact.Channel = req.Channel
	contact.Value = req.Value

	if err := s.contactRepo.Update(ctx, contact); err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}

	// Create audit log
	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  userID,
		Action:  "contact_update",
		Details: fmt.Sprintf("contact updated: %s", contactID),
	})

	return contact, nil
}

func (s *ContactService) Delete(ctx context.Context, contactID string, userID string) error {
	// Get existing contact
	contact, err := s.contactRepo.GetByID(ctx, contactID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrContactNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to get contact: %w", err)
	}

	// Verify ownership
	if contact.UserID != userID {
		return errors.New("контакт не принадлежит пользователю")
	}

	// Soft delete
	if err := s.contactRepo.SoftDelete(ctx, contactID); err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}

	// Create audit log
	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  userID,
		Action:  "contact_delete",
		Details: fmt.Sprintf("contact deleted: %s", contactID),
	})

	return nil
}

func (s *ContactService) validateContact(channel, value string) error {
	switch channel {
	case "email":
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(value) {
			return ErrInvalidEmail
		}
	case "sms":
		phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{6,14}$`)
		if !phoneRegex.MatchString(value) {
			return ErrInvalidPhone
		}
	case "telegram":
		telegramRegex := regexp.MustCompile(`^@?[a-zA-Z0-9_]{5,32}$`)
		if !telegramRegex.MatchString(value) {
			return ErrInvalidTelegram
		}
	}
	return nil
}
