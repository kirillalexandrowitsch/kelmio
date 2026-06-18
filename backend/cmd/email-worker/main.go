package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"team-task-tracker/backend/internal/config"
	"team-task-tracker/backend/internal/database"
	"team-task-tracker/backend/internal/emailoutbox"
	"team-task-tracker/backend/internal/mailer"
	appmetrics "team-task-tracker/backend/internal/metrics"
)

const batchSize = 10
const staleProcessingAfter = 5 * time.Minute

func main() {
	cfg := config.MustLoad()
	logger := newLogger(cfg.AppEnv, os.Stdout)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbCtx, dbCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dbCancel()
	db, err := database.Connect(dbCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	store := emailoutbox.NewStore(db)
	client := mailer.NewClient(cfg)
	var metricsRecorder *appmetrics.AppMetrics
	if cfg.MetricsEnabled {
		metricsRecorder = appmetrics.NewAppMetrics()
		startMetricsServer(ctx, logger, ":"+cfg.EmailWorkerMetricsPort, cfg.MetricsAuthToken, metricsRecorder)
	}
	logger.Info("email worker started", "poll_interval", cfg.EmailWorkerPollInterval.String(), "batch_size", batchSize)

	if err := run(ctx, logger, store, client, cfg.EmailWorkerPollInterval, cfg.EmailMaxAttempts, metricsRecorder); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("email worker failed", "error", err)
		os.Exit(1)
	}
	logger.Info("email worker stopped")
}

func run(ctx context.Context, logger *slog.Logger, store *emailoutbox.Store, client mailer.Client, pollInterval time.Duration, maxAttempts int, metricsRecorder *appmetrics.AppMetrics) error {
	if pollInterval <= 0 {
		pollInterval = 10 * time.Second
	}
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			if metricsRecorder != nil {
				metricsRecorder.RecordEmailWorkerHeartbeat(time.Now().UTC())
			}
			processed, err := processBatch(ctx, logger, store, client, maxAttempts, metricsRecorder)
			if err != nil {
				if metricsRecorder != nil {
					metricsRecorder.RecordEmailWorkerBatchError()
				}
				logger.Error("email worker batch failed", "error", err)
			}
			if processed == 0 {
				timer.Reset(pollInterval)
			} else {
				timer.Reset(0)
			}
		}
	}
}

func processBatch(ctx context.Context, logger *slog.Logger, store *emailoutbox.Store, client mailer.Client, maxAttempts int, metricsRecorder *appmetrics.AppMetrics) (int, error) {
	emails, err := store.ClaimBatch(ctx, batchSize, staleProcessingAfter)
	if err != nil {
		return 0, err
	}
	for _, email := range emails {
		result, err := emailoutbox.ProcessRecord(ctx, store, client, email, maxAttempts)
		if metricsRecorder != nil {
			metricsRecorder.RecordEmailWorkerDeliveryResult(result.Status)
		}
		if err != nil {
			logger.Error("email record processing failed", "email_outbox_id", email.ID, "email_type", email.EmailType, "attempt_count", email.AttemptCount, "target_status", result.Status, "error", err)
			continue
		}
		logger.Info("email record processed", "email_outbox_id", email.ID, "email_type", email.EmailType, "attempt_count", email.AttemptCount, "status", result.Status)
	}
	return len(emails), nil
}

func startMetricsServer(ctx context.Context, logger *slog.Logger, addr string, authToken string, metricsRecorder *appmetrics.AppMetrics) {
	server := &http.Server{
		Addr:              addr,
		Handler:           metricsRecorder.Handler(authToken),
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		logger.Info("email worker metrics server starting", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("email worker metrics server failed", "error", err)
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
