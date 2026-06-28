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

func TestPostgresMigrationsCreateCoreSchema(t *testing.T) {
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

	schemaName := fmt.Sprintf("integration_%d", time.Now().UnixNano())
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
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("ping integration database: %v", err)
	}

	applied, err := migrations.Up(ctx, db, "../../migrations")
	if err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("expected migrations to be applied in isolated schema")
	}

	expectedTables := []string{
		"workspaces",
		"users",
		"workspace_members",
		"projects",
		"issues",
		"labels",
		"issue_labels",
		"comments",
		"issue_links",
		"sprints",
		"saved_filters",
		"notifications",
		"team_invites",
		"project_workflow_statuses",
		"project_workflow_transitions",
		"project_members",
		"automation_rules",
		"email_outbox",
		"password_reset_tokens",
		"sessions",
		"activity_log",
		"schema_migrations",
	}
	for _, tableName := range expectedTables {
		var exists bool
		if err := db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = $1
					AND table_name = $2
			)
		`, schemaName, tableName).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", tableName, err)
		}
		if !exists {
			t.Fatalf("expected table %s to exist", tableName)
		}
	}

	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ($1)
		RETURNING id::text
	`, "Integration Workspace").Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if workspaceID == "" {
		t.Fatal("expected generated workspace id")
	}

	var hasParentIssueID bool
	if err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = $1
				AND table_name = 'issues'
				AND column_name = 'parent_issue_id'
		)
	`, schemaName).Scan(&hasParentIssueID); err != nil {
		t.Fatalf("check parent_issue_id column: %v", err)
	}
	if !hasParentIssueID {
		t.Fatal("expected issues.parent_issue_id to exist")
	}

	var hasSprintID bool
	if err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = $1
				AND table_name = 'issues'
				AND column_name = 'sprint_id'
		)
	`, schemaName).Scan(&hasSprintID); err != nil {
		t.Fatalf("check sprint_id column: %v", err)
	}
	if !hasSprintID {
		t.Fatal("expected issues.sprint_id to exist")
	}

	var hasStoryPoints bool
	if err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = $1
				AND table_name = 'issues'
				AND column_name = 'story_points'
		)
	`, schemaName).Scan(&hasStoryPoints); err != nil {
		t.Fatalf("check story_points column: %v", err)
	}
	if !hasStoryPoints {
		t.Fatal("expected issues.story_points to exist")
	}

	var hasSavedFiltersFilters bool
	if err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = $1
				AND table_name = 'saved_filters'
				AND column_name = 'filters'
		)
	`, schemaName).Scan(&hasSavedFiltersFilters); err != nil {
		t.Fatalf("check saved_filters.filters column: %v", err)
	}
	if !hasSavedFiltersFilters {
		t.Fatal("expected saved_filters.filters to exist")
	}

	var hasTeamInvitesTokenHash bool
	if err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = $1
				AND table_name = 'team_invites'
				AND column_name = 'token_hash'
		)
	`, schemaName).Scan(&hasTeamInvitesTokenHash); err != nil {
		t.Fatalf("check team_invites.token_hash column: %v", err)
	}
	if !hasTeamInvitesTokenHash {
		t.Fatal("expected team_invites.token_hash to exist")
	}

	var hasRequiredWorkflowStatusID bool
	if err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = $1
				AND table_name = 'issues'
				AND column_name = 'workflow_status_id'
				AND is_nullable = 'NO'
		)
	`, schemaName).Scan(&hasRequiredWorkflowStatusID); err != nil {
		t.Fatalf("check required issues.workflow_status_id column: %v", err)
	}
	if !hasRequiredWorkflowStatusID {
		t.Fatal("expected required issues.workflow_status_id to exist")
	}

	expectedConstraints := []string{
		"project_workflow_statuses_id_project_unique",
		"project_workflow_statuses_project_key_unique",
		"project_workflow_statuses_key_valid",
		"project_workflow_statuses_name_valid",
		"project_workflow_statuses_color_valid",
		"project_workflow_statuses_category_valid",
		"project_workflow_statuses_position_valid",
		"project_workflow_transitions_from_status_fk",
		"project_workflow_transitions_to_status_fk",
		"project_workflow_transitions_not_self",
		"issues_workflow_status_project_fk",
		"issues_status_key_valid",
		"project_members_role_valid",
		"automation_rules_name_valid",
		"automation_rules_trigger_type_valid",
		"automation_rules_conditions_valid",
		"automation_rules_actions_valid",
		"automation_rules_position_valid",
		"email_outbox_email_type_valid",
		"email_outbox_recipient_email_valid",
		"email_outbox_template_data_valid",
		"email_outbox_status_valid",
		"email_outbox_attempt_count_valid",
		"email_outbox_deduplication_key_valid",
		"email_outbox_processing_started_at_valid",
		"email_outbox_sent_at_valid",
		"password_reset_tokens_hash_valid",
		"password_reset_tokens_request_ip_hash_valid",
		"password_reset_tokens_user_agent_valid",
		"password_reset_tokens_expires_after_created",
	}
	for _, constraintName := range expectedConstraints {
		var exists bool
		if err := db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_constraint constraint_record
				JOIN pg_namespace namespace
					ON namespace.oid = constraint_record.connamespace
				WHERE namespace.nspname = $1
					AND constraint_record.conname = $2
			)
		`, schemaName, constraintName).Scan(&exists); err != nil {
			t.Fatalf("check constraint %s: %v", constraintName, err)
		}
		if !exists {
			t.Fatalf("expected constraint %s to exist", constraintName)
		}
	}

	expectedIndexes := []string{
		"idx_issues_project_created_id",
		"idx_issues_project_due_created_id",
		"idx_issues_project_priority_created_id",
		"idx_issues_sprint_status_created_id",
		"idx_notifications_workspace_user_created_id",
		"idx_notifications_unread_created_id",
		"idx_activity_log_issue_created_id",
		"idx_issue_labels_label_issue",
		"idx_project_workflow_statuses_active_name",
		"idx_project_workflow_statuses_project_position",
		"idx_project_workflow_statuses_project_category",
		"idx_project_workflow_transitions_from_status",
		"idx_project_workflow_transitions_to_status",
		"idx_issues_workflow_status_id",
		"idx_issues_project_workflow_status_created_id",
		"idx_issues_sprint_workflow_status_created_id",
		"idx_project_members_user_id",
		"idx_project_members_project_role",
		"idx_automation_rules_project_position",
		"idx_automation_rules_enabled_trigger",
		"idx_automation_rules_created_by",
		"idx_email_outbox_deduplication_key",
		"idx_email_outbox_claim",
		"idx_email_outbox_stale_processing",
		"idx_email_outbox_workspace_status",
		"idx_password_reset_tokens_token_hash",
		"idx_password_reset_tokens_user_created",
		"idx_password_reset_tokens_active_user",
		"idx_password_reset_tokens_expiry",
	}
	for _, indexName := range expectedIndexes {
		var exists bool
		if err := db.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_indexes
				WHERE schemaname = $1
					AND indexname = $2
			)
		`, schemaName, indexName).Scan(&exists); err != nil {
			t.Fatalf("check index %s: %v", indexName, err)
		}
		if !exists {
			t.Fatalf("expected index %s to exist", indexName)
		}
	}

	var userID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, "integration@example.com", "integration", "hash", "Integration User").Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	for _, notificationType := range []string{"issue_automation_assigned", "issue_automation_status_changed"} {
		if _, err := db.Exec(ctx, `
			INSERT INTO notifications (workspace_id, user_id, notification_type)
			VALUES ($1, $2, $3)
		`, workspaceID, userID, notificationType); err != nil {
			t.Fatalf("insert %s notification: %v", notificationType, err)
		}
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO notifications (workspace_id, user_id, notification_type)
		VALUES ($1, $2, 'invalid_type')
	`, workspaceID, userID); err == nil {
		t.Fatal("expected invalid notification type to be rejected")
	}

	var projectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, workspaceID, "INT", "Integration Project", userID).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	var workflowStatusCount int
	var workflowTransitionCount int
	if err := db.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM project_workflow_statuses WHERE project_id = $1),
			(SELECT count(*) FROM project_workflow_transitions WHERE project_id = $1)
	`, projectID).Scan(&workflowStatusCount, &workflowTransitionCount); err != nil {
		t.Fatalf("count default project workflow: %v", err)
	}
	if workflowStatusCount != 5 || workflowTransitionCount != 20 {
		t.Fatalf(
			"default workflow counts = statuses:%d transitions:%d, want statuses:5 transitions:20",
			workflowStatusCount,
			workflowTransitionCount,
		)
	}

	rows, err := db.Query(ctx, `
		SELECT key, name, category, color, position
		FROM project_workflow_statuses
		WHERE project_id = $1
		ORDER BY position
	`, projectID)
	if err != nil {
		t.Fatalf("query default workflow statuses: %v", err)
	}
	defer rows.Close()

	defaultWorkflowStatuses := make([]string, 0, 5)
	for rows.Next() {
		var key string
		var name string
		var category string
		var color string
		var position int
		if err := rows.Scan(&key, &name, &category, &color, &position); err != nil {
			t.Fatalf("scan default workflow status: %v", err)
		}
		defaultWorkflowStatuses = append(
			defaultWorkflowStatuses,
			fmt.Sprintf("%s|%s|%s|%s|%d", key, name, category, color, position),
		)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate default workflow statuses: %v", err)
	}

	expectedDefaultWorkflowStatuses := []string{
		"backlog|Backlog|backlog|#64748b|100",
		"todo|Todo|todo|#3b82f6|200",
		"in_progress|In progress|in_progress|#f59e0b|300",
		"blocked|Blocked|in_progress|#dc2626|400",
		"done|Done|done|#16a34a|500",
	}
	if fmt.Sprint(defaultWorkflowStatuses) != fmt.Sprint(expectedDefaultWorkflowStatuses) {
		t.Fatalf(
			"default workflow statuses = %v, want %v",
			defaultWorkflowStatuses,
			expectedDefaultWorkflowStatuses,
		)
	}

	var epicID string
	if err := db.QueryRow(ctx, `
		INSERT INTO issues (
			project_id,
			number,
			issue_key,
			title,
			issue_type,
			status,
			priority,
			reporter_id
		)
		VALUES ($1, 1, 'INT-1', 'Integration epic', 'epic', 'todo', 'medium', $2)
		RETURNING id::text
	`, projectID, userID).Scan(&epicID); err != nil {
		t.Fatalf("insert epic issue: %v", err)
	}

	var epicWorkflowStatusKey string
	if err := db.QueryRow(ctx, `
		SELECT workflow_status.key
		FROM issues issue
		JOIN project_workflow_statuses workflow_status
			ON workflow_status.id = issue.workflow_status_id
		WHERE issue.id = $1
	`, epicID).Scan(&epicWorkflowStatusKey); err != nil {
		t.Fatalf("load epic workflow status: %v", err)
	}
	if epicWorkflowStatusKey != "todo" {
		t.Fatalf("epic workflow status key = %q, want todo", epicWorkflowStatusKey)
	}

	if err := db.QueryRow(ctx, `
		UPDATE issues
		SET status = 'done'
		WHERE id = $1
		RETURNING (
			SELECT workflow_status.key
			FROM project_workflow_statuses workflow_status
			WHERE workflow_status.id = issues.workflow_status_id
		)
	`, epicID).Scan(&epicWorkflowStatusKey); err != nil {
		t.Fatalf("sync updated epic workflow status: %v", err)
	}
	if epicWorkflowStatusKey != "done" {
		t.Fatalf("updated epic workflow status key = %q, want done", epicWorkflowStatusKey)
	}

	var subtaskID string
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
			parent_issue_id
		)
		VALUES ($1, 2, 'INT-2', 'Integration subtask', 'subtask', 'todo', 'medium', $2, $3)
		RETURNING id::text
	`, projectID, userID, epicID).Scan(&subtaskID); err != nil {
		t.Fatalf("insert subtask issue: %v", err)
	}
	if subtaskID == "" {
		t.Fatal("expected generated subtask id")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO issues (
			project_id,
			number,
			issue_key,
			title,
			issue_type,
			status,
			priority,
			reporter_id
		)
		VALUES ($1, 3, 'INT-3', 'Invalid subtask', 'subtask', 'todo', 'medium', $2)
	`, projectID, userID); err == nil {
		t.Fatal("expected subtask without parent to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO issues (
			project_id,
			number,
			issue_key,
			title,
			issue_type,
			status,
			priority,
			reporter_id,
			parent_issue_id
		)
		VALUES ($1, 4, 'INT-4', 'Invalid epic', 'epic', 'todo', 'medium', $2, $3)
	`, projectID, userID, epicID); err == nil {
		t.Fatal("expected epic with parent to fail")
	}

	var linkID string
	if err := db.QueryRow(ctx, `
		INSERT INTO issue_links (source_issue_id, target_issue_id, link_type, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, epicID, subtaskID, "relates", userID).Scan(&linkID); err != nil {
		t.Fatalf("insert issue link: %v", err)
	}
	if linkID == "" {
		t.Fatal("expected generated issue link id")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO issue_links (source_issue_id, target_issue_id, link_type, created_by)
		VALUES ($1, $2, $3, $4)
	`, epicID, subtaskID, "relates", userID); err == nil {
		t.Fatal("expected duplicate issue link to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO issue_links (source_issue_id, target_issue_id, link_type, created_by)
		VALUES ($1, $2, $3, $4)
	`, subtaskID, epicID, "relates", userID); err == nil {
		t.Fatal("expected inverse relates issue link to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO issue_links (source_issue_id, target_issue_id, link_type, created_by)
		VALUES ($1, $2, $3, $4)
	`, epicID, epicID, "blocks", userID); err == nil {
		t.Fatal("expected self issue link to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO issue_links (source_issue_id, target_issue_id, link_type, created_by)
		VALUES ($1, $2, $3, $4)
	`, subtaskID, epicID, "duplicates", userID); err == nil {
		t.Fatal("expected invalid issue link type to fail")
	}

	var activeSprintID string
	if err := db.QueryRow(ctx, `
		INSERT INTO sprints (
			workspace_id,
			project_id,
			name,
			goal,
			status,
			start_date,
			end_date,
			created_by
		)
		VALUES ($1, $2, 'Integration Sprint', 'Ship sprint schema', 'active', '2026-06-01', '2026-06-14', $3)
		RETURNING id::text
	`, workspaceID, projectID, userID).Scan(&activeSprintID); err != nil {
		t.Fatalf("insert active sprint: %v", err)
	}
	if activeSprintID == "" {
		t.Fatal("expected generated active sprint id")
	}

	var issueSprintID string
	if err := db.QueryRow(ctx, `
		UPDATE issues
		SET sprint_id = $1
		WHERE id = $2
		RETURNING sprint_id::text
	`, activeSprintID, epicID).Scan(&issueSprintID); err != nil {
		t.Fatalf("assign issue to sprint: %v", err)
	}
	if issueSprintID != activeSprintID {
		t.Fatalf("issue sprint id = %q, want %q", issueSprintID, activeSprintID)
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO sprints (
			workspace_id,
			project_id,
			name,
			status,
			start_date,
			end_date,
			created_by
		)
		VALUES ($1, $2, 'Second active sprint', 'active', '2026-06-15', '2026-06-28', $3)
	`, workspaceID, projectID, userID); err == nil {
		t.Fatal("expected second active sprint in one project to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO sprints (workspace_id, project_id, name, status, created_by)
		VALUES ($1, $2, 'Invalid status sprint', 'paused', $3)
	`, workspaceID, projectID, userID); err == nil {
		t.Fatal("expected invalid sprint status to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO sprints (workspace_id, project_id, name, status, created_by)
		VALUES ($1, $2, '   ', 'planned', $3)
	`, workspaceID, projectID, userID); err == nil {
		t.Fatal("expected blank sprint name to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO sprints (
			workspace_id,
			project_id,
			name,
			status,
			start_date,
			end_date,
			created_by
		)
		VALUES ($1, $2, 'Invalid dates sprint', 'planned', '2026-06-14', '2026-06-01', $3)
	`, workspaceID, projectID, userID); err == nil {
		t.Fatal("expected sprint with end_date before start_date to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO sprints (workspace_id, project_id, name, status, created_by)
		VALUES ($1, $2, 'Incomplete completed sprint', 'completed', $3)
	`, workspaceID, projectID, userID); err == nil {
		t.Fatal("expected completed sprint without completed_at to fail")
	}

	var completedSprintID string
	if err := db.QueryRow(ctx, `
		INSERT INTO sprints (
			workspace_id,
			project_id,
			name,
			status,
			created_by,
			completed_at
		)
		VALUES ($1, $2, 'Completed Sprint', 'completed', $3, now())
		RETURNING id::text
	`, workspaceID, projectID, userID).Scan(&completedSprintID); err != nil {
		t.Fatalf("insert completed sprint: %v", err)
	}
	if completedSprintID == "" {
		t.Fatal("expected generated completed sprint id")
	}

	var otherWorkspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ('Other Integration Workspace')
		RETURNING id::text
	`).Scan(&otherWorkspaceID); err != nil {
		t.Fatalf("insert other workspace: %v", err)
	}

	var otherProjectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'OTH', 'Other Project', $2)
		RETURNING id::text
	`, otherWorkspaceID, userID).Scan(&otherProjectID); err != nil {
		t.Fatalf("insert other project: %v", err)
	}

	var projectStatusID string
	var otherProjectStatusID string
	if err := db.QueryRow(ctx, `
		SELECT id::text
		FROM project_workflow_statuses
		WHERE project_id = $1
			AND key = 'todo'
	`, projectID).Scan(&projectStatusID); err != nil {
		t.Fatalf("load project workflow status: %v", err)
	}
	if err := db.QueryRow(ctx, `
		SELECT id::text
		FROM project_workflow_statuses
		WHERE project_id = $1
			AND key = 'done'
	`, otherProjectID).Scan(&otherProjectStatusID); err != nil {
		t.Fatalf("load other project workflow status: %v", err)
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO project_workflow_transitions (
			project_id,
			from_status_id,
			to_status_id
		)
		VALUES ($1, $2, $3)
	`, projectID, projectStatusID, otherProjectStatusID); err == nil {
		t.Fatal("expected cross-project workflow transition to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO project_workflow_transitions (
			project_id,
			from_status_id,
			to_status_id
		)
		VALUES ($1, $2, $2)
	`, projectID, projectStatusID); err == nil {
		t.Fatal("expected self workflow transition to fail")
	}

	invalidWorkflowStatuses := []struct {
		name     string
		key      string
		color    string
		category string
	}{
		{name: "invalid key", key: "Invalid-Key", color: "#123456", category: "todo"},
		{name: "invalid color", key: "invalid_color", color: "red", category: "todo"},
		{name: "invalid category", key: "invalid_category", color: "#123456", category: "paused"},
	}
	for _, invalidStatus := range invalidWorkflowStatuses {
		if _, err := db.Exec(ctx, `
			INSERT INTO project_workflow_statuses (
				project_id,
				key,
				name,
				color,
				category,
				position
			)
			VALUES ($1, $2, $3, $4, $5, 900)
		`, projectID, invalidStatus.key, invalidStatus.name, invalidStatus.color, invalidStatus.category); err == nil {
			t.Fatalf("expected %s workflow status to fail", invalidStatus.name)
		}
	}

	if _, err := db.Exec(ctx, `
		UPDATE project_workflow_statuses
		SET key = 'renamed'
		WHERE id = $1
	`, projectStatusID); err == nil {
		t.Fatal("expected workflow status key update to fail")
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO sprints (workspace_id, project_id, name, status, created_by)
		VALUES ($1, $2, 'Cross workspace sprint', 'planned', $3)
	`, workspaceID, otherProjectID, userID); err == nil {
		t.Fatal("expected sprint project/workspace mismatch to fail")
	}
}

func TestProjectWorkflowMigrationBackfillsLegacyIssues(t *testing.T) {
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

	schemaName := fmt.Sprintf("workflow_upgrade_%d", time.Now().UnixNano())
	quotedSchemaName := pgx.Identifier{schemaName}.Sanitize()
	if _, err := adminDB.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`); err != nil {
		t.Fatalf("ensure pgcrypto extension: %v", err)
	}
	if _, err := adminDB.Exec(ctx, `CREATE SCHEMA `+quotedSchemaName); err != nil {
		t.Fatalf("create workflow upgrade schema: %v", err)
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
		t.Fatalf("connect to workflow upgrade schema: %v", err)
	}
	defer db.Close()

	legacyMigrationsDir := t.TempDir()
	migrationEntries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	for _, entry := range migrationEntries {
		if entry.IsDir() || entry.Name() >= "000011_" {
			continue
		}

		contents, err := os.ReadFile(filepath.Join("../../migrations", entry.Name()))
		if err != nil {
			t.Fatalf("read legacy migration %s: %v", entry.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(legacyMigrationsDir, entry.Name()), contents, 0o600); err != nil {
			t.Fatalf("copy legacy migration %s: %v", entry.Name(), err)
		}
	}

	applied, err := migrations.Up(ctx, db, legacyMigrationsDir)
	if err != nil {
		t.Fatalf("apply legacy migrations: %v", err)
	}
	if len(applied) != 10 {
		t.Fatalf("legacy migrations applied = %d, want 10", len(applied))
	}

	var workspaceID string
	var userID string
	var projectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ('Legacy Workflow Workspace')
		RETURNING id::text
	`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert legacy workspace: %v", err)
	}
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ('legacy-workflow@example.com', 'legacy_workflow', 'hash', 'Legacy Workflow')
		RETURNING id::text
	`).Scan(&userID); err != nil {
		t.Fatalf("insert legacy user: %v", err)
	}
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'LEG', 'Legacy Project', $2)
		RETURNING id::text
	`, workspaceID, userID).Scan(&projectID); err != nil {
		t.Fatalf("insert legacy project: %v", err)
	}

	legacyStatuses := []string{"backlog", "todo", "in_progress", "blocked", "done"}
	for index, status := range legacyStatuses {
		if _, err := db.Exec(ctx, `
			INSERT INTO issues (
				project_id,
				number,
				issue_key,
				title,
				issue_type,
				status,
				priority,
				reporter_id
			)
			VALUES ($1, $2, $3, $4, 'task', $5, 'medium', $6)
		`, projectID, index+1, fmt.Sprintf("LEG-%d", index+1), "Legacy "+status, status, userID); err != nil {
			t.Fatalf("insert legacy issue with status %s: %v", status, err)
		}
	}

	applied, err = migrations.Up(ctx, db, "../../migrations")
	if err != nil {
		t.Fatalf("apply workflow migration: %v", err)
	}
	if len(applied) == 0 || applied[0].Version != 11 {
		t.Fatalf("post-legacy migrations applied = %#v, want to start at version 11", applied)
	}

	var workflowStatuses int
	var workflowTransitions int
	var syncedIssues int
	if err := db.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM project_workflow_statuses WHERE project_id = $1),
			(SELECT count(*) FROM project_workflow_transitions WHERE project_id = $1),
			(
				SELECT count(*)
				FROM issues issue
				JOIN project_workflow_statuses workflow_status
					ON workflow_status.id = issue.workflow_status_id
				WHERE issue.project_id = $1
					AND issue.status = workflow_status.key
			)
	`, projectID).Scan(&workflowStatuses, &workflowTransitions, &syncedIssues); err != nil {
		t.Fatalf("check workflow backfill: %v", err)
	}
	if workflowStatuses != 5 || workflowTransitions != 20 || syncedIssues != 5 {
		t.Fatalf(
			"backfill counts = statuses:%d transitions:%d synced issues:%d, want 5/20/5",
			workflowStatuses,
			workflowTransitions,
			syncedIssues,
		)
	}

	applied, err = migrations.Up(ctx, db, "../../migrations")
	if err != nil {
		t.Fatalf("repeat workflow migration: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("repeat workflow migrations applied = %#v, want none", applied)
	}

	var newProjectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'NEW', 'Post Migration Project', $2)
		RETURNING id::text
	`, workspaceID, userID).Scan(&newProjectID); err != nil {
		t.Fatalf("insert post-migration project: %v", err)
	}

	if err := db.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM project_workflow_statuses WHERE project_id = $1),
			(SELECT count(*) FROM project_workflow_transitions WHERE project_id = $1)
	`, newProjectID).Scan(&workflowStatuses, &workflowTransitions); err != nil {
		t.Fatalf("check post-migration project workflow: %v", err)
	}
	if workflowStatuses != 5 || workflowTransitions != 20 {
		t.Fatalf(
			"post-migration workflow counts = statuses:%d transitions:%d, want 5/20",
			workflowStatuses,
			workflowTransitions,
		)
	}

	var postMigrationIssueID string
	if err := db.QueryRow(ctx, `
		INSERT INTO issues (
			project_id,
			number,
			issue_key,
			title,
			issue_type,
			status,
			priority,
			reporter_id
		)
		VALUES ($1, 1, 'NEW-1', 'Post migration issue', 'task', 'blocked', 'medium', $2)
		RETURNING id::text
	`, newProjectID, userID).Scan(&postMigrationIssueID); err != nil {
		t.Fatalf("insert post-migration issue: %v", err)
	}

	var workflowStatusKey string
	if err := db.QueryRow(ctx, `
		SELECT workflow_status.key
		FROM issues issue
		JOIN project_workflow_statuses workflow_status
			ON workflow_status.id = issue.workflow_status_id
		WHERE issue.id = $1
	`, postMigrationIssueID).Scan(&workflowStatusKey); err != nil {
		t.Fatalf("load post-migration issue workflow status: %v", err)
	}
	if workflowStatusKey != "blocked" {
		t.Fatalf("post-migration issue workflow status = %q, want blocked", workflowStatusKey)
	}

	if err := db.QueryRow(ctx, `
		UPDATE issues
		SET status = 'done'
		WHERE id = $1
		RETURNING (
			SELECT workflow_status.key
			FROM project_workflow_statuses workflow_status
			WHERE workflow_status.id = issues.workflow_status_id
		)
	`, postMigrationIssueID).Scan(&workflowStatusKey); err != nil {
		t.Fatalf("update post-migration issue workflow status: %v", err)
	}
	if workflowStatusKey != "done" {
		t.Fatalf("updated post-migration issue workflow status = %q, want done", workflowStatusKey)
	}
}
