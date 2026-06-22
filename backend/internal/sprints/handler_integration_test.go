//go:build integration

package sprints

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"kelmio/backend/internal/auth"
	"kelmio/backend/internal/database"
	"kelmio/backend/internal/migrations"
)

func TestSprintLifecycleIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newSprintIntegrationDB(t, ctx)
	handler := NewHandler(db, nil)

	user, projectID, otherProjectID, issueID := seedSprintIntegrationWorkspace(t, ctx, db)

	sprint, err := handler.createSprint(ctx, user, normalizedCreateSprint{
		ProjectID: projectID,
		Name:      "Sprint 1",
		Goal:      "Ship sprint API",
		StartDate: "2026-06-01",
		EndDate:   "2026-06-14",
	})
	if err != nil {
		t.Fatalf("create sprint: %v", err)
	}
	if sprint.Status != "planned" {
		t.Fatalf("Status = %q, want planned", sprint.Status)
	}
	if sprint.IssueCount != 0 {
		t.Fatalf("IssueCount = %d, want 0", sprint.IssueCount)
	}

	sprints, err := handler.listSprints(ctx, user.WorkspaceID, projectID, "planned")
	if err != nil {
		t.Fatalf("list planned project sprints: %v", err)
	}
	if !hasSprintID(sprints, sprint.ID) {
		t.Fatalf("expected sprint list to contain %s", sprint.ID)
	}

	gotSprint, err := handler.getSprint(ctx, user.WorkspaceID, sprint.ID)
	if err != nil {
		t.Fatalf("get sprint: %v", err)
	}
	if gotSprint.ID != sprint.ID {
		t.Fatalf("got sprint id = %q, want %q", gotSprint.ID, sprint.ID)
	}

	updated, err := handler.updateSprint(ctx, user.WorkspaceID, sprint.ID, normalizedUpdateSprint{
		Name:      "Sprint 1 Updated",
		Goal:      "Updated sprint goal",
		StartDate: "2026-06-02",
		EndDate:   "2026-06-15",
	})
	if err != nil {
		t.Fatalf("update sprint: %v", err)
	}
	if updated.Name != "Sprint 1 Updated" {
		t.Fatalf("updated sprint name = %q, want Sprint 1 Updated", updated.Name)
	}

	active, err := handler.startSprint(ctx, user, sprint.ID)
	if err != nil {
		t.Fatalf("start sprint: %v", err)
	}
	if active.Status != "active" {
		t.Fatalf("started sprint status = %q, want active", active.Status)
	}

	secondSprint, err := handler.createSprint(ctx, user, normalizedCreateSprint{
		ProjectID: projectID,
		Name:      "Sprint 2",
	})
	if err != nil {
		t.Fatalf("create second sprint: %v", err)
	}
	if _, err := handler.startSprint(ctx, user, secondSprint.ID); !errors.Is(err, errActiveSprintExists) {
		t.Fatalf("start second active sprint error = %v, want %v", err, errActiveSprintExists)
	}
	if _, err := handler.completeSprint(ctx, user, secondSprint.ID); !errors.Is(err, errSprintNotActive) {
		t.Fatalf("complete planned sprint error = %v, want %v", err, errSprintNotActive)
	}

	withIssue, err := handler.addIssueToSprint(ctx, user, sprint.ID, issueID)
	if err != nil {
		t.Fatalf("add issue to sprint: %v", err)
	}
	if withIssue.IssueCount != 1 {
		t.Fatalf("IssueCount after add = %d, want 1", withIssue.IssueCount)
	}
	if withIssue.PointsTotal != 5 || withIssue.PointsOpen != 5 || withIssue.PointsDone != 0 {
		t.Fatalf("points after add = total:%d open:%d done:%d, want total:5 open:5 done:0", withIssue.PointsTotal, withIssue.PointsOpen, withIssue.PointsDone)
	}
	var shippedStatusID string
	if err := db.QueryRow(ctx, `
		INSERT INTO project_workflow_statuses (project_id, key, name, color, category, position)
		VALUES ($1, 'shipped', 'Shipped', '#16a34a', 'done', 600)
		RETURNING id::text
	`, projectID).Scan(&shippedStatusID); err != nil {
		t.Fatalf("create custom done status: %v", err)
	}
	if _, err := db.Exec(ctx, `UPDATE issues SET workflow_status_id = $2 WHERE id = $1`, issueID, shippedStatusID); err != nil {
		t.Fatalf("move issue to custom done status: %v", err)
	}
	withCustomDone, err := handler.getSprint(ctx, user.WorkspaceID, sprint.ID)
	if err != nil {
		t.Fatalf("get sprint with custom done issue: %v", err)
	}
	if withCustomDone.DoneCount != 1 || withCustomDone.PointsDone != 5 || withCustomDone.PointsOpen != 0 {
		t.Fatalf("custom done metrics = count:%d done:%d open:%d, want 1/5/0", withCustomDone.DoneCount, withCustomDone.PointsDone, withCustomDone.PointsOpen)
	}
	expectIssueSprint(t, ctx, db, issueID, sprint.ID)
	expectIssueActivity(t, ctx, db, issueID, "issue_added_to_sprint")

	if err := handler.removeIssueFromSprint(ctx, user, sprint.ID, issueID); err != nil {
		t.Fatalf("remove issue from sprint: %v", err)
	}
	expectIssueSprint(t, ctx, db, issueID, "")
	expectIssueActivity(t, ctx, db, issueID, "issue_removed_from_sprint")

	if _, err := handler.addIssueToSprint(ctx, user, sprint.ID, issueID); err != nil {
		t.Fatalf("re-add issue to sprint: %v", err)
	}

	completed, err := handler.completeSprint(ctx, user, sprint.ID)
	if err != nil {
		t.Fatalf("complete sprint: %v", err)
	}
	if completed.Status != "completed" {
		t.Fatalf("completed sprint status = %q, want completed", completed.Status)
	}
	if completed.CompletedAt == nil {
		t.Fatal("expected completed sprint to have completed_at")
	}
	expectIssueSprint(t, ctx, db, issueID, sprint.ID)

	if _, err := handler.updateSprint(ctx, user.WorkspaceID, sprint.ID, normalizedUpdateSprint{
		Name: "Completed Sprint Updated",
	}); !errors.Is(err, errSprintCompleted) {
		t.Fatalf("update completed sprint error = %v, want %v", err, errSprintCompleted)
	}
	if _, err := handler.addIssueToSprint(ctx, user, sprint.ID, issueID); !errors.Is(err, errSprintCompleted) {
		t.Fatalf("add issue to completed sprint error = %v, want %v", err, errSprintCompleted)
	}
	if _, err := handler.startSprint(ctx, user, sprint.ID); !errors.Is(err, errSprintNotPlanned) {
		t.Fatalf("start completed sprint error = %v, want %v", err, errSprintNotPlanned)
	}

	secondActive, err := handler.startSprint(ctx, user, secondSprint.ID)
	if err != nil {
		t.Fatalf("start second sprint after completing first: %v", err)
	}
	if secondActive.Status != "active" {
		t.Fatalf("second sprint status = %q, want active", secondActive.Status)
	}

	otherProjectSprint, err := handler.createSprint(ctx, user, normalizedCreateSprint{
		ProjectID: otherProjectID,
		Name:      "Other Project Sprint",
	})
	if err != nil {
		t.Fatalf("create other project sprint: %v", err)
	}
	if _, err := handler.addIssueToSprint(ctx, user, otherProjectSprint.ID, issueID); !errors.Is(err, errSprintIssueProjectMismatch) {
		t.Fatalf("add cross-project issue error = %v, want %v", err, errSprintIssueProjectMismatch)
	}
}

func newSprintIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

	schemaName := fmt.Sprintf("sprints_integration_%d", time.Now().UnixNano())
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

func seedSprintIntegrationWorkspace(t *testing.T, ctx context.Context, db *pgxpool.Pool) (auth.CurrentUser, string, string, string) {
	t.Helper()

	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ('Sprints Integration Workspace')
		RETURNING id::text
	`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}

	var userID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ('sprints-integration@example.com', 'sprints_integration', 'hash', 'Sprints Integration')
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
		VALUES ($1, 'SPRT', 'Sprint Project', $2)
		RETURNING id::text
	`, workspaceID, userID).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	var otherProjectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'OTH', 'Other Sprint Project', $2)
		RETURNING id::text
	`, workspaceID, userID).Scan(&otherProjectID); err != nil {
		t.Fatalf("insert other project: %v", err)
	}

	var issueID string
	if err := db.QueryRow(ctx, `
		INSERT INTO issues (
			project_id,
			number,
			issue_key,
			title,
			issue_type,
			status,
			priority,
			story_points,
			reporter_id
		)
		VALUES ($1, 1, 'SPRT-1', 'Sprint issue', 'task', 'todo', 'medium', 5, $2)
		RETURNING id::text
	`, projectID, userID).Scan(&issueID); err != nil {
		t.Fatalf("insert issue: %v", err)
	}

	user := auth.CurrentUser{
		ID:          userID,
		WorkspaceID: workspaceID,
		Role:        "admin",
	}

	return user, projectID, otherProjectID, issueID
}

func hasSprintID(sprints []sprintResponse, sprintID string) bool {
	for _, sprint := range sprints {
		if sprint.ID == sprintID {
			return true
		}
	}

	return false
}

func expectIssueSprint(t *testing.T, ctx context.Context, db *pgxpool.Pool, issueID string, wantSprintID string) {
	t.Helper()

	var sprintID pgtype.Text
	if err := db.QueryRow(ctx, `
		SELECT sprint_id::text
		FROM issues
		WHERE id = $1
	`, issueID).Scan(&sprintID); err != nil {
		t.Fatalf("load issue sprint id: %v", err)
	}

	if wantSprintID == "" {
		if sprintID.Valid {
			t.Fatalf("issue sprint id = %q, want NULL", sprintID.String)
		}
		return
	}

	if !sprintID.Valid {
		t.Fatalf("issue sprint id is NULL, want %q", wantSprintID)
	}
	if sprintID.String != wantSprintID {
		t.Fatalf("issue sprint id = %q, want %q", sprintID.String, wantSprintID)
	}
}

func expectIssueActivity(t *testing.T, ctx context.Context, db *pgxpool.Pool, issueID string, action string) {
	t.Helper()

	var exists bool
	if err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM activity_log
			WHERE entity_type = 'issue'
				AND entity_id = $1
				AND action = $2
		)
	`, issueID, action).Scan(&exists); err != nil {
		t.Fatalf("check issue activity %s: %v", action, err)
	}
	if !exists {
		t.Fatalf("expected issue activity %s", action)
	}
}
