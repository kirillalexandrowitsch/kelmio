//go:build integration

package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUserBySessionResolvesActiveWorkspace(t *testing.T) {
	ctx, db, userID := setupAuthIntegrationWorkspace(t)
	handler := NewHandler(db, time.Hour, false, newIntegrationCSRFManager(t), nil)

	var orgID string
	if err := db.QueryRow(ctx, `SELECT id::text FROM organizations WHERE slug = 'default'`).Scan(&orgID); err != nil {
		t.Fatalf("read default organization: %v", err)
	}

	var secondWorkspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name, organization_id, slug, status)
		VALUES ('Second Workspace', $1, 'second', 'active')
		RETURNING id::text
	`, orgID).Scan(&secondWorkspaceID); err != nil {
		t.Fatalf("insert second workspace: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'member')
	`, secondWorkspaceID, userID); err != nil {
		t.Fatalf("insert second membership: %v", err)
	}

	newSession := func(active any) string {
		token, err := newSessionToken()
		if err != nil {
			t.Fatalf("new session token: %v", err)
		}
		if _, err := db.Exec(ctx, `
			INSERT INTO sessions (user_id, token_hash, expires_at, active_workspace_id)
			VALUES ($1, $2, now() + interval '1 hour', $3)
		`, userID, hashToken(token), active); err != nil {
			t.Fatalf("insert session: %v", err)
		}
		return token
	}

	// A session pinned to the second workspace resolves to it.
	activeToken := newSession(secondWorkspaceID)
	record, err := handler.userBySession(ctx, hashToken(activeToken))
	if err != nil {
		t.Fatalf("userBySession (active): %v", err)
	}
	if record.WorkspaceID != secondWorkspaceID {
		t.Fatalf("active workspace = %q, want %q", record.WorkspaceID, secondWorkspaceID)
	}
	if record.Role != "member" {
		t.Fatalf("active workspace role = %q, want member", record.Role)
	}
	if record.OrganizationID != orgID {
		t.Fatalf("organization id = %q, want %q", record.OrganizationID, orgID)
	}

	// A session without a pinned workspace falls back to the first membership.
	fallbackToken := newSession(nil)
	record, err = handler.userBySession(ctx, hashToken(fallbackToken))
	if err != nil {
		t.Fatalf("userBySession (fallback): %v", err)
	}
	if record.WorkspaceID == secondWorkspaceID {
		t.Fatalf("fallback resolved to the pinned workspace; want the first membership")
	}
	if record.Role != "admin" {
		t.Fatalf("fallback workspace role = %q, want admin", record.Role)
	}

	// A session pinned to a workspace the user does not belong to also falls back.
	var foreignWorkspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name, organization_id, slug, status)
		VALUES ('Foreign Workspace', $1, 'foreign', 'active')
		RETURNING id::text
	`, orgID).Scan(&foreignWorkspaceID); err != nil {
		t.Fatalf("insert foreign workspace: %v", err)
	}
	foreignToken := newSession(foreignWorkspaceID)
	record, err = handler.userBySession(ctx, hashToken(foreignToken))
	if err != nil {
		t.Fatalf("userBySession (foreign): %v", err)
	}
	if record.WorkspaceID == foreignWorkspaceID {
		t.Fatalf("resolved to a workspace the user is not a member of")
	}
}

func TestSetActiveWorkspaceSwitchesScopeAndRejectsForeign(t *testing.T) {
	ctx, db, userID := setupAuthIntegrationWorkspace(t)
	handler := NewHandler(db, time.Hour, false, newIntegrationCSRFManager(t), nil)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	var orgID string
	if err := db.QueryRow(ctx, `SELECT id::text FROM organizations WHERE slug = 'default'`).Scan(&orgID); err != nil {
		t.Fatalf("read default organization: %v", err)
	}
	// Pin the bootstrap workspace to the default organization so switching
	// preserves a consistent active organization.
	if _, err := db.Exec(ctx, `
		UPDATE workspaces SET organization_id = $1
		WHERE id = (SELECT workspace_id FROM workspace_members WHERE user_id = $2 LIMIT 1)
	`, orgID, userID); err != nil {
		t.Fatalf("attach bootstrap workspace: %v", err)
	}

	var secondWorkspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name, organization_id, slug, status)
		VALUES ('Second Workspace', $1, 'second', 'active')
		RETURNING id::text
	`, orgID).Scan(&secondWorkspaceID); err != nil {
		t.Fatalf("insert second workspace: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'member')
	`, secondWorkspaceID, userID); err != nil {
		t.Fatalf("insert second membership: %v", err)
	}

	var foreignWorkspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name, organization_id, slug, status)
		VALUES ('Foreign Workspace', $1, 'foreign', 'active')
		RETURNING id::text
	`, orgID).Scan(&foreignWorkspaceID); err != nil {
		t.Fatalf("insert foreign workspace: %v", err)
	}

	loginResponse := performMuxRequest(mux, http.MethodPost, "/api/v1/auth/login", `{"login":"admin","password":"admin12345"}`, nil)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status = %d: %s", loginResponse.Code, loginResponse.Body.String())
	}
	cookies := loginResponse.Result().Cookies()
	var sessionValue string
	for _, cookie := range cookies {
		if cookie.Name == SessionCookieName {
			sessionValue = cookie.Value
		}
	}
	if sessionValue == "" {
		t.Fatal("login did not set a session cookie")
	}

	// Switching to a workspace the user belongs to rewrites the active scope.
	switchResponse := performMuxRequest(mux, http.MethodPost, "/api/v1/session/active-workspace", fmt.Sprintf(`{"workspace_id":%q}`, secondWorkspaceID), cookies)
	if switchResponse.Code != http.StatusOK {
		t.Fatalf("switch status = %d: %s", switchResponse.Code, switchResponse.Body.String())
	}
	var payload struct {
		User struct {
			Workspace struct {
				ID   string `json:"id"`
				Role string `json:"role"`
			} `json:"workspace"`
		} `json:"user"`
	}
	if err := json.Unmarshal(switchResponse.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode switch response: %v", err)
	}
	if payload.User.Workspace.ID != secondWorkspaceID {
		t.Fatalf("active workspace = %q, want %q", payload.User.Workspace.ID, secondWorkspaceID)
	}
	if payload.User.Workspace.Role != "member" {
		t.Fatalf("active workspace role = %q, want member", payload.User.Workspace.Role)
	}

	var storedActive string
	if err := db.QueryRow(ctx, `
		SELECT COALESCE(active_workspace_id::text, '')
		FROM sessions WHERE token_hash = $1
	`, hashToken(sessionValue)).Scan(&storedActive); err != nil {
		t.Fatalf("read session active workspace: %v", err)
	}
	if storedActive != secondWorkspaceID {
		t.Fatalf("session active workspace = %q, want %q", storedActive, secondWorkspaceID)
	}

	// Switching to a workspace the user is not a member of is rejected.
	foreignResponse := performMuxRequest(mux, http.MethodPost, "/api/v1/session/active-workspace", fmt.Sprintf(`{"workspace_id":%q}`, foreignWorkspaceID), cookies)
	if foreignResponse.Code != http.StatusForbidden {
		t.Fatalf("foreign switch status = %d, want 403: %s", foreignResponse.Code, foreignResponse.Body.String())
	}

	// A malformed workspace id is a bad request.
	invalidResponse := performMuxRequest(mux, http.MethodPost, "/api/v1/session/active-workspace", `{"workspace_id":"not-a-uuid"}`, cookies)
	if invalidResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid switch status = %d, want 400: %s", invalidResponse.Code, invalidResponse.Body.String())
	}
}

func performMuxRequest(mux http.Handler, method string, path string, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}
