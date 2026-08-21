// Package config loads and validates FlashFlow runtime configuration.
package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

const redacted = "[REDACTED]"

// LookupEnv matches os.LookupEnv and makes tests deterministic.
type LookupEnv func(string) (string, bool)

// Config contains the complete Phase 0 runtime contract.
type Config struct {
	Environment string
	HTTP        HTTPConfig
	Database    DatabaseConfig
	Secrets     Secrets
}

// HTTPConfig controls public HTTP timeouts and graceful shutdown.
type HTTPConfig struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	HandlerTimeout    time.Duration
	ShutdownTimeout   time.Duration
}

// DatabaseConfig identifies the PostgreSQL source of truth.
type DatabaseConfig struct{ URL string }

// Secrets holds values that must never be logged.
type Secrets struct{ JWT string }

// Load reads environment values, applies defaults, and validates them.
func Load(lookup LookupEnv) (Config, error) {
	if lookup == nil {
		return Config{}, fmt.Errorf("lookup environment function is required")
	}
	cfg := Config{
		Environment: valueOrDefault(lookup, "APP_ENV", "development"),
		HTTP: HTTPConfig{
			Address:           valueOrDefault(lookup, "HTTP_ADDR", ":8080"),
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
			HandlerTimeout:    10 * time.Second,
			ShutdownTimeout:   10 * time.Second,
		},
	}
	var err error
	for key, target := range map[string]*time.Duration{
		"HTTP_READ_HEADER_TIMEOUT": &cfg.HTTP.ReadHeaderTimeout,
		"HTTP_READ_TIMEOUT":        &cfg.HTTP.ReadTimeout,
		"HTTP_WRITE_TIMEOUT":       &cfg.HTTP.WriteTimeout,
		"HTTP_IDLE_TIMEOUT":        &cfg.HTTP.IdleTimeout,
		"HTTP_HANDLER_TIMEOUT":     &cfg.HTTP.HandlerTimeout,
		"HTTP_SHUTDOWN_TIMEOUT":    &cfg.HTTP.ShutdownTimeout,
	} {
		*target, err = durationValue(lookup, key, *target)
		if err != nil {
			return Config{}, err
		}
	}
	if cfg.Database.URL, err = required(lookup, "DATABASE_URL"); err != nil {
		return Config{}, err
	}
	if cfg.Secrets.JWT, err = required(lookup, "JWT_SECRET"); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate rejects unsafe or ambiguous startup settings.
func (c Config) Validate() error {
	switch c.Environment {
	case "development", "test", "production":
	default:
		return fmt.Errorf("APP_ENV must be development, test, or production")
	}
	if strings.TrimSpace(c.HTTP.Address) == "" {
		return fmt.Errorf("HTTP_ADDR must not be empty")
	}
	for name, value := range map[string]time.Duration{
		"HTTP_READ_HEADER_TIMEOUT": c.HTTP.ReadHeaderTimeout,
		"HTTP_READ_TIMEOUT":        c.HTTP.ReadTimeout,
		"HTTP_WRITE_TIMEOUT":       c.HTTP.WriteTimeout,
		"HTTP_IDLE_TIMEOUT":        c.HTTP.IdleTimeout,
		"HTTP_HANDLER_TIMEOUT":     c.HTTP.HandlerTimeout,
		"HTTP_SHUTDOWN_TIMEOUT":    c.HTTP.ShutdownTimeout,
	} {
		if value <= 0 {
			return fmt.Errorf("%s must be greater than zero", name)
		}
	}
	databaseURL, err := url.Parse(c.Database.URL)
	if err != nil {
		return fmt.Errorf("DATABASE_URL must be a valid PostgreSQL URL")
	}
	if databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql" {
		return fmt.Errorf("DATABASE_URL scheme must be postgres or postgresql")
	}
	if databaseURL.Host == "" || databaseURL.User == nil {
		return fmt.Errorf("DATABASE_URL must be a valid PostgreSQL URL")
	}
	minimum := 16
	if c.Environment == "production" {
		minimum = 32
	}
	if len(c.Secrets.JWT) < minimum {
		return fmt.Errorf("JWT_SECRET must contain at least %d characters for %s", minimum, c.Environment)
	}
	return nil
}

// SafeSummary returns values suitable for structured startup logs.
func (c Config) SafeSummary() map[string]any {
	return map[string]any{
		"environment":      c.Environment,
		"http_addr":        c.HTTP.Address,
		"handler_timeout":  c.HTTP.HandlerTimeout.String(),
		"shutdown_timeout": c.HTTP.ShutdownTimeout.String(),
		"database_url":     redactURL(c.Database.URL),
		"jwt_secret":       redacted,
	}
}

func valueOrDefault(lookup LookupEnv, key, fallback string) string {
	if value, ok := lookup(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func required(lookup LookupEnv, key string) (string, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return strings.TrimSpace(value), nil
}

func durationValue(lookup LookupEnv, key string, fallback time.Duration) (time.Duration, error) {
	value, ok := lookup(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}
	return duration, nil
}

func redactURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return redacted
	}
	if parsed.User != nil {
		parsed.User = url.UserPassword(redacted, redacted)
	}
	return parsed.String()
}
