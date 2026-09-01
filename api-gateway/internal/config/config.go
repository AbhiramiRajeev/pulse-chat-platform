package config

import (
	"os"
)

type Config struct {
	HTTPPort        string
	ChatServiceAddr string
	JWTSecret       string
	RedisHost       string
	RedisPort       string
}
func Load() Config {
	return Config{
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		ChatServiceAddr: getEnv("CHAT_SERVICE_ADDR", "localhost:50051"),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		RedisHost:       getEnv("REDIS_HOST", "localhost"),
		RedisPort:       getEnv("REDIS_PORT", "6379"),
	}
}

func getEnv(envVar, defaultValue string) string {
	value := os.Getenv(envVar)

	if value == "" {
		return defaultValue
	}

	return value
}	