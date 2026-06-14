package automations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrActionFailed = errors.New("automation action failed")

type Engine struct{}

type ExecuteRequest struct {
	WorkspaceID       string
	IssueID           string
	TriggerType       string
	InitiatedByUserID string
}

type ExecuteResult struct {
	AppliedRuleNames []string
	ChangedFields    []string
	FromStatus       string
	ToStatus         string
	FromAssigneeID   string
	ToAssigneeID     string
}

type runtimeIssue struct {
	IssueID          string
	ProjectID        string
	WorkspaceID      string
	IssueType        string
	WorkflowStatusID string
	Status           string
	Priority         string
	ReporterID       string
	AssigneeID       string
	LabelIDs         map[string]bool
}

type runtimeRule struct {
	ID         string
	Name       string
	Conditions []runtimeItem
	Actions    []runtimeItem
}

type runtimeItem struct {
	Type             string  `json:"type"`
	Value            string  `json:"value"`
	WorkflowStatusID string  `json:"workflow_status_id"`
	UserID           *string `json:"user_id"`
	LabelID          string  `json:"label_id"`
}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Execute(ctx context.Context, tx pgx.Tx, request ExecuteRequest) (ExecuteResult, error) {
	snapshot, err := loadRuntimeIssue(ctx, tx, request.WorkspaceID, request.IssueID)
	if err != nil {
		return ExecuteResult{}, err
	}
	rules, err := loadRuntimeRules(ctx, tx, snapshot.ProjectID, request.TriggerType)
	if err != nil {
		return ExecuteResult{}, err
	}
	current := cloneRuntimeIssue(snapshot)
	appliedRuleNames := make([]string, 0, len(rules))
	for _, rule := range rules {
		if !matchesConditions(snapshot, rule.Conditions) {
			continue
		}
		before := cloneRuntimeIssue(current)
		for _, action := range rule.Actions {
			if err := applyAction(ctx, tx, &current, action); err != nil {
				return ExecuteResult{}, fmt.Errorf("%w: rule %s: %v", ErrActionFailed, rule.ID, err)
			}
		}
		payload := automationActivityPayload(rule, request, before, current)
		if payload == nil {
			continue
		}
		if err := insertAutomationActivity(ctx, tx, request.IssueID, payload); err != nil {
			return ExecuteResult{}, err
		}
		appliedRuleNames = append(appliedRuleNames, rule.Name)
	}
	return automationExecuteResult(snapshot, current, appliedRuleNames), nil
}

func loadRuntimeIssue(ctx context.Context, tx pgx.Tx, workspaceID string, issueID string) (runtimeIssue, error) {
	var issue runtimeIssue
	var assigneeID pgtype.Text
	if err := tx.QueryRow(ctx, `
		SELECT
			issue.id::text,
			issue.project_id::text,
			project.workspace_id::text,
			issue.issue_type,
			issue.workflow_status_id::text,
			issue.status,
			issue.priority,
			issue.reporter_id::text,
			issue.assignee_id::text
		FROM issues issue
		JOIN projects project ON project.id = issue.project_id
		WHERE issue.id = $1
			AND project.workspace_id = $2
			AND project.archived_at IS NULL
			AND issue.archived_at IS NULL
		FOR UPDATE OF issue
	`, issueID, workspaceID).Scan(
		&issue.IssueID,
		&issue.ProjectID,
		&issue.WorkspaceID,
		&issue.IssueType,
		&issue.WorkflowStatusID,
		&issue.Status,
		&issue.Priority,
		&issue.ReporterID,
		&assigneeID,
	); err != nil {
		return runtimeIssue{}, err
	}
	if assigneeID.Valid {
		issue.AssigneeID = assigneeID.String
	}
	labelIDs, err := loadRuntimeLabelIDs(ctx, tx, issueID)
	issue.LabelIDs = labelIDs
	return issue, err
}

func loadRuntimeLabelIDs(ctx context.Context, tx pgx.Tx, issueID string) (map[string]bool, error) {
	rows, err := tx.Query(ctx, `SELECT label_id::text FROM issue_labels WHERE issue_id = $1`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	labelIDs := map[string]bool{}
	for rows.Next() {
		var labelID string
		if err := rows.Scan(&labelID); err != nil {
			return nil, err
		}
		labelIDs[labelID] = true
	}
	return labelIDs, rows.Err()
}

func loadRuntimeRules(ctx context.Context, tx pgx.Tx, projectID string, triggerType string) ([]runtimeRule, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, name, conditions, actions
		FROM automation_rules
		WHERE project_id = $1
			AND trigger_type = $2
			AND is_enabled = true
		ORDER BY position, id
	`, projectID, triggerType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rules := make([]runtimeRule, 0)
	for rows.Next() {
		var rule runtimeRule
		var conditionsJSON []byte
		var actionsJSON []byte
		if err := rows.Scan(&rule.ID, &rule.Name, &conditionsJSON, &actionsJSON); err != nil {
			return nil, err
		}
		if _, _, _, err := normalizeDefinition(conditionsJSON, actionsJSON, true, true); err != nil {
			return nil, fmt.Errorf("%w: rule %s has invalid definition", ErrActionFailed, rule.ID)
		}
		if err := json.Unmarshal(conditionsJSON, &rule.Conditions); err != nil {
			return nil, fmt.Errorf("%w: rule %s has invalid conditions", ErrActionFailed, rule.ID)
		}
		if err := json.Unmarshal(actionsJSON, &rule.Actions); err != nil || len(rule.Actions) == 0 {
			return nil, fmt.Errorf("%w: rule %s has invalid actions", ErrActionFailed, rule.ID)
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func matchesConditions(issue runtimeIssue, conditions []runtimeItem) bool {
	for _, condition := range conditions {
		switch condition.Type {
		case "issue_type":
			if issue.IssueType != condition.Value {
				return false
			}
		case "workflow_status":
			if issue.WorkflowStatusID != condition.WorkflowStatusID {
				return false
			}
		case "priority":
			if issue.Priority != condition.Value {
				return false
			}
		case "assignee":
			if nullableRuntimeID(condition.UserID) != issue.AssigneeID {
				return false
			}
		case "reporter":
			if nullableRuntimeID(condition.UserID) != issue.ReporterID {
				return false
			}
		case "label":
			if !issue.LabelIDs[condition.LabelID] {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func applyAction(ctx context.Context, tx pgx.Tx, issue *runtimeIssue, action runtimeItem) error {
	switch action.Type {
	case "change_workflow_status":
		return changeWorkflowStatus(ctx, tx, issue, action.WorkflowStatusID)
	case "change_assignee":
		return changeAssignee(ctx, tx, issue, nullableRuntimeID(action.UserID))
	case "change_priority":
		return changePriority(ctx, tx, issue, action.Value)
	case "add_label":
		return addLabel(ctx, tx, issue, action.LabelID)
	case "remove_label":
		return removeLabel(ctx, tx, issue, action.LabelID)
	default:
		return errors.New("action type is invalid")
	}
}

func changeWorkflowStatus(ctx context.Context, tx pgx.Tx, issue *runtimeIssue, targetID string) error {
	var targetKey string
	if err := tx.QueryRow(ctx, `
		SELECT key
		FROM project_workflow_statuses
		WHERE id = $1 AND project_id = $2 AND archived_at IS NULL
	`, targetID, issue.ProjectID).Scan(&targetKey); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("workflow status is unavailable")
		}
		return err
	}
	if targetID == issue.WorkflowStatusID {
		return nil
	}
	var allowed bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM project_workflow_transitions
			WHERE project_id = $1 AND from_status_id = $2 AND to_status_id = $3
		)
	`, issue.ProjectID, issue.WorkflowStatusID, targetID).Scan(&allowed); err != nil {
		return err
	}
	if !allowed {
		return errors.New("workflow transition is not allowed")
	}
	if _, err := tx.Exec(ctx, `
		UPDATE issues SET workflow_status_id = $2, updated_at = now() WHERE id = $1
	`, issue.IssueID, targetID); err != nil {
		return err
	}
	issue.WorkflowStatusID = targetID
	issue.Status = targetKey
	return nil
}

func changeAssignee(ctx context.Context, tx pgx.Tx, issue *runtimeIssue, targetID string) error {
	if targetID != "" {
		var valid bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM projects project
				JOIN workspace_members workspace_member
					ON workspace_member.workspace_id = project.workspace_id
					AND workspace_member.user_id = $2
				JOIN users app_user
					ON app_user.id = workspace_member.user_id
					AND app_user.is_active = true
				LEFT JOIN project_members project_member
					ON project_member.project_id = project.id
					AND project_member.user_id = app_user.id
				WHERE project.id = $1
					AND project.workspace_id = $3
					AND (workspace_member.role = 'admin' OR project_member.user_id IS NOT NULL)
			)
		`, issue.ProjectID, targetID, issue.WorkspaceID).Scan(&valid); err != nil {
			return err
		}
		if !valid {
			return errors.New("assignee is unavailable")
		}
	}
	if targetID == issue.AssigneeID {
		return nil
	}
	var value any
	if targetID != "" {
		value = targetID
	}
	if _, err := tx.Exec(ctx, `UPDATE issues SET assignee_id = $2::uuid, updated_at = now() WHERE id = $1`, issue.IssueID, value); err != nil {
		return err
	}
	issue.AssigneeID = targetID
	return nil
}

func changePriority(ctx context.Context, tx pgx.Tx, issue *runtimeIssue, priority string) error {
	if !validPriorities[priority] {
		return errors.New("priority is invalid")
	}
	if priority == issue.Priority {
		return nil
	}
	if _, err := tx.Exec(ctx, `UPDATE issues SET priority = $2, updated_at = now() WHERE id = $1`, issue.IssueID, priority); err != nil {
		return err
	}
	issue.Priority = priority
	return nil
}

func addLabel(ctx context.Context, tx pgx.Tx, issue *runtimeIssue, labelID string) error {
	if err := validateRuntimeLabel(ctx, tx, issue.WorkspaceID, labelID); err != nil {
		return err
	}
	if issue.LabelIDs[labelID] {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO issue_labels (issue_id, label_id) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, issue.IssueID, labelID); err != nil {
		return err
	}
	issue.LabelIDs[labelID] = true
	return nil
}

func removeLabel(ctx context.Context, tx pgx.Tx, issue *runtimeIssue, labelID string) error {
	if err := validateRuntimeLabel(ctx, tx, issue.WorkspaceID, labelID); err != nil {
		return err
	}
	if !issue.LabelIDs[labelID] {
		return nil
	}
	if _, err := tx.Exec(ctx, `DELETE FROM issue_labels WHERE issue_id = $1 AND label_id = $2`, issue.IssueID, labelID); err != nil {
		return err
	}
	delete(issue.LabelIDs, labelID)
	return nil
}

func validateRuntimeLabel(ctx context.Context, tx pgx.Tx, workspaceID string, labelID string) error {
	var valid bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM labels WHERE id = $1 AND workspace_id = $2)`, labelID, workspaceID).Scan(&valid); err != nil {
		return err
	}
	if !valid {
		return errors.New("label is unavailable")
	}
	return nil
}

func automationActivityPayload(rule runtimeRule, request ExecuteRequest, before runtimeIssue, after runtimeIssue) map[string]string {
	changedFields := make([]string, 0, 4)
	payload := map[string]string{
		"rule_id":              rule.ID,
		"rule_name":            rule.Name,
		"trigger_type":         request.TriggerType,
		"initiated_by_user_id": request.InitiatedByUserID,
	}
	if before.WorkflowStatusID != after.WorkflowStatusID {
		changedFields = append(changedFields, "status")
		payload["from_status"] = before.Status
		payload["to_status"] = after.Status
	}
	if before.AssigneeID != after.AssigneeID {
		changedFields = append(changedFields, "assignee")
		payload["from_assignee_id"] = before.AssigneeID
		payload["to_assignee_id"] = after.AssigneeID
	}
	if before.Priority != after.Priority {
		changedFields = append(changedFields, "priority")
		payload["from_priority"] = before.Priority
		payload["to_priority"] = after.Priority
	}
	added, removed := changedLabels(before.LabelIDs, after.LabelIDs)
	if len(added) > 0 || len(removed) > 0 {
		changedFields = append(changedFields, "labels")
		payload["added_label_ids"] = strings.Join(added, ",")
		payload["removed_label_ids"] = strings.Join(removed, ",")
	}
	if len(changedFields) == 0 {
		return nil
	}
	payload["changed_fields"] = strings.Join(changedFields, ",")
	return payload
}

func automationExecuteResult(before runtimeIssue, after runtimeIssue, appliedRuleNames []string) ExecuteResult {
	result := ExecuteResult{AppliedRuleNames: append([]string(nil), appliedRuleNames...)}
	if before.WorkflowStatusID != after.WorkflowStatusID {
		result.ChangedFields = append(result.ChangedFields, "status")
		result.FromStatus = before.Status
		result.ToStatus = after.Status
	}
	if before.AssigneeID != after.AssigneeID {
		result.ChangedFields = append(result.ChangedFields, "assignee")
		result.FromAssigneeID = before.AssigneeID
		result.ToAssigneeID = after.AssigneeID
	}
	if before.Priority != after.Priority {
		result.ChangedFields = append(result.ChangedFields, "priority")
	}
	added, removed := changedLabels(before.LabelIDs, after.LabelIDs)
	if len(added) > 0 || len(removed) > 0 {
		result.ChangedFields = append(result.ChangedFields, "labels")
	}
	return result
}

func insertAutomationActivity(ctx context.Context, tx pgx.Tx, issueID string, payload map[string]string) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO activity_log (entity_type, entity_id, action, actor_id, payload)
		VALUES ('issue', $1, 'automation_applied', NULL, $2::jsonb)
	`, issueID, string(encoded))
	return err
}

func changedLabels(before map[string]bool, after map[string]bool) ([]string, []string) {
	added := make([]string, 0)
	removed := make([]string, 0)
	for labelID := range after {
		if !before[labelID] {
			added = append(added, labelID)
		}
	}
	for labelID := range before {
		if !after[labelID] {
			removed = append(removed, labelID)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return added, removed
}

func cloneRuntimeIssue(issue runtimeIssue) runtimeIssue {
	cloned := issue
	cloned.LabelIDs = make(map[string]bool, len(issue.LabelIDs))
	for labelID := range issue.LabelIDs {
		cloned.LabelIDs[labelID] = true
	}
	return cloned
}

func nullableRuntimeID(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
