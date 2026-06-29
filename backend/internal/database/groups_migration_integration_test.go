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

func TestGroupsMigrationCreatesTablesWithConstraints(t *testing.T) {
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
	schemaName := fmt.Sprintf("groups_migration_%d", time.Now().UnixNano())
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
	if err := db.QueryRow(ctx, `
		INSERT INTO organizations (name, slug, status)
		VALUES ('Cascade Org', 'cascade-org', 'active')
		RETURNING id::text
	`).Scan(&orgID); err != nil {
		t.Fatalf("insert organization: %v", err)
	}
	var userID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name, is_active)
		VALUES ('grp@example.com', 'grp_user', 'hash', 'Group User', true)
		RETURNING id::text
	`).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var groupID string
	if err := db.QueryRow(ctx, `
		INSERT INTO groups (organization_id, name, description, created_by)
		VALUES ($1, 'Engineers', 'Builds things', $2)
		RETURNING id::text
	`, orgID, userID).Scan(&groupID); err != nil {
		t.Fatalf("insert group: %v", err)
	}

	// Group names are unique within an organization.
	if _, err := db.Exec(ctx, `
		INSERT INTO groups (organization_id, name) VALUES ($1, 'Engineers')
	`, orgID); err == nil {
		t.Fatal("expected a unique-violation for a duplicate group name in the organization")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO group_members (group_id, user_id) VALUES ($1, $2)
	`, groupID, userID); err != nil {
		t.Fatalf("insert group member: %v", err)
	}

	// Deleting the organization cascades to its groups and their members.
	if _, err := db.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, orgID); err != nil {
		t.Fatalf("delete organization: %v", err)
	}
	var groupCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM groups WHERE id = $1`, groupID).Scan(&groupCount); err != nil {
		t.Fatalf("count groups: %v", err)
	}
	if groupCount != 0 {
		t.Fatalf("group count after org delete = %d, want 0", groupCount)
	}
	var memberCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM group_members WHERE group_id = $1`, groupID).Scan(&memberCount); err != nil {
		t.Fatalf("count group members: %v", err)
	}
	if memberCount != 0 {
		t.Fatalf("group member count after org delete = %d, want 0", memberCount)
	}
}
