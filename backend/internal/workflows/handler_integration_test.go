//go:build integration

package workflows

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

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/database"
	"team-task-tracker/backend/internal/migrations"
)

func TestWorkflowLifecycleIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newWorkflowIntegrationDB(t, ctx)
	handler := NewHandler(db, nil)
	admin, _, projectID, otherProjectID := seedWorkflowIntegrationWorkspace(t, ctx, db)

	workflow, err := handler.getWorkflow(ctx, admin.WorkspaceID, projectID)
	if err != nil {
		t.Fatalf("get default workflow: %v", err)
	}
	if len(workflow.Statuses) != 5 || len(workflow.Transitions) != 20 {
		t.Fatalf("default workflow = %d statuses/%d transitions, want 5/20", len(workflow.Statuses), len(workflow.Transitions))
	}

	review, err := handler.createWorkflowStatus(ctx, admin.WorkspaceID, projectID, normalizedCreateStatus{
		Key: "review", Name: "Review", Color: "#0ea5e9", Category: "in_progress",
	})
	if err != nil {
		t.Fatalf("create workflow status: %v", err)
	}
	if review.Position != 600 {
		t.Fatalf("created status position = %d, want 600", review.Position)
	}
	if _, err := handler.createWorkflowStatus(ctx, admin.WorkspaceID, projectID, normalizedCreateStatus{
		Key: "review", Name: "Other", Color: "#0ea5e9", Category: "todo",
	}); !errors.Is(err, errStatusKeyExists) {
		t.Fatalf("duplicate key error = %v, want %v", err, errStatusKeyExists)
	}
	if _, err := handler.createWorkflowStatus(ctx, admin.WorkspaceID, projectID, normalizedCreateStatus{
		Key: "other", Name: "review", Color: "#0ea5e9", Category: "todo",
	}); !errors.Is(err, errStatusNameExists) {
		t.Fatalf("duplicate name error = %v, want %v", err, errStatusNameExists)
	}

	updated, err := handler.updateWorkflowStatus(ctx, admin.WorkspaceID, projectID, review.ID, normalizedUpdateStatus{
		Name: "Ready for review", HasName: true, Color: "#14b8a6", HasColor: true,
	})
	if err != nil {
		t.Fatalf("update workflow status: %v", err)
	}
	if updated.Name != "Ready for review" || updated.Color != "#14b8a6" || updated.Key != "review" {
		t.Fatalf("updated workflow status = %#v", updated)
	}

	workflow, err = handler.getWorkflow(ctx, admin.WorkspaceID, projectID)
	if err != nil {
		t.Fatalf("get workflow for reorder: %v", err)
	}
	activeIDs := activeStatusIDs(workflow.Statuses)
	reverseStrings(activeIDs)
	if err := handler.reorderWorkflowStatuses(ctx, admin.WorkspaceID, projectID, activeIDs); err != nil {
		t.Fatalf("reorder workflow statuses: %v", err)
	}
	if err := handler.reorderWorkflowStatuses(ctx, admin.WorkspaceID, projectID, activeIDs[:len(activeIDs)-1]); !errors.Is(err, errStatusOrderMismatch) {
		t.Fatalf("partial order error = %v, want %v", err, errStatusOrderMismatch)
	}

	todoID := workflowStatusID(workflow.Statuses, "todo")
	doneID := workflowStatusID(workflow.Statuses, "done")
	if err := handler.replaceWorkflowTransitions(ctx, admin.WorkspaceID, projectID, []normalizedTransition{{
		FromStatusID: todoID, ToStatusID: review.ID,
	}}); err != nil {
		t.Fatalf("replace workflow transitions: %v", err)
	}
	workflow, err = handler.getWorkflow(ctx, admin.WorkspaceID, projectID)
	if err != nil {
		t.Fatalf("get replaced transitions: %v", err)
	}
	if len(workflow.Transitions) != 1 {
		t.Fatalf("replaced transitions = %#v, want one", workflow.Transitions)
	}

	otherWorkflow, err := handler.getWorkflow(ctx, admin.WorkspaceID, otherProjectID)
	if err != nil {
		t.Fatalf("get other workflow: %v", err)
	}
	if err := handler.replaceWorkflowTransitions(ctx, admin.WorkspaceID, projectID, []normalizedTransition{{
		FromStatusID: todoID, ToStatusID: workflowStatusID(otherWorkflow.Statuses, "done"),
	}}); !errors.Is(err, errInvalidTransitionStatus) {
		t.Fatalf("cross-project transition error = %v, want %v", err, errInvalidTransitionStatus)
	}
	workflow, err = handler.getWorkflow(ctx, admin.WorkspaceID, projectID)
	if err != nil {
		t.Fatalf("get transitions after rejected replacement: %v", err)
	}
	if len(workflow.Transitions) != 1 {
		t.Fatalf("transitions after rejected replacement = %#v, want original graph", workflow.Transitions)
	}
	if err := handler.replaceWorkflowTransitions(ctx, admin.WorkspaceID, projectID, nil); err != nil {
		t.Fatalf("replace workflow transitions with empty graph: %v", err)
	}
	workflow, err = handler.getWorkflow(ctx, admin.WorkspaceID, projectID)
	if err != nil {
		t.Fatalf("get empty transition graph: %v", err)
	}
	if len(workflow.Transitions) != 0 {
		t.Fatalf("empty transition graph = %#v, want none", workflow.Transitions)
	}

	archivedReview, err := handler.archiveWorkflowStatus(ctx, admin, projectID, review.ID, doneID)
	if err != nil {
		t.Fatalf("archive unused custom status: %v", err)
	}
	if archivedReview.ArchivedAt == nil {
		t.Fatal("expected archived custom status")
	}
	if _, err := handler.updateWorkflowStatus(ctx, admin.WorkspaceID, projectID, review.ID, normalizedUpdateStatus{
		Name: "Archived review", HasName: true,
	}); !errors.Is(err, errStatusArchived) {
		t.Fatalf("archived status update error = %v, want %v", err, errStatusArchived)
	}
	if err := handler.replaceWorkflowTransitions(ctx, admin.WorkspaceID, projectID, []normalizedTransition{{
		FromStatusID: todoID, ToStatusID: review.ID,
	}}); !errors.Is(err, errInvalidTransitionStatus) {
		t.Fatalf("archived transition error = %v, want %v", err, errInvalidTransitionStatus)
	}

	customReplacement, err := handler.createWorkflowStatus(ctx, admin.WorkspaceID, projectID, normalizedCreateStatus{
		Key: "verify", Name: "Verify", Color: "#8b5cf6", Category: "in_progress",
	})
	if err != nil {
		t.Fatalf("create custom replacement: %v", err)
	}
	issueID := insertWorkflowTestIssue(t, ctx, db, projectID, admin.ID, "todo", 1)
	if _, err := handler.archiveWorkflowStatus(ctx, admin, projectID, todoID, customReplacement.ID); !errors.Is(err, errStatusNotAssignable) {
		t.Fatalf("custom replacement error = %v, want %v", err, errStatusNotAssignable)
	}
	expectIssueStatus(t, ctx, db, issueID, "todo")

	blockedID := workflowStatusID(workflow.Statuses, "blocked")
	blockedIssueID := insertWorkflowTestIssue(t, ctx, db, projectID, admin.ID, "blocked", 2)
	if _, err := handler.archiveWorkflowStatus(ctx, admin, projectID, blockedID, doneID); err != nil {
		t.Fatalf("archive used legacy status: %v", err)
	}
	expectIssueStatus(t, ctx, db, blockedIssueID, "done")
	var activityCount int
	if err := db.QueryRow(ctx, `
		SELECT count(*)::int
		FROM activity_log
		WHERE entity_type = 'issue'
			AND entity_id = $1
			AND action = 'status_changed'
			AND payload->>'from_status' = 'blocked'
			AND payload->>'to_status' = 'done'
	`, blockedIssueID).Scan(&activityCount); err != nil {
		t.Fatalf("count replacement activity: %v", err)
	}
	if activityCount != 1 {
		t.Fatalf("replacement activity count = %d, want 1", activityCount)
	}

	if _, err := handler.archiveWorkflowStatus(ctx, admin, projectID, doneID, todoID); !errors.Is(err, errRequiresDoneStatus) {
		t.Fatalf("last done archive error = %v, want %v", err, errRequiresDoneStatus)
	}
	if _, err := handler.updateWorkflowStatus(ctx, admin.WorkspaceID, projectID, doneID, normalizedUpdateStatus{
		Category: "in_progress", HasCategory: true,
	}); !errors.Is(err, errRequiresDoneStatus) {
		t.Fatalf("last done update error = %v, want %v", err, errRequiresDoneStatus)
	}

	if _, err := handler.getWorkflow(ctx, "6d5257d4-002e-44da-8925-d9108699c504", projectID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("workspace isolation error = %v, want %v", err, pgx.ErrNoRows)
	}
	if _, err := db.Exec(ctx, `UPDATE projects SET archived_at = now() WHERE id = $1`, otherProjectID); err != nil {
		t.Fatalf("archive other project: %v", err)
	}
	if _, err := handler.getWorkflow(ctx, admin.WorkspaceID, otherProjectID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("archived project error = %v, want %v", err, pgx.ErrNoRows)
	}
}

func TestWorkflowHTTPPermissionsIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newWorkflowIntegrationDB(t, ctx)
	_, _, projectID, _ := seedWorkflowIntegrationWorkspace(t, ctx, db)
	authHandler := auth.NewHandler(db, time.Hour, false, nil, nil)
	workflowHandler := NewHandler(db, authHandler)
	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	workflowHandler.RegisterRoutes(mux)

	memberCookies := loginWorkflowUser(t, mux, "workflow_member", "member12345")
	response := performWorkflowRequest(mux, http.MethodGet, "/api/v1/projects/"+projectID+"/workflow", "", memberCookies)
	if response.Code != http.StatusOK {
		t.Fatalf("member workflow read status = %d, want 200: %s", response.Code, response.Body.String())
	}
	response = performWorkflowRequest(
		mux,
		http.MethodPost,
		"/api/v1/projects/"+projectID+"/workflow/statuses",
		`{"key":"member_status","name":"Member status","color":"#123456","category":"todo"}`,
		memberCookies,
	)
	if response.Code != http.StatusForbidden {
		t.Fatalf("member workflow mutation status = %d, want 403: %s", response.Code, response.Body.String())
	}

	adminCookies := loginWorkflowUser(t, mux, "workflow_admin", "admin12345")
	response = performWorkflowRequest(
		mux,
		http.MethodPost,
		"/api/v1/projects/"+projectID+"/workflow/statuses",
		`{"key":"admin_status","name":"Admin status","color":"#123456","category":"todo"}`,
		adminCookies,
	)
	if response.Code != http.StatusCreated {
		t.Fatalf("admin workflow mutation status = %d, want 201: %s", response.Code, response.Body.String())
	}
	var created statusResponse
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created workflow status: %v", err)
	}

	response = performWorkflowRequest(
		mux,
		http.MethodPatch,
		"/api/v1/projects/"+projectID+"/workflow/statuses/"+created.ID,
		`{"name":"Updated admin status","color":"#abcdef","category":"in_progress"}`,
		adminCookies,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("admin workflow update status = %d, want 200: %s", response.Code, response.Body.String())
	}
	response = performWorkflowRequest(
		mux,
		http.MethodPatch,
		"/api/v1/projects/"+projectID+"/workflow/statuses/"+created.ID,
		`{}`,
		adminCookies,
	)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("empty workflow update status = %d, want 400: %s", response.Code, response.Body.String())
	}

	response = performWorkflowRequest(mux, http.MethodGet, "/api/v1/projects/"+projectID+"/workflow", "", adminCookies)
	if response.Code != http.StatusOK {
		t.Fatalf("admin workflow read status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var workflow workflowResponse
	if err := json.Unmarshal(response.Body.Bytes(), &workflow); err != nil {
		t.Fatalf("decode workflow response: %v", err)
	}
	orderPayload, err := json.Marshal(statusOrderRequest{StatusIDs: activeStatusIDs(workflow.Statuses)})
	if err != nil {
		t.Fatalf("encode workflow order: %v", err)
	}
	response = performWorkflowRequest(
		mux,
		http.MethodPut,
		"/api/v1/projects/"+projectID+"/workflow/statuses/order",
		string(orderPayload),
		adminCookies,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("workflow reorder status = %d, want 200: %s", response.Code, response.Body.String())
	}

	todoID := workflowStatusID(workflow.Statuses, "todo")
	response = performWorkflowRequest(
		mux,
		http.MethodPut,
		"/api/v1/projects/"+projectID+"/workflow/transitions",
		fmt.Sprintf(`{"transitions":[{"from_status_id":%q,"to_status_id":%q}]}`, todoID, created.ID),
		adminCookies,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("replace transitions status = %d, want 200: %s", response.Code, response.Body.String())
	}
	response = performWorkflowRequest(
		mux,
		http.MethodPost,
		"/api/v1/projects/"+projectID+"/workflow/statuses/"+created.ID+"/archive",
		fmt.Sprintf(`{"replacement_status_id":%q}`, todoID),
		adminCookies,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("archive workflow status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var archived statusResponse
	if err := json.Unmarshal(response.Body.Bytes(), &archived); err != nil {
		t.Fatalf("decode archived workflow status: %v", err)
	}
	if archived.ArchivedAt == nil {
		t.Fatal("expected API archive response to be archived")
	}
}

func newWorkflowIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://team_task_tracker:team_task_tracker@localhost:15432/team_task_tracker?sslmode=disable"
	}
	adminDB, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	t.Cleanup(adminDB.Close)
	schemaName := fmt.Sprintf("workflow_api_integration_%d", time.Now().UnixNano())
	quotedSchemaName := pgx.Identifier{schemaName}.Sanitize()
	if _, err := adminDB.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`); err != nil {
		t.Fatalf("ensure pgcrypto: %v", err)
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
		t.Fatalf("connect integration schema: %v", err)
	}
	t.Cleanup(db.Close)
	if _, err := migrations.Up(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return db
}

func seedWorkflowIntegrationWorkspace(
	t *testing.T,
	ctx context.Context,
	db *pgxpool.Pool,
) (auth.CurrentUser, auth.CurrentUser, string, string) {
	t.Helper()
	var workspaceID string
	if err := db.QueryRow(ctx, `INSERT INTO workspaces (name) VALUES ('Workflow API') RETURNING id::text`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin12345"), bcrypt.MinCost)
	memberHash, _ := bcrypt.GenerateFromPassword([]byte("member12345"), bcrypt.MinCost)
	var adminID string
	var memberID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ('workflow-admin@example.com', 'workflow_admin', $1, 'Workflow Admin')
		RETURNING id::text
	`, string(adminHash)).Scan(&adminID); err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ('workflow-member@example.com', 'workflow_member', $1, 'Workflow Member')
		RETURNING id::text
	`, string(memberHash)).Scan(&memberID); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin'), ($1, $3, 'member')
	`, workspaceID, adminID, memberID); err != nil {
		t.Fatalf("insert workspace members: %v", err)
	}
	var projectID string
	var otherProjectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'WF', 'Workflow Project', $2)
		RETURNING id::text
	`, workspaceID, adminID).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'WF2', 'Other Workflow Project', $2)
		RETURNING id::text
	`, workspaceID, adminID).Scan(&otherProjectID); err != nil {
		t.Fatalf("insert other project: %v", err)
	}
	return auth.CurrentUser{
			ID: adminID, WorkspaceID: workspaceID, Role: "admin",
		}, auth.CurrentUser{
			ID: memberID, WorkspaceID: workspaceID, Role: "member",
		}, projectID, otherProjectID
}

func insertWorkflowTestIssue(t *testing.T, ctx context.Context, db *pgxpool.Pool, projectID string, reporterID string, status string, number int) string {
	t.Helper()
	var issueID string
	if err := db.QueryRow(ctx, `
		INSERT INTO issues (project_id, number, issue_key, title, issue_type, status, priority, reporter_id)
		VALUES ($1, $2, $3, $4, 'task', $5, 'medium', $6)
		RETURNING id::text
	`, projectID, number, fmt.Sprintf("WF-%d", number), "Workflow issue", status, reporterID).Scan(&issueID); err != nil {
		t.Fatalf("insert workflow issue: %v", err)
	}
	return issueID
}

func expectIssueStatus(t *testing.T, ctx context.Context, db *pgxpool.Pool, issueID string, want string) {
	t.Helper()
	var status string
	var workflowKey string
	if err := db.QueryRow(ctx, `
		SELECT issue.status, workflow_status.key
		FROM issues issue
		JOIN project_workflow_statuses workflow_status ON workflow_status.id = issue.workflow_status_id
		WHERE issue.id = $1
	`, issueID).Scan(&status, &workflowKey); err != nil {
		t.Fatalf("load issue status: %v", err)
	}
	if status != want || workflowKey != want {
		t.Fatalf("issue status/workflow key = %q/%q, want %q/%q", status, workflowKey, want, want)
	}
}

func workflowStatusID(statuses []statusResponse, key string) string {
	for _, status := range statuses {
		if status.Key == key {
			return status.ID
		}
	}
	return ""
}

func activeStatusIDs(statuses []statusResponse) []string {
	ids := make([]string, 0, len(statuses))
	for _, status := range statuses {
		if status.ArchivedAt == nil {
			ids = append(ids, status.ID)
		}
	}
	return ids
}

func reverseStrings(values []string) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func loginWorkflowUser(t *testing.T, mux http.Handler, username string, password string) []*http.Cookie {
	t.Helper()
	response := performWorkflowRequest(
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

func performWorkflowRequest(mux http.Handler, method string, path string, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
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
