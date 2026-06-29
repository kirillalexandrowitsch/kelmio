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

func TestEffectiveWorkspaceMembersResolvesMaxRole(t *testing.T) {
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
	schemaName := fmt.Sprintf("effective_members_%d", time.Now().UnixNano())
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
		VALUES ('Effective WS', $1, 'effective-ws', 'active')
		RETURNING id::text
	`, orgID).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	var groupID string
	if err := db.QueryRow(ctx, `
		INSERT INTO groups (organization_id, name) VALUES ($1, 'Leads') RETURNING id::text
	`, orgID).Scan(&groupID); err != nil {
		t.Fatalf("insert group: %v", err)
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
	directID := insertUser("u_direct")
	groupAdminID := insertUser("u_group")
	bothID := insertUser("u_both")
	noneID := insertUser("u_none")

	// Direct member only.
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'member')
	`, workspaceID, directID); err != nil {
		t.Fatalf("insert direct member: %v", err)
	}
	// Admin only via a group assignment, with no direct membership.
	if _, err := db.Exec(ctx, `INSERT INTO group_members (group_id, user_id) VALUES ($1, $2)`, groupID, groupAdminID); err != nil {
		t.Fatalf("insert group member: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO role_assignments (scope, scope_id, subject_type, subject_id, role)
		VALUES ('workspace', $1, 'group', $2, 'admin')
	`, workspaceID, groupID); err != nil {
		t.Fatalf("insert group assignment: %v", err)
	}
	// Direct member plus a direct admin assignment: the maximum wins.
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'member')
	`, workspaceID, bothID); err != nil {
		t.Fatalf("insert both direct member: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO role_assignments (scope, scope_id, subject_type, subject_id, role)
		VALUES ('workspace', $1, 'user', $2, 'admin')
	`, workspaceID, bothID); err != nil {
		t.Fatalf("insert user assignment: %v", err)
	}

	effectiveRole := func(userID string) (string, error) {
		var role string
		err := db.QueryRow(ctx, `
			SELECT role FROM effective_workspace_members WHERE workspace_id = $1 AND user_id = $2
		`, workspaceID, userID).Scan(&role)
		return role, err
	}

	if role, err := effectiveRole(directID); err != nil || role != "member" {
		t.Fatalf("direct member role = %q (err=%v), want member", role, err)
	}
	if role, err := effectiveRole(groupAdminID); err != nil || role != "admin" {
		t.Fatalf("group-derived role = %q (err=%v), want admin", role, err)
	}
	if role, err := effectiveRole(bothID); err != nil || role != "admin" {
		t.Fatalf("combined role = %q (err=%v), want admin", role, err)
	}
	if _, err := effectiveRole(noneID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("unassigned user error = %v, want no rows", err)
	}
}
