package projectaccess

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"kelmio/backend/internal/auth"
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
	CanWrite         bool
	CanManage        bool
}

var ErrForbidden = errors.New("project access forbidden")

func Resolve(ctx context.Context, db Querier, user auth.CurrentUser, projectID string) (Access, error) {
	return resolve(ctx, db, user, `
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
	`, projectID)
}

func RequireRead(ctx context.Context, db Querier, user auth.CurrentUser, projectID string) (Access, error) {
	access, err := Resolve(ctx, db, user, projectID)
	return require(access, err, permissionRead)
}

func RequireWrite(ctx context.Context, db Querier, user auth.CurrentUser, projectID string) (Access, error) {
	access, err := Resolve(ctx, db, user, projectID)
	return require(access, err, permissionWrite)
}

func RequireWriteForUpdate(ctx context.Context, db Querier, user auth.CurrentUser, projectID string) (Access, error) {
	access, err := resolveProjectForUpdate(ctx, db, user, projectID)
	return require(access, err, permissionWrite)
}

func RequireManage(ctx context.Context, db Querier, user auth.CurrentUser, projectID string) (Access, error) {
	access, err := Resolve(ctx, db, user, projectID)
	return require(access, err, permissionManage)
}

func RequireManageForUpdate(ctx context.Context, db Querier, user auth.CurrentUser, projectID string) (Access, error) {
	access, err := resolveProjectForUpdate(ctx, db, user, projectID)
	return require(access, err, permissionManage)
}

func RequireIssueRead(ctx context.Context, db Querier, user auth.CurrentUser, issueID string) (Access, error) {
	access, err := resolveForIssue(ctx, db, user, issueID, false)
	return require(access, err, permissionRead)
}

func RequireIssueWrite(ctx context.Context, db Querier, user auth.CurrentUser, issueID string) (Access, error) {
	access, err := resolveForIssue(ctx, db, user, issueID, true)
	return require(access, err, permissionWrite)
}

func RequireSprintRead(ctx context.Context, db Querier, user auth.CurrentUser, sprintID string) (Access, error) {
	access, err := resolveForSprint(ctx, db, user, sprintID, false)
	return require(access, err, permissionRead)
}

func RequireSprintWrite(ctx context.Context, db Querier, user auth.CurrentUser, sprintID string) (Access, error) {
	access, err := resolveForSprint(ctx, db, user, sprintID, true)
	return require(access, err, permissionWrite)
}

func resolveProjectForUpdate(ctx context.Context, db Querier, user auth.CurrentUser, projectID string) (Access, error) {
	return resolve(ctx, db, user, `
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
		FOR UPDATE OF project
	`, projectID)
}

func resolveForIssue(ctx context.Context, db Querier, user auth.CurrentUser, issueID string, lockProject bool) (Access, error) {
	lockClause := ""
	if lockProject {
		lockClause = " FOR UPDATE OF project"
	}
	return resolve(ctx, db, user, `
		SELECT
			project.id::text,
			project.workspace_id::text,
			workspace_member.role,
			COALESCE(project_member.role, '')
		FROM issues issue
		JOIN projects project ON project.id = issue.project_id
		JOIN workspace_members workspace_member
			ON workspace_member.workspace_id = project.workspace_id
			AND workspace_member.user_id = $3
		JOIN users app_user
			ON app_user.id = workspace_member.user_id
			AND app_user.is_active = true
		LEFT JOIN project_members project_member
			ON project_member.project_id = project.id
			AND project_member.user_id = workspace_member.user_id
		WHERE issue.id = $1
			AND project.workspace_id = $2
			AND project.archived_at IS NULL
			AND issue.archived_at IS NULL
	`+lockClause, issueID)
}

func resolveForSprint(ctx context.Context, db Querier, user auth.CurrentUser, sprintID string, lockProject bool) (Access, error) {
	lockClause := ""
	if lockProject {
		lockClause = " FOR UPDATE OF project"
	}
	return resolve(ctx, db, user, `
		SELECT
			project.id::text,
			project.workspace_id::text,
			workspace_member.role,
			COALESCE(project_member.role, '')
		FROM sprints sprint
		JOIN projects project ON project.id = sprint.project_id
		JOIN workspace_members workspace_member
			ON workspace_member.workspace_id = project.workspace_id
			AND workspace_member.user_id = $3
		JOIN users app_user
			ON app_user.id = workspace_member.user_id
			AND app_user.is_active = true
		LEFT JOIN project_members project_member
			ON project_member.project_id = project.id
			AND project_member.user_id = workspace_member.user_id
		WHERE sprint.id = $1
			AND project.workspace_id = $2
			AND project.archived_at IS NULL
	`+lockClause, sprintID)
}

func resolve(ctx context.Context, db Querier, user auth.CurrentUser, query string, resourceID string) (Access, error) {
	var access Access
	err := db.QueryRow(ctx, query, resourceID, user.WorkspaceID, user.ID).Scan(
		&access.ProjectID,
		&access.WorkspaceID,
		&access.WorkspaceRole,
		&access.ProjectRole,
	)
	if err != nil {
		return Access{}, err
	}

	access.IsWorkspaceAdmin, access.CanRead, access.CanWrite, access.CanManage = permissionsForRoles(
		access.WorkspaceRole,
		access.ProjectRole,
	)
	return access, nil
}

type permission int

const (
	permissionRead permission = iota
	permissionWrite
	permissionManage
)

func require(access Access, err error, required permission) (Access, error) {
	if err != nil {
		return Access{}, err
	}
	if !access.CanRead {
		return Access{}, pgx.ErrNoRows
	}
	if required == permissionWrite && !access.CanWrite {
		return Access{}, ErrForbidden
	}
	if required == permissionManage && !access.CanManage {
		return Access{}, ErrForbidden
	}
	return access, nil
}

func permissionsForRoles(workspaceRole string, projectRole string) (bool, bool, bool, bool) {
	isWorkspaceAdmin := workspaceRole == "admin"
	canRead := isWorkspaceAdmin || projectRole == "lead" || projectRole == "contributor" || projectRole == "viewer"
	canWrite := isWorkspaceAdmin || projectRole == "lead" || projectRole == "contributor"
	canManage := isWorkspaceAdmin || projectRole == "lead"
	return isWorkspaceAdmin, canRead, canWrite, canManage
}
