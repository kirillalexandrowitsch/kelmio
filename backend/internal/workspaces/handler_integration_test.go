//go:build integration

package workspaces

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

	"kelmio/backend/internal/auth"
	"kelmio/backend/internal/database"
	"kelmio/backend/internal/migrations"
)

func TestListWorkspacesScopesToActiveOrganization(t *testing.T) {
	databaseURL := getDatabaseURL()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adminDB, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	defer adminDB.Close()
	schemaName := fmt.Sprintf("workspaces_api_%d", time.Now().UnixNano())
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
	cfg.MaxConns = 3
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect schema: %v", err)
	}
	defer db.Close()
	if _, err := migrations.Up(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	var defaultOrgID string
	if err := db.QueryRow(ctx, `SELECT id::text FROM organizations WHERE slug = 'default'`).Scan(&defaultOrgID); err != nil {
		t.Fatalf("read default organization: %v", err)
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("admin12345"), bcrypt.MinCost)
	var userID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name, is_active)
		VALUES ('ws-user@example.com', 'ws_user', $1, 'WS User', true)
		RETURNING id::text
	`, string(hash)).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var foreignOrgID string
	if err := db.QueryRow(ctx, `
		INSERT INTO organizations (name, slug, status, created_by)
		VALUES ('Foreign Org', 'foreign-org', 'active', $1)
		RETURNING id::text
	`, userID).Scan(&foreignOrgID); err != nil {
		t.Fatalf("insert foreign organization: %v", err)
	}

	insertWorkspace := func(name, slug, status, organizationID string) string {
		var id string
		if err := db.QueryRow(ctx, `
			INSERT INTO workspaces (name, organization_id, slug, status)
			VALUES ($1, $2, $3, $4)
			RETURNING id::text
		`, name, organizationID, slug, status).Scan(&id); err != nil {
			t.Fatalf("insert workspace %s: %v", name, err)
		}
		return id
	}
	alphaID := insertWorkspace("Alpha", "alpha", "active", defaultOrgID)
	betaID := insertWorkspace("Beta", "beta", "active", defaultOrgID)
	gammaID := insertWorkspace("Gamma", "gamma", "archived", defaultOrgID)
	deltaID := insertWorkspace("Delta", "delta", "active", foreignOrgID)

	// Order memberships so the earliest join (Alpha) is resolved as the active
	// workspace, which pins the active organization to the default organization.
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, joined_at) VALUES
			($1, $5, 'admin', now() - interval '3 minutes'),
			($2, $5, 'member', now() - interval '2 minutes'),
			($3, $5, 'member', now() - interval '1 minute'),
			($4, $5, 'member', now())
	`, alphaID, betaID, gammaID, deltaID, userID); err != nil {
		t.Fatalf("insert workspace members: %v", err)
	}

	authHandler := auth.NewHandler(db, time.Hour, false, nil, nil)
	apiHandler := NewHandler(db, authHandler)
	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	apiHandler.RegisterRoutes(mux)

	cookies := loginUser(t, mux, "ws_user", "admin12345")

	response := performRequest(mux, http.MethodGet, "/api/v1/workspaces", "", cookies)
	if response.Code != http.StatusOK {
		t.Fatalf("list status = %d: %s", response.Code, response.Body.String())
	}

	var payload listWorkspacesResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(payload.Workspaces) != 2 {
		t.Fatalf("workspaces = %#v, want only the two active default-org workspaces", payload.Workspaces)
	}

	alpha := findWorkspace(payload.Workspaces, alphaID)
	if alpha == nil {
		t.Fatal("Alpha workspace missing from list")
	}
	if !alpha.IsActive {
		t.Fatal("Alpha should be marked as the active workspace")
	}
	if alpha.Role != "admin" {
		t.Fatalf("Alpha role = %q, want admin", alpha.Role)
	}

	beta := findWorkspace(payload.Workspaces, betaID)
	if beta == nil {
		t.Fatal("Beta workspace missing from list")
	}
	if beta.IsActive {
		t.Fatal("Beta should not be marked active")
	}

	if findWorkspace(payload.Workspaces, gammaID) != nil {
		t.Fatal("archived Gamma workspace should be excluded")
	}
	if findWorkspace(payload.Workspaces, deltaID) != nil {
		t.Fatal("foreign-organization Delta workspace should be excluded")
	}
}

func TestWorkspaceLifecycleAndAuthorization(t *testing.T) {
	databaseURL := getDatabaseURL()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adminDB, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	defer adminDB.Close()
	schemaName := fmt.Sprintf("workspaces_lifecycle_%d", time.Now().UnixNano())
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
	cfg.MaxConns = 3
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect schema: %v", err)
	}
	defer db.Close()
	if _, err := migrations.Up(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	var orgID string
	if err := db.QueryRow(ctx, `SELECT id::text FROM organizations WHERE slug = 'default'`).Scan(&orgID); err != nil {
		t.Fatalf("read default organization: %v", err)
	}
	var homeID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name, organization_id, slug, status)
		VALUES ('Home', $1, 'home', 'active')
		RETURNING id::text
	`, orgID).Scan(&homeID); err != nil {
		t.Fatalf("insert home workspace: %v", err)
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("admin12345"), bcrypt.MinCost)
	insertUser := func(email, username string) string {
		var id string
		if err := db.QueryRow(ctx, `
			INSERT INTO users (email, username, password_hash, display_name, is_active)
			VALUES ($1, $2, $3, $2, true)
			RETURNING id::text
		`, email, username, string(hash)).Scan(&id); err != nil {
			t.Fatalf("insert user %s: %v", username, err)
		}
		return id
	}
	adminID := insertUser("ws-admin@example.com", "ws_admin")
	memberID := insertUser("ws-member@example.com", "ws_member")
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin'), ($1, $3, 'member')
	`, homeID, adminID, memberID); err != nil {
		t.Fatalf("insert workspace members: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, 'org_admin'), ($1, $3, 'org_member')
	`, orgID, adminID, memberID); err != nil {
		t.Fatalf("insert organization members: %v", err)
	}

	authHandler := auth.NewHandler(db, time.Hour, false, nil, nil)
	apiHandler := NewHandler(db, authHandler)
	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	apiHandler.RegisterRoutes(mux)

	adminCookies := loginUser(t, mux, "ws_admin", "admin12345")
	memberCookies := loginUser(t, mux, "ws_member", "admin12345")

	// An organization admin creates a workspace and becomes its administrator.
	createResponse := performRequest(mux, http.MethodPost, "/api/v1/workspaces", `{"name":"Marketing"}`, adminCookies)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", createResponse.Code, createResponse.Body.String())
	}
	var created workspaceResponse
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Name != "Marketing" || created.Slug != "marketing" || created.Status != "active" || created.Role != "admin" {
		t.Fatalf("created = %#v, want Marketing/marketing/active/admin", created)
	}

	// The new workspace is listed for its creator.
	listResponse := performRequest(mux, http.MethodGet, "/api/v1/workspaces", "", adminCookies)
	var listed listWorkspacesResponse
	_ = json.Unmarshal(listResponse.Body.Bytes(), &listed)
	if findWorkspace(listed.Workspaces, created.ID) == nil {
		t.Fatal("created workspace missing from the creator's list")
	}

	// Rename and then archive the workspace.
	renameResponse := performRequest(mux, http.MethodPatch, "/api/v1/workspaces/"+created.ID, `{"name":"Marketing Team"}`, adminCookies)
	if renameResponse.Code != http.StatusOK {
		t.Fatalf("rename status = %d: %s", renameResponse.Code, renameResponse.Body.String())
	}
	var renamed workspaceResponse
	_ = json.Unmarshal(renameResponse.Body.Bytes(), &renamed)
	if renamed.Name != "Marketing Team" {
		t.Fatalf("renamed name = %q, want Marketing Team", renamed.Name)
	}

	archiveResponse := performRequest(mux, http.MethodPatch, "/api/v1/workspaces/"+created.ID, `{"status":"archived"}`, adminCookies)
	if archiveResponse.Code != http.StatusOK {
		t.Fatalf("archive status = %d: %s", archiveResponse.Code, archiveResponse.Body.String())
	}
	var archived workspaceResponse
	_ = json.Unmarshal(archiveResponse.Body.Bytes(), &archived)
	if archived.Status != "archived" {
		t.Fatalf("archived status = %q, want archived", archived.Status)
	}

	// Archived workspaces drop out of the switcher list.
	afterArchive := performRequest(mux, http.MethodGet, "/api/v1/workspaces", "", adminCookies)
	var afterArchiveList listWorkspacesResponse
	_ = json.Unmarshal(afterArchive.Body.Bytes(), &afterArchiveList)
	if findWorkspace(afterArchiveList.Workspaces, created.ID) != nil {
		t.Fatal("archived workspace should be excluded from the list")
	}

	// A plain member cannot create or modify workspaces.
	memberCreate := performRequest(mux, http.MethodPost, "/api/v1/workspaces", `{"name":"Sales"}`, memberCookies)
	if memberCreate.Code != http.StatusForbidden {
		t.Fatalf("member create status = %d, want 403: %s", memberCreate.Code, memberCreate.Body.String())
	}
	memberUpdate := performRequest(mux, http.MethodPatch, "/api/v1/workspaces/"+homeID, `{"name":"Renamed Home"}`, memberCookies)
	if memberUpdate.Code != http.StatusForbidden {
		t.Fatalf("member update status = %d, want 403: %s", memberUpdate.Code, memberUpdate.Body.String())
	}

	// Updating a workspace that does not exist is a not-found.
	missing := performRequest(mux, http.MethodPatch, "/api/v1/workspaces/00000000-0000-0000-0000-000000000000", `{"name":"Nope"}`, adminCookies)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing update status = %d, want 404: %s", missing.Code, missing.Body.String())
	}
}

func findWorkspace(workspaces []workspaceResponse, id string) *workspaceResponse {
	for i := range workspaces {
		if workspaces[i].ID == id {
			return &workspaces[i]
		}
	}
	return nil
}

func getDatabaseURL() string {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}
	return "postgres://kelmio:kelmio@localhost:15432/kelmio?sslmode=disable"
}

func loginUser(t *testing.T, mux http.Handler, username string, password string) []*http.Cookie {
	t.Helper()
	response := performRequest(mux, http.MethodPost, "/api/v1/auth/login", fmt.Sprintf(`{"login":%q,"password":%q}`, username, password), nil)
	if response.Code != http.StatusOK {
		t.Fatalf("login %s status = %d: %s", username, response.Code, response.Body.String())
	}
	return response.Result().Cookies()
}

func performRequest(mux http.Handler, method string, path string, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
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
