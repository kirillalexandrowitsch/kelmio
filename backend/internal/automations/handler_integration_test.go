//go:build integration

package automations

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
	"team-task-tracker/backend/internal/projectaccess"
)

func TestAutomationRuleLifecycleIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := newAutomationIntegrationDB(t, ctx)
	seed := seedAutomationIntegration(t, ctx, db)
	handler := NewHandler(db, nil)

	first, err := handler.createRule(ctx, seed.Admin, seed.ProjectID, mustNormalizeCreate(t, createRuleRequest{
		Name:        "Route bugs",
		TriggerType: "issue_created",
		Conditions: json.RawMessage(fmt.Sprintf(`[
			{"type":"issue_type","value":"bug"},
			{"type":"workflow_status","workflow_status_id":%q},
			{"type":"label","label_id":%q}
		]`, seed.StatusID, seed.LabelID)),
		Actions: json.RawMessage(fmt.Sprintf(`[
			{"type":"change_assignee","user_id":%q},
			{"type":"change_priority","value":"critical"}
		]`, seed.Contributor.ID)),
	}))
	if err != nil {
		t.Fatalf("create automation rule: %v", err)
	}
	if first.Position != 100 || !first.IsEnabled || first.DisabledReason != nil {
		t.Fatalf("created rule = %#v", first)
	}
	second, err := handler.createRule(ctx, seed.Lead, seed.ProjectID, mustNormalizeCreate(t, createRuleRequest{
		Name: "Label high priority", TriggerType: "priority_changed",
		Conditions: json.RawMessage(`[]`),
		Actions:    json.RawMessage(fmt.Sprintf(`[{"type":"add_label","label_id":%q}]`, seed.LabelID)),
	}))
	if err != nil {
		t.Fatalf("lead create automation rule: %v", err)
	}
	if second.Position != 200 {
		t.Fatalf("second position = %d, want 200", second.Position)
	}

	rules, err := handler.listRules(ctx, seed.ProjectID)
	if err != nil || len(rules) != 2 {
		t.Fatalf("list rules = %#v, %v", rules, err)
	}
	if err := handler.reorderRules(ctx, seed.Admin, seed.ProjectID, []string{second.ID, first.ID}); err != nil {
		t.Fatalf("reorder rules: %v", err)
	}
	if err := handler.reorderRules(ctx, seed.Admin, seed.ProjectID, []string{first.ID}); !errors.Is(err, errRuleOrderMismatch) {
		t.Fatalf("partial reorder error = %v, want order mismatch", err)
	}

	disabled := false
	updated, err := handler.updateRule(ctx, seed.Admin, seed.ProjectID, first.ID, normalizedUpdateRule{
		HasName: true, Name: "Updated route bugs", HasIsEnabled: true, IsEnabled: disabled,
	})
	if err != nil {
		t.Fatalf("disable automation rule: %v", err)
	}
	if updated.IsEnabled || updated.Name != "Updated route bugs" || updated.DisabledReason != nil {
		t.Fatalf("updated rule = %#v", updated)
	}
	enabled := true
	updated, err = handler.updateRule(ctx, seed.Admin, seed.ProjectID, first.ID, normalizedUpdateRule{
		HasIsEnabled: true, IsEnabled: enabled,
	})
	if err != nil || !updated.IsEnabled {
		t.Fatalf("re-enable automation rule = %#v, %v", updated, err)
	}

	if _, err := handler.createRule(ctx, seed.Contributor, seed.ProjectID, mustNormalizeCreate(t, validCreateRuleRequest())); !errors.Is(err, projectaccess.ErrForbidden) {
		t.Fatalf("contributor create error = %v, want forbidden", err)
	}
	if _, err := handler.createRule(ctx, seed.Outsider, seed.ProjectID, mustNormalizeCreate(t, validCreateRuleRequest())); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("outsider create error = %v, want no rows", err)
	}
	invalid := mustNormalizeCreate(t, createRuleRequest{
		Name: "Invalid status", TriggerType: "status_changed", Conditions: json.RawMessage(`[]`),
		Actions: json.RawMessage(fmt.Sprintf(`[{"type":"change_workflow_status","workflow_status_id":%q}]`, seed.OtherStatusID)),
	})
	if _, err := handler.createRule(ctx, seed.Admin, seed.ProjectID, invalid); !errors.Is(err, errInvalidDependency) {
		t.Fatalf("cross-project dependency error = %v, want invalid dependency", err)
	}
	if err := handler.deleteRule(ctx, seed.Admin, seed.ProjectID, second.ID); err != nil {
		t.Fatalf("delete rule: %v", err)
	}
	if err := handler.deleteRule(ctx, seed.Admin, seed.ProjectID, second.ID); !errors.Is(err, errRuleNotFound) {
		t.Fatalf("repeat delete error = %v, want rule not found", err)
	}
}

func TestAutomationDependencyInvalidationIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := newAutomationIntegrationDB(t, ctx)
	seed := seedAutomationIntegration(t, ctx, db)
	handler := NewHandler(db, nil)

	statusRule := createIntegrationRule(t, ctx, handler, seed.Admin, seed.ProjectID, "Status dependency",
		fmt.Sprintf(`[{"type":"workflow_status","workflow_status_id":%q}]`, seed.StatusID),
		`[{"type":"change_priority","value":"high"}]`)
	labelRule := createIntegrationRule(t, ctx, handler, seed.Admin, seed.ProjectID, "Label dependency",
		fmt.Sprintf(`[{"type":"label","label_id":%q}]`, seed.LabelID),
		`[{"type":"change_priority","value":"high"}]`)
	userRule := createIntegrationRule(t, ctx, handler, seed.Admin, seed.ProjectID, "User dependency",
		`[]`, fmt.Sprintf(`[{"type":"change_assignee","user_id":%q}]`, seed.Contributor.ID))
	leadRule := createIntegrationRule(t, ctx, handler, seed.Admin, seed.ProjectID, "Inactive user dependency",
		fmt.Sprintf(`[{"type":"reporter","user_id":%q}]`, seed.Lead.ID),
		`[{"type":"change_priority","value":"high"}]`)
	adminRule := createIntegrationRule(t, ctx, handler, seed.Admin, seed.ProjectID, "Workspace admin dependency",
		`[]`, fmt.Sprintf(`[{"type":"change_assignee","user_id":%q}]`, seed.Admin.ID))
	unrelated := createIntegrationRule(t, ctx, handler, seed.Admin, seed.ProjectID, "Unrelated",
		`[]`, `[{"type":"change_priority","value":"medium"}]`)

	tx, _ := db.Begin(ctx)
	if err := DisableRulesForWorkflowStatus(ctx, tx, seed.ProjectID, seed.StatusID); err != nil {
		t.Fatalf("disable status rules: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit status invalidation: %v", err)
	}
	expectRuleDisabled(t, ctx, db, statusRule.ID, DisabledWorkflowStatusUnavailable)

	tx, _ = db.Begin(ctx)
	if err := DisableRulesForLabel(ctx, tx, seed.WorkspaceID, seed.LabelID); err != nil {
		t.Fatalf("disable label rules: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit label invalidation: %v", err)
	}
	expectRuleDisabled(t, ctx, db, labelRule.ID, DisabledLabelUnavailable)

	tx, _ = db.Begin(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`, seed.ProjectID, seed.Contributor.ID); err != nil {
		t.Fatalf("remove contributor membership: %v", err)
	}
	if err := DisableRulesForProjectUser(ctx, tx, seed.ProjectID, seed.Contributor.ID); err != nil {
		t.Fatalf("disable project user rules: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit user invalidation: %v", err)
	}
	expectRuleDisabled(t, ctx, db, userRule.ID, DisabledProjectAccessRemoved)

	tx, _ = db.Begin(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`, seed.ProjectID, seed.Admin.ID); err != nil {
		t.Fatalf("remove workspace admin project row: %v", err)
	}
	if err := DisableRulesForProjectUser(ctx, tx, seed.ProjectID, seed.Admin.ID); err != nil {
		t.Fatalf("reconcile workspace admin project access: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit workspace admin reconciliation: %v", err)
	}
	expectRuleEnabled(t, ctx, db, adminRule.ID)

	tx, _ = db.Begin(ctx)
	if _, err := tx.Exec(ctx, `UPDATE users SET is_active = false WHERE id = $1`, seed.Lead.ID); err != nil {
		t.Fatalf("deactivate project lead: %v", err)
	}
	if err := DisableRulesForWorkspaceUser(ctx, tx, seed.WorkspaceID, seed.Lead.ID); err != nil {
		t.Fatalf("disable inactive user rules: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit inactive user invalidation: %v", err)
	}
	expectRuleDisabled(t, ctx, db, leadRule.ID, DisabledUserUnavailable)
	expectRuleEnabled(t, ctx, db, unrelated.ID)
}

func TestAutomationRuleHTTPPermissionsIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := newAutomationIntegrationDB(t, ctx)
	seed := seedAutomationIntegration(t, ctx, db)
	authHandler := auth.NewHandler(db, time.Hour, false, nil, nil)
	handler := NewHandler(db, authHandler)
	mux := http.NewServeMux()
	authHandler.RegisterRoutes(mux)
	handler.RegisterRoutes(mux)

	adminCookies := loginAutomationUser(t, mux, seed.Admin.Username)
	response := performAutomationRequest(
		mux,
		http.MethodPost,
		"/api/v1/projects/"+seed.ProjectID+"/automation-rules",
		`{"name":"API rule","trigger_type":"issue_created","conditions":[],"actions":[{"type":"change_priority","value":"high"}]}`,
		adminCookies,
	)
	if response.Code != http.StatusCreated {
		t.Fatalf("admin create status = %d, want 201: %s", response.Code, response.Body.String())
	}
	leadCookies := loginAutomationUser(t, mux, seed.Lead.Username)
	response = performAutomationRequest(mux, http.MethodGet, "/api/v1/projects/"+seed.ProjectID+"/automation-rules", "", leadCookies)
	if response.Code != http.StatusOK {
		t.Fatalf("lead list status = %d, want 200: %s", response.Code, response.Body.String())
	}
	contributorCookies := loginAutomationUser(t, mux, seed.Contributor.Username)
	response = performAutomationRequest(mux, http.MethodGet, "/api/v1/projects/"+seed.ProjectID+"/automation-rules", "", contributorCookies)
	if response.Code != http.StatusForbidden {
		t.Fatalf("contributor list status = %d, want 403: %s", response.Code, response.Body.String())
	}
	outsiderCookies := loginAutomationUser(t, mux, seed.Outsider.Username)
	response = performAutomationRequest(mux, http.MethodGet, "/api/v1/projects/"+seed.ProjectID+"/automation-rules", "", outsiderCookies)
	if response.Code != http.StatusNotFound {
		t.Fatalf("outsider list status = %d, want 404: %s", response.Code, response.Body.String())
	}
}

type automationSeed struct {
	WorkspaceID   string
	ProjectID     string
	OtherProject  string
	StatusID      string
	OtherStatusID string
	LabelID       string
	Admin         auth.CurrentUser
	Lead          auth.CurrentUser
	Contributor   auth.CurrentUser
	Outsider      auth.CurrentUser
}

func newAutomationIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
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
	schemaName := fmt.Sprintf("automations_integration_%d", time.Now().UnixNano())
	quoted := pgx.Identifier{schemaName}.Sanitize()
	if _, err := adminDB.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`); err != nil {
		t.Fatalf("ensure pgcrypto: %v", err)
	}
	if _, err := adminDB.Exec(ctx, `CREATE SCHEMA `+quoted); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = adminDB.Exec(cleanupCtx, `DROP SCHEMA IF EXISTS `+quoted+` CASCADE`)
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

func seedAutomationIntegration(t *testing.T, ctx context.Context, db *pgxpool.Pool) automationSeed {
	t.Helper()
	var seed automationSeed
	if err := db.QueryRow(ctx, `INSERT INTO workspaces (name) VALUES ('Automation') RETURNING id::text`).Scan(&seed.WorkspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("member12345"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash integration password: %v", err)
	}
	insertUser := func(email, username string) auth.CurrentUser {
		var user auth.CurrentUser
		if err := db.QueryRow(ctx, `
			INSERT INTO users (email, username, password_hash, display_name)
			VALUES ($1, $2, $3, $2)
			RETURNING id::text, email, username, display_name
		`, email, username, string(passwordHash)).Scan(&user.ID, &user.Email, &user.Username, &user.DisplayName); err != nil {
			t.Fatalf("insert user %s: %v", username, err)
		}
		user.WorkspaceID = seed.WorkspaceID
		return user
	}
	seed.Admin = insertUser("automation-admin@example.com", "automation_admin")
	seed.Admin.Role = "admin"
	seed.Lead = insertUser("automation-lead@example.com", "automation_lead")
	seed.Lead.Role = "member"
	seed.Contributor = insertUser("automation-contributor@example.com", "automation_contributor")
	seed.Contributor.Role = "member"
	seed.Outsider = insertUser("automation-outsider@example.com", "automation_outsider")
	seed.Outsider.Role = "member"
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin'), ($1, $3, 'member'), ($1, $4, 'member'), ($1, $5, 'member')
	`, seed.WorkspaceID, seed.Admin.ID, seed.Lead.ID, seed.Contributor.ID, seed.Outsider.ID); err != nil {
		t.Fatalf("insert workspace members: %v", err)
	}
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'AUT', 'Automation Project', $2)
		RETURNING id::text
	`, seed.WorkspaceID, seed.Admin.ID).Scan(&seed.ProjectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'AUO', 'Other Automation Project', $2)
		RETURNING id::text
	`, seed.WorkspaceID, seed.Admin.ID).Scan(&seed.OtherProject); err != nil {
		t.Fatalf("insert other project: %v", err)
	}
	if _, err := db.Exec(ctx, `
		UPDATE project_members SET role = 'lead' WHERE project_id = $1 AND user_id = $2
	`, seed.ProjectID, seed.Lead.ID); err != nil {
		t.Fatalf("promote project lead: %v", err)
	}
	if _, err := db.Exec(ctx, `
		DELETE FROM project_members WHERE project_id = $1 AND user_id = $2
	`, seed.ProjectID, seed.Outsider.ID); err != nil {
		t.Fatalf("remove project outsider: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT id::text FROM project_workflow_statuses WHERE project_id = $1 AND key = 'todo'`, seed.ProjectID).Scan(&seed.StatusID); err != nil {
		t.Fatalf("load status: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT id::text FROM project_workflow_statuses WHERE project_id = $1 AND key = 'todo'`, seed.OtherProject).Scan(&seed.OtherStatusID); err != nil {
		t.Fatalf("load other status: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO labels (workspace_id, name, color) VALUES ($1, 'automation', '#123456') RETURNING id::text`, seed.WorkspaceID).Scan(&seed.LabelID); err != nil {
		t.Fatalf("insert label: %v", err)
	}
	return seed
}

func mustNormalizeCreate(t *testing.T, req createRuleRequest) normalizedCreateRule {
	t.Helper()
	input, err := normalizeCreateRule(req)
	if err != nil {
		t.Fatalf("normalize create rule: %v", err)
	}
	return input
}

func createIntegrationRule(t *testing.T, ctx context.Context, handler *Handler, user auth.CurrentUser, projectID, name, conditions, actions string) ruleResponse {
	t.Helper()
	rule, err := handler.createRule(ctx, user, projectID, mustNormalizeCreate(t, createRuleRequest{
		Name: name, TriggerType: "status_changed", Conditions: json.RawMessage(conditions), Actions: json.RawMessage(actions),
	}))
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	return rule
}

func expectRuleDisabled(t *testing.T, ctx context.Context, db *pgxpool.Pool, ruleID, reason string) {
	t.Helper()
	var enabled bool
	var gotReason *string
	if err := db.QueryRow(ctx, `SELECT is_enabled, disabled_reason FROM automation_rules WHERE id = $1`, ruleID).Scan(&enabled, &gotReason); err != nil {
		t.Fatalf("load disabled rule: %v", err)
	}
	if enabled || gotReason == nil || *gotReason != reason {
		t.Fatalf("rule state = enabled:%v reason:%v, want false/%q", enabled, gotReason, reason)
	}
}

func expectRuleEnabled(t *testing.T, ctx context.Context, db *pgxpool.Pool, ruleID string) {
	t.Helper()
	var enabled bool
	var reason *string
	if err := db.QueryRow(ctx, `SELECT is_enabled, disabled_reason FROM automation_rules WHERE id = $1`, ruleID).Scan(&enabled, &reason); err != nil {
		t.Fatalf("load enabled rule: %v", err)
	}
	if !enabled || reason != nil {
		t.Fatalf("rule state = enabled:%v reason:%v, want true/nil", enabled, reason)
	}
}

func loginAutomationUser(t *testing.T, mux *http.ServeMux, username string) []*http.Cookie {
	t.Helper()
	response := performAutomationRequest(
		mux,
		http.MethodPost,
		"/api/v1/auth/login",
		fmt.Sprintf(`{"login":%q,"password":"member12345"}`, username),
		nil,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("login %s status = %d: %s", username, response.Code, response.Body.String())
	}
	return response.Result().Cookies()
}

func performAutomationRequest(mux *http.ServeMux, method string, path string, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
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
