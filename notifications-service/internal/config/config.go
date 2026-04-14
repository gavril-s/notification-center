package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServiceName       string
	Port              string
	DatabaseURL       string
	RabbitMQURL       string
	JWTSecret         string
	UnsubscribeSecret string
	RecipientsURL     string
	SourcesURL        string
}

func Load() *Config {
	return &Config{
		ServiceName:       getEnv("SERVICE_NAME", "notifications-service"),
		Port:              getEnv("PORT", "8083"),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
		RabbitMQURL:       getEnv("RABBITMQ_URL", ""),
		JWTSecret:         getEnv("JWT_SECRET", "default-secret-change-in-prod"),
		UnsubscribeSecret: getEnv("UNSUBSCRIBE_SECRET", "unsubscribe-secret-change-in-prod"),
		RecipientsURL:     getEnv("RECIPIENTS_URL", "http://recipients:8081"),
		SourcesURL:        getEnv("SOURCES_URL", "http://sources:8082"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return fallback
}
