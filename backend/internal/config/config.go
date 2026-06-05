package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

type Config struct {
	AppEnv                  string
	Port                    string
	DatabaseURL             string
	FrontendURL             string
	PublicAppURL            string
	TrustedOrigins          []string
	SessionTTL              time.Duration
	SessionCookieSecure     bool
	CSRFSecret              string
	RateLimitLoginPerMinute int
	AppVersion              string
	BuildCommit             string
	BuildTime               string
}

func Load() (Config, error) {
	appEnv := env("APP_ENV", EnvDevelopment)
	cfg := Config{
		AppEnv:                  appEnv,
		Port:                    env("BACKEND_PORT", "8080"),
		DatabaseURL:             databaseURL(appEnv),
		FrontendURL:             frontendURL(appEnv),
		PublicAppURL:            publicAppURL(appEnv),
		TrustedOrigins:          trustedOrigins(appEnv),
		SessionTTL:              durationEnv("SESSION_TTL", 168*time.Hour),
		SessionCookieSecure:     boolEnv("SESSION_COOKIE_SECURE", false),
		CSRFSecret:              env("CSRF_SECRET", ""),
		RateLimitLoginPerMinute: intEnv("RATE_LIMIT_LOGIN_PER_MINUTE", 10),
		AppVersion:              env("APP_VERSION", "development"),
		BuildCommit:             env("BUILD_COMMIT", "local"),
		BuildTime:               env("BUILD_TIME", ""),
	}

	if cfg.AppEnv == EnvProduction && cfg.FrontendURL == "" {
		cfg.FrontendURL = cfg.PublicAppURL
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}
	return cfg
}

func (cfg Config) Validate() error {
	var problems []string

	if cfg.AppEnv != EnvDevelopment && cfg.AppEnv != EnvProduction {
		problems = append(problems, "APP_ENV must be development or production")
	}
	if err := validatePort(cfg.Port); err != nil {
		problems = append(problems, err.Error())
	}
	if err := validateDatabaseURL(cfg.DatabaseURL); err != nil {
		problems = append(problems, err.Error())
	}
	if cfg.SessionTTL <= 0 {
		problems = append(problems, "SESSION_TTL must be greater than 0")
	}
	if cfg.RateLimitLoginPerMinute <= 0 {
		problems = append(problems, "RATE_LIMIT_LOGIN_PER_MINUTE must be greater than 0")
	}

	if cfg.AppEnv == EnvProduction {
		problems = append(problems, validateProduction(cfg)...)
	}

	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func databaseURL(appEnv string) string {
	if appEnv == EnvProduction {
		return env("DATABASE_URL", "")
	}
	return env("DATABASE_URL", "postgres://team_task_tracker:team_task_tracker@localhost:15432/team_task_tracker?sslmode=disable")
}

func publicAppURL(appEnv string) string {
	if appEnv == EnvProduction {
		return env("PUBLIC_APP_URL", "")
	}
	return env("PUBLIC_APP_URL", frontendURL(appEnv))
}

func frontendURL(appEnv string) string {
	if appEnv == EnvProduction {
		return env("FRONTEND_URL", "")
	}
	return env("FRONTEND_URL", "http://localhost:5173")
}

func trustedOrigins(appEnv string) []string {
	raw := env("TRUSTED_ORIGINS", "")
	if raw == "" && appEnv == EnvDevelopment {
		raw = frontendURL(appEnv)
	}
	return splitCSV(raw)
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := env(key, "")
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0
	}
	return duration
}

func boolEnv(key string, fallback bool) bool {
	value := env(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return parsed
}

func intEnv(key string, fallback int) int {
	value := env(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func validatePort(port string) error {
	value, err := strconv.Atoi(port)
	if err != nil || value < 1 || value > 65535 {
		return errors.New("BACKEND_PORT must be a valid port from 1 to 65535")
	}
	return nil
}

func validateDatabaseURL(databaseURL string) error {
	if strings.TrimSpace(databaseURL) == "" {
		return errors.New("DATABASE_URL is required")
	}

	parsed, err := url.Parse(databaseURL)
	if err != nil || parsed.Scheme == "" {
		return errors.New("DATABASE_URL must be a valid PostgreSQL URL")
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return errors.New("DATABASE_URL must use postgres or postgresql scheme")
	}
	return nil
}

func validateProduction(cfg Config) []string {
	var problems []string

	if cfg.PublicAppURL == "" {
		problems = append(problems, "PUBLIC_APP_URL is required in production")
	} else if err := validatePublicAppURL(cfg.PublicAppURL); err != nil {
		problems = append(problems, err.Error())
	}

	if len(cfg.TrustedOrigins) == 0 {
		problems = append(problems, "TRUSTED_ORIGINS is required in production")
	}
	for _, origin := range cfg.TrustedOrigins {
		if err := validateOrigin(origin, true); err != nil {
			problems = append(problems, err.Error())
		}
	}

	if !cfg.SessionCookieSecure {
		problems = append(problems, "SESSION_COOKIE_SECURE must be true in production")
	}
	if len(cfg.CSRFSecret) < 32 {
		problems = append(problems, "CSRF_SECRET must be at least 32 characters in production")
	}

	return problems
}

func validatePublicAppURL(value string) error {
	parsed, err := parseHTTPURL(value, "PUBLIC_APP_URL")
	if err != nil {
		return err
	}
	if parsed.Scheme != "https" {
		return errors.New("PUBLIC_APP_URL must use https in production")
	}
	if isLocalhost(parsed.Hostname()) {
		return errors.New("PUBLIC_APP_URL must not use localhost in production")
	}
	return nil
}

func validateOrigin(value string, production bool) error {
	if value == "*" {
		return errors.New("TRUSTED_ORIGINS must not contain wildcard origins")
	}

	parsed, err := parseHTTPURL(value, "TRUSTED_ORIGINS")
	if err != nil {
		return err
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return fmt.Errorf("TRUSTED_ORIGINS origin %q must not include a path", value)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("TRUSTED_ORIGINS origin %q must not include query or fragment", value)
	}
	if production && isLocalhost(parsed.Hostname()) {
		return fmt.Errorf("TRUSTED_ORIGINS origin %q must not use localhost in production", value)
	}
	return nil
}

func parseHTTPURL(value string, name string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%s must be a valid URL", name)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%s must use http or https scheme", name)
	}
	return parsed, nil
}

func isLocalhost(host string) bool {
	normalized := strings.ToLower(strings.TrimSpace(host))
	if normalized == "localhost" || normalized == "127.0.0.1" || normalized == "::1" {
		return true
	}
	ip := net.ParseIP(normalized)
	return ip != nil && ip.IsLoopback()
}
