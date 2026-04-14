package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"notification-center/recipients/internal/domain"
)

type UnsubscribeRuleRepository struct {
	db *pgxpool.Pool
}

func NewUnsubscribeRuleRepository(db *pgxpool.Pool) *UnsubscribeRuleRepository {
	return &UnsubscribeRuleRepository{db: db}
}

func (r *UnsubscribeRuleRepository) Create(ctx context.Context, rule *domain.UnsubscribeRule) error {
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	query := `
		INSERT INTO recipients.unsubscribe_rules (id, contact_id, user_id, sender_id, scope_type, scope_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`

	_, err := r.db.Exec(ctx, query,
		rule.ID, rule.ContactID, rule.UserID, rule.SenderID, rule.ScopeType, rule.ScopeID,
	)
	if err != nil {
		return fmt.Errorf("failed to create unsubscribe rule: %w", err)
	}

	return nil
}

func (r *UnsubscribeRuleRepository) GetByContactID(ctx context.Context, contactID string) ([]domain.UnsubscribeRule, error) {
	query := `
		SELECT id, contact_id, user_id, sender_id, scope_type, scope_id, created_at
		FROM recipients.unsubscribe_rules
		WHERE contact_id = $1
	`

	rows, err := r.db.Query(ctx, query, contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unsubscribe rules: %w", err)
	}
	defer rows.Close()

	var rules []domain.UnsubscribeRule
	for rows.Next() {
		var rule domain.UnsubscribeRule
		err := rows.Scan(
			&rule.ID, &rule.ContactID, &rule.UserID, &rule.SenderID, &rule.ScopeType, &rule.ScopeID, &rule.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan unsubscribe rule: %w", err)
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func (r *UnsubscribeRuleRepository) ExistsByContactAndScope(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM recipients.unsubscribe_rules
			WHERE contact_id = $1 AND scope_type = $2 AND COALESCE(sender_id, '') = COALESCE($3, '') AND COALESCE(scope_id, '') = COALESCE($4, '')
		)
	`

	var exists bool
	err := r.db.QueryRow(ctx, query, contactID, scopeType, senderID, scopeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check unsubscribe rule exists: %w", err)
	}

	return exists, nil
}

func (r *UnsubscribeRuleRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM recipients.unsubscribe_rules WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete unsubscribe rule: %w", err)
	}
	return nil
}

func (r *UnsubscribeRuleRepository) DeleteByContactAndScope(ctx context.Context, contactID, scopeType string, senderID, scopeID *string) error {
	query := `
		DELETE FROM recipients.unsubscribe_rules
		WHERE contact_id = $1 AND scope_type = $2 AND COALESCE(sender_id, '') = COALESCE($3, '') AND COALESCE(scope_id, '') = COALESCE($4, '')
	`
	_, err := r.db.Exec(ctx, query, contactID, scopeType, senderID, scopeID)
	if err != nil {
		return fmt.Errorf("failed to delete unsubscribe rule by scope: %w", err)
	}
	return nil
}

// RefreshTokenRepository handles refresh tokens
type RefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepository(db *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}

	query := `
		INSERT INTO recipients.refresh_tokens (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	_, err := r.db.Exec(ctx, query, token.ID, token.UserID, token.Token, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) GetByToken(ctx context.Context, tokenStr string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token, expires_at, created_at
		FROM recipients.refresh_tokens
		WHERE token = $1 AND expires_at > NOW()
	`

	var token domain.RefreshToken
	err := r.db.QueryRow(ctx, query, tokenStr).Scan(
		&token.ID, &token.UserID, &token.Token, &token.ExpiresAt, &token.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return &token, nil
}

func (r *RefreshTokenRepository) Delete(ctx context.Context, tokenStr string) error {
	query := `DELETE FROM recipients.refresh_tokens WHERE token = $1`
	_, err := r.db.Exec(ctx, query, tokenStr)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) DeleteByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM recipients.refresh_tokens WHERE user_id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete refresh tokens by user: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) CleanupExpired(ctx context.Context) error {
	query := `DELETE FROM recipients.refresh_tokens WHERE expires_at < NOW()`
	_, err := r.db.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}
	return nil
}

// GuestClaimTokenRepository handles guest claim tokens
type GuestClaimTokenRepository struct {
	db *pgxpool.Pool
}

func NewGuestClaimTokenRepository(db *pgxpool.Pool) *GuestClaimTokenRepository {
	return &GuestClaimTokenRepository{db: db}
}

func (r *GuestClaimTokenRepository) Create(ctx context.Context, token *domain.GuestClaimToken) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}

	query := `
		INSERT INTO recipients.guest_claim_tokens (id, token, contact_id, user_id, used_at, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`

	_, err := r.db.Exec(ctx, query, token.ID, token.Token, token.ContactID, token.UserID, token.UsedAt, token.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to create guest claim token: %w", err)
	}

	return nil
}

func (r *GuestClaimTokenRepository) GetByToken(ctx context.Context, tokenStr string) (*domain.GuestClaimToken, error) {
	query := `
		SELECT id, token, contact_id, user_id, used_at, expires_at, created_at
		FROM recipients.guest_claim_tokens
		WHERE token = $1 AND used_at IS NULL AND expires_at > NOW()
	`

	var token domain.GuestClaimToken
	err := r.db.QueryRow(ctx, query, tokenStr).Scan(
		&token.ID, &token.Token, &token.ContactID, &token.UserID, &token.UsedAt, &token.ExpiresAt, &token.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get guest claim token: %w", err)
	}

	return &token, nil
}

func (r *GuestClaimTokenRepository) MarkUsed(ctx context.Context, id string) error {
	query := `UPDATE recipients.guest_claim_tokens SET used_at = $2 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to mark guest claim token as used: %w", err)
	}
	return nil
}

// UnsubscribeTokenRepository handles unsubscribe tokens
type UnsubscribeTokenRepository struct {
	db *pgxpool.Pool
}

func NewUnsubscribeTokenRepository(db *pgxpool.Pool) *UnsubscribeTokenRepository {
	return &UnsubscribeTokenRepository{db: db}
}

func (r *UnsubscribeTokenRepository) GetByToken(ctx context.Context, tokenStr string) (*domain.UnsubscribeToken, error) {
	query := `
		SELECT id, token, contact_id, sender_id, scope_type, scope_id, expires_at, created_at
		FROM recipients.unsubscribe_tokens
		WHERE token = $1 AND expires_at > NOW()
	`

	var token domain.UnsubscribeToken
	err := r.db.QueryRow(ctx, query, tokenStr).Scan(
		&token.ID, &token.Token, &token.ContactID, &token.SenderID, &token.ScopeType, &token.ScopeID, &token.ExpiresAt, &token.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get unsubscribe token: %w", err)
	}

	return &token, nil
}

// AuditLogRepository handles audit logs
type AuditLogRepository struct {
	db *pgxpool.Pool
}

func NewAuditLogRepository(db *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}

	query := `
		INSERT INTO recipients.audit_log (id, user_id, action, details, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	_, err := r.db.Exec(ctx, query, log.ID, log.UserID, log.Action, log.Details)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

func (r *AuditLogRepository) GetByUserID(ctx context.Context, userID string, limit int) ([]domain.AuditLog, error) {
	query := `
		SELECT id, user_id, action, details, created_at
		FROM recipients.audit_log
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var log domain.AuditLog
		err := rows.Scan(&log.ID, &log.UserID, &log.Action, &log.Details, &log.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}
