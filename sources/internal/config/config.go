package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServiceName string
	Port        string
	DatabaseURL string
}

func Load() *Config {
	return &Config{
		ServiceName: getEnv("SERVICE_NAME", "sources"),
		Port:        getEnv("PORT", "8082"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
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
