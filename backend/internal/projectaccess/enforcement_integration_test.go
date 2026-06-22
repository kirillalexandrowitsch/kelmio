//go:build integration

package projectaccess_test

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

	"kelmio/backend/internal/auth"
	"kelmio/backend/internal/database"
	"kelmio/backend/internal/issues"
	"kelmio/backend/internal/migrations"
	"kelmio/backend/internal/notifications"
	"kelmio/backend/internal/projectmembers"
	"kelmio/backend/internal/projects"
	"kelmio/backend/internal/sprints"
	"kelmio/backend/internal/workflows"
)

func TestProjectPermissionEnforcementIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newAccessIntegrationDB(t, ctx)
	seed := seedAccessIntegrationWorkspace(t, ctx, db)
	mux := accessIntegrationMux(db)

	admin := loginAccessUser(t, mux, "access_admin")
	lead := loginAccessUser(t, mux, "access_lead")
	contributor := loginAccessUser(t, mux, "access_contributor")
	viewer := loginAccessUser(t, mux, "access_viewer")
	outsider := loginAccessUser(t, mux, "access_outsider")

	assertAccessStatus(t, mux, admin, http.MethodGet, "/api/v1/projects/"+seed.projectA, "", http.StatusOK)
	viewerProject := assertAccessStatus(t, mux, viewer, http.MethodGet, "/api/v1/projects/"+seed.projectA, "", http.StatusOK)
	if !strings.Contains(viewerProject.Body.String(), `"project_role":"viewer"`) ||
		!strings.Contains(viewerProject.Body.String(), `"can_write":false`) ||
		!strings.Contains(viewerProject.Body.String(), `"can_manage":false`) {
		t.Fatalf("viewer project metadata = %s", viewerProject.Body.String())
	}
	outsiderProjects := assertAccessStatus(t, mux, outsider, http.MethodGet, "/api/v1/projects", "", http.StatusOK)
	if strings.Contains(outsiderProjects.Body.String(), seed.projectA) {
		t.Fatalf("outsider project list leaked project A: %s", outsiderProjects.Body.String())
	}
	assertAccessStatus(t, mux, outsider, http.MethodGet, "/api/v1/projects/"+seed.projectA, "", http.StatusNotFound)
	assertAccessStatus(t, mux, lead, http.MethodPatch, "/api/v1/projects/"+seed.projectA, `{"name":"Lead edit","description":""}`, http.StatusForbidden)

	assertAccessStatus(t, mux, viewer, http.MethodGet, "/api/v1/projects/"+seed.projectA+"/workflow", "", http.StatusOK)
	assertAccessStatus(t, mux, outsider, http.MethodGet, "/api/v1/projects/"+seed.projectA+"/workflow", "", http.StatusNotFound)
	assertAccessStatus(t, mux, viewer, http.MethodPost, "/api/v1/projects/"+seed.projectA+"/workflow/statuses", `{"key":"viewer_status","name":"Viewer status","color":"#123456","category":"todo"}`, http.StatusForbidden)
	assertAccessStatus(t, mux, lead, http.MethodPost, "/api/v1/projects/"+seed.projectA+"/workflow/statuses", `{"key":"lead_status","name":"Lead status","color":"#123456","category":"todo"}`, http.StatusCreated)

	assertAccessStatus(t, mux, viewer, http.MethodGet, "/api/v1/issues/"+seed.issueA, "", http.StatusOK)
	leadIssue := assertAccessStatus(t, mux, lead, http.MethodGet, "/api/v1/issues/"+seed.issueA, "", http.StatusOK)
	if strings.Contains(leadIssue.Body.String(), seed.issueB) {
		t.Fatalf("issue detail leaked inaccessible parent: %s", leadIssue.Body.String())
	}
	leadIssues := assertAccessStatus(t, mux, lead, http.MethodGet, "/api/v1/issues", "", http.StatusOK)
	if strings.Contains(leadIssues.Body.String(), seed.issueB) {
		t.Fatalf("issue list leaked inaccessible parent: %s", leadIssues.Body.String())
	}
	assertAccessStatus(t, mux, outsider, http.MethodGet, "/api/v1/issues/"+seed.issueA, "", http.StatusNotFound)
	outsiderIssues := assertAccessStatus(t, mux, outsider, http.MethodGet, "/api/v1/issues", "", http.StatusOK)
	if strings.Contains(outsiderIssues.Body.String(), seed.issueA) {
		t.Fatalf("outsider issue list leaked issue A: %s", outsiderIssues.Body.String())
	}
	assertAccessStatus(t, mux, viewer, http.MethodPost, "/api/v1/issues", fmt.Sprintf(`{"project_id":%q,"title":"Viewer issue","issue_type":"task","status":"todo","priority":"medium"}`, seed.projectA), http.StatusForbidden)
	assertAccessStatus(t, mux, contributor, http.MethodPost, "/api/v1/issues", fmt.Sprintf(`{"project_id":%q,"title":"Contributor issue","issue_type":"task","status":"todo","priority":"medium"}`, seed.projectA), http.StatusCreated)
	assertAccessStatus(t, mux, contributor, http.MethodPost, "/api/v1/issues/"+seed.issueA+"/assign", fmt.Sprintf(`{"assignee_id":%q}`, seed.outsiderID), http.StatusBadRequest)
	leadLinks := assertAccessStatus(t, mux, lead, http.MethodGet, "/api/v1/issues/"+seed.issueA+"/links", "", http.StatusOK)
	if strings.Contains(leadLinks.Body.String(), seed.issueB) {
		t.Fatalf("issue links leaked inaccessible target: %s", leadLinks.Body.String())
	}
	assertAccessStatus(t, mux, contributor, http.MethodPost, "/api/v1/issues/"+seed.issueA+"/links", fmt.Sprintf(`{"target_issue_id":%q,"link_type":"relates"}`, seed.issueB), http.StatusNotFound)
	assertAccessStatus(t, mux, viewer, http.MethodPost, "/api/v1/issues/"+seed.issueA+"/comments", `{"body":"Viewer comment"}`, http.StatusForbidden)
	assertAccessStatus(t, mux, contributor, http.MethodPost, "/api/v1/issues/"+seed.issueA+"/comments", `{"body":"Contributor comment"}`, http.StatusCreated)

	assertAccessStatus(t, mux, viewer, http.MethodGet, "/api/v1/sprints/"+seed.sprintA, "", http.StatusOK)
	assertAccessStatus(t, mux, outsider, http.MethodGet, "/api/v1/sprints/"+seed.sprintA, "", http.StatusNotFound)
	outsiderSprints := assertAccessStatus(t, mux, outsider, http.MethodGet, "/api/v1/sprints", "", http.StatusOK)
	if strings.Contains(outsiderSprints.Body.String(), seed.sprintA) {
		t.Fatalf("outsider sprint list leaked sprint A: %s", outsiderSprints.Body.String())
	}
	assertAccessStatus(t, mux, viewer, http.MethodPatch, "/api/v1/sprints/"+seed.sprintA, `{"name":"Viewer sprint","goal":"","start_date":"","end_date":""}`, http.StatusForbidden)
	assertAccessStatus(t, mux, contributor, http.MethodPatch, "/api/v1/sprints/"+seed.sprintA, `{"name":"Contributor sprint","goal":"","start_date":"","end_date":""}`, http.StatusOK)
}

type accessSeed struct {
	projectA   string
	projectB   string
	issueA     string
	issueB     string
	sprintA    string
	outsiderID string
}

func accessIntegrationMux(db *pgxpool.Pool) http.Handler {
	authHandler := auth.NewHandler(db, time.Hour, false, nil, nil)
	notificationService := notifications.NewService()
	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	projects.NewHandler(db, authHandler).RegisterRoutes(mux)
	projectmembers.NewHandler(db, authHandler).RegisterRoutes(mux)
	workflows.NewHandler(db, authHandler).RegisterRoutes(mux)
	issues.NewHandler(db, authHandler, notificationService).RegisterRoutes(mux)
	sprints.NewHandler(db, authHandler, notificationService).RegisterRoutes(mux)
	return mux
}

func seedAccessIntegrationWorkspace(t *testing.T, ctx context.Context, db *pgxpool.Pool) accessSeed {
	t.Helper()
	var workspaceID string
	if err := db.QueryRow(ctx, `INSERT INTO workspaces (name) VALUES ('Access Integration') RETURNING id::text`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("access12345"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	insertUser := func(username string, role string) string {
		var userID string
		if err := db.QueryRow(ctx, `
			INSERT INTO users (email, username, password_hash, display_name)
			VALUES ($1, $2, $3, $2)
			RETURNING id::text
		`, username+"@example.com", username, string(passwordHash)).Scan(&userID); err != nil {
			t.Fatalf("insert user %s: %v", username, err)
		}
		if _, err := db.Exec(ctx, `
			INSERT INTO workspace_members (workspace_id, user_id, role)
			VALUES ($1, $2, $3)
		`, workspaceID, userID, role); err != nil {
			t.Fatalf("insert workspace member %s: %v", username, err)
		}
		return userID
	}
	adminID := insertUser("access_admin", "admin")
	leadID := insertUser("access_lead", "member")
	contributorID := insertUser("access_contributor", "member")
	viewerID := insertUser("access_viewer", "member")
	outsiderID := insertUser("access_outsider", "member")

	insertProject := func(key string, creatorID string) string {
		var projectID string
		if err := db.QueryRow(ctx, `
			INSERT INTO projects (workspace_id, key, name, created_by)
			VALUES ($1, $2, $2, $3)
			RETURNING id::text
		`, workspaceID, key, creatorID).Scan(&projectID); err != nil {
			t.Fatalf("insert project %s: %v", key, err)
		}
		return projectID
	}
	projectA := insertProject("ACCA", leadID)
	projectB := insertProject("ACCB", adminID)
	if _, err := db.Exec(ctx, `UPDATE project_members SET role = 'viewer' WHERE project_id = $1 AND user_id = $2`, projectA, viewerID); err != nil {
		t.Fatalf("configure viewer membership: %v", err)
	}
	if _, err := db.Exec(ctx, `DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`, projectA, outsiderID); err != nil {
		t.Fatalf("remove outsider membership: %v", err)
	}
	if _, err := db.Exec(ctx, `DELETE FROM project_members WHERE project_id = $1 AND user_id IN ($2, $3, $4)`, projectB, outsiderID, leadID, contributorID); err != nil {
		t.Fatalf("configure inaccessible project memberships: %v", err)
	}

	insertIssue := func(projectID string, key string, reporterID string) string {
		var issueID string
		if err := db.QueryRow(ctx, `
			INSERT INTO issues (project_id, number, issue_key, title, issue_type, status, priority, reporter_id)
			VALUES ($1, 1, $2, $2, 'task', 'todo', 'medium', $3)
			RETURNING id::text
		`, projectID, key, reporterID).Scan(&issueID); err != nil {
			t.Fatalf("insert issue %s: %v", key, err)
		}
		return issueID
	}
	issueA := insertIssue(projectA, "ACCA-1", leadID)
	issueB := insertIssue(projectB, "ACCB-1", adminID)
	if _, err := db.Exec(ctx, `UPDATE issues SET parent_issue_id = $1 WHERE id = $2`, issueB, issueA); err != nil {
		t.Fatalf("set cross-project parent: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO issue_links (source_issue_id, target_issue_id, link_type, created_by)
		VALUES ($1, $2, 'relates', $3)
	`, issueA, issueB, adminID); err != nil {
		t.Fatalf("insert cross-project link: %v", err)
	}
	var sprintA string
	if err := db.QueryRow(ctx, `
		INSERT INTO sprints (workspace_id, project_id, name, created_by)
		VALUES ($1, $2, 'Access Sprint', $3)
		RETURNING id::text
	`, workspaceID, projectA, leadID).Scan(&sprintA); err != nil {
		t.Fatalf("insert sprint: %v", err)
	}
	return accessSeed{
		projectA: projectA, projectB: projectB, issueA: issueA, issueB: issueB,
		sprintA: sprintA, outsiderID: outsiderID,
	}
}

func newAccessIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
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
	schemaName := fmt.Sprintf("access_integration_%d", time.Now().UnixNano())
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

func loginAccessUser(t *testing.T, mux http.Handler, username string) []*http.Cookie {
	t.Helper()
	response := performAccessRequest(mux, nil, http.MethodPost, "/api/v1/auth/login", fmt.Sprintf(`{"login":%q,"password":"access12345"}`, username))
	if response.Code != http.StatusOK {
		t.Fatalf("login %s status = %d: %s", username, response.Code, response.Body.String())
	}
	return response.Result().Cookies()
}

func assertAccessStatus(
	t *testing.T,
	mux http.Handler,
	cookies []*http.Cookie,
	method string,
	path string,
	body string,
	want int,
) *httptest.ResponseRecorder {
	t.Helper()
	response := performAccessRequest(mux, cookies, method, path, body)
	if response.Code != want {
		t.Fatalf("%s %s status = %d, want %d: %s", method, path, response.Code, want, response.Body.String())
	}
	return response
}

func performAccessRequest(mux http.Handler, cookies []*http.Cookie, method string, path string, body string) *httptest.ResponseRecorder {
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
