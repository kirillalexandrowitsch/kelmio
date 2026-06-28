//go:build integration

package emailoutbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kelmio/backend/internal/database"
	"kelmio/backend/internal/migrations"
)

func TestEmailOutboxIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := newEmailOutboxIntegrationDB(t, ctx)
	store := NewStore(db)
	store.now = func() time.Time { return time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC) }

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback tx: %v", err)
	}
	rolledBack, err := Enqueue(ctx, tx, validEnqueueInput("rollback@example.com", "rollback-key"))
	if err != nil {
		t.Fatalf("enqueue in tx: %v", err)
	}
	if rolledBack.ID == "" {
		t.Fatal("rolledBack ID is empty")
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback tx: %v", err)
	}
	assertOutboxCount(t, ctx, db, "rollback-key", 0)

	first, err := store.Enqueue(ctx, validEnqueueInput("member@example.com", "dedup-key"))
	if err != nil {
		t.Fatalf("enqueue first: %v", err)
	}
	second, err := store.Enqueue(ctx, validEnqueueInput("other@example.com", "dedup-key"))
	if err != nil {
		t.Fatalf("enqueue duplicate: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("duplicate enqueue ID = %q, want %q", second.ID, first.ID)
	}
	assertOutboxCount(t, ctx, db, "dedup-key", 1)

	retryEmail, err := store.Enqueue(ctx, validEnqueueInput("retry@example.com", "retry-key"))
	if err != nil {
		t.Fatalf("enqueue retry: %v", err)
	}
	failedEmail, err := store.Enqueue(ctx, validEnqueueInput("failed@example.com", "failed-key"))
	if err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	claimed, err := store.ClaimBatch(ctx, 10, 5*time.Minute)
	if err != nil {
		t.Fatalf("claim batch: %v", err)
	}
	if len(claimed) != 3 {
		t.Fatalf("claimed = %d, want 3", len(claimed))
	}
	for _, email := range claimed {
		if email.Status != StatusProcessing || email.AttemptCount != 1 || email.ProcessingStartedAt == nil {
			t.Fatalf("claimed email = %#v", email)
		}
	}

	if err := store.MarkSent(ctx, first.ID); err != nil {
		t.Fatalf("mark sent: %v", err)
	}
	assertOutboxStatus(t, ctx, db, first.ID, StatusSent)

	retryEmail.AttemptCount = 1
	if err := store.MarkRetry(ctx, retryEmail, errors.New("smtp failed password=secret-token"), 5); err != nil {
		t.Fatalf("mark retry: %v", err)
	}
	assertOutboxStatus(t, ctx, db, retryEmail.ID, StatusPending)
	var lastError string
	if err := db.QueryRow(ctx, `SELECT last_error FROM email_outbox WHERE id = $1`, retryEmail.ID).Scan(&lastError); err != nil {
		t.Fatalf("select retry last_error: %v", err)
	}
	if lastError == "" || lastError == "smtp failed password=secret-token" {
		t.Fatalf("last_error = %q, want sanitized error", lastError)
	}

	failedEmail.AttemptCount = 5
	if err := store.MarkRetry(ctx, failedEmail, errors.New("terminal failure"), 5); err != nil {
		t.Fatalf("mark terminal failure: %v", err)
	}
	assertOutboxStatus(t, ctx, db, failedEmail.ID, StatusFailed)
}

func TestEmailOutboxClaimSkipsLockedRows(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := newEmailOutboxIntegrationDB(t, ctx)
	store := NewStore(db)
	store.now = func() time.Time { return time.Date(2026, 6, 16, 11, 0, 0, 0, time.UTC) }

	first, err := store.Enqueue(ctx, validEnqueueInput("locked@example.com", "locked-key"))
	if err != nil {
		t.Fatalf("enqueue first: %v", err)
	}
	second, err := store.Enqueue(ctx, validEnqueueInput("available@example.com", "available-key"))
	if err != nil {
		t.Fatalf("enqueue second: %v", err)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("begin lock tx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT id FROM email_outbox WHERE id = $1 FOR UPDATE`, first.ID); err != nil {
		t.Fatalf("lock first row: %v", err)
	}

	claimed, err := store.ClaimBatch(ctx, 2, 5*time.Minute)
	if err != nil {
		t.Fatalf("claim batch: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != second.ID {
		t.Fatalf("claimed = %#v, want only unlocked %s", claimed, second.ID)
	}
}

func TestEmailOutboxReclaimsStaleProcessing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := newEmailOutboxIntegrationDB(t, ctx)
	store := NewStore(db)
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	email, err := store.Enqueue(ctx, validEnqueueInput("stale@example.com", "stale-key"))
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	claimed, err := store.ClaimBatch(ctx, 1, 5*time.Minute)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != email.ID || claimed[0].AttemptCount != 1 {
		t.Fatalf("first claimed = %#v", claimed)
	}

	now = now.Add(6 * time.Minute)
	claimed, err = store.ClaimBatch(ctx, 1, 5*time.Minute)
	if err != nil {
		t.Fatalf("stale claim: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != email.ID || claimed[0].AttemptCount != 2 {
		t.Fatalf("stale claimed = %#v", claimed)
	}
}

func TestEmailOutboxDiagnosticsIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := newEmailOutboxIntegrationDB(t, ctx)
	store := NewStore(db)
	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	workspaceID := insertEmailOutboxWorkspace(t, ctx, db)
	otherWorkspaceID := insertEmailOutboxWorkspace(t, ctx, db)

	pendingInput := validEnqueueInput("pending@example.com", "diag-pending")
	pendingInput.WorkspaceID = &workspaceID
	pending, err := store.Enqueue(ctx, pendingInput)
	if err != nil {
		t.Fatalf("enqueue pending: %v", err)
	}

	processingInput := validEnqueueInput("processing@example.com", "diag-processing")
	processingInput.WorkspaceID = &workspaceID
	processing, err := store.Enqueue(ctx, processingInput)
	if err != nil {
		t.Fatalf("enqueue processing: %v", err)
	}
	claimed, err := store.ClaimBatch(ctx, 10, 5*time.Minute)
	if err != nil {
		t.Fatalf("claim batch: %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("claimed = %d, want 2", len(claimed))
	}
	if err := store.MarkRetry(ctx, pending, errors.New("smtp failed"), 5); err != nil {
		t.Fatalf("mark pending retry: %v", err)
	}
	if err := store.MarkFailed(ctx, processing.ID, errors.New("smtp password=secret token=raw-token")); err != nil {
		t.Fatalf("mark failed: %v", err)
	}

	sentInput := validEnqueueInput("sent@example.com", "diag-sent")
	sentInput.WorkspaceID = &workspaceID
	sent, err := store.Enqueue(ctx, sentInput)
	if err != nil {
		t.Fatalf("enqueue sent: %v", err)
	}
	if err := store.MarkSent(ctx, sent.ID); err != nil {
		t.Fatalf("mark sent: %v", err)
	}

	otherInput := validEnqueueInput("other@example.com", "diag-other")
	otherInput.WorkspaceID = &otherWorkspaceID
	if _, err := store.Enqueue(ctx, otherInput); err != nil {
		t.Fatalf("enqueue other workspace: %v", err)
	}

	diagnostics, err := LoadDiagnostics(ctx, db, workspaceID)
	if err != nil {
		t.Fatalf("load diagnostics: %v", err)
	}
	if diagnostics.Total != 3 || diagnostics.Counts.Pending != 1 || diagnostics.Counts.Processing != 0 || diagnostics.Counts.Sent != 1 || diagnostics.Counts.Failed != 1 {
		t.Fatalf("diagnostics counts = %#v", diagnostics)
	}
	if diagnostics.OldestPendingAt == nil {
		t.Fatal("OldestPendingAt is nil")
	}
	if len(diagnostics.RecentTerminalFailures) != 1 {
		t.Fatalf("recent failures = %#v, want 1", diagnostics.RecentTerminalFailures)
	}
	failure := diagnostics.RecentTerminalFailures[0]
	if failure.RecipientEmail != "p***@example.com" {
		t.Fatalf("masked recipient = %q", failure.RecipientEmail)
	}
	if strings.Contains(failure.LastError, "secret") || strings.Contains(failure.LastError, "raw-token") {
		t.Fatalf("last error leaked sensitive value: %q", failure.LastError)
	}
}

func validEnqueueInput(recipient string, deduplicationKey string) EnqueueInput {
	return EnqueueInput{
		EmailType:      TypeSystemTest,
		RecipientEmail: recipient,
		TemplateData: map[string]any{
			"subject":   "Subject",
			"text_body": "Body",
		},
		DeduplicationKey: deduplicationKey,
	}
}

func insertEmailOutboxWorkspace(t *testing.T, ctx context.Context, db *pgxpool.Pool) string {
	t.Helper()
	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ($1)
		RETURNING id::text
	`, fmt.Sprintf("Email Outbox %d", time.Now().UnixNano())).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	return workspaceID
}

func assertOutboxCount(t *testing.T, ctx context.Context, db *pgxpool.Pool, deduplicationKey string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM email_outbox WHERE deduplication_key = $1`, deduplicationKey).Scan(&got); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if got != want {
		t.Fatalf("outbox count for %q = %d, want %d", deduplicationKey, got, want)
	}
}

func assertOutboxStatus(t *testing.T, ctx context.Context, db *pgxpool.Pool, id string, want string) {
	t.Helper()
	var got string
	if err := db.QueryRow(ctx, `SELECT status FROM email_outbox WHERE id = $1`, id).Scan(&got); err != nil {
		t.Fatalf("select status: %v", err)
	}
	if got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
}

func newEmailOutboxIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://kelmio:kelmio@localhost:15432/kelmio?sslmode=disable"
	}
	adminDB, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	t.Cleanup(adminDB.Close)

	schemaName := fmt.Sprintf("email_outbox_%d", time.Now().UnixNano())
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
		t.Fatalf("parse database URL: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schemaName
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect schema: %v", err)
	}
	t.Cleanup(db.Close)
	applied, err := migrations.Up(ctx, db, "../../migrations")
	if err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	hasEmailOutboxMigration := false
	for _, migration := range applied {
		if migration.Version == 16 {
			hasEmailOutboxMigration = true
			break
		}
	}
	if !hasEmailOutboxMigration {
		t.Fatalf("applied migrations = %#v, want the email outbox migration (version 16)", applied)
	}
	return db
}
