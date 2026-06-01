//go:build integration

package savedfilters

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/database"
	"team-task-tracker/backend/internal/migrations"
)

func TestSavedFilterLifecycleIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newSavedFilterIntegrationDB(t, ctx)
	handler := NewHandler(db, nil)
	user, otherUser := seedSavedFilterIntegrationWorkspace(t, ctx, db)

	created, err := handler.createSavedFilter(ctx, user, normalizedCreateSavedFilter{
		Name: "My Work",
		Filters: map[string]string{
			"sort":      "created_desc",
			"status":    "todo",
			"projectId": testUUID,
		},
	})
	if err != nil {
		t.Fatalf("create saved filter: %v", err)
	}
	if created.Name != "My Work" {
		t.Fatalf("created name = %q, want My Work", created.Name)
	}
	if created.Filters["status"] != "todo" {
		t.Fatalf("created status = %q, want todo", created.Filters["status"])
	}

	if _, err := handler.createSavedFilter(ctx, user, normalizedCreateSavedFilter{
		Name:    "My Work",
		Filters: map[string]string{"sort": "created_desc"},
	}); !errors.Is(err, errSavedFilterExists) {
		t.Fatalf("duplicate create error = %v, want %v", err, errSavedFilterExists)
	}

	if _, err := handler.createSavedFilter(ctx, otherUser, normalizedCreateSavedFilter{
		Name:    "My Work",
		Filters: map[string]string{"sort": "created_desc"},
	}); err != nil {
		t.Fatalf("create same name for another user: %v", err)
	}

	userFilters, err := handler.listSavedFilters(ctx, user)
	if err != nil {
		t.Fatalf("list user filters: %v", err)
	}
	if len(userFilters) != 1 || userFilters[0].ID != created.ID {
		t.Fatalf("user filters = %#v, want created filter only", userFilters)
	}

	otherUserFilters, err := handler.listSavedFilters(ctx, otherUser)
	if err != nil {
		t.Fatalf("list other user filters: %v", err)
	}
	if len(otherUserFilters) != 1 || otherUserFilters[0].UserID != otherUser.ID {
		t.Fatalf("other user filters = %#v, want own filter only", otherUserFilters)
	}

	updated, err := handler.updateSavedFilter(ctx, user, created.ID, normalizedUpdateSavedFilter{
		Name:       "Critical Work",
		HasName:    true,
		Filters:    map[string]string{"sort": "priority_desc", "priority": "critical"},
		HasFilters: true,
	})
	if err != nil {
		t.Fatalf("update saved filter: %v", err)
	}
	if updated.Name != "Critical Work" {
		t.Fatalf("updated name = %q, want Critical Work", updated.Name)
	}
	if updated.Filters["priority"] != "critical" {
		t.Fatalf("updated priority = %q, want critical", updated.Filters["priority"])
	}

	if _, err := handler.updateSavedFilter(ctx, otherUser, created.ID, normalizedUpdateSavedFilter{
		Name:    "No Access",
		HasName: true,
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("cross-user update error = %v, want %v", err, pgx.ErrNoRows)
	}

	if err := handler.deleteSavedFilter(ctx, otherUser, created.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("cross-user delete error = %v, want %v", err, pgx.ErrNoRows)
	}

	if err := handler.deleteSavedFilter(ctx, user, created.ID); err != nil {
		t.Fatalf("delete saved filter: %v", err)
	}

	userFilters, err = handler.listSavedFilters(ctx, user)
	if err != nil {
		t.Fatalf("list user filters after delete: %v", err)
	}
	if len(userFilters) != 0 {
		t.Fatalf("user filters after delete = %#v, want empty", userFilters)
	}
}

func newSavedFilterIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://team_task_tracker:team_task_tracker@localhost:15432/team_task_tracker?sslmode=disable"
	}

	adminDB, err := database.Connect(ctx, databaseURL)
	if err != nil {
		t.Skipf("postgres is not available: %v", err)
	}
	t.Cleanup(adminDB.Close)

	schemaName := fmt.Sprintf("saved_filters_integration_%d", time.Now().UnixNano())
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

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database url: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schemaName
	cfg.MaxConns = 2

	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect to integration schema: %v", err)
	}
	t.Cleanup(db.Close)

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping integration database: %v", err)
	}

	if _, err := migrations.Up(ctx, db, "../../migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return db
}

func seedSavedFilterIntegrationWorkspace(t *testing.T, ctx context.Context, db *pgxpool.Pool) (auth.CurrentUser, auth.CurrentUser) {
	t.Helper()

	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ('Saved Filters Integration Workspace')
		RETURNING id::text
	`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}

	var userID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ('saved-filters@example.com', 'saved_filters', 'hash', 'Saved Filters')
		RETURNING id::text
	`).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var otherUserID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ('saved-filters-other@example.com', 'saved_filters_other', 'hash', 'Saved Filters Other')
		RETURNING id::text
	`).Scan(&otherUserID); err != nil {
		t.Fatalf("insert other user: %v", err)
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin'), ($1, $3, 'member')
	`, workspaceID, userID, otherUserID); err != nil {
		t.Fatalf("insert workspace members: %v", err)
	}

	return auth.CurrentUser{
			ID:          userID,
			Email:       "saved-filters@example.com",
			Username:    "saved_filters",
			DisplayName: "Saved Filters",
			WorkspaceID: workspaceID,
			Role:        "admin",
		}, auth.CurrentUser{
			ID:          otherUserID,
			Email:       "saved-filters-other@example.com",
			Username:    "saved_filters_other",
			DisplayName: "Saved Filters Other",
			WorkspaceID: workspaceID,
			Role:        "member",
		}
}
