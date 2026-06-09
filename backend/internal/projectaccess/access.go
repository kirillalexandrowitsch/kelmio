package projectaccess

import (
	"context"

	"github.com/jackc/pgx/v5"

	"team-task-tracker/backend/internal/auth"
)

type Querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Access struct {
	ProjectID        string
	WorkspaceID      string
	WorkspaceRole    string
	ProjectRole      string
	IsWorkspaceAdmin bool
	CanRead          bool
	CanManage        bool
}

func Resolve(ctx context.Context, db Querier, user auth.CurrentUser, projectID string) (Access, error) {
	var access Access
	err := db.QueryRow(ctx, `
		SELECT
			project.id::text,
			project.workspace_id::text,
			workspace_member.role,
			COALESCE(project_member.role, '')
		FROM projects project
		JOIN workspace_members workspace_member
			ON workspace_member.workspace_id = project.workspace_id
			AND workspace_member.user_id = $3
		JOIN users app_user
			ON app_user.id = workspace_member.user_id
			AND app_user.is_active = true
		LEFT JOIN project_members project_member
			ON project_member.project_id = project.id
			AND project_member.user_id = workspace_member.user_id
		WHERE project.id = $1
			AND project.workspace_id = $2
			AND project.archived_at IS NULL
	`, projectID, user.WorkspaceID, user.ID).Scan(
		&access.ProjectID,
		&access.WorkspaceID,
		&access.WorkspaceRole,
		&access.ProjectRole,
	)
	if err != nil {
		return Access{}, err
	}

	access.IsWorkspaceAdmin, access.CanRead, access.CanManage = permissionsForRoles(
		access.WorkspaceRole,
		access.ProjectRole,
	)
	return access, nil
}

func permissionsForRoles(workspaceRole string, projectRole string) (bool, bool, bool) {
	isWorkspaceAdmin := workspaceRole == "admin"
	canRead := isWorkspaceAdmin || projectRole == "lead" || projectRole == "contributor" || projectRole == "viewer"
	canManage := isWorkspaceAdmin || projectRole == "lead"
	return isWorkspaceAdmin, canRead, canManage
}
