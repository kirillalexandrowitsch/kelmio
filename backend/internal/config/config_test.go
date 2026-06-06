package config

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestLoadDevelopmentDefaults(t *testing.T) {
	clearConfigEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AppEnv != EnvDevelopment {
		t.Fatalf("AppEnv = %q, want %q", cfg.AppEnv, EnvDevelopment)
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want 8080", cfg.Port)
	}
	if !strings.HasPrefix(cfg.DatabaseURL, "postgres://") {
		t.Fatalf("DatabaseURL = %q, want postgres URL", cfg.DatabaseURL)
	}
	if cfg.FrontendURL != "http://localhost:5173" {
		t.Fatalf("FrontendURL = %q, want localhost frontend", cfg.FrontendURL)
	}
	if cfg.PublicAppURL != cfg.FrontendURL {
		t.Fatalf("PublicAppURL = %q, want frontend URL", cfg.PublicAppURL)
	}
	if len(cfg.TrustedOrigins) != 1 || cfg.TrustedOrigins[0] != cfg.FrontendURL {
		t.Fatalf("TrustedOrigins = %#v, want frontend URL", cfg.TrustedOrigins)
	}
	if cfg.SessionTTL != 168*time.Hour {
		t.Fatalf("SessionTTL = %s, want 168h", cfg.SessionTTL)
	}
	if cfg.SessionCookieSecure {
		t.Fatal("SessionCookieSecure = true, want false in development")
	}
	if cfg.CSRFSecret != "" {
		t.Fatalf("CSRFSecret = %q, want empty in development", cfg.CSRFSecret)
	}
	if cfg.RateLimitLoginPerMinute != 10 {
		t.Fatalf("RateLimitLoginPerMinute = %d, want 10", cfg.RateLimitLoginPerMinute)
	}
	if cfg.BuildTime != "" {
		t.Fatalf("BuildTime = %q, want empty default", cfg.BuildTime)
	}
}

func TestLoadProductionRequiresPublicAppURL(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("PUBLIC_APP_URL", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "PUBLIC_APP_URL is required in production") {
		t.Fatalf("Load() error = %v, want PUBLIC_APP_URL required", err)
	}
}

func TestLoadProductionRejectsLocalhostPublicAppURL(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("PUBLIC_APP_URL", "https://localhost")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "PUBLIC_APP_URL must not use localhost in production") {
		t.Fatalf("Load() error = %v, want localhost public URL rejection", err)
	}
}

func TestLoadProductionRequiresTrustedOrigins(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("TRUSTED_ORIGINS", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "TRUSTED_ORIGINS is required in production") {
		t.Fatalf("Load() error = %v, want TRUSTED_ORIGINS required", err)
	}
}

func TestLoadProductionRequiresSecureSessionCookie(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("SESSION_COOKIE_SECURE", "false")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "SESSION_COOKIE_SECURE must be true in production") {
		t.Fatalf("Load() error = %v, want secure cookie requirement", err)
	}
}

func TestLoadProductionRequiresLongCSRFSecret(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("CSRF_SECRET", "too-short")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "CSRF_SECRET must be at least 32 characters in production") {
		t.Fatalf("Load() error = %v, want CSRF secret length requirement", err)
	}
}

func TestLoadValidProductionConfig(t *testing.T) {
	setValidProductionEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AppEnv != EnvProduction {
		t.Fatalf("AppEnv = %q, want production", cfg.AppEnv)
	}
	if cfg.PublicAppURL != "https://tasks.example.com" {
		t.Fatalf("PublicAppURL = %q, want production URL", cfg.PublicAppURL)
	}
	if cfg.FrontendURL != cfg.PublicAppURL {
		t.Fatalf("FrontendURL = %q, want public URL fallback", cfg.FrontendURL)
	}
	if len(cfg.TrustedOrigins) != 1 || cfg.TrustedOrigins[0] != "https://tasks.example.com" {
		t.Fatalf("TrustedOrigins = %#v, want production origin", cfg.TrustedOrigins)
	}
	if cfg.SessionTTL != 24*time.Hour {
		t.Fatalf("SessionTTL = %s, want 24h", cfg.SessionTTL)
	}
	if !cfg.SessionCookieSecure {
		t.Fatal("SessionCookieSecure = false, want true")
	}
	if cfg.RateLimitLoginPerMinute != 5 {
		t.Fatalf("RateLimitLoginPerMinute = %d, want 5", cfg.RateLimitLoginPerMinute)
	}
	if cfg.BuildTime != "2026-06-05T20:00:00Z" {
		t.Fatalf("BuildTime = %q, want configured build time", cfg.BuildTime)
	}
}

func TestLoadProductionBuildsEncodedDatabaseURLFromPostgresEnv(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("DATABASE_URL", "")
	t.Setenv("POSTGRES_HOST", "postgres")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_DB", "team_task_tracker")
	t.Setenv("POSTGRES_USER", "team_task_tracker")
	t.Setenv("POSTGRES_PASSWORD", "strong:pass@word# value")
	t.Setenv("POSTGRES_SSLMODE", "disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	parsed, err := url.Parse(cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("parse DatabaseURL: %v", err)
	}
	if parsed.Hostname() != "postgres" || parsed.Path != "/team_task_tracker" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	password, ok := parsed.User.Password()
	if !ok || password != "strong:pass@word# value" {
		t.Fatalf("DatabaseURL password = %q, ok = %t", password, ok)
	}
	if parsed.Fragment != "" {
		t.Fatalf("DatabaseURL fragment = %q, want empty", parsed.Fragment)
	}
}

func TestLoadDatabaseURLOverrideTakesPriority(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgres://override:password@database.example:5432/override_db?sslmode=require")
	t.Setenv("POSTGRES_HOST", "ignored")
	t.Setenv("POSTGRES_DB", "ignored")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseURL != "postgres://override:password@database.example:5432/override_db?sslmode=require" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestLoadProductionRequiresDatabaseHost(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL must include a database host") {
		t.Fatalf("Load() error = %v, want missing database host", err)
	}
}

func TestLoadRejectsDatabaseURLWithoutDatabaseName(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgres://user:password@postgres:5432?sslmode=disable")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL must include a database name") {
		t.Fatalf("Load() error = %v, want missing database name", err)
	}
}

func TestLoadRejectsDatabaseURLFragment(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("DATABASE_URL", "postgres://user:password@postgres:5432/team_task_tracker?sslmode=disable#fragment")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL must not include a fragment") {
		t.Fatalf("Load() error = %v, want fragment rejection", err)
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("BACKEND_PORT", "70000")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "BACKEND_PORT must be a valid port") {
		t.Fatalf("Load() error = %v, want invalid port error", err)
	}
}

func TestLoadRejectsInvalidSessionTTL(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("SESSION_TTL", "not-a-duration")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "SESSION_TTL must be greater than 0") {
		t.Fatalf("Load() error = %v, want invalid SESSION_TTL error", err)
	}
}

func TestLoadRejectsInvalidRateLimit(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("RATE_LIMIT_LOGIN_PER_MINUTE", "0")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "RATE_LIMIT_LOGIN_PER_MINUTE must be greater than 0") {
		t.Fatalf("Load() error = %v, want invalid rate limit error", err)
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"APP_ENV",
		"BACKEND_PORT",
		"DATABASE_URL",
		"FRONTEND_URL",
		"PUBLIC_APP_URL",
		"TRUSTED_ORIGINS",
		"SESSION_TTL",
		"SESSION_COOKIE_SECURE",
		"CSRF_SECRET",
		"RATE_LIMIT_LOGIN_PER_MINUTE",
		"APP_VERSION",
		"BUILD_COMMIT",
		"BUILD_TIME",
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_DB",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_SSLMODE",
	} {
		t.Setenv(key, "")
	}
}

func setValidProductionEnv(t *testing.T) {
	t.Helper()
	clearConfigEnv(t)
	t.Setenv("APP_ENV", EnvProduction)
	t.Setenv("BACKEND_PORT", "8080")
	t.Setenv("DATABASE_URL", "postgres://team_task_tracker:team_task_tracker@postgres:5432/team_task_tracker?sslmode=disable")
	t.Setenv("PUBLIC_APP_URL", "https://tasks.example.com")
	t.Setenv("TRUSTED_ORIGINS", "https://tasks.example.com")
	t.Setenv("SESSION_TTL", "24h")
	t.Setenv("SESSION_COOKIE_SECURE", "true")
	t.Setenv("CSRF_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("RATE_LIMIT_LOGIN_PER_MINUTE", "5")
	t.Setenv("BUILD_TIME", "2026-06-05T20:00:00Z")
}
