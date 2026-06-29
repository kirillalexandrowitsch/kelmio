//go:build integration

package groups

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

func TestGroupsAPILifecycleAndAuthorization(t *testing.T) {
	databaseURL := getDatabaseURL()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adminDB, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	defer adminDB.Close()
	schemaName := fmt.Sprintf("groups_api_%d", time.Now().UnixNano())
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
	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name, organization_id, slug, status)
		VALUES ('Groups WS', $1, 'groups-ws', 'active')
		RETURNING id::text
	`, orgID).Scan(&workspaceID); err != nil {
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
	adminID := insertUser("grp-admin@example.com", "grp_admin", true)
	memberID := insertUser("grp-member@example.com", "grp_member", true)
	outsiderID := insertUser("grp-outsider@example.com", "grp_outsider", true)
	inactiveID := insertUser("grp-inactive@example.com", "grp_inactive", false)
	strangerID := insertUser("grp-stranger@example.com", "grp_stranger", true)

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
			($1, $4, 'org_member'),
			($1, $5, 'org_member')
	`, orgID, adminID, memberID, outsiderID, inactiveID); err != nil {
		t.Fatalf("insert organization members: %v", err)
	}

	authHandler := auth.NewHandler(db, time.Hour, false, nil, nil)
	apiHandler := NewHandler(db, authHandler)
	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	apiHandler.RegisterRoutes(mux)

	adminCookies := loginUser(t, mux, "grp_admin", "admin12345")
	memberCookies := loginUser(t, mux, "grp_member", "admin12345")

	// Create a group.
	createResponse := performRequest(mux, http.MethodPost, "/api/v1/groups", `{"name":"Engineers","description":"Builders"}`, adminCookies)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create status = %d: %s", createResponse.Code, createResponse.Body.String())
	}
	var group groupResponse
	_ = json.Unmarshal(createResponse.Body.Bytes(), &group)
	if group.Name != "Engineers" || group.MemberCount != 0 {
		t.Fatalf("created group = %#v, want Engineers with 0 members", group)
	}

	// Duplicate name is rejected.
	dup := performRequest(mux, http.MethodPost, "/api/v1/groups", `{"name":"Engineers"}`, adminCookies)
	if dup.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409: %s", dup.Code, dup.Body.String())
	}

	// Rename the group.
	renameResponse := performRequest(mux, http.MethodPatch, "/api/v1/groups/"+group.ID, `{"name":"Engineering"}`, adminCookies)
	if renameResponse.Code != http.StatusOK {
		t.Fatalf("rename status = %d: %s", renameResponse.Code, renameResponse.Body.String())
	}

	// Add an organization member.
	addResponse := performRequest(mux, http.MethodPost, "/api/v1/groups/"+group.ID+"/members", fmt.Sprintf(`{"user_id":%q}`, outsiderID), adminCookies)
	if addResponse.Code != http.StatusOK {
		t.Fatalf("add member status = %d: %s", addResponse.Code, addResponse.Body.String())
	}

	// Adding a non-organization member is a not-found; an inactive member is a bad request.
	stranger := performRequest(mux, http.MethodPost, "/api/v1/groups/"+group.ID+"/members", fmt.Sprintf(`{"user_id":%q}`, strangerID), adminCookies)
	if stranger.Code != http.StatusNotFound {
		t.Fatalf("stranger add status = %d, want 404: %s", stranger.Code, stranger.Body.String())
	}
	inactive := performRequest(mux, http.MethodPost, "/api/v1/groups/"+group.ID+"/members", fmt.Sprintf(`{"user_id":%q}`, inactiveID), adminCookies)
	if inactive.Code != http.StatusBadRequest {
		t.Fatalf("inactive add status = %d, want 400: %s", inactive.Code, inactive.Body.String())
	}

	// The group now reports one member.
	listResponse := performRequest(mux, http.MethodGet, "/api/v1/groups", "", adminCookies)
	var groups listGroupsResponse
	_ = json.Unmarshal(listResponse.Body.Bytes(), &groups)
	if len(groups.Groups) != 1 || groups.Groups[0].Name != "Engineering" || groups.Groups[0].MemberCount != 1 {
		t.Fatalf("groups = %#v, want one Engineering group with 1 member", groups.Groups)
	}

	// Remove the member (idempotency: second removal is a not-found).
	removeResponse := performRequest(mux, http.MethodDelete, "/api/v1/groups/"+group.ID+"/members/"+outsiderID, "", adminCookies)
	if removeResponse.Code != http.StatusNoContent {
		t.Fatalf("remove member status = %d, want 204: %s", removeResponse.Code, removeResponse.Body.String())
	}
	removeAgain := performRequest(mux, http.MethodDelete, "/api/v1/groups/"+group.ID+"/members/"+outsiderID, "", adminCookies)
	if removeAgain.Code != http.StatusNotFound {
		t.Fatalf("second remove status = %d, want 404: %s", removeAgain.Code, removeAgain.Body.String())
	}

	// A role assignment referencing the group must be cleaned up when the group
	// is deleted, since role_assignments is polymorphic and cannot cascade.
	if _, err := db.Exec(ctx, `
		INSERT INTO role_assignments (scope, scope_id, subject_type, subject_id, role)
		VALUES ('workspace', $1, 'group', $2, 'member')
	`, workspaceID, group.ID); err != nil {
		t.Fatalf("insert group role assignment: %v", err)
	}

	// Delete the group.
	deleteResponse := performRequest(mux, http.MethodDelete, "/api/v1/groups/"+group.ID, "", adminCookies)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204: %s", deleteResponse.Code, deleteResponse.Body.String())
	}
	var orphanCount int
	if err := db.QueryRow(ctx, `
		SELECT count(*) FROM role_assignments WHERE subject_type = 'group' AND subject_id = $1
	`, group.ID).Scan(&orphanCount); err != nil {
		t.Fatalf("count role assignments: %v", err)
	}
	if orphanCount != 0 {
		t.Fatalf("group role assignments after delete = %d, want 0", orphanCount)
	}
	afterDelete := performRequest(mux, http.MethodGet, "/api/v1/groups", "", adminCookies)
	var afterGroups listGroupsResponse
	_ = json.Unmarshal(afterDelete.Body.Bytes(), &afterGroups)
	if len(afterGroups.Groups) != 0 {
		t.Fatalf("groups after delete = %#v, want empty", afterGroups.Groups)
	}

	// Plain organization members cannot manage groups.
	memberList := performRequest(mux, http.MethodGet, "/api/v1/groups", "", memberCookies)
	if memberList.Code != http.StatusForbidden {
		t.Fatalf("member list status = %d, want 403: %s", memberList.Code, memberList.Body.String())
	}
	memberCreate := performRequest(mux, http.MethodPost, "/api/v1/groups", `{"name":"Sneaky"}`, memberCookies)
	if memberCreate.Code != http.StatusForbidden {
		t.Fatalf("member create status = %d, want 403: %s", memberCreate.Code, memberCreate.Body.String())
	}
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
