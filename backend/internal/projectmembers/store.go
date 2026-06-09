package projectmembers

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"team-task-tracker/backend/internal/auth"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func (h *Handler) listProjectMembers(ctx context.Context, projectID string) ([]memberResponse, error) {
	rows, err := h.db.Query(ctx, `
		SELECT
			project_member.project_id::text,
			project_member.user_id::text,
			app_user.email,
			app_user.username,
			app_user.display_name,
			project_member.role,
			workspace_member.role,
			app_user.is_active,
			project_member.created_at,
			project_member.updated_at
		FROM project_members project_member
		JOIN projects project
			ON project.id = project_member.project_id
		JOIN workspace_members workspace_member
			ON workspace_member.workspace_id = project.workspace_id
			AND workspace_member.user_id = project_member.user_id
		JOIN users app_user
			ON app_user.id = project_member.user_id
		WHERE project_member.project_id = $1
		ORDER BY
			app_user.is_active DESC,
			CASE project_member.role
				WHEN 'lead' THEN 1
				WHEN 'contributor' THEN 2
				ELSE 3
			END,
			lower(app_user.display_name),
			project_member.user_id
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]memberResponse, 0)
	for rows.Next() {
		member, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (h *Handler) putProjectMember(
	ctx context.Context,
	actor auth.CurrentUser,
	projectID string,
	userID string,
	role string,
) (memberResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return memberResponse{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockActiveProject(ctx, tx, actor.WorkspaceID, projectID); err != nil {
		return memberResponse{}, err
	}
	if err := h.requireManager(ctx, tx, actor, projectID); err != nil {
		return memberResponse{}, err
	}

	target, err := activeWorkspaceMemberForUpdate(ctx, tx, actor.WorkspaceID, userID)
	if err != nil {
		return memberResponse{}, err
	}

	var currentRole string
	err = tx.QueryRow(ctx, `
		SELECT role
		FROM project_members
		WHERE project_id = $1
			AND user_id = $2
		FOR UPDATE
	`, projectID, userID).Scan(&currentRole)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return memberResponse{}, err
	}
	if err == nil && currentRole == "lead" && role != "lead" {
		if err := ensureProjectKeepsManager(ctx, tx, actor.WorkspaceID, projectID, userID); err != nil {
			return memberResponse{}, err
		}
	}

	var member memberResponse
	if err := tx.QueryRow(ctx, `
		INSERT INTO project_members (project_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (project_id, user_id) DO UPDATE
		SET role = EXCLUDED.role,
			updated_at = now()
		RETURNING
			project_id::text,
			user_id::text,
			role,
			created_at,
			updated_at
	`, projectID, userID, role).Scan(
		&member.ProjectID,
		&member.UserID,
		&member.Role,
		&member.CreatedAt,
		&member.UpdatedAt,
	); err != nil {
		return memberResponse{}, err
	}
	member.Email = target.Email
	member.Username = target.Username
	member.DisplayName = target.DisplayName
	member.WorkspaceRole = target.WorkspaceRole
	member.IsActive = target.IsActive

	if err := tx.Commit(ctx); err != nil {
		return memberResponse{}, err
	}
	return member, nil
}

func (h *Handler) deleteProjectMember(
	ctx context.Context,
	actor auth.CurrentUser,
	projectID string,
	userID string,
) error {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockActiveProject(ctx, tx, actor.WorkspaceID, projectID); err != nil {
		return err
	}
	if err := h.requireManager(ctx, tx, actor, projectID); err != nil {
		return err
	}

	var role string
	var isActive bool
	if err := tx.QueryRow(ctx, `
		SELECT project_member.role, app_user.is_active
		FROM project_members project_member
		JOIN users app_user ON app_user.id = project_member.user_id
		WHERE project_member.project_id = $1
			AND project_member.user_id = $2
		FOR UPDATE OF project_member, app_user
	`, projectID, userID).Scan(&role, &isActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errProjectMemberNotFound
		}
		return err
	}
	if role == "lead" && isActive {
		if err := ensureProjectKeepsManager(ctx, tx, actor.WorkspaceID, projectID, userID); err != nil {
			return err
		}
	}
	commandTag, err := tx.Exec(ctx, `
		DELETE FROM project_members
		WHERE project_id = $1
			AND user_id = $2
	`, projectID, userID)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return errProjectMemberNotFound
	}
	return tx.Commit(ctx)
}

type workspaceMemberRecord struct {
	Email         string
	Username      string
	DisplayName   string
	WorkspaceRole string
	IsActive      bool
}

func activeWorkspaceMemberForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	workspaceID string,
	userID string,
) (workspaceMemberRecord, error) {
	var member workspaceMemberRecord
	err := tx.QueryRow(ctx, `
		SELECT
			app_user.email,
			app_user.username,
			app_user.display_name,
			workspace_member.role,
			app_user.is_active
		FROM workspace_members workspace_member
		JOIN users app_user ON app_user.id = workspace_member.user_id
		WHERE workspace_member.workspace_id = $1
			AND workspace_member.user_id = $2
			AND app_user.is_active = true
		FOR UPDATE OF workspace_member, app_user
	`, workspaceID, userID).Scan(
		&member.Email,
		&member.Username,
		&member.DisplayName,
		&member.WorkspaceRole,
		&member.IsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return workspaceMemberRecord{}, errWorkspaceMemberNotFound
	}
	return member, err
}

func ensureProjectKeepsManager(
	ctx context.Context,
	tx pgx.Tx,
	workspaceID string,
	projectID string,
	excludedUserID string,
) error {
	var managerCount int
	if err := tx.QueryRow(ctx, `
		SELECT (
			(
				SELECT count(*)
				FROM workspace_members workspace_member
				JOIN users app_user ON app_user.id = workspace_member.user_id
				WHERE workspace_member.workspace_id = $1
					AND workspace_member.role = 'admin'
					AND app_user.is_active = true
			)
			+
			(
				SELECT count(*)
				FROM project_members project_member
				JOIN users app_user ON app_user.id = project_member.user_id
				WHERE project_member.project_id = $2
					AND project_member.user_id <> $3
					AND project_member.role = 'lead'
					AND app_user.is_active = true
			)
		)::int
	`, workspaceID, projectID, excludedUserID).Scan(&managerCount); err != nil {
		return err
	}
	if managerCount == 0 {
		return errProjectRequiresLead
	}
	return nil
}

func lockActiveProject(ctx context.Context, tx pgx.Tx, workspaceID string, projectID string) error {
	var id string
	return tx.QueryRow(ctx, `
		SELECT id::text
		FROM projects
		WHERE id = $1
			AND workspace_id = $2
			AND archived_at IS NULL
		FOR UPDATE
	`, projectID, workspaceID).Scan(&id)
}

func scanMember(row rowScanner) (memberResponse, error) {
	var member memberResponse
	err := row.Scan(
		&member.ProjectID,
		&member.UserID,
		&member.Email,
		&member.Username,
		&member.DisplayName,
		&member.Role,
		&member.WorkspaceRole,
		&member.IsActive,
		&member.CreatedAt,
		&member.UpdatedAt,
	)
	return member, err
}
