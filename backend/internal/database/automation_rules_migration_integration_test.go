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

func TestAutomationRulesMigrationConstraintsAndCascade(t *testing.T) {
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
	schemaName := fmt.Sprintf("automation_rules_migration_%d", time.Now().UnixNano())
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
		t.Fatalf("parse database URL: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schemaName
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect schema: %v", err)
	}
	defer db.Close()
	applied, err := migrations.Up(ctx, db, "../../migrations")
	if err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if len(applied) != 15 || applied[len(applied)-1].Version != 15 {
		t.Fatalf("applied migrations = %#v, want through version 15", applied)
	}

	var workspaceID, userID, projectID string
	if err := db.QueryRow(ctx, `INSERT INTO workspaces (name) VALUES ('Automation Schema') RETURNING id::text`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO users (email, username, password_hash, display_name) VALUES ('schema-auto@example.com', 'schema_auto', 'hash', 'Schema') RETURNING id::text`).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'admin')`, workspaceID, userID); err != nil {
		t.Fatalf("insert workspace member: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO projects (workspace_id, key, name, created_by) VALUES ($1, 'ARS', 'Rules', $2) RETURNING id::text`, workspaceID, userID).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	insert := func(name, trigger, conditions, actions string, position int) error {
		_, err := db.Exec(ctx, `
			INSERT INTO automation_rules (project_id, name, trigger_type, conditions, actions, position, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, projectID, name, trigger, conditions, actions, position, userID)
		return err
	}
	if err := insert("Valid", "issue_created", `[]`, `[{"type":"change_priority","value":"high"}]`, 100); err != nil {
		t.Fatalf("insert valid rule: %v", err)
	}
	for name, err := range map[string]error{
		"invalid trigger":      insert("Bad trigger", "commented", `[]`, `[{}]`, 200),
		"non-array conditions": insert("Bad conditions", "issue_created", `{}`, `[{}]`, 200),
		"empty actions":        insert("Bad actions", "issue_created", `[]`, `[]`, 200),
		"invalid position":     insert("Bad position", "issue_created", `[]`, `[{}]`, 0),
	} {
		if err == nil {
			t.Fatalf("expected %s to fail", name)
		}
	}
	if _, err := db.Exec(ctx, `DELETE FROM projects WHERE id = $1`, projectID); err != nil {
		t.Fatalf("delete project: %v", err)
	}
	var count int
	if err := db.QueryRow(ctx, `SELECT count(*)::int FROM automation_rules WHERE project_id = $1`, projectID).Scan(&count); err != nil {
		t.Fatalf("count rules after project delete: %v", err)
	}
	if count != 0 {
		t.Fatalf("rules after project delete = %d, want 0", count)
	}
	applied, err = migrations.Up(ctx, db, "../../migrations")
	if err != nil {
		t.Fatalf("repeat migrations: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("repeat migrations applied = %#v, want none", applied)
	}
}
