//go:build integration

package notifications

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
	"team-task-tracker/backend/internal/pagination"
)

func TestNotificationServiceIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newNotificationIntegrationDB(t, ctx)
	service := NewService()
	seed := seedNotificationIntegrationWorkspace(t, ctx, db)

	if err := service.NotifyIssueAssigned(ctx, db, seed.workspaceID, seed.actor.ID, seed.issue, seed.assignee.ID); err != nil {
		t.Fatalf("notify assignment: %v", err)
	}

	if err := service.NotifyIssueComment(ctx, db, seed.workspaceID, seed.actor.ID, seed.issue.ID, seed.commentID, "Please check @mentioned_user and @assignee_user"); err != nil {
		t.Fatalf("notify comment: %v", err)
	}

	if err := service.NotifySprintEvent(ctx, db, seed.workspaceID, seed.actor.ID, seed.sprint, TypeSprintStarted); err != nil {
		t.Fatalf("notify sprint started: %v", err)
	}

	assigneeNotifications, err := service.List(ctx, db, seed.assignee)
	if err != nil {
		t.Fatalf("list assignee notifications: %v", err)
	}
	if !hasNotificationType(assigneeNotifications, TypeIssueAssigned) {
		t.Fatalf("assignee notifications missing assignment: %#v", assigneeNotifications)
	}
	if !hasNotificationType(assigneeNotifications, TypeIssueMentioned) {
		t.Fatalf("assignee notifications missing mention: %#v", assigneeNotifications)
	}
	if hasDuplicateCommentNotification(assigneeNotifications, seed.commentID) {
		t.Fatalf("assignee received duplicate comment notification: %#v", assigneeNotifications)
	}

	firstPage, nextCursor, err := service.ListPage(ctx, db, seed.assignee, pagination.Params{Limit: 1})
	if err != nil {
		t.Fatalf("list first assignee notification page: %v", err)
	}
	if len(firstPage) != 1 {
		t.Fatalf("first assignee notification page len = %d, want 1", len(firstPage))
	}
	if nextCursor == nil {
		t.Fatal("expected assignee notification next cursor")
	}
	nextOffset, err := pagination.DecodeCursor(*nextCursor)
	if err != nil {
		t.Fatalf("decode assignee notification next cursor: %v", err)
	}
	secondPage, _, err := service.ListPage(ctx, db, seed.assignee, pagination.Params{Limit: 1, Offset: nextOffset})
	if err != nil {
		t.Fatalf("list second assignee notification page: %v", err)
	}
	if len(secondPage) != 1 {
		t.Fatalf("second assignee notification page len = %d, want 1", len(secondPage))
	}
	if firstPage[0].ID == secondPage[0].ID {
		t.Fatalf("notification %s appeared on both pages", firstPage[0].ID)
	}

	reporterNotifications, err := service.List(ctx, db, seed.reporter)
	if err != nil {
		t.Fatalf("list reporter notifications: %v", err)
	}
	if !hasNotificationType(reporterNotifications, TypeIssueCommented) {
		t.Fatalf("reporter notifications missing direct comment: %#v", reporterNotifications)
	}

	actorCount, err := service.UnreadCount(ctx, db, seed.actor)
	if err != nil {
		t.Fatalf("actor unread count: %v", err)
	}
	if actorCount != 0 {
		t.Fatalf("actor unread count = %d, want 0", actorCount)
	}

	unreadCount, err := service.UnreadCount(ctx, db, seed.assignee)
	if err != nil {
		t.Fatalf("assignee unread count: %v", err)
	}
	if unreadCount == 0 {
		t.Fatal("expected assignee unread notifications")
	}

	readNotification, err := service.MarkRead(ctx, db, seed.assignee, assigneeNotifications[0].ID)
	if err != nil {
		t.Fatalf("mark read: %v", err)
	}
	if readNotification.ReadAt == nil {
		t.Fatal("expected read_at after mark read")
	}

	if _, err := service.MarkRead(ctx, db, seed.reporter, assigneeNotifications[0].ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("cross-user mark read error = %v, want %v", err, pgx.ErrNoRows)
	}

	if err := service.MarkAllRead(ctx, db, seed.assignee); err != nil {
		t.Fatalf("mark all read: %v", err)
	}
	unreadCount, err = service.UnreadCount(ctx, db, seed.assignee)
	if err != nil {
		t.Fatalf("assignee unread count after mark all: %v", err)
	}
	if unreadCount != 0 {
		t.Fatalf("assignee unread count after mark all = %d, want 0", unreadCount)
	}
}

type notificationSeed struct {
	workspaceID string
	actor       auth.CurrentUser
	reporter    auth.CurrentUser
	assignee    auth.CurrentUser
	mentioned   auth.CurrentUser
	issue       IssueContext
	sprint      SprintContext
	commentID   string
}

func newNotificationIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

	schemaName := fmt.Sprintf("notifications_integration_%d", time.Now().UnixNano())
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

func seedNotificationIntegrationWorkspace(t *testing.T, ctx context.Context, db *pgxpool.Pool) notificationSeed {
	t.Helper()

	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ('Notifications Integration Workspace')
		RETURNING id::text
	`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}

	actor := insertNotificationUser(t, ctx, db, workspaceID, "actor_user", true)
	reporter := insertNotificationUser(t, ctx, db, workspaceID, "reporter_user", true)
	assignee := insertNotificationUser(t, ctx, db, workspaceID, "assignee_user", true)
	mentioned := insertNotificationUser(t, ctx, db, workspaceID, "mentioned_user", true)
	_ = insertNotificationUser(t, ctx, db, workspaceID, "inactive_user", false)

	var projectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'NTF', 'Notifications Project', $2)
		RETURNING id::text
	`, workspaceID, actor.ID).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	var issueID string
	if err := db.QueryRow(ctx, `
		INSERT INTO issues (
			project_id,
			number,
			issue_key,
			title,
			issue_type,
			status,
			priority,
			reporter_id,
			assignee_id
		)
		VALUES ($1, 1, 'NTF-1', 'Notification issue', 'task', 'todo', 'medium', $2, $3)
		RETURNING id::text
	`, projectID, reporter.ID, assignee.ID).Scan(&issueID); err != nil {
		t.Fatalf("insert issue: %v", err)
	}

	var sprintID string
	if err := db.QueryRow(ctx, `
		INSERT INTO sprints (workspace_id, project_id, name, created_by)
		VALUES ($1, $2, 'Notification Sprint', $3)
		RETURNING id::text
	`, workspaceID, projectID, actor.ID).Scan(&sprintID); err != nil {
		t.Fatalf("insert sprint: %v", err)
	}

	var commentID string
	if err := db.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&commentID); err != nil {
		t.Fatalf("generate comment id: %v", err)
	}

	return notificationSeed{
		workspaceID: workspaceID,
		actor:       actor,
		reporter:    reporter,
		assignee:    assignee,
		mentioned:   mentioned,
		commentID:   commentID,
		issue: IssueContext{
			ID:         issueID,
			IssueKey:   "NTF-1",
			Title:      "Notification issue",
			ReporterID: reporter.ID,
			AssigneeID: assignee.ID,
		},
		sprint: SprintContext{
			ID:         sprintID,
			Name:       "Notification Sprint",
			ProjectKey: "NTF",
		},
	}
}

func insertNotificationUser(t *testing.T, ctx context.Context, db *pgxpool.Pool, workspaceID string, username string, active bool) auth.CurrentUser {
	t.Helper()

	var userID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name, is_active)
		VALUES ($1, $2, 'hash', $3, $4)
		RETURNING id::text
	`, username+"@example.com", username, username, active).Scan(&userID); err != nil {
		t.Fatalf("insert user %s: %v", username, err)
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'member')
	`, workspaceID, userID); err != nil {
		t.Fatalf("insert workspace member %s: %v", username, err)
	}

	return auth.CurrentUser{
		ID:          userID,
		Username:    username,
		DisplayName: username,
		WorkspaceID: workspaceID,
		Role:        "member",
	}
}

func hasNotificationType(notifications []Notification, notificationType string) bool {
	for _, notification := range notifications {
		if notification.NotificationType == notificationType {
			return true
		}
	}

	return false
}

func hasDuplicateCommentNotification(notifications []Notification, commentID string) bool {
	count := 0
	for _, notification := range notifications {
		if notification.Payload["comment_id"] == commentID {
			count++
		}
	}

	return count > 1
}
