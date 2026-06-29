//go:build integration

package auth

import (
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

	"kelmio/backend/internal/csrf"
	"kelmio/backend/internal/database"
	"kelmio/backend/internal/migrations"
	"kelmio/backend/internal/ratelimit"
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

func TestLoginIncludesSiteAdminAndOrganization(t *testing.T) {
	ctx, db, userID := setupAuthIntegrationWorkspace(t)

	var organizationID string
	if err := db.QueryRow(ctx, `
		SELECT id::text FROM organizations WHERE slug = 'default'
	`).Scan(&organizationID); err != nil {
		t.Fatalf("read default organization: %v", err)
	}
	if _, err := db.Exec(ctx, `
		UPDATE workspaces SET organization_id = $1
		WHERE id = (SELECT workspace_id FROM workspace_members WHERE user_id = $2 LIMIT 1)
	`, organizationID, userID); err != nil {
		t.Fatalf("attach workspace to organization: %v", err)
	}
	if _, err := db.Exec(ctx, `UPDATE users SET is_site_admin = true WHERE id = $1`, userID); err != nil {
		t.Fatalf("set site admin: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, 'org_admin')
	`, organizationID, userID); err != nil {
		t.Fatalf("insert organization membership: %v", err)
	}

	handler := NewHandler(db, time.Hour, false, newIntegrationCSRFManager(t), nil)
	response := performLogin(handler, `{"login":"admin","password":"admin12345"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("login status = %d: %s", response.Code, response.Body.String())
	}

	var payload struct {
		User struct {
			IsSiteAdmin  bool `json:"is_site_admin"`
			Organization struct {
				ID   string `json:"id"`
				Role string `json:"role"`
			} `json:"organization"`
		} `json:"user"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if !payload.User.IsSiteAdmin {
		t.Fatal("login response is_site_admin = false, want true")
	}
	if payload.User.Organization.ID != organizationID {
		t.Fatalf("login organization id = %q, want %q", payload.User.Organization.ID, organizationID)
	}
	if payload.User.Organization.Role != "org_admin" {
		t.Fatalf("login organization role = %q, want org_admin", payload.User.Organization.Role)
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

func TestPasswordResetLifecycle(t *testing.T) {
	ctx, db, userID := setupAuthIntegrationWorkspace(t)
	if _, err := db.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, now() + interval '1 hour')
	`, userID, hashToken("existing-session")); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	handler := NewHandler(
		db,
		time.Hour,
		false,
		newIntegrationCSRFManager(t),
		nil,
		WithPasswordResetTTL(30*time.Minute),
		WithPasswordResetBaseURL("http://localhost:5173"),
	)
	requestRecorder := performPasswordResetRequest(handler, `{"email":" Admin@Example.COM "}`)
	if requestRecorder.Code != http.StatusAccepted {
		t.Fatalf("request status = %d, want %d: %s", requestRecorder.Code, http.StatusAccepted, requestRecorder.Body.String())
	}

	token := resetTokenFromOutbox(t, ctx, db)
	var tokenHash string
	if err := db.QueryRow(ctx, `SELECT token_hash FROM password_reset_tokens WHERE user_id = $1`, userID).Scan(&tokenHash); err != nil {
		t.Fatalf("select reset token hash: %v", err)
	}
	if tokenHash != hashToken(token) {
		t.Fatalf("stored token hash = %q, want hash of outbox token", tokenHash)
	}

	previewRecorder := performPasswordResetPreview(handler, token)
	if previewRecorder.Code != http.StatusOK {
		t.Fatalf("preview status = %d, want %d: %s", previewRecorder.Code, http.StatusOK, previewRecorder.Body.String())
	}
	if body := previewRecorder.Body.String(); !strings.Contains(body, "admin@example.com") {
		t.Fatalf("preview body = %q, want normalized email", body)
	}

	completeRecorder := performPasswordResetComplete(handler, token, `{"password":"new-admin-password","confirm_password":"new-admin-password"}`)
	if completeRecorder.Code != http.StatusNoContent {
		t.Fatalf("complete status = %d, want %d: %s", completeRecorder.Code, http.StatusNoContent, completeRecorder.Body.String())
	}

	var sessionCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE user_id = $1`, userID).Scan(&sessionCount); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessionCount != 0 {
		t.Fatalf("sessionCount = %d, want 0 after reset", sessionCount)
	}

	oldLogin := performLogin(handler, `{"login":"admin","password":"admin12345"}`)
	if oldLogin.Code != http.StatusUnauthorized {
		t.Fatalf("old login status = %d, want %d", oldLogin.Code, http.StatusUnauthorized)
	}
	newLogin := performLogin(handler, `{"login":"admin","password":"new-admin-password"}`)
	if newLogin.Code != http.StatusOK {
		t.Fatalf("new login status = %d, want %d: %s", newLogin.Code, http.StatusOK, newLogin.Body.String())
	}

	reuseRecorder := performPasswordResetComplete(handler, token, `{"password":"another-password","confirm_password":"another-password"}`)
	if reuseRecorder.Code != http.StatusBadRequest || !strings.Contains(reuseRecorder.Body.String(), "password_reset_used") {
		t.Fatalf("reuse response = %d %s, want password_reset_used", reuseRecorder.Code, reuseRecorder.Body.String())
	}
}

func TestPasswordResetPrivacyAndRevokesPreviousTokens(t *testing.T) {
	ctx, db, userID := setupAuthIntegrationWorkspace(t)
	handler := NewHandler(db, time.Hour, false, newIntegrationCSRFManager(t), nil, WithPasswordResetTTL(30*time.Minute))

	unknownRecorder := performPasswordResetRequest(handler, `{"email":"unknown@example.com"}`)
	if unknownRecorder.Code != http.StatusAccepted {
		t.Fatalf("unknown status = %d, want %d", unknownRecorder.Code, http.StatusAccepted)
	}
	var unknownOutboxCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM email_outbox`).Scan(&unknownOutboxCount); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if unknownOutboxCount != 0 {
		t.Fatalf("unknownOutboxCount = %d, want 0", unknownOutboxCount)
	}

	firstRecorder := performPasswordResetRequest(handler, `{"email":"admin@example.com"}`)
	if firstRecorder.Code != http.StatusAccepted {
		t.Fatalf("first status = %d, want %d", firstRecorder.Code, http.StatusAccepted)
	}
	secondRecorder := performPasswordResetRequest(handler, `{"email":"admin@example.com"}`)
	if secondRecorder.Code != http.StatusAccepted {
		t.Fatalf("second status = %d, want %d", secondRecorder.Code, http.StatusAccepted)
	}

	var totalTokens int
	var revokedTokens int
	if err := db.QueryRow(ctx, `
		SELECT count(*)::int, count(*) FILTER (WHERE revoked_at IS NOT NULL)::int
		FROM password_reset_tokens
		WHERE user_id = $1
	`, userID).Scan(&totalTokens, &revokedTokens); err != nil {
		t.Fatalf("count tokens: %v", err)
	}
	if totalTokens != 2 || revokedTokens != 1 {
		t.Fatalf("totalTokens=%d revokedTokens=%d, want 2/1", totalTokens, revokedTokens)
	}
	var outboxCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM email_outbox WHERE email_type = 'password_reset'`).Scan(&outboxCount); err != nil {
		t.Fatalf("count reset outbox: %v", err)
	}
	if outboxCount != 2 {
		t.Fatalf("outboxCount = %d, want 2", outboxCount)
	}
}

func TestPasswordResetTokenStates(t *testing.T) {
	ctx, db, userID := setupAuthIntegrationWorkspace(t)
	handler := NewHandler(db, time.Hour, false, newIntegrationCSRFManager(t), nil)
	now := time.Now().UTC()

	insertResetTokenState(t, ctx, db, userID, "expired-token", now.Add(-time.Hour), nil, nil)
	usedAt := now.Add(-time.Minute)
	insertResetTokenState(t, ctx, db, userID, "used-token", now.Add(time.Hour), &usedAt, nil)
	revokedAt := now.Add(-time.Minute)
	insertResetTokenState(t, ctx, db, userID, "revoked-token", now.Add(time.Hour), nil, &revokedAt)

	tests := []struct {
		name      string
		token     string
		wantCode  int
		wantError string
	}{
		{name: "unknown", token: "unknown-token", wantCode: http.StatusNotFound, wantError: "password_reset_not_found"},
		{name: "expired", token: "expired-token", wantCode: http.StatusBadRequest, wantError: "password_reset_expired"},
		{name: "used", token: "used-token", wantCode: http.StatusBadRequest, wantError: "password_reset_used"},
		{name: "revoked", token: "revoked-token", wantCode: http.StatusBadRequest, wantError: "password_reset_revoked"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := performPasswordResetPreview(handler, tt.token)
			if recorder.Code != tt.wantCode || !strings.Contains(recorder.Body.String(), tt.wantError) {
				t.Fatalf("response = %d %s, want %d/%s", recorder.Code, recorder.Body.String(), tt.wantCode, tt.wantError)
			}
		})
	}
}

func performLogin(handler *Handler, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.login(recorder, request)
	return recorder
}

func performPasswordResetRequest(handler *Handler, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-reset/request", strings.NewReader(body))
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("User-Agent", "integration-test")
	recorder := httptest.NewRecorder()
	handler.requestPasswordReset(recorder, request)
	return recorder
}

func performPasswordResetPreview(handler *Handler, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/password-reset/"+token, nil)
	request.SetPathValue("token", token)
	recorder := httptest.NewRecorder()
	handler.previewPasswordReset(recorder, request)
	return recorder
}

func performPasswordResetComplete(handler *Handler, token string, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-reset/"+token+"/complete", strings.NewReader(body))
	request.SetPathValue("token", token)
	recorder := httptest.NewRecorder()
	handler.completePasswordReset(recorder, request)
	return recorder
}

func resetTokenFromOutbox(t *testing.T, ctx context.Context, db *pgxpool.Pool) string {
	t.Helper()
	var resetURLPath string
	if err := db.QueryRow(ctx, `
		SELECT template_data->>'reset_url_path'
		FROM email_outbox
		WHERE email_type = 'password_reset'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&resetURLPath); err != nil {
		t.Fatalf("select reset url path: %v", err)
	}
	const tokenPrefix = "/reset-password?token="
	if !strings.HasPrefix(resetURLPath, tokenPrefix) {
		t.Fatalf("resetURLPath = %q, want prefix %q", resetURLPath, tokenPrefix)
	}
	return strings.TrimPrefix(resetURLPath, tokenPrefix)
}

func insertResetTokenState(t *testing.T, ctx context.Context, db *pgxpool.Pool, userID string, token string, expiresAt time.Time, usedAt *time.Time, revokedAt *time.Time) {
	t.Helper()
	createdAt := expiresAt.Add(-time.Hour)
	if _, err := db.Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token_hash, created_at, expires_at, used_at, revoked_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, hashToken(token), createdAt, expiresAt, usedAt, revokedAt); err != nil {
		t.Fatalf("insert reset token %s: %v", token, err)
	}
}

func setupAuthIntegrationWorkspace(t *testing.T) (context.Context, *pgxpool.Pool, string) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://kelmio:kelmio@localhost:15432/kelmio?sslmode=disable"
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
