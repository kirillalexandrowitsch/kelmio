package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"kelmio/backend/internal/bootstrap"
	"kelmio/backend/internal/config"
	"kelmio/backend/internal/database"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := config.MustLoad()
	bootstrapCfg, err := bootstrap.LoadConfigFromEnv()
	if err != nil {
		logger.Error("bootstrap configuration invalid", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	result, err := bootstrap.NewService(db).Bootstrap(ctx, bootstrapCfg)
	if err != nil {
		if errors.Is(err, bootstrap.ErrDatabaseNotEmpty) {
			logger.Error("bootstrap refused because production data already exists", "error", err)
		} else {
			logger.Error("bootstrap failed", "error", err)
		}
		os.Exit(1)
	}

	logger.Info("production admin bootstrap completed", "workspace_id", result.WorkspaceID, "admin_user_id", result.AdminUserID)
}
