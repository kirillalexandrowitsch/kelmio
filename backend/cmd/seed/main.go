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
	Number            int
	Title             string
	Description       string
	IssueType         string
	Status            string
	Priority          string
	StoryPoints       int
	AssigneeID        string
	ParentIssueNumber int
	DueOffsetDays     *int
	LabelIDs          []string
}

type demoSprint struct {
	Name            string
	Goal            string
	Status          string
	StartOffsetDays int
	EndOffsetDays   int
}

type demoIssueLink struct {
	SourceIssueNumber int
	TargetIssueNumber int
	LinkType          string
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg := config.MustLoad()
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
	nextWeek := 7
	nextSprint := 10
	issues := []demoIssue{
		{
			Number:      1,
			Title:       "Set up project structure",
			Description: "Initial repository, Docker Compose, backend, frontend, and database foundation.",
			IssueType:   "task",
			Status:      "done",
			Priority:    "medium",
			StoryPoints: 3,
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
			StoryPoints:   5,
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
			StoryPoints:   2,
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
			StoryPoints:   3,
			AssigneeID:    adminID,
			DueOffsetDays: &overdue,
			LabelIDs:      []string{bugLabelID},
		},
		{
			Number:      5,
			Title:       "Launch V2 planning workflow",
			Description: "Coordinate hierarchy, sprint planning, saved views, and notifications for the V2 localhost release.",
			IssueType:   "epic",
			Status:      "in_progress",
			Priority:    "high",
			StoryPoints: 13,
			AssigneeID:  adminID,
			LabelIDs:    []string{backendLabelID, frontendLabelID},
		},
		{
			Number:            6,
			Title:             "Build sprint planning experience",
			Description:       "Connect backlog planning, active sprint board, story points, and sprint summary into one daily workflow.",
			IssueType:         "story",
			Status:            "in_progress",
			Priority:          "high",
			StoryPoints:       8,
			AssigneeID:        demoMemberID,
			ParentIssueNumber: 5,
			DueOffsetDays:     &dueSoon,
			LabelIDs:          []string{frontendLabelID},
		},
		{
			Number:            7,
			Title:             "Document V2 operating scenarios",
			Description:       "Prepare scenario notes for hierarchy, sprints, saved filters, and notifications before the README update.",
			IssueType:         "story",
			Status:            "todo",
			Priority:          "medium",
			StoryPoints:       5,
			AssigneeID:        adminID,
			ParentIssueNumber: 5,
			DueOffsetDays:     &nextSprint,
			LabelIDs:          []string{backendLabelID},
		},
		{
			Number:            8,
			Title:             "Add saved planning view copy",
			Description:       "Create a focused subtask that demonstrates task decomposition under the sprint planning story.",
			IssueType:         "subtask",
			Status:            "todo",
			Priority:          "medium",
			StoryPoints:       2,
			AssigneeID:        demoMemberID,
			ParentIssueNumber: 6,
			DueOffsetDays:     &nextWeek,
			LabelIDs:          []string{frontendLabelID},
		},
		{
			Number:        9,
			Title:         "Resolve notification blocker",
			Description:   "A seeded blocked bug that demonstrates links, notifications, priority sorting, and active sprint blockers.",
			IssueType:     "bug",
			Status:        "blocked",
			Priority:      "critical",
			StoryPoints:   3,
			AssigneeID:    adminID,
			DueOffsetDays: &overdue,
			LabelIDs:      []string{bugLabelID, backendLabelID},
		},
		{
			Number:        10,
			Title:         "Prepare V2 README walkthrough",
			Description:   "Backlog task reserved for the documentation pass after smoke and e2e scenarios are extended.",
			IssueType:     "task",
			Status:        "backlog",
			Priority:      "low",
			StoryPoints:   1,
			AssigneeID:    adminID,
			DueOffsetDays: &nextSprint,
			LabelIDs:      []string{backendLabelID},
		},
	}

	issueIDs := make(map[int]string, len(issues))
	for _, issue := range issues {
		parentIssueID := ""
		if issue.ParentIssueNumber != 0 {
			parentID, ok := issueIDs[issue.ParentIssueNumber]
			if !ok {
				logger.Error("parent demo issue was not seeded", "issue", issue.Title, "parent_number", issue.ParentIssueNumber)
				os.Exit(1)
			}
			parentIssueID = parentID
		}

		issueID, err := ensureIssue(ctx, tx, projectID, adminID, issue, parentIssueID)
		if err != nil {
			logger.Error("ensure demo issue failed", "issue", issue.Title, "error", err)
			os.Exit(1)
		}
		issueIDs[issue.Number] = issueID

		if issue.Status != "todo" {
			if err := ensureIssueActivity(ctx, tx, issueID, adminID, "status_changed", map[string]string{
				"from_status": "todo",
				"to_status":   issue.Status,
			}); err != nil {
				logger.Error("ensure status activity failed", "issue", issue.Title, "error", err)
				os.Exit(1)
			}
		}

		if issue.AssigneeID != "" {
			if err := ensureIssueActivity(ctx, tx, issueID, adminID, "assignee_changed", map[string]string{
				"from_assignee_id": "",
				"to_assignee_id":   issue.AssigneeID,
			}); err != nil {
				logger.Error("ensure assignee activity failed", "issue", issue.Title, "error", err)
				os.Exit(1)
			}
		}

		if parentIssueID != "" {
			if err := ensureIssueActivity(ctx, tx, issueID, adminID, "issue_parent_changed", map[string]string{
				"from_parent_issue_id": "",
				"to_parent_issue_id":   parentIssueID,
			}); err != nil {
				logger.Error("ensure parent activity failed", "issue", issue.Title, "error", err)
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

	activeSprintID, err := ensureSprint(ctx, tx, workspaceID, projectID, adminID, demoSprint{
		Name:            "Demo Active Sprint",
		Goal:            "Ship the core V2 planning workflow on localhost.",
		Status:          "active",
		StartOffsetDays: -1,
		EndOffsetDays:   nextWeek,
	})
	if err != nil {
		logger.Error("ensure active sprint failed", "error", err)
		os.Exit(1)
	}

	nextSprintID, err := ensureSprint(ctx, tx, workspaceID, projectID, adminID, demoSprint{
		Name:            "Demo Next Sprint",
		Goal:            "Prepare documentation and follow-up polish after V2 validation.",
		Status:          "planned",
		StartOffsetDays: nextSprint,
		EndOffsetDays:   nextSprint + 7,
	})
	if err != nil {
		logger.Error("ensure next sprint failed", "error", err)
		os.Exit(1)
	}

	completedSprintID, err := ensureSprint(ctx, tx, workspaceID, projectID, adminID, demoSprint{
		Name:            "Demo Completed Sprint",
		Goal:            "Reference completed work for sprint history and summary states.",
		Status:          "completed",
		StartOffsetDays: -14,
		EndOffsetDays:   -7,
	})
	if err != nil {
		logger.Error("ensure completed sprint failed", "error", err)
		os.Exit(1)
	}

	for _, assignment := range []struct {
		IssueNumber int
		SprintID    string
	}{
		{IssueNumber: 1, SprintID: completedSprintID},
		{IssueNumber: 2, SprintID: activeSprintID},
		{IssueNumber: 3},
		{IssueNumber: 4},
		{IssueNumber: 5},
		{IssueNumber: 6, SprintID: activeSprintID},
		{IssueNumber: 7, SprintID: nextSprintID},
		{IssueNumber: 8, SprintID: activeSprintID},
		{IssueNumber: 9, SprintID: activeSprintID},
		{IssueNumber: 10, SprintID: nextSprintID},
	} {
		if err := ensureIssueSprint(ctx, tx, issueIDs[assignment.IssueNumber], assignment.SprintID); err != nil {
			logger.Error("ensure issue sprint failed", "issue_number", assignment.IssueNumber, "error", err)
			os.Exit(1)
		}
	}

	for _, link := range []demoIssueLink{
		{SourceIssueNumber: 9, TargetIssueNumber: 6, LinkType: "blocks"},
		{SourceIssueNumber: 7, TargetIssueNumber: 6, LinkType: "relates"},
	} {
		if err := ensureIssueLink(ctx, tx, issueIDs[link.SourceIssueNumber], issueIDs[link.TargetIssueNumber], adminID, link); err != nil {
			logger.Error("ensure issue link failed", "source_number", link.SourceIssueNumber, "target_number", link.TargetIssueNumber, "error", err)
			os.Exit(1)
		}
	}

	if err := ensureComment(ctx, tx, issueIDs[6], adminID, "@demo_member Can you verify the active sprint planning flow before QA?"); err != nil {
		logger.Error("ensure V2 mention comment failed", "error", err)
		os.Exit(1)
	}
	if err := ensureIssueActivity(ctx, tx, issueIDs[6], adminID, "comment_added", map[string]string{
		"preview": "@demo_member Can you verify the active sprint planning flow before QA?",
	}); err != nil {
		logger.Error("ensure V2 mention comment activity failed", "error", err)
		os.Exit(1)
	}

	if err := ensureComment(ctx, tx, issueIDs[9], demoMemberID, "This blocker still affects the active sprint notification path."); err != nil {
		logger.Error("ensure V2 blocker comment failed", "error", err)
		os.Exit(1)
	}
	if err := ensureIssueActivity(ctx, tx, issueIDs[9], demoMemberID, "comment_added", map[string]string{
		"preview": "This blocker still affects the active sprint notification path.",
	}); err != nil {
		logger.Error("ensure V2 blocker comment activity failed", "error", err)
		os.Exit(1)
	}

	if err := ensureSavedFilter(ctx, tx, workspaceID, adminID, "Active sprint blockers", map[string]string{
		"sort":     "priority_desc",
		"sprintId": activeSprintID,
		"status":   "blocked",
	}); err != nil {
		logger.Error("ensure admin blocker saved filter failed", "error", err)
		os.Exit(1)
	}
	if err := ensureSavedFilter(ctx, tx, workspaceID, adminID, "My V2 planning work", map[string]string{
		"sort":       "due_date_asc",
		"projectId":  projectID,
		"assigneeId": adminID,
	}); err != nil {
		logger.Error("ensure admin planning saved filter failed", "error", err)
		os.Exit(1)
	}
	if err := ensureSavedFilter(ctx, tx, workspaceID, demoMemberID, "My active sprint", map[string]string{
		"sort":       "created_desc",
		"sprintId":   activeSprintID,
		"assigneeId": demoMemberID,
	}); err != nil {
		logger.Error("ensure demo member saved filter failed", "error", err)
		os.Exit(1)
	}

	if err := resetSeedNotifications(ctx, tx, workspaceID); err != nil {
		logger.Error("reset seed notifications failed", "error", err)
		os.Exit(1)
	}
	if err := ensureNotification(ctx, tx, workspaceID, adminID, demoMemberID, issueIDs[9], "issue_commented", map[string]string{
		"seed_key": "v2_seed:admin:issue_commented:demo-9",
		"message":  "commented on your issue",
		"preview":  "This blocker still affects the active sprint notification path.",
	}); err != nil {
		logger.Error("ensure admin comment notification failed", "error", err)
		os.Exit(1)
	}
	if err := ensureNotification(ctx, tx, workspaceID, adminID, demoMemberID, "", "sprint_started", map[string]string{
		"seed_key":    "v2_seed:admin:sprint_started:active",
		"message":     "started a sprint",
		"sprint_id":   activeSprintID,
		"sprint_name": "Demo Active Sprint",
		"project_key": "DEMO",
	}); err != nil {
		logger.Error("ensure admin sprint notification failed", "error", err)
		os.Exit(1)
	}
	if err := ensureNotification(ctx, tx, workspaceID, demoMemberID, adminID, issueIDs[6], "issue_assigned", map[string]string{
		"seed_key": "v2_seed:demo_member:issue_assigned:demo-6",
		"message":  "assigned you to an issue",
	}); err != nil {
		logger.Error("ensure demo member assignment notification failed", "error", err)
		os.Exit(1)
	}
	if err := ensureNotification(ctx, tx, workspaceID, demoMemberID, adminID, issueIDs[6], "issue_mentioned", map[string]string{
		"seed_key": "v2_seed:demo_member:issue_mentioned:demo-6",
		"message":  "mentioned you in a comment",
		"preview":  "@demo_member Can you verify the active sprint planning flow before QA?",
	}); err != nil {
		logger.Error("ensure demo member mention notification failed", "error", err)
		os.Exit(1)
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
		VALUES ($1, 'DEMO', 'Demo Project', 'Seeded project with tasks for local V1 and V2 testing.', $2)
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

func ensureIssue(ctx context.Context, tx pgx.Tx, projectID string, reporterID string, issue demoIssue, parentIssueID string) (string, error) {
	var assigneeID any
	if issue.AssigneeID != "" {
		assigneeID = issue.AssigneeID
	}

	var parentID any
	if parentIssueID != "" {
		parentID = parentIssueID
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
			story_points,
			reporter_id,
			assignee_id,
			parent_issue_id,
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
			$11,
			$12::uuid,
			CASE WHEN $13::integer IS NULL THEN NULL ELSE CURRENT_DATE + $13::integer END
		)
		ON CONFLICT (issue_key) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			issue_type = EXCLUDED.issue_type,
			status = EXCLUDED.status,
			priority = EXCLUDED.priority,
			story_points = EXCLUDED.story_points,
			reporter_id = EXCLUDED.reporter_id,
			assignee_id = EXCLUDED.assignee_id,
			parent_issue_id = EXCLUDED.parent_issue_id,
			due_date = EXCLUDED.due_date,
			updated_at = now(),
			archived_at = NULL
		RETURNING id::text
	`, projectID, issue.Number, issueKey, issue.Title, issue.Description, issue.IssueType, issue.Status, issue.Priority, issue.StoryPoints, reporterID, assigneeID, parentID, dueOffset).Scan(&issueID); err != nil {
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

func ensureSprint(ctx context.Context, tx pgx.Tx, workspaceID string, projectID string, createdBy string, sprint demoSprint) (string, error) {
	var sprintID string
	err := tx.QueryRow(ctx, `
		UPDATE sprints
		SET goal = $4,
			status = $5,
			start_date = CURRENT_DATE + $6::integer,
			end_date = CURRENT_DATE + $7::integer,
			completed_at = CASE
				WHEN $5::text = 'completed' THEN COALESCE(completed_at, now())
				ELSE NULL
			END
		WHERE workspace_id = $1
			AND project_id = $2
			AND name = $3
		RETURNING id::text
	`, workspaceID, projectID, sprint.Name, sprint.Goal, sprint.Status, sprint.StartOffsetDays, sprint.EndOffsetDays).Scan(&sprintID)
	if err == nil {
		return sprintID, nil
	}
	if err != pgx.ErrNoRows {
		return "", err
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO sprints (
			workspace_id,
			project_id,
			name,
			goal,
			status,
			start_date,
			end_date,
			created_by,
			completed_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			CURRENT_DATE + $6::integer,
			CURRENT_DATE + $7::integer,
			$8,
			CASE WHEN $5::text = 'completed' THEN now() ELSE NULL END
		)
		RETURNING id::text
	`, workspaceID, projectID, sprint.Name, sprint.Goal, sprint.Status, sprint.StartOffsetDays, sprint.EndOffsetDays, createdBy).Scan(&sprintID); err != nil {
		return "", err
	}

	return sprintID, nil
}

func ensureIssueSprint(ctx context.Context, tx pgx.Tx, issueID string, sprintID string) error {
	var currentSprintID any
	if sprintID != "" {
		currentSprintID = sprintID
	}

	_, err := tx.Exec(ctx, `
		UPDATE issues
		SET sprint_id = $1::uuid,
			updated_at = now()
		WHERE id = $2
	`, currentSprintID, issueID)

	return err
}

func ensureIssueLink(ctx context.Context, tx pgx.Tx, sourceIssueID string, targetIssueID string, createdBy string, link demoIssueLink) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO issue_links (source_issue_id, target_issue_id, link_type, created_by)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (
			SELECT 1
			FROM issue_links
			WHERE link_type = $3
				AND (
					(source_issue_id = $1 AND target_issue_id = $2)
					OR ($3 = 'relates' AND source_issue_id = $2 AND target_issue_id = $1)
				)
		)
		ON CONFLICT DO NOTHING
	`, sourceIssueID, targetIssueID, link.LinkType, createdBy)
	if err != nil {
		return err
	}

	return ensureIssueActivity(ctx, tx, sourceIssueID, createdBy, "issue_link_created", map[string]string{
		"source_issue_key": fmt.Sprintf("DEMO-%d", link.SourceIssueNumber),
		"target_issue_key": fmt.Sprintf("DEMO-%d", link.TargetIssueNumber),
		"link_type":        link.LinkType,
	})
}

func ensureSavedFilter(ctx context.Context, tx pgx.Tx, workspaceID string, userID string, name string, filters map[string]string) error {
	filtersJSON, err := json.Marshal(filters)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO saved_filters (workspace_id, user_id, name, filters)
		VALUES ($1, $2, $3, $4::jsonb)
		ON CONFLICT (workspace_id, user_id, name) DO UPDATE SET
			filters = EXCLUDED.filters,
			updated_at = now()
	`, workspaceID, userID, name, string(filtersJSON))

	return err
}

func resetSeedNotifications(ctx context.Context, tx pgx.Tx, workspaceID string) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM notifications
		WHERE workspace_id = $1
			AND payload->>'seed_key' LIKE 'v2_seed:%'
	`, workspaceID)

	return err
}

func ensureNotification(
	ctx context.Context,
	tx pgx.Tx,
	workspaceID string,
	userID string,
	actorID string,
	issueID string,
	notificationType string,
	payload map[string]string,
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var notificationIssueID any
	if issueID != "" {
		notificationIssueID = issueID
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO notifications (
			workspace_id,
			user_id,
			actor_id,
			issue_id,
			notification_type,
			payload
		)
		VALUES ($1, $2, $3, $4::uuid, $5, $6::jsonb)
	`, workspaceID, userID, actorID, notificationIssueID, notificationType, string(payloadJSON))

	return err
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
