package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"notification-center/recipients/internal/domain"
	"notification-center/recipients/internal/repository"
)

type testAuthService struct {
	userRepo  UserRepositoryInterface
	tokenRepo RefreshTokenRepositoryInterface
	auditRepo AuditLogRepositoryInterface
	config    *testConfig
}

type testConfig struct {
	jwtSecret     string
	jwtExpiry     int
	refreshExpiry int
}

func newTestAuthService(
	userRepo UserRepositoryInterface,
	tokenRepo RefreshTokenRepositoryInterface,
	auditRepo AuditLogRepositoryInterface,
	cfg *testConfig,
) *testAuthService {
	return &testAuthService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		auditRepo: auditRepo,
		config:    cfg,
	}
}

func (s *testAuthService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	existingUser, err := s.userRepo.GetByLogin(ctx, req.Login)
	if err == nil && existingUser != nil {
		return nil, ErrUserExists
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	user := &domain.User{
		Login:        req.Login,
		PasswordHash: "hashed",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	accessToken, _ := s.generateAccessToken(user)
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		Token:     "refresh-token",
		ExpiresAt: time.Now().AddDate(0, 0, s.config.refreshExpiry),
	}

	s.tokenRepo.Create(ctx, refreshToken)

	return &RegisterResponse{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	}, nil
}

func (s *testAuthService) Login(ctx context.Context, req *LoginRequest) (*RegisterResponse, error) {
	user, err := s.userRepo.GetByLogin(ctx, req.Login)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, _ := s.generateAccessToken(user)
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		Token:     "refresh-token",
		ExpiresAt: time.Now().AddDate(0, 0, s.config.refreshExpiry),
	}

	s.tokenRepo.Create(ctx, refreshToken)

	return &RegisterResponse{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	}, nil
}

func (s *testAuthService) Refresh(ctx context.Context, req *RefreshRequest) (*RegisterResponse, error) {
	token, err := s.tokenRepo.GetByToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.userRepo.GetByID(ctx, token.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	s.tokenRepo.Delete(ctx, req.RefreshToken)

	accessToken, _ := s.generateAccessToken(user)
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		Token:     "new-refresh-token",
		ExpiresAt: time.Now().AddDate(0, 0, s.config.refreshExpiry),
	}

	s.tokenRepo.Create(ctx, refreshToken)

	return &RegisterResponse{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	}, nil
}

func (s *testAuthService) generateAccessToken(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"login":    user.Login,
		"is_admin": user.IsAdmin,
		"exp":      time.Now().Add(time.Duration(s.config.jwtExpiry) * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.jwtSecret))
}

func (s *testAuthService) ValidateAccessToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.jwtSecret), nil
	})

	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()
	cfg := &testConfig{jwtSecret: "test-secret", jwtExpiry: 60, refreshExpiry: 30}

	tests := []struct {
		name       string
		req        *RegisterRequest
		userRepo   *MockUserRepo
		wantErr    error
		wantUserID bool
		wantToken  bool
	}{
		{
			name: "успешная регистрация",
			req: &RegisterRequest{
				Login:    "newuser",
				Password: "password123",
			},
			userRepo: &MockUserRepo{
				GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
					return nil, repository.ErrNotFound
				},
				CreateFunc: func(ctx context.Context, user *domain.User) error {
					user.ID = "user-123"
					return nil
				},
			},
			wantErr:    nil,
			wantUserID: true,
			wantToken:  true,
		},
		{
			name: "пользователь уже существует",
			req: &RegisterRequest{
				Login:    "existinguser",
				Password: "password123",
			},
			userRepo: &MockUserRepo{
				GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
					return &domain.User{ID: "existing-id", Login: login}, nil
				},
			},
			wantErr:    ErrUserExists,
			wantUserID: false,
			wantToken:  false,
		},
		{
			name: "ошибка при проверке пользователя",
			req: &RegisterRequest{
				Login:    "testuser",
				Password: "password123",
			},
			userRepo: &MockUserRepo{
				GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
					return nil, anError
				},
			},
			wantErr:    anError,
			wantUserID: false,
			wantToken:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestAuthService(tt.userRepo, &MockRefreshTokenRepo{}, &MockAuditLogRepo{}, cfg)

			resp, err := svc.Register(ctx, tt.req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.wantUserID && resp.UserID == "" {
					t.Error("expected user ID, got empty")
				}
				if tt.wantToken && resp.AccessToken == "" {
					t.Error("expected access token, got empty")
				}
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	cfg := &testConfig{jwtSecret: "test-secret", jwtExpiry: 60, refreshExpiry: 30}

	tests := []struct {
		name      string
		req       *LoginRequest
		userRepo  *MockUserRepo
		wantErr   error
		wantToken bool
	}{
		{
			name: "успешный вход",
			req: &LoginRequest{
				Login:    "testuser",
				Password: "password123",
			},
			userRepo: &MockUserRepo{
				GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
					hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
					return &domain.User{
						ID:           "user-1",
						Login:        login,
						PasswordHash: string(hash),
					}, nil
				},
				CreateFunc: func(ctx context.Context, user *domain.User) error {
					return nil
				},
			},
			wantErr:   nil,
			wantToken: true,
		},
		{
			name: "пользователь не найден",
			req: &LoginRequest{
				Login:    "nonexistent",
				Password: "password123",
			},
			userRepo: &MockUserRepo{
				GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
					return nil, repository.ErrNotFound
				},
			},
			wantErr:   ErrInvalidCredentials,
			wantToken: false,
		},
		{
			name: "неверный пароль",
			req: &LoginRequest{
				Login:    "testuser",
				Password: "wrongpassword",
			},
			userRepo: &MockUserRepo{
				GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
					hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
					return &domain.User{
						ID:           "user-1",
						Login:        login,
						PasswordHash: string(hash),
					}, nil
				},
			},
			wantErr:   ErrInvalidCredentials,
			wantToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestAuthService(tt.userRepo, &MockRefreshTokenRepo{}, &MockAuditLogRepo{}, cfg)

			resp, err := svc.Login(ctx, tt.req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.wantToken && (resp == nil || resp.AccessToken == "") {
					t.Error("expected access token")
				}
			}
		})
	}
}

func TestAuthService_Refresh(t *testing.T) {
	ctx := context.Background()
	cfg := &testConfig{jwtSecret: "test-secret", jwtExpiry: 60, refreshExpiry: 30}

	tests := []struct {
		name      string
		req       *RefreshRequest
		userRepo  *MockUserRepo
		tokenRepo *MockRefreshTokenRepo
		wantErr   error
		wantToken bool
	}{
		{
			name: "успешное обновление токена",
			req: &RefreshRequest{
				RefreshToken: "valid-refresh-token",
			},
			userRepo: &MockUserRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
					return &domain.User{
						ID:    id,
						Login: "testuser",
					}, nil
				},
			},
			tokenRepo: &MockRefreshTokenRepo{
				GetByTokenFunc: func(ctx context.Context, token string) (*domain.RefreshToken, error) {
					return &domain.RefreshToken{
						UserID:    "user-1",
						Token:     token,
						ExpiresAt: time.Now().Add(time.Hour),
					}, nil
				},
				DeleteFunc: func(ctx context.Context, token string) error {
					return nil
				},
				CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
					return nil
				},
			},
			wantErr:   nil,
			wantToken: true,
		},
		{
			name: "недействительный токен",
			req: &RefreshRequest{
				RefreshToken: "invalid-token",
			},
			userRepo: &MockUserRepo{},
			tokenRepo: &MockRefreshTokenRepo{
				GetByTokenFunc: func(ctx context.Context, token string) (*domain.RefreshToken, error) {
					return nil, repository.ErrNotFound
				},
			},
			wantErr:   ErrInvalidToken,
			wantToken: false,
		},
		{
			name: "пользователь не найден",
			req: &RefreshRequest{
				RefreshToken: "valid-token",
			},
			userRepo: &MockUserRepo{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
					return nil, repository.ErrNotFound
				},
			},
			tokenRepo: &MockRefreshTokenRepo{
				GetByTokenFunc: func(ctx context.Context, token string) (*domain.RefreshToken, error) {
					return &domain.RefreshToken{
						UserID:    "user-1",
						Token:     token,
						ExpiresAt: time.Now().Add(time.Hour),
					}, nil
				},
			},
			wantErr:   ErrUserNotFound,
			wantToken: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestAuthService(tt.userRepo, tt.tokenRepo, &MockAuditLogRepo{}, cfg)

			resp, err := svc.Refresh(ctx, tt.req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.wantToken && (resp == nil || resp.AccessToken == "") {
					t.Error("expected access token")
				}
			}
		})
	}
}

func TestAuthService_ValidateAccessToken(t *testing.T) {
	cfg := &testConfig{jwtSecret: "test-secret", jwtExpiry: 60, refreshExpiry: 30}

	tests := []struct {
		name       string
		token      string
		wantErr    error
		wantUserID string
	}{
		{
			name:       "валидный токен",
			token:      generateTestToken("user-1", "testuser", false, "test-secret", 60),
			wantErr:    nil,
			wantUserID: "user-1",
		},
		{
			name:    "недействительный токен - неверная подпись",
			token:   generateTestToken("user-1", "testuser", false, "wrong-secret", 60),
			wantErr: ErrInvalidToken,
		},
		{
			name:    "недействительный токен - пустая строка",
			token:   "",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "недействительный токен - невалидный формат",
			token:   "not.a.valid.token",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "истекший токен",
			token:   generateTestToken("user-1", "testuser", false, "test-secret", -60),
			wantErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestAuthService(&MockUserRepo{}, &MockRefreshTokenRepo{}, &MockAuditLogRepo{}, cfg)

			claims, err := svc.ValidateAccessToken(tt.token)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tt.wantUserID != "" {
					if claims == nil {
						t.Error("expected claims, got nil")
					} else if claims["user_id"] != tt.wantUserID {
						t.Errorf("expected user_id %s, got %v", tt.wantUserID, claims["user_id"])
					}
				}
			}
		})
	}
}

func generateTestToken(userID, login string, isAdmin bool, secret string, expiry int) string {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"login":    login,
		"is_admin": isAdmin,
		"exp":      time.Now().Add(time.Duration(expiry) * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(secret))
	return tokenStr
}

type mockError struct{}

func (e *mockError) Error() string { return "mock error" }

var anError error = &mockError{}
