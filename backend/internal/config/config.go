// Package config defines the runtime configuration boundary for the API.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config contains settings that must be shared consistently by the server components.
type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	CORSOrigin      string
	CookieSecure    bool
	AvatarDir       string
}

// Load reads environment variables, applies safe local defaults, and rejects missing secrets and database settings.
func Load() (Config, error) {
	config := Config{
		HTTPAddr:        stringValue("HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  durationValue("ACCESS_TOKEN_TTL", 24*time.Hour),
		RefreshTokenTTL: durationValue("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		CORSOrigin:      stringValue("CORS_ORIGIN", "http://localhost:3005"),
		AvatarDir:       stringValue("AVATAR_DIR", "uploads/avatars"),
	}

	var err error
	if config.CookieSecure, err = boolValue("COOKIE_SECURE", false); err != nil {
		return Config{}, err
	}
	if config.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if len(config.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	return config, nil
}

func stringValue(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationValue(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil && duration > 0 {
			return duration
		}
	}
	return fallback
}

func boolValue(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}
