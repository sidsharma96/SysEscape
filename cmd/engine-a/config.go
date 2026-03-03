package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultRunTokenSecret = "dev-run-token-secret-not-for-production"

type Config struct {
	DatabaseURL      string
	Port             string
	RunTokenSecret   string
	S3Endpoint       string
	S3Bucket         string
	S3AccessKey      string
	S3SecretKey      string
	S3Region         string
	S3ForcePathStyle bool
}

func LoadConfigFromEnv() (Config, error) {
	forcePathStyle, err := parseBoolEnv("S3_FORCE_PATH_STYLE", true)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		DatabaseURL:      envOrDefault("DATABASE_URL", "postgres://ser:ser@localhost:5432/ser?sslmode=disable"),
		Port:             envOrDefault("ENGINE_A_PORT", "8081"),
		RunTokenSecret:   envOrDefault("RUN_TOKEN_SECRET", defaultRunTokenSecret),
		S3Endpoint:       envOrDefault("S3_ENDPOINT", "http://localhost:9000"),
		S3Bucket:         envOrDefault("S3_BUCKET", "ser-bundles"),
		S3AccessKey:      envOrDefault("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:      envOrDefault("S3_SECRET_KEY", "minioadmin"),
		S3Region:         envOrDefault("S3_REGION", "us-east-1"),
		S3ForcePathStyle: forcePathStyle,
	}
	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}

func parseBoolEnv(key string, fallback bool) (bool, error) {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return false, fmt.Errorf("invalid %s value %q: %w", key, val, err)
	}
	return parsed, nil
}
