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

func TestSiteAdminMigrationPromotesExistingWorkspaceAdmins(t *testing.T) {
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
	schemaName := fmt.Sprintf("site_admin_upgrade_%d", time.Now().UnixNano())
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

	// Apply every migration before the site admin migration.
	legacyDir := t.TempDir()
	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() >= "000020_" {
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
		t.Fatalf("apply legacy migrations: %v", err)
	}

	var workspaceID string
	if err := db.QueryRow(ctx, `INSERT INTO workspaces (name) VALUES ('Site Admin Upgrade') RETURNING id::text`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	insertUser := func(email, username string) string {
		var id string
		if err := db.QueryRow(ctx, `
			INSERT INTO users (email, username, password_hash, display_name, is_active)
			VALUES ($1, $2, 'hash', $2, true)
			RETURNING id::text
		`, email, username).Scan(&id); err != nil {
			t.Fatalf("insert user %s: %v", username, err)
		}
		return id
	}
	adminID := insertUser("site-admin@example.com", "site_admin")
	memberID := insertUser("site-member@example.com", "site_member")
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin'), ($1, $3, 'member')
	`, workspaceID, adminID, memberID); err != nil {
		t.Fatalf("insert workspace members: %v", err)
	}

	applied, err := migrations.Up(ctx, db, "../../migrations")
	if err != nil {
		t.Fatalf("apply site admin migration: %v", err)
	}
	hasSiteAdminMigration := false
	for _, migration := range applied {
		if migration.Version == 20 {
			hasSiteAdminMigration = true
			break
		}
	}
	if !hasSiteAdminMigration {
		t.Fatalf("applied migrations = %#v, want the site admin migration (version 20)", applied)
	}

	var adminIsSiteAdmin bool
	var memberIsSiteAdmin bool
	if err := db.QueryRow(ctx, `
		SELECT
			(SELECT is_site_admin FROM users WHERE id = $1),
			(SELECT is_site_admin FROM users WHERE id = $2)
	`, adminID, memberID).Scan(&adminIsSiteAdmin, &memberIsSiteAdmin); err != nil {
		t.Fatalf("read site admin backfill: %v", err)
	}
	if !adminIsSiteAdmin {
		t.Fatal("workspace admin was not promoted to site admin")
	}
	if memberIsSiteAdmin {
		t.Fatal("workspace member was unexpectedly promoted to site admin")
	}
}
