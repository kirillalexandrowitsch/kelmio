//go:build integration

package auth

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
	"golang.org/x/crypto/bcrypt"

	"team-task-tracker/backend/internal/csrf"
	"team-task-tracker/backend/internal/database"
	"team-task-tracker/backend/internal/migrations"
	"team-task-tracker/backend/internal/ratelimit"
)

func TestLoginRateLimitBlocksAndSuccessfulLoginResets(t *testing.T) {
	ctx, db, userID := setupAuthIntegrationWorkspace(t)
	_ = userID

	csrfManager := newIntegrationCSRFManager(t)
	now := time.Date(2026, time.June, 4, 12, 0, 0, 0, time.UTC)
	limiter := ratelimit.NewLimiter(2, time.Minute, func() time.Time { return now })
	handler := NewHandler(db, time.Hour, false, csrfManager, limiter)

	firstInvalid := performLogin(handler, `{"login":" admin ","password":"wrong-password"}`)
	if firstInvalid.Code != http.StatusUnauthorized {
		t.Fatalf("first invalid status = %d, want %d", firstInvalid.Code, http.StatusUnauthorized)
	}

	valid := performLogin(handler, `{"login":"ADMIN","password":"admin12345"}`)
	if valid.Code != http.StatusOK {
		t.Fatalf("valid status = %d, want %d: %s", valid.Code, http.StatusOK, valid.Body.String())
	}

	invalidAfterReset := performLogin(handler, `{"login":"admin","password":"wrong-password"}`)
	if invalidAfterReset.Code != http.StatusUnauthorized {
		t.Fatalf("invalid after reset status = %d, want %d", invalidAfterReset.Code, http.StatusUnauthorized)
	}

	secondInvalidAfterReset := performLogin(handler, `{"login":"admin","password":"wrong-password"}`)
	if secondInvalidAfterReset.Code != http.StatusUnauthorized {
		t.Fatalf("second invalid after reset status = %d, want %d", secondInvalidAfterReset.Code, http.StatusUnauthorized)
	}

	blocked := performLogin(handler, `{"login":"admin","password":"wrong-password"}`)
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("blocked status = %d, want %d", blocked.Code, http.StatusTooManyRequests)
	}
	if got := blocked.Header().Get("Retry-After"); got == "" {
		t.Fatal("Retry-After header is empty")
	}
	if body := blocked.Body.String(); !strings.Contains(body, "rate_limited") {
		t.Fatalf("body = %q, want rate_limited", body)
	}

	var sessionCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE user_id = $1`, userID).Scan(&sessionCount); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessionCount != 1 {
		t.Fatalf("sessionCount = %d, want 1", sessionCount)
	}
}

func TestMeCleansExpiredSessionsAndKeepsActiveSessions(t *testing.T) {
	ctx, db, userID := setupAuthIntegrationWorkspace(t)

	activeToken := "active-session-token"
	expiredToken := "expired-session-token"
	if _, err := db.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES
			($1, $2, now() + interval '1 hour'),
			($1, $3, now() - interval '1 hour')
	`, userID, hashToken(activeToken), hashToken(expiredToken)); err != nil {
		t.Fatalf("insert sessions: %v", err)
	}

	handler := NewHandler(db, time.Hour, false, newIntegrationCSRFManager(t), nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	request.AddCookie(&http.Cookie{Name: SessionCookieName, Value: activeToken})
	recorder := httptest.NewRecorder()
	handler.me(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var activeExists bool
	if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM sessions WHERE token_hash = $1)`, hashToken(activeToken)).Scan(&activeExists); err != nil {
		t.Fatalf("check active session: %v", err)
	}
	if !activeExists {
		t.Fatal("active session was removed")
	}

	var expiredExists bool
	if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM sessions WHERE token_hash = $1)`, hashToken(expiredToken)).Scan(&expiredExists); err != nil {
		t.Fatalf("check expired session: %v", err)
	}
	if expiredExists {
		t.Fatal("expired session was not removed")
	}
}

func performLogin(handler *Handler, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.login(recorder, request)
	return recorder
}

func setupAuthIntegrationWorkspace(t *testing.T) (context.Context, *pgxpool.Pool, string) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://team_task_tracker:team_task_tracker@localhost:15432/team_task_tracker?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	adminDB, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	t.Cleanup(adminDB.Close)

	schemaName := fmt.Sprintf("auth_integration_%d", time.Now().UnixNano())
	quotedSchemaName := pgx.Identifier{schemaName}.Sanitize()

	if _, err := adminDB.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`); err != nil {
		t.Fatalf("ensure pgcrypto extension: %v", err)
	}
	if _, err := adminDB.Exec(ctx, `CREATE SCHEMA `+quotedSchemaName); err != nil {
		t.Fatalf("create integration schema: %v", err)
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
		t.Fatalf("connect to integration schema: %v", err)
	}
	t.Cleanup(db.Close)

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping integration database: %v", err)
	}
	if _, err := migrations.Up(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ($1)
		RETURNING id::text
	`, "Auth Integration Workspace").Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("admin12345"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	var userID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, "admin@example.com", "admin", string(passwordHash), "Admin").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin')
	`, workspaceID, userID); err != nil {
		t.Fatalf("insert workspace member: %v", err)
	}

	return ctx, db, userID
}

func newIntegrationCSRFManager(t *testing.T) *csrf.Manager {
	t.Helper()

	manager, err := csrf.NewManager("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("create csrf manager: %v", err)
	}
	return manager
}
