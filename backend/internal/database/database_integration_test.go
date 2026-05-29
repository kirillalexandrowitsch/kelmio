//go:build integration

package database

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"team-task-tracker/backend/internal/migrations"
)

func TestPostgresMigrationsCreateCoreSchema(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://team_task_tracker:team_task_tracker@localhost:15432/team_task_tracker?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminDB, err := Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	defer adminDB.Close()

	schemaName := fmt.Sprintf("integration_%d", time.Now().UnixNano())
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
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping integration database: %v", err)
	}

	applied, err := migrations.Up(ctx, db, "../../migrations")
	if err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected migrations to be applied in isolated schema")
	}

	expectedTables := []string{
		"workspaces",
		"users",
		"workspace_members",
		"projects",
		"issues",
		"labels",
		"issue_labels",
		"comments",
		"issue_links",
		"sessions",
		"activity_log",
		"schema_migrations",
	}
	for _, tableName := range expectedTables {
		var exists bool
		if err := db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = $1
					AND table_name = $2
			)
		`, schemaName, tableName).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", tableName, err)
		}
		if !exists {
			t.Fatalf("expected table %s to exist", tableName)
		}
	}

	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ($1)
		RETURNING id::text
	`, "Integration Workspace").Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if workspaceID == "" {
		t.Fatal("expected generated workspace id")
	}

	var hasParentIssueID bool
	if err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = $1
				AND table_name = 'issues'
				AND column_name = 'parent_issue_id'
		)
	`, schemaName).Scan(&hasParentIssueID); err != nil {
		t.Fatalf("check parent_issue_id column: %v", err)
	}
	if !hasParentIssueID {
		t.Fatal("expected issues.parent_issue_id to exist")
	}

	var userID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, "integration@example.com", "integration", "hash", "Integration User").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var projectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, workspaceID, "INT", "Integration Project", userID).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	var epicID string
	if err := db.QueryRow(ctx, `
		INSERT INTO issues (
			project_id,
			number,
			issue_key,
			title,
			issue_type,
			status,
			priority,
			reporter_id
		)
		VALUES ($1, 1, 'INT-1', 'Integration epic', 'epic', 'todo', 'medium', $2)
		RETURNING id::text
	`, projectID, userID).Scan(&epicID); err != nil {
		t.Fatalf("insert epic issue: %v", err)
	}

	var subtaskID string
	if err := db.QueryRow(ctx, `
		INSERT INTO issues (
			project_id,
			number,
			issue_key,
			title,
			issue_type,
			status,
			priority,
			reporter_id,
			parent_issue_id
		)
		VALUES ($1, 2, 'INT-2', 'Integration subtask', 'subtask', 'todo', 'medium', $2, $3)
		RETURNING id::text
	`, projectID, userID, epicID).Scan(&subtaskID); err != nil {
		t.Fatalf("insert subtask issue: %v", err)
	}
	if subtaskID == "" {
		t.Fatal("expected generated subtask id")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO issues (
			project_id,
			number,
			issue_key,
			title,
			issue_type,
			status,
			priority,
			reporter_id
		)
		VALUES ($1, 3, 'INT-3', 'Invalid subtask', 'subtask', 'todo', 'medium', $2)
	`, projectID, userID); err == nil {
		t.Fatal("expected subtask without parent to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO issues (
			project_id,
			number,
			issue_key,
			title,
			issue_type,
			status,
			priority,
			reporter_id,
			parent_issue_id
		)
		VALUES ($1, 4, 'INT-4', 'Invalid epic', 'epic', 'todo', 'medium', $2, $3)
	`, projectID, userID, epicID); err == nil {
		t.Fatal("expected epic with parent to fail")
	}

	var linkID string
	if err := db.QueryRow(ctx, `
		INSERT INTO issue_links (source_issue_id, target_issue_id, link_type, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, epicID, subtaskID, "relates", userID).Scan(&linkID); err != nil {
		t.Fatalf("insert issue link: %v", err)
	}
	if linkID == "" {
		t.Fatal("expected generated issue link id")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO issue_links (source_issue_id, target_issue_id, link_type, created_by)
		VALUES ($1, $2, $3, $4)
	`, epicID, subtaskID, "relates", userID); err == nil {
		t.Fatal("expected duplicate issue link to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO issue_links (source_issue_id, target_issue_id, link_type, created_by)
		VALUES ($1, $2, $3, $4)
	`, subtaskID, epicID, "relates", userID); err == nil {
		t.Fatal("expected inverse relates issue link to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO issue_links (source_issue_id, target_issue_id, link_type, created_by)
		VALUES ($1, $2, $3, $4)
	`, epicID, epicID, "blocks", userID); err == nil {
		t.Fatal("expected self issue link to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO issue_links (source_issue_id, target_issue_id, link_type, created_by)
		VALUES ($1, $2, $3, $4)
	`, subtaskID, epicID, "duplicates", userID); err == nil {
		t.Fatal("expected invalid issue link type to fail")
	}
}
