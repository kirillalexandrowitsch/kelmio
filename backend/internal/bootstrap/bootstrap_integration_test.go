//go:build integration

package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"team-task-tracker/backend/internal/database"
	"team-task-tracker/backend/internal/migrations"
)

func TestBootstrapCreatesFirstAdminAndRejectsRepeat(t *testing.T) {
	ctx, db := newBootstrapIntegrationDB(t)
	service := NewService(db)
	cfg := Config{
		WorkspaceName:    "Production Workspace",
		AdminEmail:       "admin@example.com",
		AdminUsername:    "production_admin",
		AdminDisplayName: "Production Admin",
		AdminPassword:    "secure-password",
	}

	result, err := service.Bootstrap(ctx, cfg)
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if result.WorkspaceID == "" || result.AdminUserID == "" {
		t.Fatalf("Bootstrap() result = %#v", result)
	}

	var (
		workspaceName string
		email         string
		username      string
		displayName   string
		passwordHash  string
		role          string
		isActive      bool
	)
	if err := db.QueryRow(ctx, `
		SELECT w.name, u.email, u.username, u.display_name, u.password_hash, u.is_active, wm.role
		FROM workspace_members wm
		JOIN workspaces w ON w.id = wm.workspace_id
		JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = $1 AND wm.user_id = $2
	`, result.WorkspaceID, result.AdminUserID).Scan(
		&workspaceName,
		&email,
		&username,
		&displayName,
		&passwordHash,
		&isActive,
		&role,
	); err != nil {
		t.Fatalf("load bootstrapped admin: %v", err)
	}
	if workspaceName != cfg.WorkspaceName || email != cfg.AdminEmail || username != cfg.AdminUsername || displayName != cfg.AdminDisplayName {
		t.Fatalf("unexpected bootstrapped values: %q %q %q %q", workspaceName, email, username, displayName)
	}
	if !isActive || role != "admin" {
		t.Fatalf("bootstrapped admin active=%t role=%q", isActive, role)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(cfg.AdminPassword)); err != nil {
		t.Fatalf("bootstrapped password mismatch: %v", err)
	}

	repeatCfg := cfg
	repeatCfg.AdminPassword = "replacement-password"
	if _, err := service.Bootstrap(ctx, repeatCfg); !errors.Is(err, ErrDatabaseNotEmpty) {
		t.Fatalf("repeat Bootstrap() error = %v, want ErrDatabaseNotEmpty", err)
	}
	var passwordHashAfterRepeat string
	if err := db.QueryRow(ctx, `SELECT password_hash FROM users WHERE id = $1`, result.AdminUserID).Scan(&passwordHashAfterRepeat); err != nil {
		t.Fatalf("load password hash after repeat bootstrap: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHashAfterRepeat), []byte(cfg.AdminPassword)); err != nil {
		t.Fatalf("original password hash changed after repeat bootstrap: %v", err)
	}
	assertCoreCounts(t, ctx, db, 1, 1, 1)
}

func TestBootstrapRejectsAnyExistingCoreData(t *testing.T) {
	ctx, db := newBootstrapIntegrationDB(t)
	if _, err := db.Exec(ctx, `INSERT INTO workspaces (name) VALUES ('Existing Workspace')`); err != nil {
		t.Fatalf("insert existing workspace: %v", err)
	}

	_, err := NewService(db).Bootstrap(ctx, Config{
		WorkspaceName:    "Production Workspace",
		AdminEmail:       "admin@example.com",
		AdminUsername:    "production_admin",
		AdminDisplayName: "Production Admin",
		AdminPassword:    "secure-password",
	})
	if !errors.Is(err, ErrDatabaseNotEmpty) {
		t.Fatalf("Bootstrap() error = %v, want ErrDatabaseNotEmpty", err)
	}

	var workspaceName string
	if err := db.QueryRow(ctx, `SELECT name FROM workspaces`).Scan(&workspaceName); err != nil {
		t.Fatalf("load existing workspace after bootstrap refusal: %v", err)
	}
	if workspaceName != "Existing Workspace" {
		t.Fatalf("existing workspace name = %q", workspaceName)
	}
	assertCoreCounts(t, ctx, db, 1, 0, 0)
}

func newBootstrapIntegrationDB(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://team_task_tracker:team_task_tracker@localhost:15432/team_task_tracker?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	adminDB, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	t.Cleanup(adminDB.Close)

	schemaName := fmt.Sprintf("bootstrap_integration_%d", time.Now().UnixNano())
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

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database url: %v", err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schemaName
	poolConfig.MaxConns = 2

	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("connect to integration schema: %v", err)
	}
	t.Cleanup(db.Close)
	if _, err := migrations.Up(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return ctx, db
}

func assertCoreCounts(t *testing.T, ctx context.Context, db *pgxpool.Pool, wantWorkspaces int, wantUsers int, wantMembers int) {
	t.Helper()

	var workspaces, users, members int
	if err := db.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM workspaces),
			(SELECT count(*) FROM users),
			(SELECT count(*) FROM workspace_members)
	`).Scan(&workspaces, &users, &members); err != nil {
		t.Fatalf("load core counts: %v", err)
	}
	if workspaces != wantWorkspaces || users != wantUsers || members != wantMembers {
		t.Fatalf(
			"core counts = workspaces:%d users:%d members:%d, want workspaces:%d users:%d members:%d",
			workspaces,
			users,
			members,
			wantWorkspaces,
			wantUsers,
			wantMembers,
		)
	}
}
