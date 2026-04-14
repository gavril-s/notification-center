package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"notification-center/recipients/internal/domain"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	query := `
		INSERT INTO recipients.users (id, login, password_hash, is_admin, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
	`

	_, err := r.db.Exec(ctx, query, user.ID, user.Login, user.PasswordHash, user.IsAdmin)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, login, password_hash, is_admin, created_at, updated_at
		FROM recipients.users
		WHERE id = $1
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	query := `
		SELECT id, login, password_hash, is_admin, created_at, updated_at
		FROM recipients.users
		WHERE login = $1
	`

	var user domain.User
	err := r.db.QueryRow(ctx, query, login).Scan(
		&user.ID, &user.Login, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by login: %w", err)
	}

	return &user, nil
}
