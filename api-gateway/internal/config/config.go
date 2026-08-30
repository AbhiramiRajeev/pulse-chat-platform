package config

import (
	"os"
)

type Config struct {
	HTTPPort        string
	ChatServiceAddr string
}

func Load() Config {
	return Config{
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		ChatServiceAddr: getEnv("CHAT_SERVICE_ADDR", "localhost:50051"),
	}
}

func getEnv(envVar, defaultValue string) string {
	value := os.Getenv(envVar)

	if value == "" {
		return defaultValue
	}

	return value
}