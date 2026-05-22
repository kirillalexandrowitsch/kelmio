package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"team-task-tracker/backend/internal/config"
	"team-task-tracker/backend/internal/database"
)

type seedConfig struct {
	WorkspaceName string
	AdminEmail    string
	AdminUsername string
	AdminPassword string
	AdminName     string
	DemoEmail     string
	DemoUsername  string
	DemoPassword  string
	DemoName      string
}

type demoIssue struct {
	Number        int
	Title         string
	Description   string
	IssueType     string
	Status        string
	Priority      string
	AssigneeID    string
	DueOffsetDays *int
	LabelIDs      []string
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := config.Load()
	seed := seedConfig{
		WorkspaceName: env("SEED_WORKSPACE_NAME", "Local Workspace"),
		AdminEmail:    env("SEED_ADMIN_EMAIL", "admin@example.com"),
		AdminUsername: env("SEED_ADMIN_USERNAME", "admin"),
		AdminPassword: env("SEED_ADMIN_PASSWORD", "admin12345"),
		AdminName:     env("SEED_ADMIN_NAME", "Admin"),
		DemoEmail:     env("SEED_DEMO_EMAIL", "demo.member@example.com"),
		DemoUsername:  env("SEED_DEMO_USERNAME", "demo_member"),
		DemoPassword:  env("SEED_DEMO_PASSWORD", "demo12345"),
		DemoName:      env("SEED_DEMO_NAME", "Demo Member"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	tx, err := db.Begin(ctx)
	if err != nil {
		logger.Error("begin seed transaction failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	workspaceID, err := ensureWorkspace(ctx, tx, seed.WorkspaceName)
	if err != nil {
		logger.Error("ensure workspace failed", "error", err)
		os.Exit(1)
	}

	adminID, err := ensureUser(ctx, tx, workspaceID, seed.AdminEmail, seed.AdminUsername, seed.AdminPassword, seed.AdminName, "admin")
	if err != nil {
		logger.Error("ensure admin user failed", "error", err)
		os.Exit(1)
	}

	demoMemberID, err := ensureUser(ctx, tx, workspaceID, seed.DemoEmail, seed.DemoUsername, seed.DemoPassword, seed.DemoName, "member")
	if err != nil {
		logger.Error("ensure demo member failed", "error", err)
		os.Exit(1)
	}

	projectID, err := ensureProject(ctx, tx, workspaceID, adminID)
	if err != nil {
		logger.Error("ensure demo project failed", "error", err)
		os.Exit(1)
	}

	frontendLabelID, err := ensureLabel(ctx, tx, workspaceID, "frontend", "#4e795d")
	if err != nil {
		logger.Error("ensure frontend label failed", "error", err)
		os.Exit(1)
	}
	backendLabelID, err := ensureLabel(ctx, tx, workspaceID, "backend", "#3662a1")
	if err != nil {
		logger.Error("ensure backend label failed", "error", err)
		os.Exit(1)
	}
	bugLabelID, err := ensureLabel(ctx, tx, workspaceID, "bug", "#923c2d")
	if err != nil {
		logger.Error("ensure bug label failed", "error", err)
		os.Exit(1)
	}

	overdue := -1
	dueSoon := 2
	later := 5
	issues := []demoIssue{
		{
			Number:      1,
			Title:       "Set up project structure",
			Description: "Initial repository, Docker Compose, backend, frontend, and database foundation.",
			IssueType:   "task",
			Status:      "done",
			Priority:    "medium",
			AssigneeID:  adminID,
			LabelIDs:    []string{backendLabelID},
		},
		{
			Number:        2,
			Title:         "Build issue list filters",
			Description:   "Add filters by project, status, priority, assignee, label, due date, and search query.",
			IssueType:     "story",
			Status:        "in_progress",
			Priority:      "high",
			AssigneeID:    demoMemberID,
			DueOffsetDays: &dueSoon,
			LabelIDs:      []string{frontendLabelID},
		},
		{
			Number:        3,
			Title:         "Polish empty states",
			Description:   "Make empty project, label, issue, comment, and activity states clear for first-time users.",
			IssueType:     "task",
			Status:        "todo",
			Priority:      "medium",
			DueOffsetDays: &later,
			LabelIDs:      []string{frontendLabelID},
		},
		{
			Number:        4,
			Title:         "Fix login validation message",
			Description:   "Show a clear validation error when login credentials are missing or invalid.",
			IssueType:     "bug",
			Status:        "blocked",
			Priority:      "critical",
			AssigneeID:    adminID,
			DueOffsetDays: &overdue,
			LabelIDs:      []string{bugLabelID},
		},
	}

	for _, issue := range issues {
		issueID, err := ensureIssue(ctx, tx, projectID, adminID, issue)
		if err != nil {
			logger.Error("ensure demo issue failed", "issue", issue.Title, "error", err)
			os.Exit(1)
		}

		if issue.Status != "todo" {
			if err := ensureIssueActivity(ctx, tx, issueID, adminID, "status_changed", map[string]string{
				"from_status": "todo",
				"to_status":   issue.Status,
			}); err != nil {
				logger.Error("ensure status activity failed", "issue", issue.Title, "error", err)
				os.Exit(1)
			}
		}

		if issue.Number == 2 {
			if err := ensureComment(ctx, tx, issueID, demoMemberID, "I started this one and will keep the board status updated."); err != nil {
				logger.Error("ensure demo comment failed", "error", err)
				os.Exit(1)
			}
			if err := ensureIssueActivity(ctx, tx, issueID, demoMemberID, "comment_added", map[string]string{
				"preview": "I started this one and will keep the board status updated.",
			}); err != nil {
				logger.Error("ensure comment activity failed", "error", err)
				os.Exit(1)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error("commit seed transaction failed", "error", err)
		os.Exit(1)
	}

	logger.Info(
		"seed data ready",
		"workspace", seed.WorkspaceName,
		"admin_email", seed.AdminEmail,
		"admin_username", seed.AdminUsername,
		"demo_username", seed.DemoUsername,
		"demo_project", "DEMO",
	)
}

func ensureWorkspace(ctx context.Context, tx pgx.Tx, name string) (string, error) {
	var workspaceID string
	err := tx.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		SELECT $1
		WHERE NOT EXISTS (
			SELECT 1 FROM workspaces WHERE name = $1
		)
		RETURNING id::text
	`, name).Scan(&workspaceID)
	if err == nil {
		return workspaceID, nil
	}

	if err := tx.QueryRow(ctx, `
		SELECT id::text FROM workspaces WHERE name = $1
	`, name).Scan(&workspaceID); err != nil {
		return "", err
	}

	return workspaceID, nil
}

func ensureUser(
	ctx context.Context,
	tx pgx.Tx,
	workspaceID string,
	email string,
	username string,
	password string,
	displayName string,
	role string,
) (string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	var userID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name, is_active)
		VALUES ($1, $2, $3, $4, true)
		ON CONFLICT (email) DO UPDATE SET
			username = EXCLUDED.username,
			password_hash = EXCLUDED.password_hash,
			display_name = EXCLUDED.display_name,
			is_active = true
		RETURNING id::text
	`, email, username, string(passwordHash), displayName).Scan(&userID); err != nil {
		return "", err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (workspace_id, user_id) DO UPDATE SET role = EXCLUDED.role
	`, workspaceID, userID, role); err != nil {
		return "", err
	}

	return userID, nil
}

func ensureProject(ctx context.Context, tx pgx.Tx, workspaceID string, createdBy string) (string, error) {
	var projectID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, description, created_by)
		VALUES ($1, 'DEMO', 'Demo Project', 'Seeded project with tasks for local V1 testing.', $2)
		ON CONFLICT (workspace_id, key) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			archived_at = NULL
		RETURNING id::text
	`, workspaceID, createdBy).Scan(&projectID); err != nil {
		return "", err
	}

	return projectID, nil
}

func ensureLabel(ctx context.Context, tx pgx.Tx, workspaceID string, name string, color string) (string, error) {
	var labelID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO labels (workspace_id, name, color)
		VALUES ($1, $2, $3)
		ON CONFLICT (workspace_id, name) DO UPDATE SET color = EXCLUDED.color
		RETURNING id::text
	`, workspaceID, name, color).Scan(&labelID); err != nil {
		return "", err
	}

	return labelID, nil
}

func ensureIssue(ctx context.Context, tx pgx.Tx, projectID string, reporterID string, issue demoIssue) (string, error) {
	var assigneeID any
	if issue.AssigneeID != "" {
		assigneeID = issue.AssigneeID
	}

	var dueOffset any
	if issue.DueOffsetDays != nil {
		dueOffset = *issue.DueOffsetDays
	}

	issueKey := fmt.Sprintf("DEMO-%d", issue.Number)
	var issueID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO issues (
			project_id,
			number,
			issue_key,
			title,
			description,
			issue_type,
			status,
			priority,
			reporter_id,
			assignee_id,
			due_date
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			CASE WHEN $11::integer IS NULL THEN NULL ELSE CURRENT_DATE + $11::integer END
		)
		ON CONFLICT (issue_key) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			issue_type = EXCLUDED.issue_type,
			status = EXCLUDED.status,
			priority = EXCLUDED.priority,
			reporter_id = EXCLUDED.reporter_id,
			assignee_id = EXCLUDED.assignee_id,
			due_date = EXCLUDED.due_date,
			updated_at = now(),
			archived_at = NULL
		RETURNING id::text
	`, projectID, issue.Number, issueKey, issue.Title, issue.Description, issue.IssueType, issue.Status, issue.Priority, reporterID, assigneeID, dueOffset).Scan(&issueID); err != nil {
		return "", err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM issue_labels WHERE issue_id = $1`, issueID); err != nil {
		return "", err
	}

	for _, labelID := range issue.LabelIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO issue_labels (issue_id, label_id)
			VALUES ($1, $2)
			ON CONFLICT (issue_id, label_id) DO NOTHING
		`, issueID, labelID); err != nil {
			return "", err
		}
	}

	if err := ensureIssueActivity(ctx, tx, issueID, reporterID, "issue_created", map[string]string{
		"issue_key": issueKey,
		"title":     issue.Title,
		"status":    issue.Status,
		"priority":  issue.Priority,
	}); err != nil {
		return "", err
	}

	return issueID, nil
}

func ensureComment(ctx context.Context, tx pgx.Tx, issueID string, authorID string, body string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO comments (issue_id, author_id, body)
		SELECT $1, $2, $3
		WHERE NOT EXISTS (
			SELECT 1
			FROM comments
			WHERE issue_id = $1
				AND author_id = $2
				AND body = $3
		)
	`, issueID, authorID, body)

	return err
}

func ensureIssueActivity(
	ctx context.Context,
	tx pgx.Tx,
	issueID string,
	actorID string,
	action string,
	payload map[string]string,
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO activity_log (entity_type, entity_id, action, actor_id, payload)
		SELECT 'issue', $1, $2, $3, $4::jsonb
		WHERE NOT EXISTS (
			SELECT 1
			FROM activity_log
			WHERE entity_type = 'issue'
				AND entity_id = $1
				AND action = $2
		)
	`, issueID, action, actorID, string(payloadJSON))

	return err
}

func env(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
