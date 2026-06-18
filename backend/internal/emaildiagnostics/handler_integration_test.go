//go:build integration

package emaildiagnostics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/database"
	"team-task-tracker/backend/internal/emailoutbox"
	"team-task-tracker/backend/internal/migrations"
)

func TestEmailDiagnosticsAPIIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newEmailDiagnosticsIntegrationDB(t, ctx)
	seed := seedEmailDiagnosticsWorkspace(t, ctx, db)
	insertEmailDiagnosticsRows(t, ctx, db, seed.workspaceID)

	authHandler := auth.NewHandler(db, time.Hour, false, nil, nil)
	diagnosticsHandler := NewHandler(db, authHandler)

	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	diagnosticsHandler.RegisterRoutes(mux)

	memberCookie := loginEmailDiagnosticsUser(t, mux, "diag_member", "member12345")
	memberRequest := performEmailDiagnosticsRequest(mux, http.MethodGet, "/api/v1/email/diagnostics", "", memberCookie)
	if memberRequest.Code != http.StatusForbidden {
		t.Fatalf("member status = %d, want %d: %s", memberRequest.Code, http.StatusForbidden, memberRequest.Body.String())
	}

	adminCookie := loginEmailDiagnosticsUser(t, mux, "diag_admin", "admin12345")
	adminRequest := performEmailDiagnosticsRequest(mux, http.MethodGet, "/api/v1/email/diagnostics", "", adminCookie)
	if adminRequest.Code != http.StatusOK {
		t.Fatalf("admin status = %d, want %d: %s", adminRequest.Code, http.StatusOK, adminRequest.Body.String())
	}

	body := adminRequest.Body.String()
	if strings.Contains(body, "admin@example.com") || strings.Contains(body, "raw-token-in-template") || strings.Contains(body, "secret-password") {
		t.Fatalf("diagnostics response leaked sensitive data: %s", body)
	}

	var diagnostics emailoutbox.Diagnostics
	if err := json.Unmarshal(adminRequest.Body.Bytes(), &diagnostics); err != nil {
		t.Fatalf("decode diagnostics: %v", err)
	}
	if diagnostics.Total != 3 || diagnostics.Counts.Pending != 1 || diagnostics.Counts.Sent != 1 || diagnostics.Counts.Failed != 1 {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if len(diagnostics.RecentTerminalFailures) != 1 {
		t.Fatalf("recent failures = %#v, want 1", diagnostics.RecentTerminalFailures)
	}
	if diagnostics.RecentTerminalFailures[0].RecipientEmail != "a***@example.com" {
		t.Fatalf("masked recipient = %q", diagnostics.RecentTerminalFailures[0].RecipientEmail)
	}
}

type emailDiagnosticsSeed struct {
	workspaceID string
}

func seedEmailDiagnosticsWorkspace(t *testing.T, ctx context.Context, db *pgxpool.Pool) emailDiagnosticsSeed {
	t.Helper()

	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ('Email Diagnostics Workspace')
		RETURNING id::text
	`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}

	insertEmailDiagnosticsUser(t, ctx, db, workspaceID, "diag_admin", "admin", "admin12345")
	insertEmailDiagnosticsUser(t, ctx, db, workspaceID, "diag_member", "member", "member12345")
	return emailDiagnosticsSeed{workspaceID: workspaceID}
}

func insertEmailDiagnosticsUser(t *testing.T, ctx context.Context, db *pgxpool.Pool, workspaceID string, username string, role string, password string) {
	t.Helper()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	var userID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name, is_active)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id::text
	`, username+"@example.com", username, string(passwordHash), username).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, $3)
	`, workspaceID, userID, role); err != nil {
		t.Fatalf("insert workspace member: %v", err)
	}
}

func insertEmailDiagnosticsRows(t *testing.T, ctx context.Context, db *pgxpool.Pool, workspaceID string) {
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
			sent_at,
			created_at,
			updated_at
		)
		VALUES
			($1, 'password_reset', 'admin@example.com', '{"reset_url":"raw-token-in-template"}'::jsonb, 'failed', 5, now(), 'smtp password=secret-password token=raw-token', 'diag-failed', NULL, now() - interval '3 minutes', now() - interval '1 minute'),
			($1, 'team_invite', 'pending@example.com', '{}'::jsonb, 'pending', 0, now(), NULL, 'diag-pending', NULL, now() - interval '2 minutes', now() - interval '2 minutes'),
			($1, 'team_invite', 'sent@example.com', '{}'::jsonb, 'sent', 1, now(), NULL, 'diag-sent', now(), now() - interval '4 minutes', now())
	`, workspaceID); err != nil {
		t.Fatalf("insert outbox rows: %v", err)
	}
}

func newEmailDiagnosticsIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

	schemaName := fmt.Sprintf("email_diagnostics_%d", time.Now().UnixNano())
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

func loginEmailDiagnosticsUser(t *testing.T, mux http.Handler, username string, password string) []*http.Cookie {
	t.Helper()
	body := fmt.Sprintf(`{"login":%q,"password":%q}`, username, password)
	recorder := performEmailDiagnosticsRequest(mux, http.MethodPost, "/api/v1/auth/login", body, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	return recorder.Result().Cookies()
}

func performEmailDiagnosticsRequest(mux http.Handler, method string, path string, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	reader := bytes.NewReader([]byte(body))
	request := httptest.NewRequest(method, path, reader)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	return recorder
}
