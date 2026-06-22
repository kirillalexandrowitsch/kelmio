//go:build integration

package projectmembers

import (
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
	"kelmio/backend/internal/projectaccess"
)

func TestProjectMemberLifecycleIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newProjectMemberIntegrationDB(t, ctx)
	handler := NewHandler(db, nil)
	users := seedProjectMemberIntegrationWorkspace(t, ctx, db, true)

	members, err := handler.listProjectMembers(ctx, users.ProjectID)
	if err != nil {
		t.Fatalf("list project members: %v", err)
	}
	if len(members) != 5 {
		t.Fatalf("backfilled project members = %d, want 5", len(members))
	}
	if roleForUser(members, users.Admin.ID) != "lead" {
		t.Fatalf("creator role = %q, want lead", roleForUser(members, users.Admin.ID))
	}

	lead, err := handler.putProjectMember(ctx, users.Admin, users.ProjectID, users.Lead.ID, "lead")
	if err != nil {
		t.Fatalf("promote project lead: %v", err)
	}
	if lead.Role != "lead" {
		t.Fatalf("promoted role = %q, want lead", lead.Role)
	}
	viewer, err := handler.putProjectMember(ctx, users.Admin, users.ProjectID, users.Viewer.ID, "viewer")
	if err != nil {
		t.Fatalf("set project viewer: %v", err)
	}
	if viewer.Role != "viewer" {
		t.Fatalf("viewer role = %q, want viewer", viewer.Role)
	}

	if _, err := handler.putProjectMember(ctx, users.Admin, users.ProjectID, users.CrossWorkspaceID, "contributor"); !errors.Is(err, errWorkspaceMemberNotFound) {
		t.Fatalf("cross-workspace member error = %v, want %v", err, errWorkspaceMemberNotFound)
	}
	if _, err := db.Exec(ctx, `UPDATE users SET is_active = false WHERE id = $1`, users.Inactive.ID); err != nil {
		t.Fatalf("deactivate project member: %v", err)
	}
	members, err = handler.listProjectMembers(ctx, users.ProjectID)
	if err != nil {
		t.Fatalf("list inactive project member: %v", err)
	}
	if memberIsActive(members, users.Inactive.ID) {
		t.Fatal("expected inactive project membership row to remain visible")
	}
	if _, err := handler.putProjectMember(ctx, users.Admin, users.ProjectID, users.Inactive.ID, "viewer"); !errors.Is(err, errWorkspaceMemberNotFound) {
		t.Fatalf("inactive member update error = %v, want %v", err, errWorkspaceMemberNotFound)
	}
	if _, err := projectaccess.Resolve(ctx, db, users.Inactive, users.ProjectID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("inactive effective access error = %v, want %v", err, pgx.ErrNoRows)
	}

	if err := handler.deleteProjectMember(ctx, users.Admin, users.ProjectID, users.Viewer.ID); err != nil {
		t.Fatalf("delete project member: %v", err)
	}
	access, err := projectaccess.Resolve(ctx, db, users.Viewer, users.ProjectID)
	if err != nil {
		t.Fatalf("resolve removed project member access: %v", err)
	}
	if access.CanRead || access.CanManage || access.ProjectRole != "" {
		t.Fatalf("removed project member access = %#v", access)
	}
	if err := handler.deleteProjectMember(ctx, users.Admin, users.ProjectID, users.Viewer.ID); !errors.Is(err, errProjectMemberNotFound) {
		t.Fatalf("repeat delete error = %v, want %v", err, errProjectMemberNotFound)
	}

	access, err = projectaccess.Resolve(ctx, db, users.Admin, users.ProjectID)
	if err != nil {
		t.Fatalf("resolve workspace admin access: %v", err)
	}
	if !access.IsWorkspaceAdmin || !access.CanRead || !access.CanManage {
		t.Fatalf("workspace admin access = %#v", access)
	}
	access, err = projectaccess.Resolve(ctx, db, users.Lead, users.ProjectID)
	if err != nil {
		t.Fatalf("resolve lead access: %v", err)
	}
	if access.ProjectRole != "lead" || !access.CanManage {
		t.Fatalf("lead access = %#v", access)
	}
	access, err = projectaccess.Resolve(ctx, db, users.Contributor, users.ProjectID)
	if err != nil {
		t.Fatalf("resolve contributor access: %v", err)
	}
	if access.ProjectRole != "contributor" || !access.CanRead || access.CanManage {
		t.Fatalf("contributor access = %#v", access)
	}
	var contributorRuleID string
	if err := db.QueryRow(ctx, `
		INSERT INTO automation_rules (
			project_id, name, trigger_type, conditions, actions, position, created_by
		)
		VALUES (
			$1, 'Contributor dependency', 'assignee_changed', '[]'::jsonb,
			jsonb_build_array(jsonb_build_object('type', 'change_assignee', 'user_id', $2::text)),
			100,
			$3
		)
		RETURNING id::text
	`, users.ProjectID, users.Contributor.ID, users.Admin.ID).Scan(&contributorRuleID); err != nil {
		t.Fatalf("insert contributor automation rule: %v", err)
	}
	if err := handler.deleteProjectMember(ctx, users.Admin, users.ProjectID, users.Contributor.ID); err != nil {
		t.Fatalf("delete project member for automation invalidation: %v", err)
	}
	var contributorRuleEnabled bool
	var contributorRuleReason *string
	if err := db.QueryRow(ctx, `
		SELECT is_enabled, disabled_reason
		FROM automation_rules
		WHERE id = $1
	`, contributorRuleID).Scan(&contributorRuleEnabled, &contributorRuleReason); err != nil {
		t.Fatalf("load contributor automation rule: %v", err)
	}
	if contributorRuleEnabled || contributorRuleReason == nil || *contributorRuleReason != "project_access_removed" {
		t.Fatalf("contributor automation rule = enabled:%v reason:%v", contributorRuleEnabled, contributorRuleReason)
	}
	if err := handler.deleteProjectMember(ctx, users.Admin, users.ProjectID, users.Admin.ID); err != nil {
		t.Fatalf("delete workspace admin project row: %v", err)
	}
	access, err = projectaccess.Resolve(ctx, db, users.Admin, users.ProjectID)
	if err != nil {
		t.Fatalf("resolve workspace admin after project row delete: %v", err)
	}
	if !access.IsWorkspaceAdmin || !access.CanManage || access.ProjectRole != "" {
		t.Fatalf("workspace admin access after project row delete = %#v", access)
	}
}

func TestProjectMemberLastLeadProtectionIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newProjectMemberIntegrationDB(t, ctx)
	handler := NewHandler(db, nil)
	users := seedProjectMemberIntegrationWorkspace(t, ctx, db, false)

	if _, err := handler.putProjectMember(ctx, users.Lead, users.ProjectID, users.Lead.ID, "contributor"); !errors.Is(err, errProjectRequiresLead) {
		t.Fatalf("last lead demotion error = %v, want %v", err, errProjectRequiresLead)
	}
	if err := handler.deleteProjectMember(ctx, users.Lead, users.ProjectID, users.Lead.ID); !errors.Is(err, errProjectRequiresLead) {
		t.Fatalf("last lead delete error = %v, want %v", err, errProjectRequiresLead)
	}
	if _, err := handler.putProjectMember(ctx, users.Lead, users.ProjectID, users.Contributor.ID, "lead"); err != nil {
		t.Fatalf("add replacement lead: %v", err)
	}
	if _, err := handler.putProjectMember(ctx, users.Lead, users.ProjectID, users.Lead.ID, "contributor"); err != nil {
		t.Fatalf("demote lead with replacement: %v", err)
	}
}

func TestProjectMemberHTTPPermissionsIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newProjectMemberIntegrationDB(t, ctx)
	users := seedProjectMemberIntegrationWorkspace(t, ctx, db, true)
	handler := NewHandler(db, nil)
	if _, err := handler.putProjectMember(ctx, users.Admin, users.ProjectID, users.Lead.ID, "lead"); err != nil {
		t.Fatalf("promote lead: %v", err)
	}
	if _, err := handler.putProjectMember(ctx, users.Admin, users.ProjectID, users.Viewer.ID, "viewer"); err != nil {
		t.Fatalf("set viewer: %v", err)
	}

	authHandler := auth.NewHandler(db, time.Hour, false, nil, nil)
	apiHandler := NewHandler(db, authHandler)
	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	apiHandler.RegisterRoutes(mux)

	contributorCookies := loginProjectMemberUser(t, mux, "pm_contributor", "member12345")
	response := performProjectMemberRequest(mux, http.MethodGet, "/api/v1/projects/"+users.ProjectID+"/members", "", contributorCookies)
	if response.Code != http.StatusForbidden {
		t.Fatalf("contributor list status = %d, want 403: %s", response.Code, response.Body.String())
	}
	viewerCookies := loginProjectMemberUser(t, mux, "pm_viewer", "member12345")
	response = performProjectMemberRequest(mux, http.MethodGet, "/api/v1/projects/"+users.ProjectID+"/members", "", viewerCookies)
	if response.Code != http.StatusForbidden {
		t.Fatalf("viewer list status = %d, want 403: %s", response.Code, response.Body.String())
	}

	leadCookies := loginProjectMemberUser(t, mux, "pm_lead", "member12345")
	response = performProjectMemberRequest(mux, http.MethodGet, "/api/v1/projects/"+users.ProjectID+"/members", "", leadCookies)
	if response.Code != http.StatusOK {
		t.Fatalf("lead list status = %d, want 200: %s", response.Code, response.Body.String())
	}
	response = performProjectMemberRequest(
		mux,
		http.MethodPut,
		"/api/v1/projects/"+users.ProjectID+"/members/"+users.Contributor.ID,
		`{"role":"viewer"}`,
		leadCookies,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("lead update status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var updated memberResponse
	if err := json.Unmarshal(response.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated project member: %v", err)
	}
	if updated.Role != "viewer" {
		t.Fatalf("updated project role = %q, want viewer", updated.Role)
	}

	adminCookies := loginProjectMemberUser(t, mux, "pm_admin", "admin12345")
	response = performProjectMemberRequest(
		mux,
		http.MethodDelete,
		"/api/v1/projects/"+users.ProjectID+"/members/"+users.Viewer.ID,
		"",
		adminCookies,
	)
	if response.Code != http.StatusNoContent {
		t.Fatalf("admin delete status = %d, want 204: %s", response.Code, response.Body.String())
	}
	response = performProjectMemberRequest(mux, http.MethodGet, "/api/v1/projects/"+users.ProjectID+"/members", "", viewerCookies)
	if response.Code != http.StatusNotFound {
		t.Fatalf("nonmember list status = %d, want 404: %s", response.Code, response.Body.String())
	}
	if _, err := db.Exec(ctx, `UPDATE projects SET archived_at = now() WHERE id = $1`, users.ProjectID); err != nil {
		t.Fatalf("archive project: %v", err)
	}
	response = performProjectMemberRequest(mux, http.MethodGet, "/api/v1/projects/"+users.ProjectID+"/members", "", adminCookies)
	if response.Code != http.StatusNotFound {
		t.Fatalf("archived project status = %d, want 404: %s", response.Code, response.Body.String())
	}
	response = performProjectMemberRequest(mux, http.MethodGet, "/api/v1/projects/6d5257d4-002e-44da-8925-d9108699c504/members", "", adminCookies)
	if response.Code != http.StatusNotFound {
		t.Fatalf("missing project status = %d, want 404: %s", response.Code, response.Body.String())
	}
}

type projectMemberTestUsers struct {
	Admin            auth.CurrentUser
	Lead             auth.CurrentUser
	Contributor      auth.CurrentUser
	Viewer           auth.CurrentUser
	Inactive         auth.CurrentUser
	CrossWorkspaceID string
	ProjectID        string
}

func newProjectMemberIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
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
	schemaName := fmt.Sprintf("project_members_integration_%d", time.Now().UnixNano())
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
		t.Fatalf("connect schema: %v", err)
	}
	t.Cleanup(db.Close)
	if _, err := migrations.Up(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}

func seedProjectMemberIntegrationWorkspace(
	t *testing.T,
	ctx context.Context,
	db *pgxpool.Pool,
	withAdmin bool,
) projectMemberTestUsers {
	t.Helper()
	var workspaceID string
	if err := db.QueryRow(ctx, `INSERT INTO workspaces (name) VALUES ('Project Members') RETURNING id::text`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("member12345"), bcrypt.MinCost)
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin12345"), bcrypt.MinCost)
	insertUser := func(email, username string, hash []byte, active bool) string {
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
	adminID := insertUser("pm-admin@example.com", "pm_admin", adminHash, true)
	leadID := insertUser("pm-lead@example.com", "pm_lead", passwordHash, true)
	contributorID := insertUser("pm-contributor@example.com", "pm_contributor", passwordHash, true)
	viewerID := insertUser("pm-viewer@example.com", "pm_viewer", passwordHash, true)
	inactiveID := insertUser("pm-inactive@example.com", "pm_inactive", passwordHash, true)
	workspaceRole := "member"
	if withAdmin {
		workspaceRole = "admin"
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES
			($1, $2, $7),
			($1, $3, 'member'),
			($1, $4, 'member'),
			($1, $5, 'member'),
			($1, $6, 'member')
	`, workspaceID, adminID, leadID, contributorID, viewerID, inactiveID, workspaceRole); err != nil {
		t.Fatalf("insert workspace members: %v", err)
	}
	creatorID := adminID
	if !withAdmin {
		creatorID = leadID
	}
	var projectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'PM', 'Project Members', $2)
		RETURNING id::text
	`, workspaceID, creatorID).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	var otherWorkspaceID string
	if err := db.QueryRow(ctx, `INSERT INTO workspaces (name) VALUES ('Other Project Members') RETURNING id::text`).Scan(&otherWorkspaceID); err != nil {
		t.Fatalf("insert other workspace: %v", err)
	}
	crossWorkspaceID := insertUser("pm-cross@example.com", "pm_cross", passwordHash, true)
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'member')
	`, otherWorkspaceID, crossWorkspaceID); err != nil {
		t.Fatalf("insert cross workspace member: %v", err)
	}

	return projectMemberTestUsers{
		Admin:            auth.CurrentUser{ID: adminID, WorkspaceID: workspaceID, Role: workspaceRole},
		Lead:             auth.CurrentUser{ID: leadID, WorkspaceID: workspaceID, Role: "member"},
		Contributor:      auth.CurrentUser{ID: contributorID, WorkspaceID: workspaceID, Role: "member"},
		Viewer:           auth.CurrentUser{ID: viewerID, WorkspaceID: workspaceID, Role: "member"},
		Inactive:         auth.CurrentUser{ID: inactiveID, WorkspaceID: workspaceID, Role: "member"},
		CrossWorkspaceID: crossWorkspaceID,
		ProjectID:        projectID,
	}
}

func roleForUser(members []memberResponse, userID string) string {
	for _, member := range members {
		if member.UserID == userID {
			return member.Role
		}
	}
	return ""
}

func memberIsActive(members []memberResponse, userID string) bool {
	for _, member := range members {
		if member.UserID == userID {
			return member.IsActive
		}
	}
	return false
}

func loginProjectMemberUser(t *testing.T, mux http.Handler, username string, password string) []*http.Cookie {
	t.Helper()
	response := performProjectMemberRequest(
		mux,
		http.MethodPost,
		"/api/v1/auth/login",
		fmt.Sprintf(`{"login":%q,"password":%q}`, username, password),
		nil,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("login %s status = %d: %s", username, response.Code, response.Body.String())
	}
	return response.Result().Cookies()
}

func performProjectMemberRequest(mux http.Handler, method string, path string, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
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
