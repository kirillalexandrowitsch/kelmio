//go:build integration

package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kelmio/backend/internal/migrations"
)

func TestEffectiveProjectMembersResolvesMaxRole(t *testing.T) {
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
	schemaName := fmt.Sprintf("effective_project_%d", time.Now().UnixNano())
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
	if _, err := migrations.Up(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	var orgID string
	if err := db.QueryRow(ctx, `SELECT id::text FROM organizations WHERE slug = 'default'`).Scan(&orgID); err != nil {
		t.Fatalf("read default organization: %v", err)
	}
	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name, organization_id, slug, status)
		VALUES ('Project Eff WS', $1, 'project-eff-ws', 'active')
		RETURNING id::text
	`, orgID).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}

	insertUser := func(username string) string {
		var id string
		if err := db.QueryRow(ctx, `
			INSERT INTO users (email, username, password_hash, display_name, is_active)
			VALUES ($1, $1, 'hash', $1, true)
			RETURNING id::text
		`, username).Scan(&id); err != nil {
			t.Fatalf("insert user %s: %v", username, err)
		}
		return id
	}
	creatorID := insertUser("p_creator")
	directID := insertUser("p_direct")
	groupOnlyID := insertUser("p_group")
	userOnlyID := insertUser("p_user")

	// Only creator and direct are workspace members; the project-create trigger
	// seeds them as lead / contributor project members respectively.
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'admin'), ($1, $3, 'member')
	`, workspaceID, creatorID, directID); err != nil {
		t.Fatalf("insert workspace members: %v", err)
	}
	var projectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'PEF', 'Project Eff', $2)
		RETURNING id::text
	`, workspaceID, creatorID).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	// Groups carrying project role assignments.
	insertGroup := func(name string) string {
		var id string
		if err := db.QueryRow(ctx, `INSERT INTO groups (organization_id, name) VALUES ($1, $2) RETURNING id::text`, orgID, name).Scan(&id); err != nil {
			t.Fatalf("insert group %s: %v", name, err)
		}
		return id
	}
	leadGroupID := insertGroup("Leads")
	viewGroupID := insertGroup("Viewers")
	if _, err := db.Exec(ctx, `INSERT INTO group_members (group_id, user_id) VALUES ($1, $2), ($3, $4)`, leadGroupID, directID, viewGroupID, groupOnlyID); err != nil {
		t.Fatalf("insert group members: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO role_assignments (scope, scope_id, subject_type, subject_id, role) VALUES
			('project', $1, 'group', $2, 'lead'),
			('project', $1, 'group', $3, 'viewer'),
			('project', $1, 'user', $4, 'contributor')
	`, projectID, leadGroupID, viewGroupID, userOnlyID); err != nil {
		t.Fatalf("insert role assignments: %v", err)
	}

	effectiveRole := func(userID string) (string, error) {
		var role string
		err := db.QueryRow(ctx, `
			SELECT role FROM effective_project_members WHERE project_id = $1 AND user_id = $2
		`, projectID, userID).Scan(&role)
		return role, err
	}

	if role, err := effectiveRole(creatorID); err != nil || role != "lead" {
		t.Fatalf("creator role = %q (err=%v), want lead", role, err)
	}
	// Direct contributor plus a group-derived lead resolves to the maximum.
	if role, err := effectiveRole(directID); err != nil || role != "lead" {
		t.Fatalf("direct+group role = %q (err=%v), want lead", role, err)
	}
	if role, err := effectiveRole(groupOnlyID); err != nil || role != "viewer" {
		t.Fatalf("group-only role = %q (err=%v), want viewer", role, err)
	}
	if role, err := effectiveRole(userOnlyID); err != nil || role != "contributor" {
		t.Fatalf("user-only role = %q (err=%v), want contributor", role, err)
	}
	// A user with neither membership nor assignment has no effective role.
	if _, err := effectiveRole(insertUser("p_none")); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("unassigned user error = %v, want no rows", err)
	}
}
