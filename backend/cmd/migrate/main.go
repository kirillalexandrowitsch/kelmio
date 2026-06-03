package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"team-task-tracker/backend/internal/config"
	"team-task-tracker/backend/internal/database"
	"team-task-tracker/backend/internal/migrations"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	cfg := config.MustLoad()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	applied, err := migrations.Up(ctx, db, "migrations")
	if err != nil {
		logger.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	if len(applied) == 0 {
		logger.Info("database schema is up to date")
		return
	}

	for _, migration := range applied {
		logger.Info("migration applied", "version", migration.Version, "name", migration.Name)
	}
}
