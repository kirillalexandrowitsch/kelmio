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
	if cfg.SMTPHost != "localhost" {
		t.Fatalf("SMTPHost = %q, want localhost", cfg.SMTPHost)
	}
	if cfg.SMTPPort != 1025 {
		t.Fatalf("SMTPPort = %d, want 1025", cfg.SMTPPort)
	}
	if cfg.SMTPTLSMode != SMTPTLSModeNone {
		t.Fatalf("SMTPTLSMode = %q, want none", cfg.SMTPTLSMode)
	}
	if cfg.SMTPFromEmail != "no-reply@team-task-tracker.local" {
		t.Fatalf("SMTPFromEmail = %q, want local sender", cfg.SMTPFromEmail)
	}
	if cfg.SMTPFromName != "Team Task Tracker" {
		t.Fatalf("SMTPFromName = %q, want product sender", cfg.SMTPFromName)
	}
	if !cfg.EmailDeliveryEnabled {
		t.Fatal("EmailDeliveryEnabled = false, want true in development")
	}
	if cfg.EmailWorkerPollInterval != 10*time.Second {
		t.Fatalf("EmailWorkerPollInterval = %s, want 10s", cfg.EmailWorkerPollInterval)
	}
	if cfg.EmailMaxAttempts != 5 {
		t.Fatalf("EmailMaxAttempts = %d, want 5", cfg.EmailMaxAttempts)
	}
	if cfg.PasswordResetTTL != 30*time.Minute {
		t.Fatalf("PasswordResetTTL = %s, want 30m", cfg.PasswordResetTTL)
	}
	if !cfg.MetricsEnabled {
		t.Fatal("MetricsEnabled = false, want true in development")
	}
	if cfg.MetricsAuthToken != "" {
		t.Fatalf("MetricsAuthToken = %q, want empty development token", cfg.MetricsAuthToken)
	}
	if cfg.EmailWorkerMetricsPort != "9091" {
		t.Fatalf("EmailWorkerMetricsPort = %q, want 9091", cfg.EmailWorkerMetricsPort)
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
	if cfg.EmailDeliveryEnabled {
		t.Fatal("EmailDeliveryEnabled = true, want false default in production")
	}
	if cfg.MetricsEnabled {
		t.Fatal("MetricsEnabled = true, want false production default without token")
	}
}

func TestLoadValidProductionMetricsConfig(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("METRICS_AUTH_TOKEN", "0123456789abcdef0123456789abcdef")
	t.Setenv("EMAIL_WORKER_METRICS_PORT", "19091")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.MetricsEnabled {
		t.Fatal("MetricsEnabled = false, want true when production token is configured")
	}
	if cfg.MetricsAuthToken != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("MetricsAuthToken = %q, want configured token", cfg.MetricsAuthToken)
	}
	if cfg.EmailWorkerMetricsPort != "19091" {
		t.Fatalf("EmailWorkerMetricsPort = %q, want configured port", cfg.EmailWorkerMetricsPort)
	}
}

func TestLoadProductionMetricsEnabledRequiresLongToken(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("METRICS_ENABLED", "true")
	t.Setenv("METRICS_AUTH_TOKEN", "short")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "METRICS_AUTH_TOKEN must be at least 32 characters in production") {
		t.Fatalf("Load() error = %v, want metrics token length error", err)
	}
}

func TestLoadRejectsInvalidWorkerMetricsPort(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("EMAIL_WORKER_METRICS_PORT", "70000")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "EMAIL_WORKER_METRICS_PORT must be a valid port") {
		t.Fatalf("Load() error = %v, want invalid worker metrics port error", err)
	}
}

func TestLoadValidProductionSMTPConfig(t *testing.T) {
	setValidProductionEnv(t)
	setValidProductionEmailEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.EmailDeliveryEnabled {
		t.Fatal("EmailDeliveryEnabled = false, want true")
	}
	if cfg.SMTPHost != "smtp.example.com" || cfg.SMTPPort != 587 {
		t.Fatalf("SMTP host/port = %s:%d, want smtp.example.com:587", cfg.SMTPHost, cfg.SMTPPort)
	}
	if cfg.SMTPTLSMode != SMTPTLSModeStartTLS {
		t.Fatalf("SMTPTLSMode = %q, want starttls", cfg.SMTPTLSMode)
	}
	if cfg.SMTPFromEmail != "no-reply@example.com" {
		t.Fatalf("SMTPFromEmail = %q, want configured sender", cfg.SMTPFromEmail)
	}
	if cfg.EmailWorkerPollInterval != 5*time.Second {
		t.Fatalf("EmailWorkerPollInterval = %s, want 5s", cfg.EmailWorkerPollInterval)
	}
	if cfg.EmailMaxAttempts != 7 {
		t.Fatalf("EmailMaxAttempts = %d, want 7", cfg.EmailMaxAttempts)
	}
	if cfg.PasswordResetTTL != 45*time.Minute {
		t.Fatalf("PasswordResetTTL = %s, want 45m", cfg.PasswordResetTTL)
	}
}

func TestLoadProductionEmailDeliveryDisabledDoesNotRequireSMTPHost(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("EMAIL_DELIVERY_ENABLED", "false")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_PORT", "")
	t.Setenv("SMTP_FROM_EMAIL", "")
	t.Setenv("SMTP_FROM_NAME", "")

	if _, err := Load(); err != nil {
		t.Fatalf("Load() error = %v, want disabled email delivery to allow empty SMTP host and sender", err)
	}
}

func TestLoadProductionEmailDeliveryRequiresSMTPHost(t *testing.T) {
	setValidProductionEnv(t)
	setValidProductionEmailEnv(t)
	t.Setenv("SMTP_HOST", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "SMTP_HOST is required when EMAIL_DELIVERY_ENABLED=true") {
		t.Fatalf("Load() error = %v, want SMTP_HOST required", err)
	}
}

func TestLoadProductionEmailDeliveryRejectsInvalidSMTPPort(t *testing.T) {
	setValidProductionEnv(t)
	setValidProductionEmailEnv(t)
	t.Setenv("SMTP_PORT", "70000")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "SMTP_PORT must be a valid port") {
		t.Fatalf("Load() error = %v, want invalid SMTP_PORT error", err)
	}
}

func TestLoadRejectsInvalidSMTPTLSMode(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("SMTP_TLS_MODE", "ssl")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "SMTP_TLS_MODE must be none, starttls, or tls") {
		t.Fatalf("Load() error = %v, want invalid SMTP_TLS_MODE error", err)
	}
}

func TestLoadProductionEmailDeliveryRequiresSafeTLSMode(t *testing.T) {
	setValidProductionEnv(t)
	setValidProductionEmailEnv(t)
	t.Setenv("SMTP_TLS_MODE", "none")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "SMTP_TLS_MODE must be starttls or tls in production") {
		t.Fatalf("Load() error = %v, want production safe TLS error", err)
	}
}

func TestLoadRejectsInvalidSMTPFromEmail(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("SMTP_FROM_EMAIL", "not-email")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "SMTP_FROM_EMAIL must be a valid email address") {
		t.Fatalf("Load() error = %v, want invalid sender error", err)
	}
}

func TestLoadRejectsInvalidEmailWorkerSettings(t *testing.T) {
	tests := []struct {
		name string
		key  string
		val  string
		want string
	}{
		{name: "poll interval", key: "EMAIL_WORKER_POLL_INTERVAL", val: "0s", want: "EMAIL_WORKER_POLL_INTERVAL must be greater than 0"},
		{name: "max attempts", key: "EMAIL_MAX_ATTEMPTS", val: "0", want: "EMAIL_MAX_ATTEMPTS must be greater than 0"},
		{name: "password reset ttl", key: "PASSWORD_RESET_TTL", val: "bad-duration", want: "PASSWORD_RESET_TTL must be greater than 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearConfigEnv(t)
			t.Setenv(tt.key, tt.val)

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load() error = %v, want %q", err, tt.want)
			}
		})
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
		"SMTP_HOST",
		"SMTP_PORT",
		"SMTP_USERNAME",
		"SMTP_PASSWORD",
		"SMTP_FROM_EMAIL",
		"SMTP_FROM_NAME",
		"SMTP_TLS_MODE",
		"EMAIL_DELIVERY_ENABLED",
		"EMAIL_WORKER_POLL_INTERVAL",
		"EMAIL_MAX_ATTEMPTS",
		"PASSWORD_RESET_TTL",
		"METRICS_ENABLED",
		"METRICS_AUTH_TOKEN",
		"EMAIL_WORKER_METRICS_PORT",
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

func setValidProductionEmailEnv(t *testing.T) {
	t.Helper()
	t.Setenv("EMAIL_DELIVERY_ENABLED", "true")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USERNAME", "smtp-user")
	t.Setenv("SMTP_PASSWORD", "smtp-password")
	t.Setenv("SMTP_FROM_EMAIL", "no-reply@example.com")
	t.Setenv("SMTP_FROM_NAME", "Team Task Tracker")
	t.Setenv("SMTP_TLS_MODE", SMTPTLSModeStartTLS)
	t.Setenv("EMAIL_WORKER_POLL_INTERVAL", "5s")
	t.Setenv("EMAIL_MAX_ATTEMPTS", "7")
	t.Setenv("PASSWORD_RESET_TTL", "45m")
}
