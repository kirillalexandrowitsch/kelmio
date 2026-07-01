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

func TestAuditLogMigrationNullsReferencesOnDelete(t *testing.T) {
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
	schemaName := fmt.Sprintf("audit_log_migration_%d", time.Now().UnixNano())
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
		INSERT INTO organizations (name, slug, status) VALUES ('Audit Org', 'audit-org', 'active')
		RETURNING id::text
	`).Scan(&orgID); err != nil {
		t.Fatalf("insert organization: %v", err)
	}
	var actorID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name, is_active)
		VALUES ('auditor@example.com', 'auditor', 'hash', 'Auditor', true)
		RETURNING id::text
	`).Scan(&actorID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var entryID string
	if err := db.QueryRow(ctx, `
		INSERT INTO audit_log (organization_id, actor_id, action, target_type, target_id, metadata)
		VALUES ($1, $2, 'organization.created', 'organization', $1, '{"k":"v"}'::jsonb)
		RETURNING id::text
	`, orgID, actorID).Scan(&entryID); err != nil {
		t.Fatalf("insert audit entry: %v", err)
	}

	// Deleting the actor and the organization preserves the audit row but nulls
	// the references.
	if _, err := db.Exec(ctx, `DELETE FROM users WHERE id = $1`, actorID); err != nil {
		t.Fatalf("delete actor: %v", err)
	}
	if _, err := db.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, orgID); err != nil {
		t.Fatalf("delete organization: %v", err)
	}

	var actorIsNull, orgIsNull bool
	var action string
	if err := db.QueryRow(ctx, `
		SELECT actor_id IS NULL, organization_id IS NULL, action
		FROM audit_log WHERE id = $1
	`, entryID).Scan(&actorIsNull, &orgIsNull, &action); err != nil {
		t.Fatalf("read audit entry: %v", err)
	}
	if !actorIsNull || !orgIsNull {
		t.Fatalf("references after delete: actorNull=%v orgNull=%v, want both null", actorIsNull, orgIsNull)
	}
	if action != "organization.created" {
		t.Fatalf("action = %q, want organization.created", action)
	}
}
