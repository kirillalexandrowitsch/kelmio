package automations

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"kelmio/backend/internal/auth"
	"kelmio/backend/internal/projectaccess"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func (h *Handler) listRules(ctx context.Context, projectID string) ([]ruleResponse, error) {
	rows, err := h.db.Query(ctx, ruleSelect+`
		WHERE automation_rule.project_id = $1
		ORDER BY automation_rule.position, automation_rule.id
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rules := make([]ruleResponse, 0)
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (h *Handler) createRule(ctx context.Context, user auth.CurrentUser, projectID string, input normalizedCreateRule) (ruleResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ruleResponse{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := projectaccess.RequireManageForUpdate(ctx, tx, user, projectID); err != nil {
		return ruleResponse{}, err
	}
	if err := validateDependencies(ctx, tx, projectID, user.WorkspaceID, input.Definition.Dependencies); err != nil {
		return ruleResponse{}, err
	}
	rule, err := scanRule(tx.QueryRow(ctx, `
		WITH next_position AS (
			SELECT COALESCE(MAX(position), 0) + 100 AS position
			FROM automation_rules
			WHERE project_id = $1
		)
		INSERT INTO automation_rules (
			project_id, name, trigger_type, conditions, actions, position, is_enabled, created_by
		)
		SELECT $1, $2, $3, $4, $5, next_position.position, $6, $7
		FROM next_position
		RETURNING
			id::text, project_id::text, name, trigger_type, conditions, actions, position,
			is_enabled, disabled_reason, created_by::text, created_at, updated_at
	`, projectID, input.Name, input.TriggerType, input.Conditions, input.Actions, input.IsEnabled, user.ID))
	if err != nil {
		return ruleResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ruleResponse{}, err
	}
	return rule, nil
}

func (h *Handler) updateRule(ctx context.Context, user auth.CurrentUser, projectID string, ruleID string, input normalizedUpdateRule) (ruleResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ruleResponse{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := projectaccess.RequireManageForUpdate(ctx, tx, user, projectID); err != nil {
		return ruleResponse{}, err
	}
	current, err := getRuleForUpdate(ctx, tx, projectID, ruleID)
	if err != nil {
		return ruleResponse{}, err
	}

	conditions := current.Conditions
	actions := current.Actions
	if input.HasConditions {
		conditions = input.Conditions
	}
	if input.HasActions {
		actions = input.Actions
	}
	if input.HasConditions || input.HasActions || (input.HasIsEnabled && input.IsEnabled) {
		definition, _, _, err := normalizeDefinition(conditions, actions, true, true)
		if err != nil {
			return ruleResponse{}, err
		}
		if err := validateDependencies(ctx, tx, projectID, user.WorkspaceID, definition.Dependencies); err != nil {
			return ruleResponse{}, err
		}
	}
	clearReason := input.HasConditions || input.HasActions || (input.HasIsEnabled && input.IsEnabled)
	rule, err := scanRule(tx.QueryRow(ctx, `
		UPDATE automation_rules
		SET name = CASE WHEN $3 THEN $4 ELSE name END,
			trigger_type = CASE WHEN $5 THEN $6 ELSE trigger_type END,
			conditions = CASE WHEN $7 THEN $8 ELSE conditions END,
			actions = CASE WHEN $9 THEN $10 ELSE actions END,
			is_enabled = CASE WHEN $11 THEN $12 ELSE is_enabled END,
			disabled_reason = CASE
				WHEN $11 AND $12 THEN NULL
				WHEN $11 AND NOT $12 THEN NULL
				WHEN $13 THEN NULL
				ELSE disabled_reason
			END,
			updated_at = now()
		WHERE project_id = $1
			AND id = $2
		RETURNING
			id::text, project_id::text, name, trigger_type, conditions, actions, position,
			is_enabled, disabled_reason, created_by::text, created_at, updated_at
	`, projectID, ruleID,
		input.HasName, input.Name,
		input.HasTriggerType, input.TriggerType,
		input.HasConditions, conditions,
		input.HasActions, actions,
		input.HasIsEnabled, input.IsEnabled,
		clearReason,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return ruleResponse{}, errRuleNotFound
	}
	if err != nil {
		return ruleResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ruleResponse{}, err
	}
	return rule, nil
}

func (h *Handler) deleteRule(ctx context.Context, user auth.CurrentUser, projectID string, ruleID string) error {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := projectaccess.RequireManageForUpdate(ctx, tx, user, projectID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `DELETE FROM automation_rules WHERE project_id = $1 AND id = $2`, projectID, ruleID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errRuleNotFound
	}
	return tx.Commit(ctx)
}

func (h *Handler) reorderRules(ctx context.Context, user auth.CurrentUser, projectID string, ruleIDs []string) error {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := projectaccess.RequireManageForUpdate(ctx, tx, user, projectID); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `
		SELECT id::text
		FROM automation_rules
		WHERE project_id = $1
		ORDER BY position, id
		FOR UPDATE
	`, projectID)
	if err != nil {
		return err
	}
	current := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		current = append(current, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if !sameIDSet(current, ruleIDs) {
		return errRuleOrderMismatch
	}
	for index, ruleID := range ruleIDs {
		if _, err := tx.Exec(ctx, `
			UPDATE automation_rules
			SET position = $3, updated_at = now()
			WHERE project_id = $1 AND id = $2
		`, projectID, ruleID, (index+1)*100); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func getRuleForUpdate(ctx context.Context, tx pgx.Tx, projectID string, ruleID string) (ruleResponse, error) {
	rule, err := scanRule(tx.QueryRow(ctx, ruleSelect+`
		WHERE automation_rule.project_id = $1
			AND automation_rule.id = $2
		FOR UPDATE
	`, projectID, ruleID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ruleResponse{}, errRuleNotFound
	}
	return rule, err
}

func validateDependencies(ctx context.Context, tx pgx.Tx, projectID string, workspaceID string, dependencies []dependency) error {
	seen := map[string]bool{}
	for _, dependency := range dependencies {
		key := dependency.Kind + ":" + dependency.ID
		if seen[key] {
			continue
		}
		seen[key] = true
		var valid bool
		var err error
		switch dependency.Kind {
		case "workflow_status":
			err = tx.QueryRow(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM project_workflow_statuses
					WHERE id = $1 AND project_id = $2 AND archived_at IS NULL
				)
			`, dependency.ID, projectID).Scan(&valid)
		case "label":
			err = tx.QueryRow(ctx, `
				SELECT EXISTS (SELECT 1 FROM labels WHERE id = $1 AND workspace_id = $2)
			`, dependency.ID, workspaceID).Scan(&valid)
		case "user":
			err = tx.QueryRow(ctx, `
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
			`, projectID, dependency.ID, workspaceID).Scan(&valid)
		}
		if err != nil {
			return err
		}
		if !valid {
			return fmtInvalidDependency(dependency.Kind)
		}
	}
	return nil
}

func fmtInvalidDependency(kind string) error {
	return errors.Join(errInvalidDependency, errors.New(kind+" dependency is unavailable"))
}

const ruleSelect = `
	SELECT
		automation_rule.id::text,
		automation_rule.project_id::text,
		automation_rule.name,
		automation_rule.trigger_type,
		automation_rule.conditions,
		automation_rule.actions,
		automation_rule.position,
		automation_rule.is_enabled,
		automation_rule.disabled_reason,
		automation_rule.created_by::text,
		automation_rule.created_at,
		automation_rule.updated_at
	FROM automation_rules automation_rule
`

func scanRule(row rowScanner) (ruleResponse, error) {
	var rule ruleResponse
	err := row.Scan(
		&rule.ID, &rule.ProjectID, &rule.Name, &rule.TriggerType, &rule.Conditions, &rule.Actions,
		&rule.Position, &rule.IsEnabled, &rule.DisabledReason, &rule.CreatedBy, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err == nil {
		rule.Conditions = append(json.RawMessage(nil), rule.Conditions...)
		rule.Actions = append(json.RawMessage(nil), rule.Actions...)
	}
	return rule, err
}

func sameIDSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]bool, len(left))
	for _, id := range left {
		seen[id] = true
	}
	for _, id := range right {
		if !seen[id] {
			return false
		}
	}
	return true
}
