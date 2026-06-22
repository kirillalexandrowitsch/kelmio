package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"kelmio/backend/internal/auth"
	"kelmio/backend/internal/automations"
	"kelmio/backend/internal/projectaccess"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func (h *Handler) getWorkflow(ctx context.Context, workspaceID string, projectID string, users ...auth.CurrentUser) (workflowResponse, error) {
	if len(users) > 0 {
		if _, err := projectaccess.RequireRead(ctx, h.db, users[0], projectID); err != nil {
			return workflowResponse{}, err
		}
	} else if err := h.requireActiveProject(ctx, h.db, workspaceID, projectID, false); err != nil {
		return workflowResponse{}, err
	}

	statuses, err := listStatuses(ctx, h.db, projectID)
	if err != nil {
		return workflowResponse{}, err
	}
	transitions, err := listTransitions(ctx, h.db, projectID)
	if err != nil {
		return workflowResponse{}, err
	}
	return workflowResponse{ProjectID: projectID, Statuses: statuses, Transitions: transitions}, nil
}

func (h *Handler) createWorkflowStatus(
	ctx context.Context,
	workspaceID string,
	projectID string,
	input normalizedCreateStatus,
	users ...auth.CurrentUser,
) (statusResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return statusResponse{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := h.requireManageProject(ctx, tx, workspaceID, projectID, users); err != nil {
		return statusResponse{}, err
	}

	status, err := scanStatus(tx.QueryRow(ctx, `
		INSERT INTO project_workflow_statuses (
			project_id,
			key,
			name,
			color,
			category,
			position
		)
		SELECT
			$1,
			$2,
			$3,
			$4,
			$5,
			COALESCE(MAX(position), 0) + 100
		FROM project_workflow_statuses
		WHERE project_id = $1
		RETURNING
			id::text,
			project_id::text,
			key,
			name,
			color,
			category,
			position,
			created_at,
			updated_at,
			archived_at
	`, projectID, input.Key, input.Name, input.Color, input.Category))
	if err != nil {
		return statusResponse{}, mapStatusConstraintError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return statusResponse{}, err
	}
	return status, nil
}

func (h *Handler) updateWorkflowStatus(
	ctx context.Context,
	workspaceID string,
	projectID string,
	statusID string,
	input normalizedUpdateStatus,
	users ...auth.CurrentUser,
) (statusResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return statusResponse{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := h.requireManageProject(ctx, tx, workspaceID, projectID, users); err != nil {
		return statusResponse{}, err
	}
	current, err := getStatusForUpdate(ctx, tx, projectID, statusID)
	if err != nil {
		return statusResponse{}, err
	}
	if current.ArchivedAt != nil {
		return statusResponse{}, errStatusArchived
	}
	if current.Category == "done" && input.HasCategory && input.Category != "done" {
		if err := ensureAnotherDoneStatus(ctx, tx, projectID, statusID); err != nil {
			return statusResponse{}, err
		}
	}

	status, err := scanStatus(tx.QueryRow(ctx, `
		UPDATE project_workflow_statuses
		SET name = CASE WHEN $3 THEN $4 ELSE name END,
			color = CASE WHEN $5 THEN $6 ELSE color END,
			category = CASE WHEN $7 THEN $8 ELSE category END,
			updated_at = now()
		WHERE project_id = $1
			AND id = $2
			AND archived_at IS NULL
		RETURNING
			id::text,
			project_id::text,
			key,
			name,
			color,
			category,
			position,
			created_at,
			updated_at,
			archived_at
	`, projectID, statusID, input.HasName, input.Name, input.HasColor, input.Color, input.HasCategory, input.Category))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return statusResponse{}, errStatusNotFound
		}
		return statusResponse{}, mapStatusConstraintError(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return statusResponse{}, err
	}
	return status, nil
}

func (h *Handler) reorderWorkflowStatuses(
	ctx context.Context,
	workspaceID string,
	projectID string,
	statusIDs []string,
	users ...auth.CurrentUser,
) error {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := h.requireManageProject(ctx, tx, workspaceID, projectID, users); err != nil {
		return err
	}
	activeIDs, err := activeStatusIDsForUpdate(ctx, tx, projectID)
	if err != nil {
		return err
	}
	if !sameIDSet(activeIDs, statusIDs) {
		return errStatusOrderMismatch
	}
	for index, statusID := range statusIDs {
		if _, err := tx.Exec(ctx, `
			UPDATE project_workflow_statuses
			SET position = $3,
				updated_at = now()
			WHERE project_id = $1
				AND id = $2
				AND archived_at IS NULL
		`, projectID, statusID, (index+1)*100); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (h *Handler) archiveWorkflowStatus(
	ctx context.Context,
	user auth.CurrentUser,
	projectID string,
	statusID string,
	replacementStatusID string,
) (statusResponse, error) {
	if statusID == replacementStatusID {
		return statusResponse{}, errors.New("replacement status must be different from archived status")
	}

	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return statusResponse{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := projectaccess.RequireManageForUpdate(ctx, tx, user, projectID); err != nil {
		return statusResponse{}, err
	}
	status, err := getStatusForUpdate(ctx, tx, projectID, statusID)
	if err != nil {
		return statusResponse{}, err
	}
	if status.ArchivedAt != nil {
		return statusResponse{}, errStatusArchived
	}
	replacement, err := getStatusForUpdate(ctx, tx, projectID, replacementStatusID)
	if err != nil {
		return statusResponse{}, err
	}
	if replacement.ArchivedAt != nil {
		return statusResponse{}, errStatusArchived
	}
	if status.Category == "done" {
		if err := ensureAnotherDoneStatus(ctx, tx, projectID, statusID); err != nil {
			return statusResponse{}, err
		}
	}

	payload, err := json.Marshal(map[string]string{
		"from_status": status.Key,
		"to_status":   replacement.Key,
	})
	if err != nil {
		return statusResponse{}, err
	}
	if _, err := tx.Exec(ctx, `
		WITH updated_issues AS (
			UPDATE issues
			SET workflow_status_id = $3,
				updated_at = now()
			WHERE project_id = $1
				AND workflow_status_id = $2
			RETURNING id
		)
		INSERT INTO activity_log (entity_type, entity_id, action, actor_id, payload)
		SELECT 'issue', id, 'status_changed', $4, $5::jsonb
		FROM updated_issues
	`, projectID, statusID, replacement.ID, user.ID, string(payload)); err != nil {
		return statusResponse{}, err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM project_workflow_transitions
		WHERE project_id = $1
			AND (from_status_id = $2 OR to_status_id = $2)
	`, projectID, statusID); err != nil {
		return statusResponse{}, err
	}

	archived, err := scanStatus(tx.QueryRow(ctx, `
		UPDATE project_workflow_statuses
		SET archived_at = now(),
			updated_at = now()
		WHERE project_id = $1
			AND id = $2
			AND archived_at IS NULL
		RETURNING
			id::text,
			project_id::text,
			key,
			name,
			color,
			category,
			position,
			created_at,
			updated_at,
			archived_at
	`, projectID, statusID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return statusResponse{}, errStatusNotFound
		}
		return statusResponse{}, err
	}
	if err := automations.DisableRulesForWorkflowStatus(ctx, tx, projectID, statusID); err != nil {
		return statusResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return statusResponse{}, err
	}
	return archived, nil
}

func (h *Handler) replaceWorkflowTransitions(
	ctx context.Context,
	workspaceID string,
	projectID string,
	transitions []normalizedTransition,
	users ...auth.CurrentUser,
) error {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := h.requireManageProject(ctx, tx, workspaceID, projectID, users); err != nil {
		return err
	}
	activeIDs, err := activeStatusIDsForUpdate(ctx, tx, projectID)
	if err != nil {
		return err
	}
	active := make(map[string]bool, len(activeIDs))
	for _, statusID := range activeIDs {
		active[statusID] = true
	}
	for _, transition := range transitions {
		if !active[transition.FromStatusID] || !active[transition.ToStatusID] {
			return errInvalidTransitionStatus
		}
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM project_workflow_transitions
		WHERE project_id = $1
	`, projectID); err != nil {
		return err
	}
	for _, transition := range transitions {
		if _, err := tx.Exec(ctx, `
			INSERT INTO project_workflow_transitions (project_id, from_status_id, to_status_id)
			VALUES ($1, $2, $3)
		`, projectID, transition.FromStatusID, transition.ToStatusID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (h *Handler) requireManageProject(
	ctx context.Context,
	querier projectaccess.Querier,
	workspaceID string,
	projectID string,
	users []auth.CurrentUser,
) error {
	if len(users) > 0 {
		_, err := projectaccess.RequireManageForUpdate(ctx, querier, users[0], projectID)
		return err
	}
	return h.requireActiveProject(ctx, querier, workspaceID, projectID, true)
}

func (h *Handler) requireActiveProject(
	ctx context.Context,
	querier interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	workspaceID string,
	projectID string,
	forUpdate bool,
) error {
	lock := ""
	if forUpdate {
		lock = " FOR UPDATE"
	}
	var id string
	return querier.QueryRow(ctx, `
		SELECT id::text
		FROM projects
		WHERE id = $1
			AND workspace_id = $2
			AND archived_at IS NULL
	`+lock, projectID, workspaceID).Scan(&id)
}

func getStatusForUpdate(ctx context.Context, tx pgx.Tx, projectID string, statusID string) (statusResponse, error) {
	status, err := scanStatus(tx.QueryRow(ctx, `
		SELECT
			id::text,
			project_id::text,
			key,
			name,
			color,
			category,
			position,
			created_at,
			updated_at,
			archived_at
		FROM project_workflow_statuses
		WHERE project_id = $1
			AND id = $2
		FOR UPDATE
	`, projectID, statusID))
	if errors.Is(err, pgx.ErrNoRows) {
		return statusResponse{}, errStatusNotFound
	}
	return status, err
}

func ensureAnotherDoneStatus(ctx context.Context, tx pgx.Tx, projectID string, excludedStatusID string) error {
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)::int
		FROM project_workflow_statuses
		WHERE project_id = $1
			AND id <> $2
			AND category = 'done'
			AND archived_at IS NULL
	`, projectID, excludedStatusID).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return errRequiresDoneStatus
	}
	return nil
}

func activeStatusIDsForUpdate(ctx context.Context, tx pgx.Tx, projectID string) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text
		FROM project_workflow_statuses
		WHERE project_id = $1
			AND archived_at IS NULL
		ORDER BY position, id
		FOR UPDATE
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func listStatuses(ctx context.Context, querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, projectID string) ([]statusResponse, error) {
	rows, err := querier.Query(ctx, `
		SELECT
			id::text,
			project_id::text,
			key,
			name,
			color,
			category,
			position,
			created_at,
			updated_at,
			archived_at
		FROM project_workflow_statuses
		WHERE project_id = $1
		ORDER BY archived_at NULLS FIRST, position, id
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	statuses := make([]statusResponse, 0)
	for rows.Next() {
		status, err := scanStatus(rows)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}
	return statuses, rows.Err()
}

func listTransitions(ctx context.Context, querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, projectID string) ([]transitionResponse, error) {
	rows, err := querier.Query(ctx, `
		SELECT
			transition.from_status_id::text,
			transition.to_status_id::text,
			transition.created_at
		FROM project_workflow_transitions transition
		JOIN project_workflow_statuses source ON source.id = transition.from_status_id
		JOIN project_workflow_statuses target ON target.id = transition.to_status_id
		WHERE transition.project_id = $1
			AND source.archived_at IS NULL
			AND target.archived_at IS NULL
		ORDER BY source.position, target.position, transition.from_status_id, transition.to_status_id
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	transitions := make([]transitionResponse, 0)
	for rows.Next() {
		var transition transitionResponse
		if err := rows.Scan(&transition.FromStatusID, &transition.ToStatusID, &transition.CreatedAt); err != nil {
			return nil, err
		}
		transitions = append(transitions, transition)
	}
	return transitions, rows.Err()
}

func scanStatus(row rowScanner) (statusResponse, error) {
	var status statusResponse
	err := row.Scan(
		&status.ID,
		&status.ProjectID,
		&status.Key,
		&status.Name,
		&status.Color,
		&status.Category,
		&status.Position,
		&status.CreatedAt,
		&status.UpdatedAt,
		&status.ArchivedAt,
	)
	return status, err
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

func mapStatusConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return err
	}
	switch pgErr.ConstraintName {
	case "project_workflow_statuses_project_key_unique":
		return errStatusKeyExists
	case "idx_project_workflow_statuses_active_name":
		return errStatusNameExists
	default:
		return fmt.Errorf("workflow status conflict: %w", err)
	}
}
