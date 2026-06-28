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

func TestOrganizationMigrationBackfillsExistingWorkspace(t *testing.T) {
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
	schemaName := fmt.Sprintf("organization_upgrade_%d", time.Now().UnixNano())
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

	// Apply every migration before the organization migration to reproduce a
	// legacy single-workspace database.
	legacyDir := t.TempDir()
	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() >= "000018_" {
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
	if err := db.QueryRow(ctx, `INSERT INTO workspaces (name) VALUES ('V6 Tenancy') RETURNING id::text`).Scan(&workspaceID); err != nil {
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
	adminID := insertUser("v6-admin@example.com", "v6_admin")
	memberID := insertUser("v6-member@example.com", "v6_member")
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin'), ($1, $3, 'member')
	`, workspaceID, adminID, memberID); err != nil {
		t.Fatalf("insert workspace members: %v", err)
	}

	applied, err := migrations.Up(ctx, db, "../../migrations")
	if err != nil {
		t.Fatalf("apply organization migration: %v", err)
	}
	if len(applied) == 0 || applied[0].Version != 18 {
		t.Fatalf("organization migrations applied = %#v, want to start at version 18", applied)
	}

	var organizationID string
	var orgStatus string
	if err := db.QueryRow(ctx, `
		SELECT id::text, status FROM organizations WHERE slug = 'default'
	`).Scan(&organizationID, &orgStatus); err != nil {
		t.Fatalf("default organization not created: %v", err)
	}
	if orgStatus != "active" {
		t.Fatalf("default organization status = %q, want active", orgStatus)
	}

	var workspaceOrgID string
	var workspaceSlug string
	var workspaceStatus string
	if err := db.QueryRow(ctx, `
		SELECT organization_id::text, slug, status FROM workspaces WHERE id = $1
	`, workspaceID).Scan(&workspaceOrgID, &workspaceSlug, &workspaceStatus); err != nil {
		t.Fatalf("read migrated workspace: %v", err)
	}
	if workspaceOrgID != organizationID {
		t.Fatalf("workspace organization_id = %q, want %q", workspaceOrgID, organizationID)
	}
	if workspaceSlug != "v6-tenancy" {
		t.Fatalf("workspace slug = %q, want v6-tenancy", workspaceSlug)
	}
	if workspaceStatus != "active" {
		t.Fatalf("workspace status = %q, want active", workspaceStatus)
	}

	var adminRole string
	var memberRole string
	if err := db.QueryRow(ctx, `
		SELECT
			(SELECT role FROM organization_members WHERE organization_id = $1 AND user_id = $2),
			(SELECT role FROM organization_members WHERE organization_id = $1 AND user_id = $3)
	`, organizationID, adminID, memberID).Scan(&adminRole, &memberRole); err != nil {
		t.Fatalf("check organization membership backfill: %v", err)
	}
	if adminRole != "org_admin" || memberRole != "org_member" {
		t.Fatalf("organization membership = admin:%q member:%q, want org_admin/org_member", adminRole, memberRole)
	}
}
