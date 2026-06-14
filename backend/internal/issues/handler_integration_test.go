//go:build integration

package issues

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
	"team-task-tracker/backend/internal/automations"
	"team-task-tracker/backend/internal/database"
	"team-task-tracker/backend/internal/migrations"
	"team-task-tracker/backend/internal/notifications"
	"team-task-tracker/backend/internal/pagination"
)

func TestIssueHierarchyIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newIssueIntegrationDB(t, ctx)
	handler := NewHandler(db, nil)

	user, projectID := seedIssueIntegrationWorkspace(t, ctx, db)

	parent, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID,
		Title:     "Parent issue",
		IssueType: "task",
		Status:    "todo",
		Priority:  "medium",
	})
	if err != nil {
		t.Fatalf("create parent issue: %v", err)
	}

	epic, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID,
		Title:     "Epic issue",
		IssueType: "epic",
		Status:    "todo",
		Priority:  "medium",
	})
	if err != nil {
		t.Fatalf("create epic issue: %v", err)
	}

	epicChild, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID:     projectID,
		ParentIssueID: epic.ID,
		Title:         "Story under epic",
		IssueType:     "story",
		Status:        "todo",
		Priority:      "medium",
	})
	if err != nil {
		t.Fatalf("create epic child issue: %v", err)
	}
	expectIssueParent(t, epicChild, epic.ID)

	subtask, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID:     projectID,
		ParentIssueID: parent.ID,
		Title:         "Subtask issue",
		IssueType:     "subtask",
		Status:        "todo",
		Priority:      "medium",
	})
	if err != nil {
		t.Fatalf("create subtask issue: %v", err)
	}
	expectIssueParent(t, subtask, parent.ID)

	children, err := handler.listIssueChildren(ctx, user.WorkspaceID, parent.ID)
	if err != nil {
		t.Fatalf("list issue children: %v", err)
	}
	if !hasIssueID(children, subtask.ID) {
		t.Fatalf("expected children to contain subtask %s", subtask.ID)
	}

	if _, err := handler.setIssueParent(ctx, user, parent.ID, subtask.ID); !errors.Is(err, errIssueParentCycle) {
		t.Fatalf("set parent to descendant error = %v, want %v", err, errIssueParentCycle)
	}

	if _, err := handler.setIssueParent(ctx, user, subtask.ID, ""); !errors.Is(err, errIssueParentRequired) {
		t.Fatalf("clear subtask parent error = %v, want %v", err, errIssueParentRequired)
	}

	if _, err := handler.setIssueParent(ctx, user, epic.ID, parent.ID); !errors.Is(err, errIssueParentForbidden) {
		t.Fatalf("set epic parent error = %v, want %v", err, errIssueParentForbidden)
	}

	moved, err := handler.setIssueParent(ctx, user, epicChild.ID, parent.ID)
	if err != nil {
		t.Fatalf("move epic child under parent: %v", err)
	}
	expectIssueParent(t, moved, parent.ID)

	activity, err := handler.listIssueActivity(ctx, user.WorkspaceID, epicChild.ID)
	if err != nil {
		t.Fatalf("list issue activity: %v", err)
	}
	if !hasActivityAction(activity, "issue_parent_changed") {
		t.Fatal("expected issue_parent_changed activity")
	}
}

func TestIssueWorkflowStatusesIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newIssueIntegrationDB(t, ctx)
	handler := NewHandler(db, nil)
	user, projectID := seedIssueIntegrationWorkspace(t, ctx, db)

	var todoID, reviewID string
	if err := db.QueryRow(ctx, `SELECT id::text FROM project_workflow_statuses WHERE project_id = $1 AND key = 'todo'`, projectID).Scan(&todoID); err != nil {
		t.Fatalf("load todo status: %v", err)
	}
	if err := db.QueryRow(ctx, `
		INSERT INTO project_workflow_statuses (project_id, key, name, color, category, position)
		VALUES ($1, 'review', 'Ready for review', '#0ea5e9', 'in_progress', 600)
		RETURNING id::text
	`, projectID).Scan(&reviewID); err != nil {
		t.Fatalf("create review status: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO project_workflow_transitions (project_id, from_status_id, to_status_id)
		VALUES ($1, $2, $3)
	`, projectID, todoID, reviewID); err != nil {
		t.Fatalf("create review transition: %v", err)
	}

	issue, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID:        projectID,
		Title:            "Workflow issue",
		IssueType:        "task",
		Status:           "done",
		WorkflowStatusID: reviewID,
		Priority:         "medium",
	})
	if err != nil {
		t.Fatalf("create issue by workflow status id: %v", err)
	}
	if issue.Status != "review" || issue.WorkflowStatus.ID != reviewID || issue.WorkflowStatus.Category != "in_progress" {
		t.Fatalf("created workflow issue = %#v", issue)
	}

	legacy, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID,
		Title:     "Legacy custom key issue",
		IssueType: "task",
		Status:    "review",
		Priority:  "medium",
	})
	if err != nil {
		t.Fatalf("create issue by custom key: %v", err)
	}
	if legacy.WorkflowStatus.ID != reviewID {
		t.Fatalf("legacy custom key workflow status = %#v", legacy.WorkflowStatus)
	}

	todo, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID,
		Title:     "Transition issue",
		IssueType: "task",
		Status:    "todo",
		Priority:  "medium",
	})
	if err != nil {
		t.Fatalf("create transition issue: %v", err)
	}
	transitioned, err := handler.transitionIssueStatus(ctx, user, todo.ID, normalizedTransitionIssue{WorkflowStatusID: reviewID})
	if err != nil {
		t.Fatalf("allowed transition: %v", err)
	}
	if transitioned.Status != "review" {
		t.Fatalf("transitioned status = %q, want review", transitioned.Status)
	}
	if _, err := handler.transitionIssueStatus(ctx, user, todo.ID, normalizedTransitionIssue{Status: "done"}); !errors.Is(err, errTransitionNotAllowed) {
		t.Fatalf("forbidden transition error = %v, want %v", err, errTransitionNotAllowed)
	}
	if _, err := handler.transitionIssueStatus(ctx, user, todo.ID, normalizedTransitionIssue{WorkflowStatusID: reviewID}); err != nil {
		t.Fatalf("no-op transition: %v", err)
	}
	var transitionActivityCount int
	if err := db.QueryRow(ctx, `
		SELECT count(*)::int
		FROM activity_log
		WHERE entity_type = 'issue'
			AND entity_id = $1
			AND action = 'status_changed'
	`, todo.ID).Scan(&transitionActivityCount); err != nil {
		t.Fatalf("count transition activity: %v", err)
	}
	if transitionActivityCount != 1 {
		t.Fatalf("transition activity count = %d, want 1", transitionActivityCount)
	}

	filtered, err := handler.listIssues(ctx, user.WorkspaceID, map[string][]string{"workflow_status_id": {reviewID}})
	if err != nil {
		t.Fatalf("filter issues by workflow status id: %v", err)
	}
	if len(filtered) != 3 {
		t.Fatalf("workflow status filter returned %d issues, want 3", len(filtered))
	}
}

func TestIssueAutomationIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newIssueIntegrationDB(t, ctx)
	handler := NewHandler(db, nil, notifications.NewService())
	user, projectID := seedIssueIntegrationWorkspace(t, ctx, db)

	var todoID, inProgressID, doneID, labelID, notificationTargetID string
	if err := db.QueryRow(ctx, `SELECT id::text FROM project_workflow_statuses WHERE project_id = $1 AND key = 'todo'`, projectID).Scan(&todoID); err != nil {
		t.Fatalf("load todo status: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT id::text FROM project_workflow_statuses WHERE project_id = $1 AND key = 'in_progress'`, projectID).Scan(&inProgressID); err != nil {
		t.Fatalf("load in progress status: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT id::text FROM project_workflow_statuses WHERE project_id = $1 AND key = 'done'`, projectID).Scan(&doneID); err != nil {
		t.Fatalf("load done status: %v", err)
	}
	if err := db.QueryRow(ctx, `
		INSERT INTO labels (workspace_id, name, color) VALUES ($1, 'automated', '#123456') RETURNING id::text
	`, user.WorkspaceID).Scan(&labelID); err != nil {
		t.Fatalf("create automation label: %v", err)
	}
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ('automation-target@example.com', 'automation_target', 'hash', 'Automation Target')
		RETURNING id::text
	`).Scan(&notificationTargetID); err != nil {
		t.Fatalf("create automation notification target: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'member')
	`, user.WorkspaceID, notificationTargetID); err != nil {
		t.Fatalf("add automation notification target to workspace: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id, role) VALUES ($1, $2, 'contributor')
	`, projectID, notificationTargetID); err != nil {
		t.Fatalf("add automation notification target to project: %v", err)
	}

	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 100, "Create first", "issue_created",
		`[{"type":"issue_type","value":"bug"},{"type":"priority","value":"medium"}]`,
		fmt.Sprintf(`[{"type":"change_priority","value":"high"},{"type":"change_assignee","user_id":%q},{"type":"add_label","label_id":%q}]`, user.ID, labelID))
	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 200, "Frozen snapshot does not match", "issue_created",
		`[{"type":"priority","value":"high"}]`, `[{"type":"change_priority","value":"low"}]`)
	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 300, "Create later wins", "issue_created",
		`[{"type":"priority","value":"medium"}]`, `[{"type":"change_priority","value":"critical"}]`)
	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 400, "No priority cascade", "priority_changed",
		`[{"type":"priority","value":"critical"}]`, `[{"type":"change_priority","value":"low"}]`)
	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 500, "Direct priority", "priority_changed",
		`[{"type":"priority","value":"high"}]`, `[{"type":"change_priority","value":"low"}]`)
	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 550, "Remove label", "priority_changed",
		`[{"type":"priority","value":"high"}]`, fmt.Sprintf(`[{"type":"remove_label","label_id":%q}]`, labelID))
	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 600, "Direct status", "status_changed",
		fmt.Sprintf(`[{"type":"workflow_status","workflow_status_id":%q}]`, inProgressID),
		fmt.Sprintf(`[{"type":"change_workflow_status","workflow_status_id":%q}]`, doneID))
	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 700, "No assignee cascade", "assignee_changed",
		fmt.Sprintf(`[{"type":"assignee","user_id":%q}]`, user.ID), `[{"type":"change_priority","value":"low"}]`)
	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 750, "Override direct assignment", "assignee_changed",
		fmt.Sprintf(`[{"type":"assignee","user_id":%q}]`, notificationTargetID), `[{"type":"change_assignee","user_id":null}]`)

	issue, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID, Title: "Automated bug", IssueType: "bug", Status: "todo", Priority: "medium",
	})
	if err != nil {
		t.Fatalf("create automated issue: %v", err)
	}
	if issue.Priority != "critical" || stringOrEmpty(issue.AssigneeID) != user.ID || len(issue.Labels) != 1 {
		t.Fatalf("created automated issue = %#v", issue)
	}

	updated, err := handler.updateIssue(ctx, user, issue.ID, normalizedUpdateIssue{
		Title: issue.Title, Description: issue.Description, IssueType: issue.IssueType,
		Priority: "high", StoryPoints: issue.StoryPoints,
	})
	if err != nil {
		t.Fatalf("update automated issue: %v", err)
	}
	if updated.Priority != "low" {
		t.Fatalf("updated priority = %q, want low", updated.Priority)
	}
	if len(updated.Labels) != 0 {
		t.Fatalf("updated labels = %#v, want removed automation label", updated.Labels)
	}

	transitioned, err := handler.transitionIssueStatus(ctx, user, issue.ID, normalizedTransitionIssue{WorkflowStatusID: inProgressID})
	if err != nil {
		t.Fatalf("transition automated issue: %v", err)
	}
	if transitioned.WorkflowStatus.ID != doneID || transitioned.Status != "done" {
		t.Fatalf("automated transition = %#v", transitioned.WorkflowStatus)
	}

	if _, err := handler.assignIssue(ctx, user, issue.ID, ""); err != nil {
		t.Fatalf("clear automated assignee: %v", err)
	}
	assigned, err := handler.assignIssue(ctx, user, issue.ID, user.ID)
	if err != nil {
		t.Fatalf("assign automated issue: %v", err)
	}
	if assigned.Priority != "low" {
		t.Fatalf("assigned automated priority = %q, want low", assigned.Priority)
	}
	overridden, err := handler.assignIssue(ctx, user, issue.ID, notificationTargetID)
	if err != nil {
		t.Fatalf("assign notification target: %v", err)
	}
	if overridden.AssigneeID != nil {
		t.Fatalf("overridden assignee = %v, want nil", overridden.AssigneeID)
	}
	var assignmentNotifications int
	if err := db.QueryRow(ctx, `
		SELECT count(*)::int FROM notifications
		WHERE user_id = $1 AND issue_id = $2 AND notification_type = 'issue_assigned'
	`, notificationTargetID, issue.ID).Scan(&assignmentNotifications); err != nil {
		t.Fatalf("count suppressed assignment notifications: %v", err)
	}
	if assignmentNotifications != 0 {
		t.Fatalf("assignment notifications = %d, want 0", assignmentNotifications)
	}

	var automationActivityCount, systemActivityCount int
	if err := db.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE action = 'automation_applied')::int,
			count(*) FILTER (WHERE action = 'automation_applied' AND actor_id IS NULL)::int
		FROM activity_log WHERE entity_type = 'issue' AND entity_id = $1
	`, issue.ID).Scan(&automationActivityCount, &systemActivityCount); err != nil {
		t.Fatalf("count automation activity: %v", err)
	}
	if automationActivityCount != 6 || systemActivityCount != automationActivityCount {
		t.Fatalf("automation activities = %d/%d, want six System entries and no entry for no-op rule", automationActivityCount, systemActivityCount)
	}

	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 800, "Fail create atomically", "issue_created",
		`[]`, fmt.Sprintf(`[{"type":"change_workflow_status","workflow_status_id":%q}]`, doneID))
	if _, err := db.Exec(ctx, `
		DELETE FROM project_workflow_transitions
		WHERE project_id = $1 AND from_status_id = $2 AND to_status_id = $3
	`, projectID, todoID, doneID); err != nil {
		t.Fatalf("remove automation transition: %v", err)
	}
	var beforeCount int
	if err := db.QueryRow(ctx, `SELECT count(*)::int FROM issues WHERE project_id = $1`, projectID).Scan(&beforeCount); err != nil {
		t.Fatalf("count issues before rollback: %v", err)
	}
	if _, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID, Title: "Must rollback", IssueType: "task", Status: "todo", Priority: "medium",
	}); !errors.Is(err, automations.ErrActionFailed) {
		t.Fatalf("failed automation error = %v, want action failed", err)
	}
	var afterCount int
	if err := db.QueryRow(ctx, `SELECT count(*)::int FROM issues WHERE project_id = $1`, projectID).Scan(&afterCount); err != nil {
		t.Fatalf("count issues after rollback: %v", err)
	}
	if afterCount != beforeCount {
		t.Fatalf("issue count after failed automation = %d, want %d", afterCount, beforeCount)
	}

	if _, err := db.Exec(ctx, `UPDATE automation_rules SET is_enabled = false WHERE project_id = $1 AND name = 'Fail create atomically'`, projectID); err != nil {
		t.Fatalf("disable transition failure rule: %v", err)
	}
	var unavailableLabelID string
	if err := db.QueryRow(ctx, `
		INSERT INTO labels (workspace_id, name, color) VALUES ($1, 'unavailable', '#654321') RETURNING id::text
	`, user.WorkspaceID).Scan(&unavailableLabelID); err != nil {
		t.Fatalf("create unavailable label: %v", err)
	}
	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 900, "Unavailable label", "issue_created",
		`[]`, fmt.Sprintf(`[{"type":"add_label","label_id":%q}]`, unavailableLabelID))
	if _, err := db.Exec(ctx, `DELETE FROM labels WHERE id = $1`, unavailableLabelID); err != nil {
		t.Fatalf("delete unavailable label: %v", err)
	}
	if _, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID, Title: "Missing label must rollback", IssueType: "task", Status: "todo", Priority: "medium",
	}); !errors.Is(err, automations.ErrActionFailed) {
		t.Fatalf("unavailable label error = %v, want action failed", err)
	}

	if _, err := db.Exec(ctx, `UPDATE automation_rules SET is_enabled = false WHERE project_id = $1 AND name = 'Unavailable label'`, projectID); err != nil {
		t.Fatalf("disable unavailable label rule: %v", err)
	}
	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 1000, "Unavailable user", "issue_created",
		`[]`, fmt.Sprintf(`[{"type":"change_assignee","user_id":%q}]`, notificationTargetID))
	if _, err := db.Exec(ctx, `UPDATE users SET is_active = false WHERE id = $1`, notificationTargetID); err != nil {
		t.Fatalf("deactivate unavailable user: %v", err)
	}
	if _, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID, Title: "Missing user must rollback", IssueType: "task", Status: "todo", Priority: "medium",
	}); !errors.Is(err, automations.ErrActionFailed) {
		t.Fatalf("unavailable user error = %v, want action failed", err)
	}

	if _, err := db.Exec(ctx, `UPDATE automation_rules SET is_enabled = false WHERE project_id = $1 AND name = 'Unavailable user'`, projectID); err != nil {
		t.Fatalf("disable unavailable user rule: %v", err)
	}
	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 1100, "Invalid stored action", "issue_created",
		`[]`, `[{"type":"unknown"}]`)
	if _, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID, Title: "Invalid rule must rollback", IssueType: "task", Status: "todo", Priority: "medium",
	}); !errors.Is(err, automations.ErrActionFailed) {
		t.Fatalf("invalid stored action error = %v, want action failed", err)
	}
}

func TestIssueAutomationNotificationsAreAtomicIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newIssueIntegrationDB(t, ctx)
	handler := NewHandler(db, nil, notifications.NewService())
	user, projectID := seedIssueIntegrationWorkspace(t, ctx, db)

	var targetID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ('automation-recipient@example.com', 'automation_recipient', 'hash', 'Automation Recipient')
		RETURNING id::text
	`).Scan(&targetID); err != nil {
		t.Fatalf("create automation recipient: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role) VALUES ($1, $2, 'member')
	`, user.WorkspaceID, targetID); err != nil {
		t.Fatalf("grant automation recipient workspace access: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO project_members (project_id, user_id, role) VALUES ($1, $2, 'contributor')
	`, projectID, targetID); err != nil {
		t.Fatalf("grant automation recipient project access: %v", err)
	}
	insertIssueAutomationRule(t, ctx, db, projectID, user.ID, 100, "Assign recipient", "issue_created",
		`[]`, fmt.Sprintf(`[{"type":"change_assignee","user_id":%q}]`, targetID))

	issue, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID, Title: "Notify atomically", IssueType: "task", Status: "todo", Priority: "medium",
	})
	if err != nil {
		t.Fatalf("create automated notification issue: %v", err)
	}
	if stringOrEmpty(issue.AssigneeID) != targetID {
		t.Fatalf("automated assignee = %v, want %s", issue.AssigneeID, targetID)
	}
	var notificationType string
	var actorIsNull bool
	if err := db.QueryRow(ctx, `
		SELECT notification_type, actor_id IS NULL
		FROM notifications
		WHERE issue_id = $1 AND user_id = $2
	`, issue.ID, targetID).Scan(&notificationType, &actorIsNull); err != nil {
		t.Fatalf("load automation notification: %v", err)
	}
	if notificationType != notifications.TypeIssueAutomationAssigned || !actorIsNull {
		t.Fatalf("automation notification = %s/null:%t", notificationType, actorIsNull)
	}

	if _, err := db.Exec(ctx, `ALTER TABLE notifications DROP CONSTRAINT notifications_notification_type_check`); err != nil {
		t.Fatalf("drop automation notification constraint: %v", err)
	}
	if _, err := db.Exec(ctx, `
		ALTER TABLE notifications ADD CONSTRAINT notifications_notification_type_check CHECK (
			notification_type IN ('issue_assigned', 'issue_mentioned', 'issue_commented', 'sprint_started', 'sprint_completed')
		) NOT VALID
	`); err != nil {
		t.Fatalf("restrict notification types: %v", err)
	}
	var beforeCount int
	if err := db.QueryRow(ctx, `SELECT count(*)::int FROM issues WHERE project_id = $1`, projectID).Scan(&beforeCount); err != nil {
		t.Fatalf("count issues before notification rollback: %v", err)
	}
	if _, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID, Title: "Rollback notification failure", IssueType: "task", Status: "todo", Priority: "medium",
	}); err == nil {
		t.Fatal("expected automation notification insert failure")
	}
	var afterCount int
	if err := db.QueryRow(ctx, `SELECT count(*)::int FROM issues WHERE project_id = $1`, projectID).Scan(&afterCount); err != nil {
		t.Fatalf("count issues after notification rollback: %v", err)
	}
	if afterCount != beforeCount {
		t.Fatalf("issue count after notification failure = %d, want %d", afterCount, beforeCount)
	}
}

func TestIssueLinksIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newIssueIntegrationDB(t, ctx)
	handler := NewHandler(db, nil)

	user, projectID := seedIssueIntegrationWorkspace(t, ctx, db)

	source, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID,
		Title:     "Source issue",
		IssueType: "task",
		Status:    "todo",
		Priority:  "medium",
	})
	if err != nil {
		t.Fatalf("create source issue: %v", err)
	}

	target, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID,
		Title:     "Target issue",
		IssueType: "bug",
		Status:    "todo",
		Priority:  "high",
	})
	if err != nil {
		t.Fatalf("create target issue: %v", err)
	}

	link, err := handler.createIssueLink(ctx, user, source.ID, normalizedCreateIssueLink{
		TargetIssueID: target.ID,
		LinkType:      "relates",
	})
	if err != nil {
		t.Fatalf("create issue link: %v", err)
	}
	if link.SourceIssueID != source.ID || link.TargetIssueID != target.ID {
		t.Fatalf("unexpected link endpoints: %#v", link)
	}

	sourceLinks, err := handler.listIssueLinks(ctx, user.WorkspaceID, source.ID)
	if err != nil {
		t.Fatalf("list source issue links: %v", err)
	}
	if !hasIssueLinkID(sourceLinks, link.ID) {
		t.Fatalf("expected source links to contain link %s", link.ID)
	}

	targetLinks, err := handler.listIssueLinks(ctx, user.WorkspaceID, target.ID)
	if err != nil {
		t.Fatalf("list target issue links: %v", err)
	}
	if !hasIssueLinkID(targetLinks, link.ID) {
		t.Fatalf("expected target links to contain link %s", link.ID)
	}

	if _, err := handler.createIssueLink(ctx, user, source.ID, normalizedCreateIssueLink{
		TargetIssueID: target.ID,
		LinkType:      "relates",
	}); !errors.Is(err, errIssueLinkDuplicate) {
		t.Fatalf("duplicate link error = %v, want %v", err, errIssueLinkDuplicate)
	}

	if _, err := handler.createIssueLink(ctx, user, target.ID, normalizedCreateIssueLink{
		TargetIssueID: source.ID,
		LinkType:      "relates",
	}); !errors.Is(err, errIssueLinkDuplicate) {
		t.Fatalf("inverse relates link error = %v, want %v", err, errIssueLinkDuplicate)
	}

	if _, err := handler.createIssueLink(ctx, user, source.ID, normalizedCreateIssueLink{
		TargetIssueID: source.ID,
		LinkType:      "blocks",
	}); !errors.Is(err, errIssueLinkSelf) {
		t.Fatalf("self link error = %v, want %v", err, errIssueLinkSelf)
	}

	if _, err := handler.createIssueLink(ctx, user, source.ID, normalizedCreateIssueLink{
		TargetIssueID: "6d5257d4-002e-44da-8925-d9108699c504",
		LinkType:      "blocks",
	}); !errors.Is(err, errInvalidIssueLinkTarget) {
		t.Fatalf("invalid target link error = %v, want %v", err, errInvalidIssueLinkTarget)
	}

	if err := handler.deleteIssueLink(ctx, user, target.ID, link.ID); err != nil {
		t.Fatalf("delete issue link from target side: %v", err)
	}

	sourceLinks, err = handler.listIssueLinks(ctx, user.WorkspaceID, source.ID)
	if err != nil {
		t.Fatalf("list source issue links after delete: %v", err)
	}
	if hasIssueLinkID(sourceLinks, link.ID) {
		t.Fatalf("expected source links to not contain deleted link %s", link.ID)
	}

	sourceActivity, err := handler.listIssueActivity(ctx, user.WorkspaceID, source.ID)
	if err != nil {
		t.Fatalf("list source activity: %v", err)
	}
	if !hasActivityAction(sourceActivity, "issue_link_created") {
		t.Fatal("expected source activity to contain issue_link_created")
	}
	if !hasActivityAction(sourceActivity, "issue_link_deleted") {
		t.Fatal("expected source activity to contain issue_link_deleted")
	}

	targetActivity, err := handler.listIssueActivity(ctx, user.WorkspaceID, target.ID)
	if err != nil {
		t.Fatalf("list target activity: %v", err)
	}
	if !hasActivityAction(targetActivity, "issue_link_created") {
		t.Fatal("expected target activity to contain issue_link_created")
	}
	if !hasActivityAction(targetActivity, "issue_link_deleted") {
		t.Fatal("expected target activity to contain issue_link_deleted")
	}
}

func TestIssueListSprintFiltersIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newIssueIntegrationDB(t, ctx)
	handler := NewHandler(db, nil)

	user, projectID := seedIssueIntegrationWorkspace(t, ctx, db)

	var sprintID string
	if err := db.QueryRow(ctx, `
		INSERT INTO sprints (workspace_id, project_id, name, created_by)
		VALUES ($1, $2, 'Sprint filter test', $3)
		RETURNING id::text
	`, user.WorkspaceID, projectID, user.ID).Scan(&sprintID); err != nil {
		t.Fatalf("insert sprint: %v", err)
	}

	inSprint, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID,
		Title:     "Issue in sprint",
		IssueType: "task",
		Status:    "todo",
		Priority:  "medium",
	})
	if err != nil {
		t.Fatalf("create sprint issue: %v", err)
	}
	if _, err := db.Exec(ctx, `
		UPDATE issues
		SET sprint_id = $1
		WHERE id = $2
	`, sprintID, inSprint.ID); err != nil {
		t.Fatalf("assign issue to sprint: %v", err)
	}

	withoutSprint, err := handler.createIssue(ctx, user, normalizedCreateIssue{
		ProjectID: projectID,
		Title:     "Issue without sprint",
		IssueType: "task",
		Status:    "todo",
		Priority:  "medium",
	})
	if err != nil {
		t.Fatalf("create no sprint issue: %v", err)
	}

	sprintIssues, err := handler.listIssues(ctx, user.WorkspaceID, map[string][]string{
		"sprint_id": {sprintID},
	})
	if err != nil {
		t.Fatalf("list sprint issues: %v", err)
	}
	if !hasIssueID(sprintIssues, inSprint.ID) {
		t.Fatalf("expected sprint filter to contain issue %s", inSprint.ID)
	}
	if hasIssueID(sprintIssues, withoutSprint.ID) {
		t.Fatalf("expected sprint filter to exclude issue %s", withoutSprint.ID)
	}

	noSprintIssues, err := handler.listIssues(ctx, user.WorkspaceID, map[string][]string{
		"project_id": {projectID},
		"sprint_id":  {"none"},
	})
	if err != nil {
		t.Fatalf("list no sprint issues: %v", err)
	}
	if !hasIssueID(noSprintIssues, withoutSprint.ID) {
		t.Fatalf("expected no sprint filter to contain issue %s", withoutSprint.ID)
	}
	if hasIssueID(noSprintIssues, inSprint.ID) {
		t.Fatalf("expected no sprint filter to exclude issue %s", inSprint.ID)
	}
}

func TestIssuePaginationIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db := newIssueIntegrationDB(t, ctx)
	handler := NewHandler(db, nil)

	user, projectID := seedIssueIntegrationWorkspace(t, ctx, db)

	createdIDs := make([]string, 0, 3)
	for index := 0; index < 3; index++ {
		issue, err := handler.createIssue(ctx, user, normalizedCreateIssue{
			ProjectID: projectID,
			Title:     fmt.Sprintf("Paginated issue %d", index),
			IssueType: "task",
			Status:    "todo",
			Priority:  "medium",
		})
		if err != nil {
			t.Fatalf("create paginated issue %d: %v", index, err)
		}
		createdIDs = append(createdIDs, issue.ID)
	}

	firstPage, nextCursor, err := handler.listIssuesPage(ctx, user.WorkspaceID, map[string][]string{
		"project_id": {projectID},
		"sort":       {"created_asc"},
	}, pagination.Params{Limit: 2})
	if err != nil {
		t.Fatalf("list first issue page: %v", err)
	}
	if len(firstPage) != 2 {
		t.Fatalf("first issue page len = %d, want 2", len(firstPage))
	}
	if nextCursor == nil {
		t.Fatal("expected issue next cursor")
	}

	nextOffset, err := pagination.DecodeCursor(*nextCursor)
	if err != nil {
		t.Fatalf("decode issue next cursor: %v", err)
	}
	secondPage, secondNextCursor, err := handler.listIssuesPage(ctx, user.WorkspaceID, map[string][]string{
		"project_id": {projectID},
		"sort":       {"created_asc"},
	}, pagination.Params{Limit: 2, Offset: nextOffset})
	if err != nil {
		t.Fatalf("list second issue page: %v", err)
	}
	if len(secondPage) != 1 {
		t.Fatalf("second issue page len = %d, want 1", len(secondPage))
	}
	if secondNextCursor != nil {
		t.Fatalf("second issue next cursor = %q, want nil", *secondNextCursor)
	}
	if hasIssueID(firstPage, secondPage[0].ID) {
		t.Fatalf("issue %s appeared on both pages", secondPage[0].ID)
	}

	for index := 0; index < 3; index++ {
		if _, err := db.Exec(ctx, `
			INSERT INTO activity_log (entity_type, entity_id, action, actor_id, payload)
			VALUES ('issue', $1, 'status_changed', $2, $3::jsonb)
		`, createdIDs[0], user.ID, fmt.Sprintf(`{"index":"%d"}`, index)); err != nil {
			t.Fatalf("insert activity %d: %v", index, err)
		}
	}

	firstActivityPage, activityNextCursor, err := handler.listIssueActivityPage(ctx, user.WorkspaceID, createdIDs[0], pagination.Params{Limit: 2})
	if err != nil {
		t.Fatalf("list first activity page: %v", err)
	}
	if len(firstActivityPage) != 2 {
		t.Fatalf("first activity page len = %d, want 2", len(firstActivityPage))
	}
	if activityNextCursor == nil {
		t.Fatal("expected activity next cursor")
	}

	activityNextOffset, err := pagination.DecodeCursor(*activityNextCursor)
	if err != nil {
		t.Fatalf("decode activity next cursor: %v", err)
	}
	secondActivityPage, _, err := handler.listIssueActivityPage(ctx, user.WorkspaceID, createdIDs[0], pagination.Params{Limit: 2, Offset: activityNextOffset})
	if err != nil {
		t.Fatalf("list second activity page: %v", err)
	}
	if len(secondActivityPage) == 0 {
		t.Fatal("expected second activity page")
	}
	if hasActivityID(firstActivityPage, secondActivityPage[0].ID) {
		t.Fatalf("activity %s appeared on both pages", secondActivityPage[0].ID)
	}
}

func newIssueIntegrationDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
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

	schemaName := fmt.Sprintf("issues_integration_%d", time.Now().UnixNano())
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

func seedIssueIntegrationWorkspace(t *testing.T, ctx context.Context, db *pgxpool.Pool) (auth.CurrentUser, string) {
	t.Helper()

	var workspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name)
		VALUES ('Issues Integration Workspace')
		RETURNING id::text
	`).Scan(&workspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}

	var userID string
	if err := db.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name)
		VALUES ('issues-integration@example.com', 'issues_integration', 'hash', 'Issues Integration')
		RETURNING id::text
	`).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin')
	`, workspaceID, userID); err != nil {
		t.Fatalf("insert workspace member: %v", err)
	}

	var projectID string
	if err := db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, created_by)
		VALUES ($1, 'HIER', 'Hierarchy Project', $2)
		RETURNING id::text
	`, workspaceID, userID).Scan(&projectID); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	return auth.CurrentUser{
		ID:          userID,
		WorkspaceID: workspaceID,
		Role:        "admin",
	}, projectID
}

func insertIssueAutomationRule(
	t *testing.T,
	ctx context.Context,
	db *pgxpool.Pool,
	projectID string,
	createdBy string,
	position int,
	name string,
	triggerType string,
	conditions string,
	actions string,
) {
	t.Helper()
	if _, err := db.Exec(ctx, `
		INSERT INTO automation_rules (
			project_id, name, trigger_type, conditions, actions, position, created_by
		)
		VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6, $7)
	`, projectID, name, triggerType, conditions, actions, position, createdBy); err != nil {
		t.Fatalf("insert automation rule %q: %v", name, err)
	}
}

func expectIssueParent(t *testing.T, issue issueResponse, want string) {
	t.Helper()

	if issue.ParentIssueID == nil {
		t.Fatalf("ParentIssueID is nil, want %q", want)
	}
	if *issue.ParentIssueID != want {
		t.Fatalf("ParentIssueID = %q, want %q", *issue.ParentIssueID, want)
	}
}

func hasIssueID(issues []issueResponse, issueID string) bool {
	for _, issue := range issues {
		if issue.ID == issueID {
			return true
		}
	}

	return false
}

func hasIssueLinkID(links []issueLinkResponse, linkID string) bool {
	for _, link := range links {
		if link.ID == linkID {
			return true
		}
	}

	return false
}

func hasActivityAction(activity []issueActivityResponse, action string) bool {
	for _, entry := range activity {
		if entry.Action == action {
			return true
		}
	}

	return false
}

func hasActivityID(activity []issueActivityResponse, activityID string) bool {
	for _, entry := range activity {
		if entry.ID == activityID {
			return true
		}
	}

	return false
}
