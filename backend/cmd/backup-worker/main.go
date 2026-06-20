package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"team-task-tracker/backend/internal/backups"
	"team-task-tracker/backend/internal/config"
	appmetrics "team-task-tracker/backend/internal/metrics"
)

type backupRunner interface {
	RunOnce(context.Context) (backups.Result, error)
}

func main() {
	once := flag.Bool("once", false, "create one scheduled backup and exit")
	flag.Parse()

	cfg := config.MustLoad()
	logger := newLogger(cfg.AppEnv, os.Stdout)
	runner := backups.NewRunner(cfg.DatabaseURL, cfg.BackupDir, cfg.BackupRetentionCount)
	latest, found, err := backups.Latest(cfg.BackupDir)
	if err != nil {
		logger.Error("scheduled backup inventory failed", "error", err)
		os.Exit(1)
	}
	artifacts, err := backups.List(cfg.BackupDir)
	if err != nil {
		logger.Error("scheduled backup inventory failed", "error", err)
		os.Exit(1)
	}
	lastSuccess := time.Time{}
	if found {
		lastSuccess = latest.CreatedAt
	}
	metricsRecorder := appmetrics.NewBackupMetrics(lastSuccess, len(artifacts))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if *once {
		success, retentionOK := performBackup(ctx, logger, runner, metricsRecorder)
		if !success || !retentionOK {
			os.Exit(1)
		}
		return
	}

	startMetricsServer(ctx, logger, ":"+cfg.BackupMetricsPort, cfg.MetricsAuthToken, cfg.MetricsEnabled, metricsRecorder)
	logger.Info(
		"backup worker started",
		"interval", cfg.BackupInterval.String(),
		"retry_interval", cfg.BackupRetryInterval.String(),
		"retention_count", cfg.BackupRetentionCount,
	)
	if err := run(ctx, logger, runner, metricsRecorder, cfg.BackupInterval, cfg.BackupRetryInterval); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("backup worker failed", "error", err)
		os.Exit(1)
	}
	logger.Info("backup worker stopped")
}

func run(ctx context.Context, logger *slog.Logger, runner backupRunner, metricsRecorder *appmetrics.BackupMetrics, interval time.Duration, retryInterval time.Duration) error {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	if retryInterval <= 0 {
		retryInterval = 5 * time.Minute
	}
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			success, _ := performBackup(ctx, logger, runner, metricsRecorder)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if success {
				timer.Reset(interval)
			} else {
				timer.Reset(retryInterval)
			}
		}
	}
}

func performBackup(ctx context.Context, logger *slog.Logger, runner backupRunner, metricsRecorder *appmetrics.BackupMetrics) (bool, bool) {
	startedAt := time.Now().UTC()
	result, err := runner.RunOnce(ctx)
	if err != nil {
		metricsRecorder.RecordFailure(time.Now().UTC(), time.Since(startedAt))
		logger.Error("scheduled backup failed", "error", err)
		return false, true
	}

	metricsRecorder.RecordSuccess(result.CompletedAt, result.Duration, result.ArtifactCount)
	logger.Info(
		"scheduled backup created",
		"backup_file", filepath.Base(result.Artifact.Path),
		"size_bytes", result.Artifact.Size,
		"duration_ms", result.Duration.Milliseconds(),
		"artifact_count", result.ArtifactCount,
		"removed_count", result.RemovedCount,
	)
	if result.RetentionError != nil {
		metricsRecorder.RecordRetentionFailure()
		logger.Error("scheduled backup retention failed", "error", result.RetentionError)
		return true, false
	}
	return true, true
}

func startMetricsServer(ctx context.Context, logger *slog.Logger, addr string, authToken string, enabled bool, metricsRecorder *appmetrics.BackupMetrics) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	if enabled {
		mux.Handle("GET /metrics", metricsRecorder.Handler(authToken))
	}
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		logger.Info("backup worker metrics server starting", "addr", addr, "metrics_enabled", enabled)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("backup worker metrics server failed", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
}

func newLogger(appEnv string, output io.Writer) *slog.Logger {
	options := &slog.HandlerOptions{Level: slog.LevelInfo}
	if appEnv == config.EnvProduction {
		return slog.New(slog.NewJSONHandler(output, options))
	}
	return slog.New(slog.NewTextHandler(output, options))
}
