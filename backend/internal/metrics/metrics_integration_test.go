//go:build integration

package metrics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"team-task-tracker/backend/internal/database"
	"team-task-tracker/backend/internal/migrations"
)

func TestMetricsCollectorsIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newMetricsIntegrationDB(t, ctx)
	workspaceID := insertMetricsWorkspace(t, ctx, db)
	insertMetricsOutboxRows(t, ctx, db, workspaceID)

	appMetrics := NewAppMetrics()
	appMetrics.RegisterDatabaseReadyCollector(db)
	appMetrics.RegisterEmailOutboxCollector(db)

	recorder := httptest.NewRecorder()
	appMetrics.Handler("").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("metrics status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	contentType := recorder.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Fatalf("Content-Type = %q, want Prometheus text", contentType)
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		"team_task_tracker_database_ready 1",
		`team_task_tracker_email_outbox_records{status="pending"} 1`,
		`team_task_tracker_email_outbox_records{status="processing"} 1`,
		`team_task_tracker_email_outbox_records{status="sent"} 1`,
		`team_task_tracker_email_outbox_records{status="failed"} 1`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics output missing %q:\n%s", expected, body)
		}
	}
	for _, forbidden := range []string{"metrics-user@example.com", "raw-token", "smtp-password"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("metrics output leaked %q:\n%s", forbidden, body)
		}
	}
}

func newMetricsIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://team_task_tracker:team_task_tracker@localhost:15432/team_task_tracker?sslmode=disable"
	}

	adminDB, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	t.Cleanup(adminDB.Close)

	schemaName := fmt.Sprintf("metrics_%d", time.Now().UnixNano())
	quotedSchemaName := pgx.Identifier{schemaName}.Sanitize()
	if _, err := adminDB.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`); err != nil {
		t.Fatalf("ensure pgcrypto: %v", err)
	}
	if _, err := adminDB.Exec(ctx, `CREATE SCHEMA `+quotedSchemaName); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = adminDB.Exec(cleanupCtx, `DROP SCHEMA IF EXISTS `+quotedSchemaName+` CASCADE`)
	})

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database url: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schemaName
	cfg.MaxConns = 2

	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect integration db: %v", err)
	}
	t.Cleanup(db.Close)

	if _, err := migrations.Up(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}

func insertMetricsWorkspace(t *testing.T, ctx context.Context, db *pgxpool.Pool) string {
	t.Helper()

	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ('Metrics Workspace')
		RETURNING id::text
	`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	return workspaceID
}

func insertMetricsOutboxRows(t *testing.T, ctx context.Context, db *pgxpool.Pool, workspaceID string) {
	t.Helper()

	if _, err := db.Exec(ctx, `
		INSERT INTO email_outbox (
			workspace_id,
			email_type,
			recipient_email,
			template_data,
			status,
			attempt_count,
			next_attempt_at,
			last_error,
			deduplication_key,
			processing_started_at,
			sent_at
		)
		VALUES
			($1, 'password_reset', 'metrics-user@example.com', '{"reset_url":"raw-token"}'::jsonb, 'pending', 0, now(), NULL, 'metrics-pending', NULL, NULL),
			($1, 'team_invite', 'metrics-user@example.com', '{}'::jsonb, 'processing', 1, now(), NULL, 'metrics-processing', now(), NULL),
			($1, 'team_invite', 'metrics-user@example.com', '{}'::jsonb, 'sent', 1, now(), NULL, 'metrics-sent', NULL, now()),
			($1, 'password_reset', 'metrics-user@example.com', '{}'::jsonb, 'failed', 5, now(), 'smtp-password raw-token', 'metrics-failed', NULL, NULL)
	`, workspaceID); err != nil {
		t.Fatalf("insert outbox rows: %v", err)
	}
}
