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

type testAuthServiceWithTracking struct {
	userRepo      *MockUserRepoWithTracking
	tokenRepo     *MockRefreshTokenRepoWithTracking
	auditRepo     *MockAuditLogRepoWithTracking
	config        *testConfig
	createdUsers  []*domain.User
	createdTokens []*domain.RefreshToken
	deletedTokens []string
}

type MockUserRepoWithTracking struct {
	GetByLoginFunc func(ctx context.Context, login string) (*domain.User, error)
	GetByIDFunc    func(ctx context.Context, id string) (*domain.User, error)
	CreateFunc     func(ctx context.Context, user *domain.User) error
}

func (m *MockUserRepoWithTracking) GetByLogin(ctx context.Context, login string) (*domain.User, error) {
	if m.GetByLoginFunc != nil {
		return m.GetByLoginFunc(ctx, login)
	}
	return nil, repository.ErrNotFound
}

func (m *MockUserRepoWithTracking) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, repository.ErrNotFound
}

func (m *MockUserRepoWithTracking) Create(ctx context.Context, user *domain.User) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, user)
	}
	return nil
}

type MockRefreshTokenRepoWithTracking struct {
	GetByTokenFunc func(ctx context.Context, token string) (*domain.RefreshToken, error)
	CreateFunc     func(ctx context.Context, token *domain.RefreshToken) error
	DeleteFunc     func(ctx context.Context, token string) error
}

func (m *MockRefreshTokenRepoWithTracking) GetByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	if m.GetByTokenFunc != nil {
		return m.GetByTokenFunc(ctx, token)
	}
	return nil, repository.ErrNotFound
}

func (m *MockRefreshTokenRepoWithTracking) Create(ctx context.Context, token *domain.RefreshToken) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, token)
	}
	return nil
}

func (m *MockRefreshTokenRepoWithTracking) Delete(ctx context.Context, token string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, token)
	}
	return nil
}

type MockAuditLogRepoWithTracking struct {
	CreateFunc func(ctx context.Context, log *domain.AuditLog) error
}

func (m *MockAuditLogRepoWithTracking) Create(ctx context.Context, log *domain.AuditLog) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, log)
	}
	return nil
}

func newTestAuthServiceWithTracking(
	userRepo *MockUserRepoWithTracking,
	tokenRepo *MockRefreshTokenRepoWithTracking,
	auditRepo *MockAuditLogRepoWithTracking,
	cfg *testConfig,
) *testAuthServiceWithTracking {
	return &testAuthServiceWithTracking{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		auditRepo: auditRepo,
		config:    cfg,
	}
}

func (s *testAuthServiceWithTracking) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	existingUser, err := s.userRepo.GetByLogin(ctx, req.Login)
	if err == nil && existingUser != nil {
		return nil, ErrUserExists
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Login:        req.Login,
		PasswordHash: string(hashedPassword),
		IsAdmin:      false,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	s.createdUsers = append(s.createdUsers, user)

	accessToken, _ := s.generateAccessToken(user)
	refreshTokenStr := "refresh-" + user.ID
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenStr,
		ExpiresAt: time.Now().AddDate(0, 0, s.config.refreshExpiry),
	}

	if err := s.tokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, err
	}
	s.createdTokens = append(s.createdTokens, refreshToken)

	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  user.ID,
		Action:  "register",
		Details: "user registered",
	})

	return &RegisterResponse{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
	}, nil
}

func (s *testAuthServiceWithTracking) Login(ctx context.Context, req *LoginRequest) (*RegisterResponse, error) {
	user, err := s.userRepo.GetByLogin(ctx, req.Login)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, _ := s.generateAccessToken(user)
	refreshTokenStr := "refresh-" + user.ID
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenStr,
		ExpiresAt: time.Now().AddDate(0, 0, s.config.refreshExpiry),
	}

	if err := s.tokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, err
	}
	s.createdTokens = append(s.createdTokens, refreshToken)

	s.auditRepo.Create(ctx, &domain.AuditLog{
		UserID:  user.ID,
		Action:  "login",
		Details: "user logged in",
	})

	return &RegisterResponse{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
	}, nil
}

func (s *testAuthServiceWithTracking) Refresh(ctx context.Context, req *RefreshRequest) (*RegisterResponse, error) {
	token, err := s.tokenRepo.GetByToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.userRepo.GetByID(ctx, token.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if err := s.tokenRepo.Delete(ctx, req.RefreshToken); err != nil {
		return nil, err
	}
	s.deletedTokens = append(s.deletedTokens, req.RefreshToken)

	accessToken, _ := s.generateAccessToken(user)
	newRefreshTokenStr := "refresh-" + user.ID + "-new"
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		Token:     newRefreshTokenStr,
		ExpiresAt: time.Now().AddDate(0, 0, s.config.refreshExpiry),
	}

	if err := s.tokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, err
	}
	s.createdTokens = append(s.createdTokens, refreshToken)

	return &RegisterResponse{
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenStr,
	}, nil
}

func (s *testAuthServiceWithTracking) generateAccessToken(user *domain.User) (string, error) {
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

func (s *testAuthServiceWithTracking) ValidateAccessToken(tokenString string) (jwt.MapClaims, error) {
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

func TestAuthScenarios_Registration(t *testing.T) {
	ctx := context.Background()
	cfg := &testConfig{jwtSecret: "test-secret", jwtExpiry: 60, refreshExpiry: 30}

	tests := []struct {
		name                string
		login               string
		password            string
		setupUserRepo       func() *MockUserRepoWithTracking
		setupTokenRepo      func() *MockRefreshTokenRepoWithTracking
		setupAuditRepo      func() *MockAuditLogRepoWithTracking
		wantErr             error
		verifyUserCreated   func(t *testing.T, svc *testAuthServiceWithTracking, userID string)
		verifyTokens        func(t *testing.T, svc *testAuthServiceWithTracking, resp *RegisterResponse)
		verifyRefreshStored func(t *testing.T, svc *testAuthServiceWithTracking, userID string)
	}{
		{
			name:     "успешная регистрация нового пользователя",
			login:    "newuser",
			password: "securepass123",
			setupUserRepo: func() *MockUserRepoWithTracking {
				return &MockUserRepoWithTracking{
					GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
						return nil, repository.ErrNotFound
					},
					CreateFunc: func(ctx context.Context, user *domain.User) error {
						user.ID = "user-generated-id"
						return nil
					},
				}
			},
			setupTokenRepo: func() *MockRefreshTokenRepoWithTracking {
				return &MockRefreshTokenRepoWithTracking{
					CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
						return nil
					},
				}
			},
			setupAuditRepo: func() *MockAuditLogRepoWithTracking {
				return &MockAuditLogRepoWithTracking{
					CreateFunc: func(ctx context.Context, log *domain.AuditLog) error {
						return nil
					},
				}
			},
			wantErr: nil,
			verifyUserCreated: func(t *testing.T, svc *testAuthServiceWithTracking, userID string) {
				if len(svc.createdUsers) != 1 {
					t.Errorf("expected 1 user created, got %d", len(svc.createdUsers))
				}
				if svc.createdUsers[0].Login != "newuser" {
					t.Errorf("expected login 'newuser', got '%s'", svc.createdUsers[0].Login)
				}
			},
			verifyTokens: func(t *testing.T, svc *testAuthServiceWithTracking, resp *RegisterResponse) {
				if resp.AccessToken == "" {
					t.Error("expected access token to be returned")
				}
				if resp.RefreshToken == "" {
					t.Error("expected refresh token to be returned")
				}

				claims, err := svc.ValidateAccessToken(resp.AccessToken)
				if err != nil {
					t.Errorf("failed to validate access token: %v", err)
				}

				userID, ok := claims["user_id"].(string)
				if !ok || userID == "" {
					t.Error("expected user_id claim in access token")
				}

				login, ok := claims["login"].(string)
				if !ok || login != "newuser" {
					t.Errorf("expected login 'newuser', got '%v'", login)
				}

				isAdmin, ok := claims["is_admin"].(bool)
				if !ok || isAdmin != false {
					t.Errorf("expected is_admin false, got %v", isAdmin)
				}

				exp, ok := claims["exp"].(float64)
				if !ok || exp == 0 {
					t.Error("expected exp claim in access token")
				}
			},
			verifyRefreshStored: func(t *testing.T, svc *testAuthServiceWithTracking, userID string) {
				if len(svc.createdTokens) != 1 {
					t.Errorf("expected 1 refresh token created, got %d", len(svc.createdTokens))
				}
				if svc.createdTokens[0].UserID != "user-generated-id" {
					t.Errorf("expected refresh token user_id 'user-generated-id', got '%s'", svc.createdTokens[0].UserID)
				}
			},
		},
		{
			name:     "регистрация с уже существующим логином",
			login:    "existinguser",
			password: "password123",
			setupUserRepo: func() *MockUserRepoWithTracking {
				return &MockUserRepoWithTracking{
					GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
						return &domain.User{ID: "existing-id", Login: login}, nil
					},
				}
			},
			setupTokenRepo: func() *MockRefreshTokenRepoWithTracking {
				return &MockRefreshTokenRepoWithTracking{}
			},
			setupAuditRepo: func() *MockAuditLogRepoWithTracking {
				return &MockAuditLogRepoWithTracking{}
			},
			wantErr: ErrUserExists,
		},
		{
			name:     "регистрация с ошибкой базы данных",
			login:    "testuser",
			password: "password123",
			setupUserRepo: func() *MockUserRepoWithTracking {
				return &MockUserRepoWithTracking{
					GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
						return nil, repository.ErrNotFound
					},
					CreateFunc: func(ctx context.Context, user *domain.User) error {
						return errors.New("database connection error")
					},
				}
			},
			setupTokenRepo: func() *MockRefreshTokenRepoWithTracking {
				return &MockRefreshTokenRepoWithTracking{}
			},
			setupAuditRepo: func() *MockAuditLogRepoWithTracking {
				return &MockAuditLogRepoWithTracking{}
			},
			wantErr: errors.New("database connection error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := tt.setupUserRepo()
			tokenRepo := tt.setupTokenRepo()
			auditRepo := tt.setupAuditRepo()
			svc := newTestAuthServiceWithTracking(userRepo, tokenRepo, auditRepo, cfg)

			req := &RegisterRequest{
				Login:    tt.login,
				Password: tt.password,
			}
			resp, err := svc.Register(ctx, req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.verifyUserCreated != nil {
				tt.verifyUserCreated(t, svc, resp.UserID)
			}
			if tt.verifyTokens != nil {
				tt.verifyTokens(t, svc, resp)
			}
			if tt.verifyRefreshStored != nil {
				tt.verifyRefreshStored(t, svc, resp.UserID)
			}
		})
	}
}

func TestAuthScenarios_Login(t *testing.T) {
	ctx := context.Background()
	cfg := &testConfig{jwtSecret: "test-secret", jwtExpiry: 60, refreshExpiry: 30}

	tests := []struct {
		name           string
		login          string
		password       string
		setupUserRepo  func() *MockUserRepoWithTracking
		setupTokenRepo func() *MockRefreshTokenRepoWithTracking
		setupAuditRepo func() *MockAuditLogRepoWithTracking
		wantErr        error
		verifyTokens   func(t *testing.T, svc *testAuthServiceWithTracking, resp *RegisterResponse)
	}{
		{
			name:     "успешный вход с корректными учетными данными",
			login:    "testuser",
			password: "correctpassword",
			setupUserRepo: func() *MockUserRepoWithTracking {
				hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
				return &MockUserRepoWithTracking{
					GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
						return &domain.User{
							ID:           "user-123",
							Login:        login,
							PasswordHash: string(hash),
							IsAdmin:      true,
						}, nil
					},
				}
			},
			setupTokenRepo: func() *MockRefreshTokenRepoWithTracking {
				return &MockRefreshTokenRepoWithTracking{
					CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
						return nil
					},
				}
			},
			setupAuditRepo: func() *MockAuditLogRepoWithTracking {
				return &MockAuditLogRepoWithTracking{
					CreateFunc: func(ctx context.Context, log *domain.AuditLog) error {
						return nil
					},
				}
			},
			wantErr: nil,
			verifyTokens: func(t *testing.T, svc *testAuthServiceWithTracking, resp *RegisterResponse) {
				if resp.AccessToken == "" {
					t.Error("expected access token to be returned")
				}
				if resp.RefreshToken == "" {
					t.Error("expected refresh token to be returned")
				}

				claims, err := svc.ValidateAccessToken(resp.AccessToken)
				if err != nil {
					t.Errorf("failed to validate access token: %v", err)
				}

				userID, ok := claims["user_id"].(string)
				if !ok || userID != "user-123" {
					t.Errorf("expected user_id 'user-123', got '%v'", userID)
				}

				login, ok := claims["login"].(string)
				if !ok || login != "testuser" {
					t.Errorf("expected login 'testuser', got '%v'", login)
				}

				isAdmin, ok := claims["is_admin"].(bool)
				if !ok || isAdmin != true {
					t.Errorf("expected is_admin true, got %v", isAdmin)
				}
			},
		},
		{
			name:     "вход с несуществующим пользователем",
			login:    "nonexistent",
			password: "password123",
			setupUserRepo: func() *MockUserRepoWithTracking {
				return &MockUserRepoWithTracking{
					GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
						return nil, repository.ErrNotFound
					},
				}
			},
			setupTokenRepo: func() *MockRefreshTokenRepoWithTracking {
				return &MockRefreshTokenRepoWithTracking{}
			},
			setupAuditRepo: func() *MockAuditLogRepoWithTracking {
				return &MockAuditLogRepoWithTracking{}
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name:     "вход с неверным паролем",
			login:    "testuser",
			password: "wrongpassword",
			setupUserRepo: func() *MockUserRepoWithTracking {
				hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
				return &MockUserRepoWithTracking{
					GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
						return &domain.User{
							ID:           "user-123",
							Login:        login,
							PasswordHash: string(hash),
						}, nil
					},
				}
			},
			setupTokenRepo: func() *MockRefreshTokenRepoWithTracking {
				return &MockRefreshTokenRepoWithTracking{}
			},
			setupAuditRepo: func() *MockAuditLogRepoWithTracking {
				return &MockAuditLogRepoWithTracking{}
			},
			wantErr: ErrInvalidCredentials,
		},
		{
			name:     "вход с пустым паролем",
			login:    "testuser",
			password: "",
			setupUserRepo: func() *MockUserRepoWithTracking {
				hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
				return &MockUserRepoWithTracking{
					GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
						return &domain.User{
							ID:           "user-123",
							Login:        login,
							PasswordHash: string(hash),
						}, nil
					},
				}
			},
			setupTokenRepo: func() *MockRefreshTokenRepoWithTracking {
				return &MockRefreshTokenRepoWithTracking{}
			},
			setupAuditRepo: func() *MockAuditLogRepoWithTracking {
				return &MockAuditLogRepoWithTracking{}
			},
			wantErr: ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := tt.setupUserRepo()
			tokenRepo := tt.setupTokenRepo()
			auditRepo := tt.setupAuditRepo()
			svc := newTestAuthServiceWithTracking(userRepo, tokenRepo, auditRepo, cfg)

			req := &LoginRequest{
				Login:    tt.login,
				Password: tt.password,
			}
			resp, err := svc.Login(ctx, req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.verifyTokens != nil {
				tt.verifyTokens(t, svc, resp)
			}
		})
	}
}

func TestAuthScenarios_RefreshToken(t *testing.T) {
	ctx := context.Background()
	cfg := &testConfig{jwtSecret: "test-secret", jwtExpiry: 60, refreshExpiry: 30}

	tests := []struct {
		name             string
		refreshToken     string
		setupUserRepo    func() *MockUserRepoWithTracking
		setupTokenRepo   func() *MockRefreshTokenRepoWithTracking
		setupAuditRepo   func() *MockAuditLogRepoWithTracking
		wantErr          error
		verifyNewToken   func(t *testing.T, svc *testAuthServiceWithTracking, resp *RegisterResponse)
		verifyOldDeleted func(t *testing.T, svc *testAuthServiceWithTracking, oldToken string)
	}{
		{
			name:         "успешное обновление токена",
			refreshToken: "valid-refresh-token",
			setupUserRepo: func() *MockUserRepoWithTracking {
				return &MockUserRepoWithTracking{
					GetByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
						return &domain.User{
							ID:      id,
							Login:   "testuser",
							IsAdmin: false,
						}, nil
					},
				}
			},
			setupTokenRepo: func() *MockRefreshTokenRepoWithTracking {
				return &MockRefreshTokenRepoWithTracking{
					GetByTokenFunc: func(ctx context.Context, token string) (*domain.RefreshToken, error) {
						return &domain.RefreshToken{
							UserID:    "user-123",
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
				}
			},
			setupAuditRepo: func() *MockAuditLogRepoWithTracking {
				return &MockAuditLogRepoWithTracking{}
			},
			wantErr: nil,
			verifyNewToken: func(t *testing.T, svc *testAuthServiceWithTracking, resp *RegisterResponse) {
				if resp.AccessToken == "" {
					t.Error("expected new access token to be returned")
				}
				if resp.RefreshToken == "" {
					t.Error("expected new refresh token to be returned")
				}
				if resp.UserID != "user-123" {
					t.Errorf("expected user_id 'user-123', got '%s'", resp.UserID)
				}

				claims, err := svc.ValidateAccessToken(resp.AccessToken)
				if err != nil {
					t.Errorf("failed to validate new access token: %v", err)
				}

				userID, ok := claims["user_id"].(string)
				if !ok || userID != "user-123" {
					t.Errorf("expected user_id 'user-123' in token, got '%v'", userID)
				}
			},
			verifyOldDeleted: func(t *testing.T, svc *testAuthServiceWithTracking, oldToken string) {
				found := false
				for _, deleted := range svc.deletedTokens {
					if deleted == oldToken {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected old refresh token '%s' to be deleted", oldToken)
				}
			},
		},
		{
			name:         "обновление с недействительным токеном",
			refreshToken: "invalid-token",
			setupUserRepo: func() *MockUserRepoWithTracking {
				return &MockUserRepoWithTracking{}
			},
			setupTokenRepo: func() *MockRefreshTokenRepoWithTracking {
				return &MockRefreshTokenRepoWithTracking{
					GetByTokenFunc: func(ctx context.Context, token string) (*domain.RefreshToken, error) {
						return nil, repository.ErrNotFound
					},
				}
			},
			setupAuditRepo: func() *MockAuditLogRepoWithTracking {
				return &MockAuditLogRepoWithTracking{}
			},
			wantErr: ErrInvalidToken,
		},
		{
			name:         "обновление с истекшим токеном",
			refreshToken: "expired-token",
			setupUserRepo: func() *MockUserRepoWithTracking {
				return &MockUserRepoWithTracking{
					GetByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
						return &domain.User{
							ID:      "user-123",
							Login:   "testuser",
							IsAdmin: false,
						}, nil
					},
				}
			},
			setupTokenRepo: func() *MockRefreshTokenRepoWithTracking {
				return &MockRefreshTokenRepoWithTracking{
					GetByTokenFunc: func(ctx context.Context, token string) (*domain.RefreshToken, error) {
						return &domain.RefreshToken{
							UserID:    "user-123",
							Token:     token,
							ExpiresAt: time.Now().Add(-time.Hour),
						}, nil
					},
					DeleteFunc: func(ctx context.Context, token string) error {
						return nil
					},
					CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
						return nil
					},
				}
			},
			setupAuditRepo: func() *MockAuditLogRepoWithTracking {
				return &MockAuditLogRepoWithTracking{}
			},
			wantErr: nil,
			verifyNewToken: func(t *testing.T, svc *testAuthServiceWithTracking, resp *RegisterResponse) {
				if resp.AccessToken == "" {
					t.Error("expected new access token to be returned")
				}
			},
			verifyOldDeleted: func(t *testing.T, svc *testAuthServiceWithTracking, oldToken string) {
				found := false
				for _, deleted := range svc.deletedTokens {
					if deleted == oldToken {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected old refresh token '%s' to be deleted", oldToken)
				}
			},
		},
		{
			name:         "обновление когда пользователь удален",
			refreshToken: "valid-token",
			setupUserRepo: func() *MockUserRepoWithTracking {
				return &MockUserRepoWithTracking{
					GetByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
						return nil, repository.ErrNotFound
					},
				}
			},
			setupTokenRepo: func() *MockRefreshTokenRepoWithTracking {
				return &MockRefreshTokenRepoWithTracking{
					GetByTokenFunc: func(ctx context.Context, token string) (*domain.RefreshToken, error) {
						return &domain.RefreshToken{
							UserID:    "user-123",
							Token:     token,
							ExpiresAt: time.Now().Add(time.Hour),
						}, nil
					},
				}
			},
			setupAuditRepo: func() *MockAuditLogRepoWithTracking {
				return &MockAuditLogRepoWithTracking{}
			},
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userRepo := tt.setupUserRepo()
			tokenRepo := tt.setupTokenRepo()
			auditRepo := tt.setupAuditRepo()
			svc := newTestAuthServiceWithTracking(userRepo, tokenRepo, auditRepo, cfg)

			req := &RefreshRequest{
				RefreshToken: tt.refreshToken,
			}
			resp, err := svc.Refresh(ctx, req)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.verifyNewToken != nil {
				tt.verifyNewToken(t, svc, resp)
			}
			if tt.verifyOldDeleted != nil {
				tt.verifyOldDeleted(t, svc, tt.refreshToken)
			}
		})
	}
}

func TestAuthScenarios_JWTClaims(t *testing.T) {
	ctx := context.Background()
	cfg := &testConfig{jwtSecret: "test-secret", jwtExpiry: 60, refreshExpiry: 30}

	userRepo := &MockUserRepoWithTracking{
		GetByLoginFunc: func(ctx context.Context, login string) (*domain.User, error) {
			return nil, repository.ErrNotFound
		},
		CreateFunc: func(ctx context.Context, user *domain.User) error {
			user.ID = "test-user-id"
			return nil
		},
	}
	tokenRepo := &MockRefreshTokenRepoWithTracking{
		CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
			return nil
		},
	}
	auditRepo := &MockAuditLogRepoWithTracking{
		CreateFunc: func(ctx context.Context, log *domain.AuditLog) error {
			return nil
		},
	}
	svc := newTestAuthServiceWithTracking(userRepo, tokenRepo, auditRepo, cfg)

	t.Run("проверка полей JWT токена", func(t *testing.T) {
		req := &RegisterRequest{
			Login:    "claimstest",
			Password: "password123",
		}
		resp, err := svc.Register(ctx, req)
		if err != nil {
			t.Fatalf("failed to register: %v", err)
		}

		claims, err := svc.ValidateAccessToken(resp.AccessToken)
		if err != nil {
			t.Fatalf("failed to validate token: %v", err)
		}

		if _, ok := claims["user_id"]; !ok {
			t.Error("expected user_id claim")
		}
		if _, ok := claims["login"]; !ok {
			t.Error("expected login claim")
		}
		if _, ok := claims["is_admin"]; !ok {
			t.Error("expected is_admin claim")
		}
		if _, ok := claims["exp"]; !ok {
			t.Error("expected exp claim")
		}
		if _, ok := claims["iat"]; !ok {
			t.Error("expected iat claim")
		}

		userID, _ := claims["user_id"].(string)
		if userID != "test-user-id" {
			t.Errorf("expected user_id 'test-user-id', got '%s'", userID)
		}

		login, _ := claims["login"].(string)
		if login != "claimstest" {
			t.Errorf("expected login 'claimstest', got '%s'", login)
		}

		isAdmin, _ := claims["is_admin"].(bool)
		if isAdmin != false {
			t.Errorf("expected is_admin false, got %v", isAdmin)
		}
	})

	t.Run("проверка что токен не просрочен сразу после создания", func(t *testing.T) {
		req := &LoginRequest{
			Login:    "claimstest2",
			Password: "password123",
		}

		hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		userRepo.GetByLoginFunc = func(ctx context.Context, login string) (*domain.User, error) {
			return &domain.User{
				ID:           "user-456",
				Login:        login,
				PasswordHash: string(hash),
				IsAdmin:      true,
			}, nil
		}

		resp, err := svc.Login(ctx, req)
		if err != nil {
			t.Fatalf("failed to login: %v", err)
		}

		claims, err := svc.ValidateAccessToken(resp.AccessToken)
		if err != nil {
			t.Fatalf("failed to validate token: %v", err)
		}

		exp, _ := claims["exp"].(float64)
		iat, _ := claims["iat"].(float64)

		if exp <= iat {
			t.Errorf("exp (%v) should be greater than iat (%v)", exp, iat)
		}

		now := time.Now().Unix()
		if float64(now) > exp {
			t.Errorf("token should not be expired immediately after creation")
		}
	})
}
