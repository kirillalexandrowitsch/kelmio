//go:build integration

package invites

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	"kelmio/backend/internal/auth"
	"kelmio/backend/internal/database"
	"kelmio/backend/internal/migrations"
)

func TestInviteLifecycleIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newInviteIntegrationDB(t, ctx)
	seed := seedInviteIntegrationWorkspace(t, ctx, db)
	handler := NewHandler(db, nil)

	created, token, err := handler.createInvite(ctx, seed.admin, normalizedCreateInvite{
		Email: "new-member@example.com",
		Role:  "member",
	})
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if created.Status != "pending" {
		t.Fatalf("created status = %q, want pending", created.Status)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
	if created.EmailDeliveryStatus != "pending" || created.EmailQueuedAt == nil {
		t.Fatalf("created delivery status = %q queued=%v, want pending with queued time", created.EmailDeliveryStatus, created.EmailQueuedAt)
	}

	var storedHash string
	if err := db.QueryRow(ctx, `SELECT token_hash FROM team_invites WHERE id = $1`, created.ID).Scan(&storedHash); err != nil {
		t.Fatalf("select token hash: %v", err)
	}
	if storedHash == token {
		t.Fatal("raw token was stored")
	}
	if storedHash != hashInviteToken(token) {
		t.Fatal("stored token hash does not match token")
	}
	var outboxCount int
	var inviteURLPath string
	if err := db.QueryRow(ctx, `
		SELECT count(*)::int, max(template_data->>'invite_url_path')
		FROM email_outbox
		WHERE email_type = 'team_invite'
			AND template_data->>'invite_id' = $1
	`, created.ID).Scan(&outboxCount, &inviteURLPath); err != nil {
		t.Fatalf("select invite outbox: %v", err)
	}
	if outboxCount != 1 || inviteURLPath != "/accept-invite?token="+token {
		t.Fatalf("outboxCount=%d inviteURLPath=%q, want one invite email with accept URL", outboxCount, inviteURLPath)
	}

	listed, err := handler.listInvites(ctx, seed.admin)
	if err != nil {
		t.Fatalf("list invites: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("listed invites = %#v, want created invite", listed)
	}
	if listed[0].EmailDeliveryStatus != "pending" || listed[0].EmailQueuedAt == nil {
		t.Fatalf("listed delivery status = %#v, want pending summary", listed[0])
	}

	if _, _, err := handler.createInvite(ctx, seed.admin, normalizedCreateInvite{
		Email: "new-member@example.com",
		Role:  "member",
	}); !errors.Is(err, errInviteExists) {
		t.Fatalf("duplicate invite error = %v, want %v", err, errInviteExists)
	}

	handler.now = func() time.Time { return time.Now().UTC().Add(2 * time.Minute) }
	resent, err := handler.resendInvite(ctx, seed.admin, created.ID)
	if err != nil {
		t.Fatalf("resend invite: %v", err)
	}
	if resent.ID != created.ID || resent.EmailDeliveryStatus != "pending" || resent.EmailQueuedAt == nil {
		t.Fatalf("resent invite = %#v, want same invite with pending delivery", resent)
	}
	if err := db.QueryRow(ctx, `
		SELECT count(*)::int
		FROM email_outbox
		WHERE email_type = 'team_invite'
			AND template_data->>'invite_id' = $1
	`, created.ID).Scan(&outboxCount); err != nil {
		t.Fatalf("count resent outbox: %v", err)
	}
	if outboxCount != 2 {
		t.Fatalf("outboxCount after resend = %d, want 2", outboxCount)
	}
	handler.now = func() time.Time { return time.Now().UTC() }
	var cooldownErr inviteResendCooldownError
	if _, err := handler.resendInvite(ctx, seed.admin, created.ID); !errors.As(err, &cooldownErr) {
		t.Fatalf("second resend error = %v, want cooldown", err)
	}

	preview, err := handler.inviteByToken(ctx, token)
	if err != nil {
		t.Fatalf("preview invite by token: %v", err)
	}
	if preview.Email != "new-member@example.com" {
		t.Fatalf("preview email = %q, want new-member@example.com", preview.Email)
	}

	accepted, err := handler.acceptInvite(ctx, token, normalizedAcceptInvite{
		Username:    "new_member",
		DisplayName: "New Member",
		Password:    "password123",
	})
	if err != nil {
		t.Fatalf("accept invite: %v", err)
	}
	if !accepted.Accepted || accepted.Username != "new_member" {
		t.Fatalf("accepted response = %#v", accepted)
	}

	assertInvitedUser(t, ctx, db, seed.workspaceID, "new-member@example.com", "new_member", "member", true, "password123")

	if _, err := handler.acceptInvite(ctx, token, normalizedAcceptInvite{
		Username:    "another_member",
		DisplayName: "Another Member",
		Password:    "password123",
	}); !errors.Is(err, errInviteAccepted) {
		t.Fatalf("second accept error = %v, want %v", err, errInviteAccepted)
	}
}

func TestInviteRejectsRevokedExpiredAlreadyMemberAndUserConflictsIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newInviteIntegrationDB(t, ctx)
	seed := seedInviteIntegrationWorkspace(t, ctx, db)
	handler := NewHandler(db, nil)

	revoked, revokedToken, err := handler.createInvite(ctx, seed.admin, normalizedCreateInvite{
		Email: "revoked@example.com",
		Role:  "member",
	})
	if err != nil {
		t.Fatalf("create revoked invite: %v", err)
	}
	if _, err := handler.revokeInvite(ctx, seed.admin, revoked.ID); err != nil {
		t.Fatalf("revoke invite: %v", err)
	}
	if _, err := handler.acceptInvite(ctx, revokedToken, normalizedAcceptInvite{
		Username:    "revoked_user",
		DisplayName: "Revoked User",
		Password:    "password123",
	}); !errors.Is(err, errInviteRevoked) {
		t.Fatalf("revoked accept error = %v, want %v", err, errInviteRevoked)
	}

	expiredToken := "expired-token"
	if _, err := db.Exec(ctx, `
		INSERT INTO team_invites (
			workspace_id,
			email,
			role,
			token_hash,
			created_by,
			created_at,
			expires_at
		)
		VALUES ($1, 'expired@example.com', 'member', $2, $3, now() - interval '48 hours', now() - interval '24 hours')
	`, seed.workspaceID, hashInviteToken(expiredToken), seed.admin.ID); err != nil {
		t.Fatalf("insert expired invite: %v", err)
	}
	if _, err := handler.acceptInvite(ctx, expiredToken, normalizedAcceptInvite{
		Username:    "expired_user",
		DisplayName: "Expired User",
		Password:    "password123",
	}); !errors.Is(err, errInviteExpired) {
		t.Fatalf("expired accept error = %v, want %v", err, errInviteExpired)
	}

	var preEmailInviteID string
	if err := db.QueryRow(ctx, `
		INSERT INTO team_invites (
			workspace_id,
			email,
			role,
			token_hash,
			created_by,
			expires_at
		)
		VALUES ($1, 'pre-email@example.com', 'member', $2, $3, now() + interval '7 days')
		RETURNING id::text
	`, seed.workspaceID, hashInviteToken("pre-email-token"), seed.admin.ID).Scan(&preEmailInviteID); err != nil {
		t.Fatalf("insert pre-email invite: %v", err)
	}
	if _, err := handler.resendInvite(ctx, seed.admin, preEmailInviteID); !errors.Is(err, errInviteEmailUnavailable) {
		t.Fatalf("pre-email resend error = %v, want %v", err, errInviteEmailUnavailable)
	}

	alreadyMemberInvite, alreadyMemberToken, err := handler.createInvite(ctx, seed.admin, normalizedCreateInvite{
		Email: seed.member.Email,
		Role:  "member",
	})
	if err != nil {
		t.Fatalf("create already member invite: %v", err)
	}
	_ = alreadyMemberInvite
	if _, err := handler.acceptInvite(ctx, alreadyMemberToken, normalizedAcceptInvite{
		Username:    "existing_member_new",
		DisplayName: "Existing Member New",
		Password:    "password123",
	}); !errors.Is(err, errAlreadyMember) {
		t.Fatalf("already member accept error = %v, want %v", err, errAlreadyMember)
	}

	conflictInvite, conflictToken, err := handler.createInvite(ctx, seed.admin, normalizedCreateInvite{
		Email: "conflict@example.com",
		Role:  "member",
	})
	if err != nil {
		t.Fatalf("create conflict invite: %v", err)
	}
	_ = conflictInvite
	if _, err := handler.acceptInvite(ctx, conflictToken, normalizedAcceptInvite{
		Username:    seed.admin.Username,
		DisplayName: "Conflict User",
		Password:    "password123",
	}); !errors.Is(err, errUserExists) {
		t.Fatalf("conflict accept error = %v, want %v", err, errUserExists)
	}
}

func TestInviteAcceptReactivatesInactiveMemberIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newInviteIntegrationDB(t, ctx)
	seed := seedInviteIntegrationWorkspace(t, ctx, db)
	handler := NewHandler(db, nil)

	_, token, err := handler.createInvite(ctx, seed.admin, normalizedCreateInvite{
		Email: "inactive_member@example.com",
		Role:  "admin",
	})
	if err != nil {
		t.Fatalf("create invite for inactive member: %v", err)
	}

	if _, err := handler.acceptInvite(ctx, token, normalizedAcceptInvite{
		Username:    "reactivated_member",
		DisplayName: "Reactivated Member",
		Password:    "password123",
	}); err != nil {
		t.Fatalf("accept inactive member invite: %v", err)
	}

	assertInvitedUser(t, ctx, db, seed.workspaceID, "inactive_member@example.com", "reactivated_member", "admin", true, "password123")
}

func TestInviteAPIAdminAccessAndTokenLeakIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newInviteIntegrationDB(t, ctx)
	seed := seedInviteIntegrationWorkspace(t, ctx, db)
	authHandler := auth.NewHandler(db, time.Hour, false, nil, nil)
	inviteHandler := NewHandler(db, authHandler)

	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	inviteHandler.RegisterRoutes(mux)

	memberCookie := loginInviteTestUser(t, mux, seed.member.Username, "member12345")
	memberCreate := performInviteRequest(mux, http.MethodPost, "/api/v1/team/invites", `{"email":"api-member@example.com"}`, memberCookie)
	if memberCreate.Code != http.StatusForbidden {
		t.Fatalf("member create status = %d, want %d: %s", memberCreate.Code, http.StatusForbidden, memberCreate.Body.String())
	}
	memberResend := performInviteRequest(mux, http.MethodPost, "/api/v1/team/invites/00000000-0000-0000-0000-000000000000/resend", "", memberCookie)
	if memberResend.Code != http.StatusForbidden {
		t.Fatalf("member resend status = %d, want %d: %s", memberResend.Code, http.StatusForbidden, memberResend.Body.String())
	}

	adminCookie := loginInviteTestUser(t, mux, seed.admin.Username, "admin12345")
	adminCreate := performInviteRequest(mux, http.MethodPost, "/api/v1/team/invites", `{"email":"api-invite@example.com","role":"member"}`, adminCookie)
	if adminCreate.Code != http.StatusCreated {
		t.Fatalf("admin create status = %d, want %d: %s", adminCreate.Code, http.StatusCreated, adminCreate.Body.String())
	}

	var createBody createInviteResponse
	if err := json.Unmarshal(adminCreate.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createBody.AcceptToken == "" {
		t.Fatal("create response accept_token is empty")
	}
	if createBody.AcceptURLPath != "/accept-invite?token="+createBody.AcceptToken {
		t.Fatalf("accept url path = %q", createBody.AcceptURLPath)
	}
	if createBody.EmailDeliveryStatus != "pending" {
		t.Fatalf("email delivery status = %q, want pending", createBody.EmailDeliveryStatus)
	}

	adminResend := performInviteRequest(mux, http.MethodPost, "/api/v1/team/invites/"+createBody.ID+"/resend", "", adminCookie)
	if adminResend.Code != http.StatusOK {
		t.Fatalf("admin resend status = %d, want %d: %s", adminResend.Code, http.StatusOK, adminResend.Body.String())
	}
	if strings.Contains(adminResend.Body.String(), createBody.AcceptToken) {
		t.Fatal("resend response leaked raw invite token")
	}

	adminList := performInviteRequest(mux, http.MethodGet, "/api/v1/team/invites", "", adminCookie)
	if adminList.Code != http.StatusOK {
		t.Fatalf("admin list status = %d, want %d: %s", adminList.Code, http.StatusOK, adminList.Body.String())
	}
	if strings.Contains(adminList.Body.String(), createBody.AcceptToken) {
		t.Fatal("list response leaked raw invite token")
	}

	publicPreview := performInviteRequest(mux, http.MethodGet, "/api/v1/auth/invites/"+createBody.AcceptToken, "", nil)
	if publicPreview.Code != http.StatusOK {
		t.Fatalf("public preview status = %d, want %d: %s", publicPreview.Code, http.StatusOK, publicPreview.Body.String())
	}
}

func newInviteIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

	schemaName := fmt.Sprintf("invites_integration_%d", time.Now().UnixNano())
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

	return db
}

type inviteSeed struct {
	workspaceID string
	admin       auth.CurrentUser
	member      auth.CurrentUser
}

func seedInviteIntegrationWorkspace(t *testing.T, ctx context.Context, db *pgxpool.Pool) inviteSeed {
	t.Helper()

	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ('Invites Integration Workspace')
		RETURNING id::text
	`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}

	admin := insertInviteUser(t, ctx, db, workspaceID, "admin_user", "admin", true, "admin12345")
	member := insertInviteUser(t, ctx, db, workspaceID, "member_user", "member", true, "member12345")
	_ = insertInviteUser(t, ctx, db, workspaceID, "inactive_member", "member", false, "inactive12345")

	return inviteSeed{
		workspaceID: workspaceID,
		admin:       admin,
		member:      member,
	}
}

func insertInviteUser(t *testing.T, ctx context.Context, db *pgxpool.Pool, workspaceID string, username string, role string, active bool, password string) auth.CurrentUser {
	t.Helper()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	var userID string
	email := username + "@example.com"
	displayName := strings.ReplaceAll(username, "_", " ")
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text
	`, email, username, string(passwordHash), displayName, active).Scan(&userID); err != nil {
		t.Fatalf("insert user %s: %v", username, err)
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, $3)
	`, workspaceID, userID, role); err != nil {
		t.Fatalf("insert workspace member %s: %v", username, err)
	}

	return auth.CurrentUser{
		ID:          userID,
		Email:       email,
		Username:    username,
		DisplayName: displayName,
		WorkspaceID: workspaceID,
		Role:        role,
	}
}

func assertInvitedUser(t *testing.T, ctx context.Context, db *pgxpool.Pool, workspaceID string, email string, username string, role string, active bool, password string) {
	t.Helper()

	var gotUsername string
	var gotPasswordHash string
	var gotRole string
	var gotActive bool
	if err := db.QueryRow(ctx, `
		SELECT u.username, u.password_hash, wm.role, u.is_active
		FROM users u
		JOIN workspace_members wm ON wm.user_id = u.id
		WHERE lower(u.email) = lower($1)
			AND wm.workspace_id = $2
	`, email, workspaceID).Scan(&gotUsername, &gotPasswordHash, &gotRole, &gotActive); err != nil {
		t.Fatalf("select invited user: %v", err)
	}

	if gotUsername != username {
		t.Fatalf("username = %q, want %q", gotUsername, username)
	}
	if gotRole != role {
		t.Fatalf("role = %q, want %q", gotRole, role)
	}
	if gotActive != active {
		t.Fatalf("active = %v, want %v", gotActive, active)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(gotPasswordHash), []byte(password)); err != nil {
		t.Fatalf("password hash does not match: %v", err)
	}
}

func loginInviteTestUser(t *testing.T, mux http.Handler, username string, password string) []*http.Cookie {
	t.Helper()

	body := fmt.Sprintf(`{"login":%q,"password":%q}`, username, password)
	recorder := performInviteRequest(mux, http.MethodPost, "/api/v1/auth/login", body, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("login %s status = %d, want %d: %s", username, recorder.Code, http.StatusOK, recorder.Body.String())
	}

	return recorder.Result().Cookies()
}

func performInviteRequest(mux http.Handler, method string, path string, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}

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
