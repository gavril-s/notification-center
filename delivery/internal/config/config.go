package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServiceName string
	Port        string
	DatabaseURL string
	RabbitMQURL string
}

func Load() *Config {
	return &Config{
		ServiceName: getEnv("SERVICE_NAME", "delivery"),
		Port:        getEnv("PORT", "8084"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RabbitMQURL: getEnv("RABBITMQ_URL", ""),
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
