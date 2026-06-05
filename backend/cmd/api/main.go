package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/config"
	"team-task-tracker/backend/internal/csrf"
	"team-task-tracker/backend/internal/database"
	"team-task-tracker/backend/internal/invites"
	"team-task-tracker/backend/internal/issues"
	"team-task-tracker/backend/internal/labels"
	"team-task-tracker/backend/internal/notifications"
	"team-task-tracker/backend/internal/projects"
	"team-task-tracker/backend/internal/ratelimit"
	"team-task-tracker/backend/internal/savedfilters"
	"team-task-tracker/backend/internal/sprints"
	"team-task-tracker/backend/internal/team"
)

const maxRequestBodyBytes int64 = 1 << 20
const requestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

func main() {
	cfg := config.MustLoad()
	logger := newLogger(cfg.AppEnv, os.Stdout)

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer dbCancel()

	db, err := database.Connect(dbCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /api/v1/health", healthHandler)
	mux.HandleFunc("GET /readyz", readinessHandler(db))
	mux.HandleFunc("GET /api/v1/ready", readinessHandler(db))
	mux.HandleFunc("GET /api/v1/version", versionHandler(cfg))

	csrfManager, err := csrf.NewManager(cfg.CSRFSecret)
	if err != nil {
		logger.Error("csrf manager setup failed", "error", err)
		os.Exit(1)
	}

	loginLimiter := ratelimit.NewLimiter(cfg.RateLimitLoginPerMinute, time.Minute, time.Now)
	authHandler := auth.NewHandler(db, cfg.SessionTTL, cfg.SessionCookieSecure, csrfManager, loginLimiter)
	authHandler.RegisterRoutes(mux)
	notificationService := notifications.NewService()

	projectsHandler := projects.NewHandler(db, authHandler)
	projectsHandler.RegisterRoutes(mux)

	issuesHandler := issues.NewHandler(db, authHandler, notificationService)
	issuesHandler.RegisterRoutes(mux)

	labelsHandler := labels.NewHandler(db, authHandler)
	labelsHandler.RegisterRoutes(mux)

	teamHandler := team.NewHandler(db, authHandler)
	teamHandler.RegisterRoutes(mux)

	invitesHandler := invites.NewHandler(db, authHandler)
	invitesHandler.RegisterRoutes(mux)

	sprintsHandler := sprints.NewHandler(db, authHandler, notificationService)
	sprintsHandler.RegisterRoutes(mux)

	savedFiltersHandler := savedfilters.NewHandler(db, authHandler)
	savedFiltersHandler.RegisterRoutes(mux)

	notificationsHandler := notifications.NewHandler(db, authHandler)
	notificationsHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      requestID(requestLogger(logger, recoverPanic(logger, securityHeaders(requestBodyLimit(maxRequestBodyBytes, cors(cfg.TrustedOrigins, csrfProtection(csrfManager, mux))))))),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logServerStarting(logger, server.Addr, cfg)
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("backend server failed", "error", err)
			os.Exit(1)
		}
	case sig := <-shutdown:
		logger.Info("shutdown signal received", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}
	}
}

func newLogger(appEnv string, output io.Writer) *slog.Logger {
	options := &slog.HandlerOptions{Level: slog.LevelInfo}
	if appEnv == config.EnvProduction {
		return slog.New(slog.NewJSONHandler(output, options))
	}

	return slog.New(slog.NewTextHandler(output, options))
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func readinessHandler(db interface {
	Ping(context.Context) error
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unavailable","database":"down"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","database":"up"}`))
	}
}

type versionResponse struct {
	Version     string  `json:"version"`
	Commit      string  `json:"commit"`
	Environment string  `json:"environment"`
	BuildTime   *string `json:"build_time"`
}

func versionHandler(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		response := versionResponse{
			Version:     cfg.AppVersion,
			Commit:      cfg.BuildCommit,
			Environment: cfg.AppEnv,
			BuildTime:   nullableString(cfg.BuildTime),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}
}

func nullableString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func logServerStarting(logger *slog.Logger, addr string, cfg config.Config) {
	logger.Info(
		"backend server starting",
		"addr", addr,
		"env", cfg.AppEnv,
		"version", cfg.AppVersion,
		"commit", cfg.BuildCommit,
		"build_time", cfg.BuildTime,
	)
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := newStatusRecorder(w)

		next.ServeHTTP(recorder, r)
		logger.Info(
			"http request",
			"request_id", requestIDFromContext(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"response_bytes", recorder.BytesWritten(),
		)
	})
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(requestIDHeader)
		if !isValidRequestID(requestID) {
			requestID = newRequestID()
		}

		w.Header().Set(requestIDHeader, requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func recoverPanic(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error(
					"http request panic",
					"request_id", requestIDFromContext(r.Context()),
					"method", r.Method,
					"path", r.URL.Path,
					"panic_type", fmt.Sprintf("%T", recovered),
				)

				if !responseHasStarted(w) {
					writeAPIError(w, http.StatusInternalServerError, "internal_server_error", "internal server error")
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func requestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDContextKey{}).(string)
	if !ok {
		return ""
	}
	return requestID
}

func isValidRequestID(value string) bool {
	if len(value) < 8 || len(value) > 128 {
		return false
	}
	for i := 0; i < len(value); i++ {
		char := value[i]
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		if char >= '0' && char <= '9' {
			continue
		}
		if char == '.' || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}

func newRequestID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	}

	return hex.EncodeToString(value[:])
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w}
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}

	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}

	written, err := r.ResponseWriter.Write(data)
	r.bytes += written
	return written, err
}

func (r *statusRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *statusRecorder) BytesWritten() int {
	return r.bytes
}

func (r *statusRecorder) Written() bool {
	return r.status != 0
}

func responseHasStarted(w http.ResponseWriter) bool {
	recorder, ok := w.(interface{ Written() bool })
	return ok && recorder.Written()
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		next.ServeHTTP(w, r)
	})
}

func requestBodyLimit(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if maxBytes > 0 && r.Body != nil {
			if r.ContentLength > maxBytes {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				_, _ = w.Write([]byte(`{"error":{"code":"request_too_large","message":"request body is too large"}}`))
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}

		next.ServeHTTP(w, r)
	})
}

func csrfProtection(manager *csrf.Manager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) || isCSRFExempt(r) {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(auth.SessionCookieName)
		if err != nil || cookie.Value == "" {
			next.ServeHTTP(w, r)
			return
		}

		token := r.Header.Get(csrf.HeaderName)
		if token == "" {
			writeAPIError(w, http.StatusForbidden, "csrf_token_required", "csrf token is required")
			return
		}
		if manager == nil || !manager.Valid(cookie.Value, token) {
			writeAPIError(w, http.StatusForbidden, "invalid_csrf_token", "csrf token is invalid")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

func isCSRFExempt(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	if r.URL.Path == "/api/v1/auth/login" {
		return true
	}

	return strings.HasPrefix(r.URL.Path, "/api/v1/auth/invites/") && strings.HasSuffix(r.URL.Path, "/accept")
}

func cors(trustedOrigins []string, next http.Handler) http.Handler {
	allowedOrigins := make(map[string]struct{}, len(trustedOrigins))
	for _, origin := range trustedOrigins {
		if origin != "" {
			allowedOrigins[origin] = struct{}{}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if _, ok := allowedOrigins[origin]; origin != "" && ok {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, "+csrf.HeaderName+", "+requestIDHeader)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeAPIError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":{"code":"` + code + `","message":"` + message + `"}}`))
}
