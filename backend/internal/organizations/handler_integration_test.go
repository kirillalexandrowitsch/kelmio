//go:build integration

package organizations

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

func TestOrganizationsAPILifecycleAndAuthorization(t *testing.T) {
	databaseURL := getDatabaseURL()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adminDB, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	defer adminDB.Close()
	schemaName := fmt.Sprintf("organizations_api_%d", time.Now().UnixNano())
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
	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name, organization_id, slug, status)
		VALUES ('Org API', $1, 'org-api', 'active')
		RETURNING id::text
	`, defaultOrgID).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin12345"), bcrypt.MinCost)
	insertUser := func(email, username string, siteAdmin bool) string {
		var id string
		if err := db.QueryRow(ctx, `
			INSERT INTO users (email, username, password_hash, display_name, is_active, is_site_admin)
			VALUES ($1, $2, $3, $2, true, $4)
			RETURNING id::text
		`, email, username, string(hash), siteAdmin).Scan(&id); err != nil {
			t.Fatalf("insert user %s: %v", username, err)
		}
		return id
	}
	siteAdminID := insertUser("org-site-admin@example.com", "org_site_admin", true)
	memberID := insertUser("org-member@example.com", "org_member", false)
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin'), ($1, $3, 'member')
	`, workspaceID, siteAdminID, memberID); err != nil {
		t.Fatalf("insert workspace members: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, 'org_admin'), ($1, $3, 'org_member')
	`, defaultOrgID, siteAdminID, memberID); err != nil {
		t.Fatalf("insert organization members: %v", err)
	}

	authHandler := auth.NewHandler(db, time.Hour, false, nil, nil)
	apiHandler := NewHandler(db, authHandler)
	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	apiHandler.RegisterRoutes(mux)

	adminCookies := loginOrgUser(t, mux, "org_site_admin", "admin12345")
	memberCookies := loginOrgUser(t, mux, "org_member", "admin12345")

	// Site admin creates an organization.
	response := performOrgRequest(mux, http.MethodPost, "/api/v1/organizations", `{"name":"Acme Corp"}`, adminCookies)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201: %s", response.Code, response.Body.String())
	}
	var created organizationResponse
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created organization: %v", err)
	}
	if created.Slug != "acme-corp" || created.Status != "active" || created.Role != "org_admin" {
		t.Fatalf("created organization = %#v, want slug acme-corp / active / org_admin", created)
	}

	// Site admin sees every organization.
	response = performOrgRequest(mux, http.MethodGet, "/api/v1/organizations", "", adminCookies)
	if response.Code != http.StatusOK {
		t.Fatalf("admin list status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var adminList listOrganizationsResponse
	if err := json.Unmarshal(response.Body.Bytes(), &adminList); err != nil {
		t.Fatalf("decode admin list: %v", err)
	}
	if !containsOrg(adminList.Organizations, created.ID) || !containsOrg(adminList.Organizations, defaultOrgID) {
		t.Fatalf("admin list = %#v, want both default and created organizations", adminList.Organizations)
	}

	// Site admin renames and archives the organization.
	response = performOrgRequest(mux, http.MethodPatch, "/api/v1/organizations/"+created.ID, `{"name":"Acme","status":"archived"}`, adminCookies)
	if response.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var updated organizationResponse
	if err := json.Unmarshal(response.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated organization: %v", err)
	}
	if updated.Name != "Acme" || updated.Status != "archived" {
		t.Fatalf("updated organization = %#v, want Acme / archived", updated)
	}

	// A non-site-admin cannot create organizations.
	response = performOrgRequest(mux, http.MethodPost, "/api/v1/organizations", `{"name":"Rogue"}`, memberCookies)
	if response.Code != http.StatusForbidden {
		t.Fatalf("member create status = %d, want 403: %s", response.Code, response.Body.String())
	}

	// A non-site-admin only sees the organizations they belong to.
	response = performOrgRequest(mux, http.MethodGet, "/api/v1/organizations", "", memberCookies)
	if response.Code != http.StatusOK {
		t.Fatalf("member list status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var memberList listOrganizationsResponse
	if err := json.Unmarshal(response.Body.Bytes(), &memberList); err != nil {
		t.Fatalf("decode member list: %v", err)
	}
	if !containsOrg(memberList.Organizations, defaultOrgID) {
		t.Fatalf("member list = %#v, want the default organization", memberList.Organizations)
	}
	if containsOrg(memberList.Organizations, created.ID) {
		t.Fatalf("member list includes an organization the user does not belong to")
	}
}

func getDatabaseURL() string {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}
	return "postgres://kelmio:kelmio@localhost:15432/kelmio?sslmode=disable"
}

func containsOrg(organizations []organizationResponse, id string) bool {
	for _, organization := range organizations {
		if organization.ID == id {
			return true
		}
	}
	return false
}

func loginOrgUser(t *testing.T, mux http.Handler, username string, password string) []*http.Cookie {
	t.Helper()
	response := performOrgRequest(mux, http.MethodPost, "/api/v1/auth/login", fmt.Sprintf(`{"login":%q,"password":%q}`, username, password), nil)
	if response.Code != http.StatusOK {
		t.Fatalf("login %s status = %d: %s", username, response.Code, response.Body.String())
	}
	return response.Result().Cookies()
}

func performOrgRequest(mux http.Handler, method string, path string, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
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
