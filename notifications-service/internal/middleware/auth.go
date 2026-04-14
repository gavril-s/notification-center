package middleware

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}
		c.Set("trace_id", traceID)
		c.Next()
	}
}

// Auth extracts user ID from JWT token in Authorization header
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]
		tokenParts := strings.Split(tokenString, ".")
		if len(tokenParts) != 3 {
			c.Next()
			return
		}

		// Decode JWT payload (second part) without verification
		payload := tokenParts[1]
		// Add padding if needed for base64url decoding
		padding := len(payload) % 4
		if padding != 0 {
			payload += strings.Repeat("=", 4-padding)
		}

		decoded, err := base64.URLEncoding.DecodeString(payload)
		if err != nil {
			c.Next()
			return
		}

		var claims map[string]any
		if err := json.Unmarshal(decoded, &claims); err != nil {
			c.Next()
			return
		}

		if userID, ok := claims["user_id"].(string); ok {
			c.Set("user_id", userID)
			c.Header("X-User-ID", userID)
		}

		c.Next()
	}
}
