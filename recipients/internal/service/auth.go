package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"notification-center/recipients/internal/config"
	"notification-center/recipients/internal/domain"
	"notification-center/recipients/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("неверный логин или пароль")
	ErrUserExists         = errors.New("пользователь уже существует")
	ErrUserNotFound       = errors.New("пользователь не найден")
	ErrContactNotFound    = errors.New("контакт не найден")
	ErrContactExists      = errors.New("контакт уже существует")
	ErrInvalidToken       = errors.New("недействительный токен")
	ErrTokenExpired       = errors.New("срок действия токена истек")
)

type AuthService struct {
	userRepo  *repository.UserRepository
	tokenRepo *repository.RefreshTokenRepository
	auditRepo *repository.AuditLogRepository
	config    *config.Config
}

func NewAuthService(
	userRepo *repository.UserRepository,
	tokenRepo *repository.RefreshTokenRepository,
	auditRepo *repository.AuditLogRepository,
	cfg *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		auditRepo: auditRepo,
		config:    cfg,
	}
}

type RegisterRequest struct {
	Login             string `json:"login" binding:"required,min=3,max=50"`
	Password          string `json:"password" binding:"required,min=6"`
	ClaimContactToken string `json:"claim_contact_token"`
}

type RegisterResponse struct {
	UserID       string `json:"user_id"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *AuthService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	// Check if user already exists
	existingUser, err := s.userRepo.GetByLogin(ctx, req.Login)
	if err == nil && existingUser != nil {
		return nil, ErrUserExists
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &domain.User{
		Login:        req.Login,
		PasswordHash: string(hashedPassword),
		IsAdmin:      false,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshTokenStr := uuid.New().String()
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenStr,
		ExpiresAt: time.Now().AddDate(0, 0, s.config.RefreshExpiry),
	}

	if err := s.tokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	// Create audit log
	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  user.ID,
		Action:  "register",
		Details: fmt.Sprintf("user registered: %s", user.Login),
	})

	return &RegisterResponse{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
	}, nil
}

type LoginRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*RegisterResponse, error) {
	// Find user
	user, err := s.userRepo.GetByLogin(ctx, req.Login)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshTokenStr := uuid.New().String()
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenStr,
		ExpiresAt: time.Now().AddDate(0, 0, s.config.RefreshExpiry),
	}

	if err := s.tokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	// Create audit log
	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  user.ID,
		Action:  "login",
		Details: fmt.Sprintf("user logged in: %s", user.Login),
	})

	return &RegisterResponse{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
	}, nil
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (s *AuthService) Refresh(ctx context.Context, req *RefreshRequest) (*RegisterResponse, error) {
	// Validate refresh token
	token, err := s.tokenRepo.GetByToken(ctx, req.RefreshToken)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidToken
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, token.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Delete old refresh token
	if err := s.tokenRepo.Delete(ctx, req.RefreshToken); err != nil {
		return nil, fmt.Errorf("failed to delete old refresh token: %w", err)
	}

	// Generate new tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshTokenStr := uuid.New().String()
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		Token:     newRefreshTokenStr,
		ExpiresAt: time.Now().AddDate(0, 0, s.config.RefreshExpiry),
	}

	if err := s.tokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return &RegisterResponse{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenStr,
	}, nil
}

func (s *AuthService) generateAccessToken(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"login":    user.Login,
		"is_admin": user.IsAdmin,
		"exp":      time.Now().Add(time.Duration(s.config.JWTExpiry) * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *AuthService) ValidateAccessToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}
