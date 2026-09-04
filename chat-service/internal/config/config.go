package config

import (
	"log"
	"os"
	"time"
)

type Config struct {
	GRPCPort string

	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string

	RedisHost string
	RedisPort string

	//JWTManager expects:expiry time.Duration
	JWTSecret string
	JWTExpiry time.Duration
}

func Load() Config {

	jwtExpiry, err := time.ParseDuration(getEnv("JWT_EXPIRY", "24h"))
	if err != nil {
		log.Fatal("invalid JWT_EXPIRY:", err)
	}
	return Config{
		GRPCPort: getEnv("GRPC_PORT", "50051"),

		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:       getEnv("POSTGRES_DB", "pulse_chat"),

		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),

		JWTSecret: getEnv("JWT_SECRET", ""),
		JWTExpiry: jwtExpiry,
	}
}

func getEnv(envVar, defaultValue string) string {
	value := os.Getenv(envVar)

	if value == "" {
		return defaultValue
	}
	return value
}
