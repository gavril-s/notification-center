package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"notification-center/recipients/internal/service"
)

type AuthMiddleware struct {
	authService *service.AuthService
}

func NewAuthMiddleware(authService *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "отсутствует заголовок авторизации"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "некорректный формат токена"})
			c.Abort()
			return
		}

		claims, err := m.authService.ValidateAccessToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "недействительный токен"})
			c.Abort()
			return
		}

		// Set user info in context
		userID, ok := claims["user_id"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "некорректный формат токена"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Set("login", claims["login"])
		c.Set("is_admin", claims["is_admin"])

		c.Next()
	}
}

func GetUserID(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return ""
	}
	return userID.(string)
}

func GetIsAdmin(c *gin.Context) bool {
	isAdmin, exists := c.Get("is_admin")
	if !exists {
		return false
	}
	return isAdmin.(bool)
}
