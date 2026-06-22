//go:build integration

package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kelmio/backend/internal/migrations"
)

func TestWorkflowIssueStatusMigrationEnablesCustomStatuses(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://kelmio:kelmio@localhost:15432/kelmio?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adminDB, err := Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	defer adminDB.Close()

	schemaName := fmt.Sprintf("workflow_issue_upgrade_%d", time.Now().UnixNano())
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
	defer db.Close()

	legacyDir := t.TempDir()
	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() >= "000013_" {
			continue
		}
		contents, err := os.ReadFile(filepath.Join("../../migrations", entry.Name()))
		if err != nil {
			t.Fatalf("read migration %s: %v", entry.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(legacyDir, entry.Name()), contents, 0o600); err != nil {
			t.Fatalf("copy migration %s: %v", entry.Name(), err)
		}
	}
	if _, err := migrations.Up(ctx, db, legacyDir); err != nil {
		t.Fatalf("apply migrations 1..12: %v", err)
	}

	var workspaceID, userID, projectID, otherProjectID, issueID, reviewID, otherReviewID string
	if err := db.QueryRow(ctx, `INSERT INTO workspaces (name) VALUES ('Workflow Upgrade') RETURNING id::text`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ('workflow-upgrade@example.com', 'workflow_upgrade', 'hash', 'Workflow Upgrade')
		RETURNING id::text
	`).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'admin')`, workspaceID, userID); err != nil {
		t.Fatalf("insert workspace member: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO projects (workspace_id, key, name, created_by) VALUES ($1, 'WFU', 'Workflow Upgrade', $2) RETURNING id::text`, workspaceID, userID).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO projects (workspace_id, key, name, created_by) VALUES ($1, 'WFO', 'Other Workflow', $2) RETURNING id::text`, workspaceID, userID).Scan(&otherProjectID); err != nil {
		t.Fatalf("insert other project: %v", err)
	}
	if err := db.QueryRow(ctx, `
		INSERT INTO issues (project_id, number, issue_key, title, issue_type, status, priority, reporter_id)
		VALUES ($1, 1, 'WFU-1', 'Legacy issue', 'task', 'todo', 'medium', $2)
		RETURNING id::text
	`, projectID, userID).Scan(&issueID); err != nil {
		t.Fatalf("insert legacy issue: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO project_workflow_statuses (project_id, key, name, color, category, position) VALUES ($1, 'review', 'Review', '#0ea5e9', 'in_progress', 600) RETURNING id::text`, projectID).Scan(&reviewID); err != nil {
		t.Fatalf("insert custom status: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO project_workflow_statuses (project_id, key, name, color, category, position) VALUES ($1, 'review', 'Review', '#0ea5e9', 'in_progress', 600) RETURNING id::text`, otherProjectID).Scan(&otherReviewID); err != nil {
		t.Fatalf("insert other custom status: %v", err)
	}

	applied, err := migrations.Up(ctx, db, "../../migrations")
	if err != nil {
		t.Fatalf("apply migration 13: %v", err)
	}
	if len(applied) != 5 || applied[0].Version != 13 || applied[1].Version != 14 || applied[2].Version != 15 || applied[3].Version != 16 || applied[4].Version != 17 {
		t.Fatalf("applied migrations = %#v, want versions 13 through 17", applied)
	}

	if _, err := db.Exec(ctx, `UPDATE issues SET workflow_status_id = $2 WHERE id = $1`, issueID, reviewID); err != nil {
		t.Fatalf("update issue by workflow id: %v", err)
	}
	expectWorkflowIssueStatus(t, ctx, db, issueID, reviewID, "review")
	if _, err := db.Exec(ctx, `UPDATE issues SET status = 'done' WHERE id = $1`, issueID); err != nil {
		t.Fatalf("update issue by key: %v", err)
	}
	var doneID string
	if err := db.QueryRow(ctx, `SELECT id::text FROM project_workflow_statuses WHERE project_id = $1 AND key = 'done'`, projectID).Scan(&doneID); err != nil {
		t.Fatalf("load done status: %v", err)
	}
	expectWorkflowIssueStatus(t, ctx, db, issueID, doneID, "done")
	if _, err := db.Exec(ctx, `UPDATE issues SET workflow_status_id = $2, status = 'backlog' WHERE id = $1`, issueID, reviewID); err != nil {
		t.Fatalf("update issue with workflow id precedence: %v", err)
	}
	expectWorkflowIssueStatus(t, ctx, db, issueID, reviewID, "review")
	if _, err := db.Exec(ctx, `UPDATE issues SET workflow_status_id = $2 WHERE id = $1`, issueID, otherReviewID); err == nil {
		t.Fatal("expected cross-project workflow status to fail")
	}
	if _, err := db.Exec(ctx, `UPDATE issues SET workflow_status_id = $2 WHERE id = $1`, issueID, doneID); err != nil {
		t.Fatalf("move issue before archiving custom status: %v", err)
	}
	if _, err := db.Exec(ctx, `UPDATE project_workflow_statuses SET archived_at = now() WHERE id = $1`, reviewID); err != nil {
		t.Fatalf("archive workflow status: %v", err)
	}
	if _, err := db.Exec(ctx, `UPDATE issues SET workflow_status_id = $2 WHERE id = $1`, issueID, reviewID); err == nil {
		t.Fatal("expected archived workflow status to fail")
	}
	applied, err = migrations.Up(ctx, db, "../../migrations")
	if err != nil {
		t.Fatalf("repeat migrations: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("repeat migrations applied = %#v, want none", applied)
	}
}

func expectWorkflowIssueStatus(t *testing.T, ctx context.Context, db *pgxpool.Pool, issueID string, wantID string, wantKey string) {
	t.Helper()
	var statusID, statusKey string
	if err := db.QueryRow(ctx, `SELECT workflow_status_id::text, status FROM issues WHERE id = $1`, issueID).Scan(&statusID, &statusKey); err != nil {
		t.Fatalf("load issue workflow status: %v", err)
	}
	if statusID != wantID || statusKey != wantKey {
		t.Fatalf("workflow status = %q/%q, want %q/%q", statusID, statusKey, wantID, wantKey)
	}
}
