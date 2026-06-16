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

	"team-task-tracker/backend/internal/migrations"
)

func TestProjectMemberMigrationBackfillsExistingProjects(t *testing.T) {
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
	schemaName := fmt.Sprintf("project_members_upgrade_%d", time.Now().UnixNano())
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
		if entry.IsDir() || entry.Name() >= "000012_" {
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
	applied, err := migrations.Up(ctx, db, legacyDir)
	if err != nil {
		t.Fatalf("apply legacy migrations: %v", err)
	}
	if len(applied) != 11 {
		t.Fatalf("legacy migrations applied = %d, want 11", len(applied))
	}

	var workspaceID string
	var otherWorkspaceID string
	if err := db.QueryRow(ctx, `INSERT INTO workspaces (name) VALUES ('Membership Upgrade') RETURNING id::text`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO workspaces (name) VALUES ('Other Membership Upgrade') RETURNING id::text`).Scan(&otherWorkspaceID); err != nil {
		t.Fatalf("insert other workspace: %v", err)
	}
	insertUser := func(email, username string, active bool) string {
		var id string
		if err := db.QueryRow(ctx, `
			INSERT INTO users (email, username, password_hash, display_name, is_active)
			VALUES ($1, $2, 'hash', $2, $3)
			RETURNING id::text
		`, email, username, active).Scan(&id); err != nil {
			t.Fatalf("insert user %s: %v", username, err)
		}
		return id
	}
	creatorID := insertUser("upgrade-creator@example.com", "upgrade_creator", true)
	memberID := insertUser("upgrade-member@example.com", "upgrade_member", true)
	inactiveID := insertUser("upgrade-inactive@example.com", "upgrade_inactive", false)
	otherUserID := insertUser("upgrade-other@example.com", "upgrade_other", true)
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'member'), ($1, $3, 'member'), ($1, $4, 'member'), ($5, $6, 'member')
	`, workspaceID, creatorID, memberID, inactiveID, otherWorkspaceID, otherUserID); err != nil {
		t.Fatalf("insert workspace members: %v", err)
	}
	var projectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'UPG', 'Upgrade Project', $2)
		RETURNING id::text
	`, workspaceID, creatorID).Scan(&projectID); err != nil {
		t.Fatalf("insert legacy project: %v", err)
	}

	applied, err = migrations.Up(ctx, db, "../../migrations")
	if err != nil {
		t.Fatalf("apply project membership migration: %v", err)
	}
	if len(applied) != 5 || applied[0].Version != 12 || applied[1].Version != 13 || applied[2].Version != 14 || applied[3].Version != 15 || applied[4].Version != 16 {
		t.Fatalf("membership migrations applied = %#v, want versions 12 through 16", applied)
	}
	var creatorRole string
	var memberRole string
	var inactiveCount int
	if err := db.QueryRow(ctx, `
		SELECT
			(SELECT role FROM project_members WHERE project_id = $1 AND user_id = $2),
			(SELECT role FROM project_members WHERE project_id = $1 AND user_id = $3),
			(SELECT count(*)::int FROM project_members WHERE project_id = $1 AND user_id = $4)
	`, projectID, creatorID, memberID, inactiveID).Scan(&creatorRole, &memberRole, &inactiveCount); err != nil {
		t.Fatalf("check membership backfill: %v", err)
	}
	if creatorRole != "lead" || memberRole != "contributor" || inactiveCount != 0 {
		t.Fatalf("backfill = creator:%q member:%q inactive:%d, want lead/contributor/0", creatorRole, memberRole, inactiveCount)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id, role)
		VALUES ($1, $2, 'viewer')
	`, projectID, otherUserID); err == nil {
		t.Fatal("expected cross-workspace project member insert to fail")
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id, role)
		VALUES ($1, $2, 'viewer')
	`, projectID, inactiveID); err == nil {
		t.Fatal("expected inactive project member insert to fail")
	}
	newWorkspaceMemberID := insertUser("upgrade-new@example.com", "upgrade_new", true)
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'member')
	`, workspaceID, newWorkspaceMemberID); err != nil {
		t.Fatalf("insert new workspace member: %v", err)
	}
	var oldProjectMembershipCount int
	if err := db.QueryRow(ctx, `
		SELECT count(*)::int
		FROM project_members
		WHERE project_id = $1
			AND user_id = $2
	`, projectID, newWorkspaceMemberID).Scan(&oldProjectMembershipCount); err != nil {
		t.Fatalf("count new workspace member in old project: %v", err)
	}
	if oldProjectMembershipCount != 0 {
		t.Fatalf("new workspace member old project memberships = %d, want 0", oldProjectMembershipCount)
	}

	var newProjectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'NEW', 'New Membership Project', $2)
		RETURNING id::text
	`, workspaceID, memberID).Scan(&newProjectID); err != nil {
		t.Fatalf("insert post-migration project: %v", err)
	}
	if err := db.QueryRow(ctx, `
		SELECT
			(SELECT role FROM project_members WHERE project_id = $1 AND user_id = $2),
			(SELECT role FROM project_members WHERE project_id = $1 AND user_id = $3),
			(SELECT count(*)::int FROM project_members WHERE project_id = $1 AND user_id = $4)
	`, newProjectID, memberID, creatorID, inactiveID).Scan(&creatorRole, &memberRole, &inactiveCount); err != nil {
		t.Fatalf("check new project memberships: %v", err)
	}
	if creatorRole != "lead" || memberRole != "contributor" || inactiveCount != 0 {
		t.Fatalf("new project memberships = creator:%q member:%q inactive:%d", creatorRole, memberRole, inactiveCount)
	}

	applied, err = migrations.Up(ctx, db, "../../migrations")
	if err != nil {
		t.Fatalf("repeat migrations: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("repeat migrations applied = %#v, want none", applied)
	}
}
