package config

import (
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"

	SMTPTLSModeNone     = "none"
	SMTPTLSModeStartTLS = "starttls"
	SMTPTLSModeTLS      = "tls"
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
	SMTPHost                string
	SMTPPort                int
	SMTPUsername            string
	SMTPPassword            string
	SMTPFromEmail           string
	SMTPFromName            string
	SMTPTLSMode             string
	EmailDeliveryEnabled    bool
	EmailWorkerPollInterval time.Duration
	EmailMaxAttempts        int
	PasswordResetTTL        time.Duration
	MetricsEnabled          bool
	MetricsAuthToken        string
	EmailWorkerMetricsPort  string
	BackupInterval          time.Duration
	BackupRetryInterval     time.Duration
	BackupRetentionCount    int
	BackupDir               string
	BackupMetricsPort       string
	RestoreDrillEnabled     bool
	RestoreDrillDatabaseURL string
	RestoreDrillTimeout     time.Duration
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
		SMTPHost:                smtpHost(appEnv),
		SMTPPort:                intEnv("SMTP_PORT", 1025),
		SMTPUsername:            env("SMTP_USERNAME", ""),
		SMTPPassword:            env("SMTP_PASSWORD", ""),
		SMTPFromEmail:           env("SMTP_FROM_EMAIL", "no-reply@team-task-tracker.local"),
		SMTPFromName:            env("SMTP_FROM_NAME", "Team Task Tracker"),
		SMTPTLSMode:             strings.ToLower(env("SMTP_TLS_MODE", SMTPTLSModeNone)),
		EmailDeliveryEnabled:    boolEnv("EMAIL_DELIVERY_ENABLED", appEnv == EnvDevelopment),
		EmailWorkerPollInterval: durationEnv("EMAIL_WORKER_POLL_INTERVAL", 10*time.Second),
		EmailMaxAttempts:        intEnv("EMAIL_MAX_ATTEMPTS", 5),
		PasswordResetTTL:        durationEnv("PASSWORD_RESET_TTL", 30*time.Minute),
		MetricsAuthToken:        env("METRICS_AUTH_TOKEN", ""),
		MetricsEnabled:          metricsEnabled(appEnv),
		EmailWorkerMetricsPort:  env("EMAIL_WORKER_METRICS_PORT", "9091"),
		BackupInterval:          durationEnv("BACKUP_INTERVAL", 24*time.Hour),
		BackupRetryInterval:     durationEnv("BACKUP_RETRY_INTERVAL", 5*time.Minute),
		BackupRetentionCount:    intEnv("BACKUP_RETENTION_COUNT", 7),
		BackupDir:               env("BACKUP_DIR", "backups"),
		BackupMetricsPort:       env("BACKUP_METRICS_PORT", "9092"),
		RestoreDrillEnabled:     boolEnv("RESTORE_DRILL_ENABLED", appEnv == EnvDevelopment),
		RestoreDrillDatabaseURL: restoreDrillDatabaseURL(appEnv),
		RestoreDrillTimeout:     durationEnv("RESTORE_DRILL_TIMEOUT", 5*time.Minute),
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
	problems = append(problems, validateEmailConfig(cfg)...)
	problems = append(problems, validateMetricsConfig(cfg)...)
	problems = append(problems, validateBackupConfig(cfg)...)

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
	if value := strings.TrimSpace(os.Getenv("DATABASE_URL")); value != "" {
		return value
	}

	host := "localhost"
	port := "15432"
	database := "team_task_tracker"
	user := "team_task_tracker"
	password := "team_task_tracker"
	if appEnv == EnvProduction {
		host = ""
		port = "5432"
		database = ""
		user = ""
		password = ""
	}

	host = env("POSTGRES_HOST", host)
	port = env("POSTGRES_PORT", port)
	database = env("POSTGRES_DB", database)
	user = env("POSTGRES_USER", user)
	if value, ok := os.LookupEnv("POSTGRES_PASSWORD"); ok {
		password = value
	}
	sslMode := env("POSTGRES_SSLMODE", "disable")

	connectionURL := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     net.JoinHostPort(host, port),
		Path:     "/" + database,
		RawQuery: url.Values{"sslmode": []string{sslMode}}.Encode(),
	}
	return connectionURL.String()
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

func smtpHost(appEnv string) string {
	if appEnv == EnvProduction {
		return env("SMTP_HOST", "")
	}
	return env("SMTP_HOST", "localhost")
}

func restoreDrillDatabaseURL(appEnv string) string {
	if value := strings.TrimSpace(os.Getenv("RESTORE_DRILL_DATABASE_URL")); value != "" {
		return value
	}
	if appEnv == EnvProduction {
		return ""
	}
	return "postgres://restore_drill:restore_drill@localhost:15433/restore_drill?sslmode=disable"
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

func metricsEnabled(appEnv string) bool {
	if value, ok := os.LookupEnv("METRICS_ENABLED"); ok {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			parsed, err := strconv.ParseBool(trimmed)
			return err == nil && parsed
		}
	}
	if appEnv == EnvProduction {
		return strings.TrimSpace(os.Getenv("METRICS_AUTH_TOKEN")) != ""
	}
	return true
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
	if parsed.Hostname() == "" {
		return errors.New("DATABASE_URL must include a database host")
	}
	if strings.Trim(parsed.EscapedPath(), "/") == "" {
		return errors.New("DATABASE_URL must include a database name")
	}
	if parsed.Fragment != "" {
		return errors.New("DATABASE_URL must not include a fragment")
	}
	return nil
}

func validateEmailConfig(cfg Config) []string {
	var problems []string

	if cfg.EmailWorkerPollInterval <= 0 {
		problems = append(problems, "EMAIL_WORKER_POLL_INTERVAL must be greater than 0")
	}
	if cfg.EmailMaxAttempts <= 0 {
		problems = append(problems, "EMAIL_MAX_ATTEMPTS must be greater than 0")
	}
	if cfg.PasswordResetTTL <= 0 {
		problems = append(problems, "PASSWORD_RESET_TTL must be greater than 0")
	}
	switch cfg.SMTPTLSMode {
	case SMTPTLSModeNone, SMTPTLSModeStartTLS, SMTPTLSModeTLS:
	default:
		problems = append(problems, "SMTP_TLS_MODE must be none, starttls, or tls")
	}

	if !cfg.EmailDeliveryEnabled {
		return problems
	}
	if strings.TrimSpace(cfg.SMTPHost) == "" {
		problems = append(problems, "SMTP_HOST is required when EMAIL_DELIVERY_ENABLED=true")
	}
	if cfg.SMTPPort < 1 || cfg.SMTPPort > 65535 {
		problems = append(problems, "SMTP_PORT must be a valid port from 1 to 65535 when EMAIL_DELIVERY_ENABLED=true")
	}
	if strings.TrimSpace(cfg.SMTPFromEmail) == "" {
		problems = append(problems, "SMTP_FROM_EMAIL is required when EMAIL_DELIVERY_ENABLED=true")
	} else if err := validateEmailAddress(cfg.SMTPFromEmail, "SMTP_FROM_EMAIL"); err != nil {
		problems = append(problems, err.Error())
	}
	if strings.TrimSpace(cfg.SMTPFromName) == "" {
		problems = append(problems, "SMTP_FROM_NAME is required when EMAIL_DELIVERY_ENABLED=true")
	}
	if cfg.AppEnv == EnvProduction && cfg.SMTPTLSMode == SMTPTLSModeNone {
		problems = append(problems, "SMTP_TLS_MODE must be starttls or tls in production when EMAIL_DELIVERY_ENABLED=true")
	}
	return problems
}

func validateMetricsConfig(cfg Config) []string {
	var problems []string
	if err := validateNamedPort(cfg.EmailWorkerMetricsPort, "EMAIL_WORKER_METRICS_PORT"); err != nil {
		problems = append(problems, err.Error())
	}
	if cfg.AppEnv == EnvProduction && cfg.MetricsEnabled && len(strings.TrimSpace(cfg.MetricsAuthToken)) < 32 {
		problems = append(problems, "METRICS_AUTH_TOKEN must be at least 32 characters in production when METRICS_ENABLED=true")
	}
	return problems
}

func validateBackupConfig(cfg Config) []string {
	var problems []string
	if cfg.BackupInterval <= 0 {
		problems = append(problems, "BACKUP_INTERVAL must be greater than 0")
	}
	if cfg.BackupRetryInterval <= 0 {
		problems = append(problems, "BACKUP_RETRY_INTERVAL must be greater than 0")
	}
	if cfg.BackupRetentionCount <= 0 {
		problems = append(problems, "BACKUP_RETENTION_COUNT must be greater than 0")
	}
	if strings.TrimSpace(cfg.BackupDir) == "" {
		problems = append(problems, "BACKUP_DIR is required")
	}
	if err := validateNamedPort(cfg.BackupMetricsPort, "BACKUP_METRICS_PORT"); err != nil {
		problems = append(problems, err.Error())
	}
	if cfg.RestoreDrillTimeout <= 0 {
		problems = append(problems, "RESTORE_DRILL_TIMEOUT must be greater than 0")
	}
	if cfg.RestoreDrillEnabled {
		if strings.TrimSpace(cfg.RestoreDrillDatabaseURL) == "" {
			problems = append(problems, "RESTORE_DRILL_DATABASE_URL is required when RESTORE_DRILL_ENABLED=true")
		} else if err := validateDatabaseURL(cfg.RestoreDrillDatabaseURL); err != nil {
			problems = append(problems, strings.Replace(err.Error(), "DATABASE_URL", "RESTORE_DRILL_DATABASE_URL", 1))
		}
	}
	return problems
}

func validateNamedPort(port string, name string) error {
	value, err := strconv.Atoi(port)
	if err != nil || value < 1 || value > 65535 {
		return fmt.Errorf("%s must be a valid port from 1 to 65535", name)
	}
	return nil
}

func validateEmailAddress(value string, name string) error {
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != strings.TrimSpace(value) {
		return fmt.Errorf("%s must be a valid email address", name)
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
