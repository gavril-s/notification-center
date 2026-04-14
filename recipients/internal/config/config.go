package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServiceName   string
	Port          string
	DatabaseURL   string
	JWTSecret     string
	JWTExpiry     int // in minutes
	RefreshExpiry int // in days
}

func Load() *Config {
	return &Config{
		ServiceName:   getEnv("SERVICE_NAME", "recipients"),
		Port:          getEnv("PORT", "8081"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		JWTSecret:     getEnv("JWT_SECRET", "recipients-secret-key-change-in-production"),
		JWTExpiry:     getEnvInt("JWT_EXPIRY_MINUTES", 60),
		RefreshExpiry: getEnvInt("REFRESH_EXPIRY_DAYS", 30),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
