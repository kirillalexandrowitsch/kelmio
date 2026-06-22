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
	"strings"
	"syscall"
	"time"

	"kelmio/backend/internal/backups"
	"kelmio/backend/internal/config"
	appmetrics "kelmio/backend/internal/metrics"
	"kelmio/backend/internal/restores"
)

type backupRunner interface {
	Create(context.Context) (backups.Result, error)
	ApplyRetention() (int, int, error)
}

type restoreRunner interface {
	Run(context.Context, string) (restores.Result, error)
}

type cycleResult struct {
	success     bool
	pendingPath string
}

func main() {
	once := flag.Bool("once", false, "create and verify one scheduled backup, then exit")
	restoreOnly := flag.Bool("restore-only", false, "verify the latest scheduled backup without creating a new one")
	flag.Parse()
	if *once && *restoreOnly {
		fatalConfig("--once and --restore-only cannot be used together")
	}

	cfg := config.MustLoad()
	logger := newLogger(cfg.AppEnv, os.Stdout)
	backupRunner := backups.NewRunner(cfg.DatabaseURL, cfg.BackupDir, cfg.BackupRetentionCount)
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
	statePath := restores.StatePath(cfg.BackupDir)
	state := restores.State{}
	if cfg.RestoreDrillEnabled {
		state, err = restores.LoadState(statePath)
		if err != nil {
			logger.Error("restore drill state load failed", "error", err)
			os.Exit(1)
		}
	}
	lastBackupSuccess := time.Time{}
	if cfg.RestoreDrillEnabled && state.LastSuccessBackupFile != "" {
		lastBackupSuccess = artifactModifiedAt(filepath.Join(cfg.BackupDir, filepath.Base(state.LastSuccessBackupFile)))
		if lastBackupSuccess.IsZero() && state.LastSuccessAt != nil {
			lastBackupSuccess = *state.LastSuccessAt
		}
	} else if !cfg.RestoreDrillEnabled && found {
		lastBackupSuccess = latest.CreatedAt
	}
	metricsRecorder := appmetrics.NewBackupMetrics(lastBackupSuccess, len(artifacts))

	var drillRunner restoreRunner
	pendingPath := ""
	if cfg.RestoreDrillEnabled {
		migrationDir := strings.TrimSpace(os.Getenv("MIGRATIONS_DIR"))
		if migrationDir == "" {
			migrationDir = "migrations"
		}
		expectedVersion, err := restores.LatestMigrationVersion(migrationDir)
		if err != nil {
			logger.Error("restore drill migration inventory failed", "error", err)
			os.Exit(1)
		}
		initializeRestoreMetrics(metricsRecorder, cfg.BackupDir, state)
		if state.LastResult == "failure" && state.LastBackupFile != "" {
			candidate := filepath.Join(cfg.BackupDir, filepath.Base(state.LastBackupFile))
			if _, err := os.Stat(candidate); err == nil {
				pendingPath = candidate
			}
		}
		drillRunner = &restores.Runner{
			Executor:                 restores.PSQLExecutor{DatabaseURL: cfg.RestoreDrillDatabaseURL},
			StatePath:                statePath,
			ExpectedMigrationVersion: expectedVersion,
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if *restoreOnly {
		if drillRunner == nil {
			fatalConfig("restore drill is disabled")
		}
		if !found {
			fatalConfig("no scheduled backup is available")
		}
		if !performRestore(ctx, logger, drillRunner, metricsRecorder, latest.Path, cfg.RestoreDrillTimeout) {
			os.Exit(1)
		}
		return
	}

	if *once {
		outcome := performCycle(ctx, logger, backupRunner, drillRunner, metricsRecorder, pendingPath, cfg.RestoreDrillTimeout)
		if !outcome.success {
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
		"restore_drill_enabled", cfg.RestoreDrillEnabled,
	)
	if err := run(ctx, logger, backupRunner, drillRunner, metricsRecorder, pendingPath, cfg.BackupInterval, cfg.BackupRetryInterval, cfg.RestoreDrillTimeout); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("backup worker failed", "error", err)
		os.Exit(1)
	}
	logger.Info("backup worker stopped")
}

func run(ctx context.Context, logger *slog.Logger, backupRunner backupRunner, drillRunner restoreRunner, metricsRecorder *appmetrics.BackupMetrics, pendingPath string, interval time.Duration, retryInterval time.Duration, restoreTimeout time.Duration) error {
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
			outcome := performCycle(ctx, logger, backupRunner, drillRunner, metricsRecorder, pendingPath, restoreTimeout)
			pendingPath = outcome.pendingPath
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if outcome.success {
				timer.Reset(interval)
			} else {
				timer.Reset(retryInterval)
			}
		}
	}
}

func performCycle(ctx context.Context, logger *slog.Logger, backupRunner backupRunner, drillRunner restoreRunner, metricsRecorder *appmetrics.BackupMetrics, pendingPath string, restoreTimeout time.Duration) cycleResult {
	cycleStartedAt := time.Now().UTC()
	artifactPath := pendingPath
	artifactSize := int64(0)
	if artifactPath == "" {
		result, err := backupRunner.Create(ctx)
		if err != nil {
			metricsRecorder.RecordFailure(time.Now().UTC(), time.Since(cycleStartedAt))
			logger.Error("scheduled backup failed", "error", err)
			return cycleResult{}
		}
		artifactPath = result.Artifact.Path
		artifactSize = result.Artifact.Size
		logger.Info("scheduled backup artifact created", "backup_file", filepath.Base(artifactPath), "size_bytes", artifactSize)
	}

	if drillRunner != nil && !performRestore(ctx, logger, drillRunner, metricsRecorder, artifactPath, restoreTimeout) {
		metricsRecorder.RecordFailure(time.Now().UTC(), time.Since(cycleStartedAt))
		return cycleResult{pendingPath: artifactPath}
	}

	removed, artifactCount, err := backupRunner.ApplyRetention()
	if err != nil {
		metricsRecorder.RecordRetentionFailure()
		metricsRecorder.RecordFailure(time.Now().UTC(), time.Since(cycleStartedAt))
		logger.Error("scheduled backup retention failed", "error", err)
		return cycleResult{pendingPath: artifactPath}
	}
	completedAt := time.Now().UTC()
	metricsRecorder.RecordSuccess(completedAt, completedAt.Sub(cycleStartedAt), artifactCount)
	logger.Info(
		"scheduled backup verified",
		"backup_file", filepath.Base(artifactPath),
		"duration_ms", completedAt.Sub(cycleStartedAt).Milliseconds(),
		"artifact_count", artifactCount,
		"removed_count", removed,
	)
	return cycleResult{success: true}
}

func performRestore(ctx context.Context, logger *slog.Logger, runner restoreRunner, metricsRecorder *appmetrics.BackupMetrics, artifactPath string, timeout time.Duration) bool {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	restoreCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result, err := runner.Run(restoreCtx, artifactPath)
	backupTime := artifactModifiedAt(artifactPath)
	if err != nil {
		metricsRecorder.RecordRestoreFailure(result.CompletedAt, result.Duration, backupTime)
		logger.Error("restore drill failed", "backup_file", filepath.Base(artifactPath), "error_code", restores.ErrorCode(err))
		return false
	}
	metricsRecorder.RecordRestoreSuccess(result.CompletedAt, result.Duration, backupTime)
	logger.Info(
		"restore drill passed",
		"backup_file", filepath.Base(artifactPath),
		"duration_ms", result.Duration.Milliseconds(),
		"migration_version", result.MigrationVersion,
	)
	return true
}

func initializeRestoreMetrics(metricsRecorder *appmetrics.BackupMetrics, backupDir string, state restores.State) {
	lastSuccess := time.Time{}
	if state.LastSuccessAt != nil {
		lastSuccess = *state.LastSuccessAt
	}
	backupTime := time.Time{}
	if state.LastBackupFile != "" {
		backupTime = artifactModifiedAt(filepath.Join(backupDir, filepath.Base(state.LastBackupFile)))
	}
	metricsRecorder.InitializeRestore(state.LastAttemptAt, lastSuccess, time.Duration(state.LastDurationSeconds*float64(time.Second)), backupTime, state.LastResult)
}

func artifactModifiedAt(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime().UTC()
}

func startMetricsServer(ctx context.Context, logger *slog.Logger, addr string, authToken string, enabled bool, metricsRecorder *appmetrics.BackupMetrics) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	if enabled {
		mux.Handle("GET /metrics", metricsRecorder.Handler(authToken))
	}
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 2 * time.Second}
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

func fatalConfig(message string) {
	_, _ = os.Stderr.WriteString(message + "\n")
	os.Exit(2)
}
