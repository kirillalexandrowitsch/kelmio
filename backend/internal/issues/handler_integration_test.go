//go:build integration

package issues

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/database"
	"team-task-tracker/backend/internal/migrations"
)

func TestIssueHierarchyIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newIssueIntegrationDB(t, ctx)
	handler := NewHandler(db, nil)

	user, projectID := seedIssueIntegrationWorkspace(t, ctx, db)

	parent, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID,
		Title:     "Parent issue",
		IssueType: "task",
		Status:    "todo",
		Priority:  "medium",
	})
	if err != nil {
		t.Fatalf("create parent issue: %v", err)
	}

	epic, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID,
		Title:     "Epic issue",
		IssueType: "epic",
		Status:    "todo",
		Priority:  "medium",
	})
	if err != nil {
		t.Fatalf("create epic issue: %v", err)
	}

	epicChild, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID:     projectID,
		ParentIssueID: epic.ID,
		Title:         "Story under epic",
		IssueType:     "story",
		Status:        "todo",
		Priority:      "medium",
	})
	if err != nil {
		t.Fatalf("create epic child issue: %v", err)
	}
	expectIssueParent(t, epicChild, epic.ID)

	subtask, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID:     projectID,
		ParentIssueID: parent.ID,
		Title:         "Subtask issue",
		IssueType:     "subtask",
		Status:        "todo",
		Priority:      "medium",
	})
	if err != nil {
		t.Fatalf("create subtask issue: %v", err)
	}
	expectIssueParent(t, subtask, parent.ID)

	children, err := handler.listIssueChildren(ctx, user.WorkspaceID, parent.ID)
	if err != nil {
		t.Fatalf("list issue children: %v", err)
	}
	if !hasIssueID(children, subtask.ID) {
		t.Fatalf("expected children to contain subtask %s", subtask.ID)
	}

	if _, err := handler.setIssueParent(ctx, user, parent.ID, subtask.ID); !errors.Is(err, errIssueParentCycle) {
		t.Fatalf("set parent to descendant error = %v, want %v", err, errIssueParentCycle)
	}

	if _, err := handler.setIssueParent(ctx, user, subtask.ID, ""); !errors.Is(err, errIssueParentRequired) {
		t.Fatalf("clear subtask parent error = %v, want %v", err, errIssueParentRequired)
	}

	if _, err := handler.setIssueParent(ctx, user, epic.ID, parent.ID); !errors.Is(err, errIssueParentForbidden) {
		t.Fatalf("set epic parent error = %v, want %v", err, errIssueParentForbidden)
	}

	moved, err := handler.setIssueParent(ctx, user, epicChild.ID, parent.ID)
	if err != nil {
		t.Fatalf("move epic child under parent: %v", err)
	}
	expectIssueParent(t, moved, parent.ID)

	activity, err := handler.listIssueActivity(ctx, user.WorkspaceID, epicChild.ID)
	if err != nil {
		t.Fatalf("list issue activity: %v", err)
	}
	if !hasActivityAction(activity, "issue_parent_changed") {
		t.Fatal("expected issue_parent_changed activity")
	}
}

func newIssueIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

	schemaName := fmt.Sprintf("issues_integration_%d", time.Now().UnixNano())
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

func seedIssueIntegrationWorkspace(t *testing.T, ctx context.Context, db *pgxpool.Pool) (auth.CurrentUser, string) {
	t.Helper()

	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ('Issues Integration Workspace')
		RETURNING id::text
	`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}

	var userID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ('issues-integration@example.com', 'issues_integration', 'hash', 'Issues Integration')
		RETURNING id::text
	`).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin')
	`, workspaceID, userID); err != nil {
		t.Fatalf("insert workspace member: %v", err)
	}

	var projectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'HIER', 'Hierarchy Project', $2)
		RETURNING id::text
	`, workspaceID, userID).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	return auth.CurrentUser{
		ID:          userID,
		WorkspaceID: workspaceID,
		Role:        "admin",
	}, projectID
}

func expectIssueParent(t *testing.T, issue issueResponse, want string) {
	t.Helper()

	if issue.ParentIssueID == nil {
		t.Fatalf("ParentIssueID is nil, want %q", want)
	}
	if *issue.ParentIssueID != want {
		t.Fatalf("ParentIssueID = %q, want %q", *issue.ParentIssueID, want)
	}
}

func hasIssueID(issues []issueResponse, issueID string) bool {
	for _, issue := range issues {
		if issue.ID == issueID {
			return true
		}
	}

	return false
}

func hasActivityAction(activity []issueActivityResponse, action string) bool {
	for _, entry := range activity {
		if entry.Action == action {
			return true
		}
	}

	return false
}
