package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/config"
	"team-task-tracker/backend/internal/database"
	"team-task-tracker/backend/internal/issues"
	"team-task-tracker/backend/internal/labels"
	"team-task-tracker/backend/internal/projects"
	"team-task-tracker/backend/internal/savedfilters"
	"team-task-tracker/backend/internal/sprints"
	"team-task-tracker/backend/internal/team"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

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

	authHandler := auth.NewHandler(db, 7*24*time.Hour)
	authHandler.RegisterRoutes(mux)

	projectsHandler := projects.NewHandler(db, authHandler)
	projectsHandler.RegisterRoutes(mux)

	issuesHandler := issues.NewHandler(db, authHandler)
	issuesHandler.RegisterRoutes(mux)

	labelsHandler := labels.NewHandler(db, authHandler)
	labelsHandler.RegisterRoutes(mux)

	teamHandler := team.NewHandler(db, authHandler)
	teamHandler.RegisterRoutes(mux)

	sprintsHandler := sprints.NewHandler(db, authHandler)
	sprintsHandler.RegisterRoutes(mux)

	savedFiltersHandler := savedfilters.NewHandler(db, authHandler)
	savedFiltersHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      requestLogger(logger, cors(cfg.FrontendURL, mux)),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("backend server starting", "addr", server.Addr, "env", cfg.AppEnv)
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

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info(
			"http request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func cors(frontendURL string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origin == frontendURL {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			w.Header().Set("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
