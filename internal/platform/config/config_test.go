package config

import (
	"fmt"
	"strings"
	"testing"
)

func TestLoadAppliesDefaultsAndRedactsSecrets(t *testing.T) {
	t.Parallel()
	values := map[string]string{
		"DATABASE_URL": "postgres://flashflow:database-password@localhost:5432/flashflow?sslmode=disable",
		"JWT_SECRET":   "jwt-secret-that-must-not-leak",
	}
	cfg, err := Load(mapLookup(values))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Address != ":8080" {
		t.Fatalf("HTTP address = %q, want :8080", cfg.HTTP.Address)
	}
	summary := fmt.Sprint(cfg.SafeSummary())
	for _, secret := range []string{"database-password", values["JWT_SECRET"]} {
		if strings.Contains(summary, secret) {
			t.Fatalf("safe summary leaked secret %q: %s", secret, summary)
		}
	}
}

func TestLoadRejectsMissingRequiredValues(t *testing.T) {
	t.Parallel()
	_, err := Load(mapLookup(map[string]string{}))
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("Load() error = %v, want missing DATABASE_URL", err)
	}
}

func TestLoadRejectsInvalidDurationAndURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		values map[string]string
		want   string
	}{
		{"duration", map[string]string{"HTTP_HANDLER_TIMEOUT": "soon"}, "HTTP_HANDLER_TIMEOUT must be a valid duration"},
		{"database URL", map[string]string{"DATABASE_URL": "https://localhost/database"}, "DATABASE_URL scheme must be postgres or postgresql"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values := map[string]string{
				"DATABASE_URL": "postgres://flashflow:password@localhost:5432/flashflow",
				"JWT_SECRET":   "long-enough-development-secret",
			}
			for key, value := range test.values {
				values[key] = value
			}
			_, err := Load(mapLookup(values))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Load() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestProductionRejectsWeakSecret(t *testing.T) {
	t.Parallel()
	_, err := Load(mapLookup(map[string]string{
		"APP_ENV":      "production",
		"DATABASE_URL": "postgres://flashflow:password@localhost:5432/flashflow",
		"JWT_SECRET":   "development-only-secret",
	}))
	if err == nil || !strings.Contains(err.Error(), "at least 32") {
		t.Fatalf("Load() error = %v", err)
	}
}

func mapLookup(values map[string]string) LookupEnv {
	return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
}
