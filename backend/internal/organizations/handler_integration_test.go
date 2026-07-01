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

func TestOrganizationMembersAPI(t *testing.T) {
	databaseURL := getDatabaseURL()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adminDB, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	defer adminDB.Close()
	schemaName := fmt.Sprintf("organization_members_%d", time.Now().UnixNano())
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
		VALUES ('Org Members', $1, 'org-members', 'active')
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
	siteAdminID := insertUser("members-site-admin@example.com", "members_site_admin", true)
	outsiderID := insertUser("members-outsider@example.com", "members_outsider", false)
	strangerID := insertUser("members-stranger@example.com", "members_stranger", false)
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin'), ($1, $3, 'member'), ($1, $4, 'member')
	`, workspaceID, siteAdminID, outsiderID, strangerID); err != nil {
		t.Fatalf("insert workspace members: %v", err)
	}

	authHandler := auth.NewHandler(db, time.Hour, false, nil, nil)
	apiHandler := NewHandler(db, authHandler)
	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	apiHandler.RegisterRoutes(mux)

	adminCookies := loginOrgUser(t, mux, "members_site_admin", "admin12345")
	strangerCookies := loginOrgUser(t, mux, "members_stranger", "admin12345")

	// Site admin creates an organization (becoming its first org admin).
	response := performOrgRequest(mux, http.MethodPost, "/api/v1/organizations", `{"name":"Members Org"}`, adminCookies)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", response.Code, response.Body.String())
	}
	var org organizationResponse
	_ = json.Unmarshal(response.Body.Bytes(), &org)

	// Members list starts with the creator as org admin.
	response = performOrgRequest(mux, http.MethodGet, "/api/v1/organizations/"+org.ID+"/members", "", adminCookies)
	if response.Code != http.StatusOK {
		t.Fatalf("list members status = %d: %s", response.Code, response.Body.String())
	}
	var members listOrganizationMembersResponse
	_ = json.Unmarshal(response.Body.Bytes(), &members)
	if len(members.Members) != 1 || members.Members[0].UserID != siteAdminID || members.Members[0].Role != "org_admin" {
		t.Fatalf("members = %#v, want only the creator as org_admin", members.Members)
	}

	// Add an outsider as an organization member.
	response = performOrgRequest(mux, http.MethodPost, "/api/v1/organizations/"+org.ID+"/members", fmt.Sprintf(`{"user_id":%q,"role":"org_member"}`, outsiderID), adminCookies)
	if response.Code != http.StatusOK {
		t.Fatalf("add member status = %d: %s", response.Code, response.Body.String())
	}

	// Removing the only org admin is rejected.
	response = performOrgRequest(mux, http.MethodDelete, "/api/v1/organizations/"+org.ID+"/members/"+siteAdminID, "", adminCookies)
	if response.Code != http.StatusConflict {
		t.Fatalf("remove last admin status = %d, want 409: %s", response.Code, response.Body.String())
	}

	// Promote the outsider to org admin, then removing the original admin is allowed.
	response = performOrgRequest(mux, http.MethodPost, "/api/v1/organizations/"+org.ID+"/members", fmt.Sprintf(`{"user_id":%q,"role":"org_admin"}`, outsiderID), adminCookies)
	if response.Code != http.StatusOK {
		t.Fatalf("promote status = %d: %s", response.Code, response.Body.String())
	}
	response = performOrgRequest(mux, http.MethodDelete, "/api/v1/organizations/"+org.ID+"/members/"+siteAdminID, "", adminCookies)
	if response.Code != http.StatusNoContent {
		t.Fatalf("remove admin status = %d, want 204: %s", response.Code, response.Body.String())
	}

	// A non-site-admin who is not a member cannot read or manage members.
	response = performOrgRequest(mux, http.MethodGet, "/api/v1/organizations/"+org.ID+"/members", "", strangerCookies)
	if response.Code != http.StatusForbidden {
		t.Fatalf("stranger list status = %d, want 403: %s", response.Code, response.Body.String())
	}
	response = performOrgRequest(mux, http.MethodPost, "/api/v1/organizations/"+org.ID+"/members", fmt.Sprintf(`{"user_id":%q,"role":"org_member"}`, strangerID), strangerCookies)
	if response.Code != http.StatusForbidden {
		t.Fatalf("stranger add status = %d, want 403: %s", response.Code, response.Body.String())
	}

	// Administrative actions are recorded in the audit log.
	var createdCount, addedCount, removedCount int
	if err := db.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE action = 'organization.created' AND actor_id = $2),
			count(*) FILTER (WHERE action = 'organization.member_added'),
			count(*) FILTER (WHERE action = 'organization.member_removed')
		FROM audit_log WHERE organization_id = $1
	`, org.ID, siteAdminID).Scan(&createdCount, &addedCount, &removedCount); err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	if createdCount != 1 {
		t.Fatalf("organization.created audit rows = %d, want 1", createdCount)
	}
	if addedCount < 2 {
		t.Fatalf("organization.member_added audit rows = %d, want >= 2", addedCount)
	}
	if removedCount != 1 {
		t.Fatalf("organization.member_removed audit rows = %d, want 1", removedCount)
	}
}

func TestDirectoryListsActiveOrganizationMembers(t *testing.T) {
	databaseURL := getDatabaseURL()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adminDB, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	defer adminDB.Close()
	schemaName := fmt.Sprintf("directory_%d", time.Now().UnixNano())
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
	var foreignOrgID string
	if err := db.QueryRow(ctx, `
		INSERT INTO organizations (name, slug, status) VALUES ('Foreign', 'foreign', 'active')
		RETURNING id::text
	`).Scan(&foreignOrgID); err != nil {
		t.Fatalf("insert foreign organization: %v", err)
	}
	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name, organization_id, slug, status)
		VALUES ('Directory WS', $1, 'directory-ws', 'active')
		RETURNING id::text
	`, defaultOrgID).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("admin12345"), bcrypt.MinCost)
	insertUser := func(email, username string, active bool) string {
		var id string
		if err := db.QueryRow(ctx, `
			INSERT INTO users (email, username, password_hash, display_name, is_active)
			VALUES ($1, $2, $3, $2, $4)
			RETURNING id::text
		`, email, username, string(hash), active).Scan(&id); err != nil {
			t.Fatalf("insert user %s: %v", username, err)
		}
		return id
	}
	adminID := insertUser("dir-admin@example.com", "dir_admin", true)
	memberID := insertUser("dir-member@example.com", "dir_member", true)
	inactiveID := insertUser("dir-inactive@example.com", "dir_inactive", false)
	foreignID := insertUser("dir-foreign@example.com", "dir_foreign", true)

	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin'), ($1, $3, 'member')
	`, workspaceID, adminID, memberID); err != nil {
		t.Fatalf("insert workspace members: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role) VALUES
			($1, $2, 'org_admin'),
			($1, $3, 'org_member'),
			($1, $4, 'org_member')
	`, defaultOrgID, adminID, memberID, inactiveID); err != nil {
		t.Fatalf("insert organization members: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role) VALUES ($1, $2, 'org_member')
	`, foreignOrgID, foreignID); err != nil {
		t.Fatalf("insert foreign organization member: %v", err)
	}

	authHandler := auth.NewHandler(db, time.Hour, false, nil, nil)
	apiHandler := NewHandler(db, authHandler)
	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	apiHandler.RegisterRoutes(mux)

	adminCookies := loginOrgUser(t, mux, "dir_admin", "admin12345")
	memberCookies := loginOrgUser(t, mux, "dir_member", "admin12345")

	response := performOrgRequest(mux, http.MethodGet, "/api/v1/directory", "", adminCookies)
	if response.Code != http.StatusOK {
		t.Fatalf("directory status = %d: %s", response.Code, response.Body.String())
	}
	var payload directoryResponse
	_ = json.Unmarshal(response.Body.Bytes(), &payload)

	ids := map[string]string{}
	for _, entry := range payload.Users {
		ids[entry.UserID] = entry.Role
	}
	if ids[adminID] != "org_admin" {
		t.Fatalf("admin role = %q, want org_admin (entries=%#v)", ids[adminID], payload.Users)
	}
	if ids[memberID] != "org_member" {
		t.Fatalf("member role = %q, want org_member", ids[memberID])
	}
	if _, present := ids[inactiveID]; present {
		t.Fatal("inactive user should be excluded from the directory")
	}
	if _, present := ids[foreignID]; present {
		t.Fatal("foreign-organization user should be excluded from the directory")
	}

	// A plain organization member cannot read the directory.
	memberResponse := performOrgRequest(mux, http.MethodGet, "/api/v1/directory", "", memberCookies)
	if memberResponse.Code != http.StatusForbidden {
		t.Fatalf("member directory status = %d, want 403: %s", memberResponse.Code, memberResponse.Body.String())
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
