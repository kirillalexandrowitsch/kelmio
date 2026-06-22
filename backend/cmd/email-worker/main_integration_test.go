//go:build integration

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kelmio/backend/internal/database"
	"kelmio/backend/internal/emailoutbox"
	"kelmio/backend/internal/mailer"
	"kelmio/backend/internal/migrations"
)

func TestEmailWorkerRetryRecoveryAndTerminalTemplateFailureIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := newEmailWorkerIntegrationDB(t, ctx)
	store := emailoutbox.NewStore(db)

	retryEmail, err := store.Enqueue(ctx, emailoutbox.EnqueueInput{
		EmailType:      emailoutbox.TypeSystemTest,
		RecipientEmail: "worker-retry@example.com",
		TemplateData: map[string]any{
			"subject":   "Worker retry",
			"text_body": "sensitive worker body",
		},
		DeduplicationKey: "worker-retry",
	})
	if err != nil {
		t.Fatalf("enqueue retry email: %v", err)
	}

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	client := &sequenceMailer{results: []error{errors.New("smtp unavailable"), nil}}

	processed, err := processBatch(ctx, logger, store, client, 5, nil)
	if err != nil {
		t.Fatalf("process failed delivery: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed failed delivery = %d, want 1", processed)
	}
	assertEmailWorkerState(t, ctx, db, retryEmail.ID, emailoutbox.StatusPending, 1)

	if _, err := db.Exec(ctx, `UPDATE email_outbox SET next_attempt_at = NOW() WHERE id = $1`, retryEmail.ID); err != nil {
		t.Fatalf("make retry eligible: %v", err)
	}
	processed, err = processBatch(ctx, logger, store, client, 5, nil)
	if err != nil {
		t.Fatalf("process recovered delivery: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed recovered delivery = %d, want 1", processed)
	}
	assertEmailWorkerState(t, ctx, db, retryEmail.ID, emailoutbox.StatusSent, 2)

	invalidEmail, err := store.Enqueue(ctx, emailoutbox.EnqueueInput{
		EmailType:        "invalid_template",
		RecipientEmail:   "invalid-template@example.com",
		TemplateData:     map[string]any{"token": "raw-secret-token"},
		DeduplicationKey: "worker-invalid-template",
	})
	if err != nil {
		t.Fatalf("enqueue invalid template: %v", err)
	}
	processed, err = processBatch(ctx, logger, store, client, 5, nil)
	if err != nil {
		t.Fatalf("process invalid template: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed invalid template = %d, want 1", processed)
	}
	assertEmailWorkerState(t, ctx, db, invalidEmail.ID, emailoutbox.StatusFailed, 1)

	if client.calls != 2 {
		t.Fatalf("mailer calls = %d, want exactly 2", client.calls)
	}
	logOutput := logs.String()
	for _, sensitive := range []string{"worker-retry@example.com", "invalid-template@example.com", "sensitive worker body", "raw-secret-token"} {
		if strings.Contains(logOutput, sensitive) {
			t.Fatalf("worker logs leaked %q: %s", sensitive, logOutput)
		}
	}
}

type sequenceMailer struct {
	results []error
	calls   int
}

func (m *sequenceMailer) Send(context.Context, mailer.Message) error {
	index := m.calls
	m.calls++
	if index < len(m.results) {
		return m.results[index]
	}
	return nil
}

func assertEmailWorkerState(t *testing.T, ctx context.Context, db *pgxpool.Pool, id, wantStatus string, wantAttempts int) {
	t.Helper()
	var status string
	var attempts int
	if err := db.QueryRow(ctx, `SELECT status, attempt_count FROM email_outbox WHERE id = $1`, id).Scan(&status, &attempts); err != nil {
		t.Fatalf("select email state: %v", err)
	}
	if status != wantStatus || attempts != wantAttempts {
		t.Fatalf("email state = %s/%d, want %s/%d", status, attempts, wantStatus, wantAttempts)
	}
}

func newEmailWorkerIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

	schemaName := fmt.Sprintf("email_worker_%d", time.Now().UnixNano())
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
	if len(applied) == 0 || applied[len(applied)-1].Version != 17 {
		t.Fatalf("applied migrations = %#v, want through version 17", applied)
	}
	return db
}
