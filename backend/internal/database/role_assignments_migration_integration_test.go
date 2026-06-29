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

	"kelmio/backend/internal/migrations"
)

func TestRoleAssignmentsMigrationCreatesTableWithUniqueness(t *testing.T) {
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
	schemaName := fmt.Sprintf("role_assignments_migration_%d", time.Now().UnixNano())
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
		VALUES ('Assignments WS', $1, 'assignments-ws', 'active')
		RETURNING id::text
	`, orgID).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	var groupID string
	if err := db.QueryRow(ctx, `
		INSERT INTO groups (organization_id, name) VALUES ($1, 'Engineers')
		RETURNING id::text
	`, orgID).Scan(&groupID); err != nil {
		t.Fatalf("insert group: %v", err)
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO role_assignments (scope, scope_id, subject_type, subject_id, role)
		VALUES ('workspace', $1, 'group', $2, 'member')
	`, workspaceID, groupID); err != nil {
		t.Fatalf("insert role assignment: %v", err)
	}

	// The same subject cannot be assigned twice within the same scope.
	if _, err := db.Exec(ctx, `
		INSERT INTO role_assignments (scope, scope_id, subject_type, subject_id, role)
		VALUES ('workspace', $1, 'group', $2, 'admin')
	`, workspaceID, groupID); err == nil {
		t.Fatal("expected a unique-violation for a duplicate subject assignment in the same scope")
	}

	// A check constraint guards the scope and subject_type vocabularies.
	if _, err := db.Exec(ctx, `
		INSERT INTO role_assignments (scope, scope_id, subject_type, subject_id, role)
		VALUES ('galaxy', $1, 'group', $2, 'member')
	`, workspaceID, groupID); err == nil {
		t.Fatal("expected a check-constraint violation for an unknown scope")
	}
}
