package automations

import (
	"context"

	"github.com/jackc/pgx/v5"
)

const (
	DisabledWorkflowStatusUnavailable = "workflow_status_unavailable"
	DisabledLabelUnavailable          = "label_unavailable"
	DisabledUserUnavailable           = "user_unavailable"
	DisabledProjectAccessRemoved      = "project_access_removed"
)

func DisableRulesForWorkflowStatus(ctx context.Context, tx pgx.Tx, projectID string, statusID string) error {
	return disableRulesForJSONDependency(ctx, tx, projectID, "workflow_status_id", statusID, DisabledWorkflowStatusUnavailable)
}

func DisableRulesForLabel(ctx context.Context, tx pgx.Tx, workspaceID string, labelID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE automation_rules automation_rule
		SET is_enabled = false,
			disabled_reason = $3,
			updated_at = now()
		FROM projects project
		WHERE project.id = automation_rule.project_id
			AND project.workspace_id = $1
			AND automation_rule.is_enabled = true
			AND (
				automation_rule.conditions @> jsonb_build_array(jsonb_build_object('label_id', $2::text))
				OR automation_rule.actions @> jsonb_build_array(jsonb_build_object('label_id', $2::text))
			)
	`, workspaceID, labelID, DisabledLabelUnavailable)
	return err
}

func DisableRulesForWorkspaceUser(ctx context.Context, tx pgx.Tx, workspaceID string, userID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE automation_rules automation_rule
		SET is_enabled = false,
			disabled_reason = CASE
				WHEN app_user.is_active = false THEN $3
				ELSE $4
			END,
			updated_at = now()
		FROM projects project
		JOIN workspace_members workspace_member
			ON workspace_member.workspace_id = project.workspace_id
			AND workspace_member.user_id = $2
		JOIN users app_user ON app_user.id = workspace_member.user_id
		LEFT JOIN project_members project_member
			ON project_member.project_id = project.id
			AND project_member.user_id = app_user.id
		WHERE automation_rule.project_id = project.id
			AND project.workspace_id = $1
			AND automation_rule.is_enabled = true
			AND (
				automation_rule.conditions @> jsonb_build_array(jsonb_build_object('user_id', $2::text))
				OR automation_rule.actions @> jsonb_build_array(jsonb_build_object('user_id', $2::text))
			)
			AND (
				app_user.is_active = false
				OR (workspace_member.role <> 'admin' AND project_member.user_id IS NULL)
			)
	`, workspaceID, userID, DisabledUserUnavailable, DisabledProjectAccessRemoved)
	return err
}

func DisableRulesForProjectUser(ctx context.Context, tx pgx.Tx, projectID string, userID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE automation_rules automation_rule
		SET is_enabled = false,
			disabled_reason = $3,
			updated_at = now()
		FROM projects project
		JOIN workspace_members workspace_member
			ON workspace_member.workspace_id = project.workspace_id
			AND workspace_member.user_id = $2
		JOIN users app_user ON app_user.id = workspace_member.user_id
		LEFT JOIN project_members project_member
			ON project_member.project_id = project.id
			AND project_member.user_id = app_user.id
		WHERE automation_rule.project_id = project.id
			AND project.id = $1
			AND automation_rule.is_enabled = true
			AND (
				automation_rule.conditions @> jsonb_build_array(jsonb_build_object('user_id', $2::text))
				OR automation_rule.actions @> jsonb_build_array(jsonb_build_object('user_id', $2::text))
			)
			AND app_user.is_active = true
			AND workspace_member.role <> 'admin'
			AND project_member.user_id IS NULL
	`, projectID, userID, DisabledProjectAccessRemoved)
	return err
}

func disableRulesForJSONDependency(ctx context.Context, tx pgx.Tx, projectID string, key string, value string, reason string) error {
	_, err := tx.Exec(ctx, `
		UPDATE automation_rules
		SET is_enabled = false,
			disabled_reason = $4,
			updated_at = now()
		WHERE project_id = $1
			AND is_enabled = true
			AND (
				conditions @> jsonb_build_array(jsonb_build_object($2::text, $3::text))
				OR actions @> jsonb_build_array(jsonb_build_object($2::text, $3::text))
			)
	`, projectID, key, value, reason)
	return err
}
