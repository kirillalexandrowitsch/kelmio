package issues

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/projectaccess"
)

func listIssueLabelIDs(ctx context.Context, tx pgx.Tx, issueID string) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT label_id::text
		FROM issue_labels
		WHERE issue_id = $1
		ORDER BY label_id ASC
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labelIDs := make([]string, 0)
	for rows.Next() {
		var labelID string
		if err := rows.Scan(&labelID); err != nil {
			return nil, err
		}

		labelIDs = append(labelIDs, labelID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return labelIDs, nil
}

func verifyWorkspaceLabels(ctx context.Context, tx pgx.Tx, workspaceID string, labelIDs []string) error {
	for _, labelID := range labelIDs {
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM labels
				WHERE workspace_id = $1
					AND id = $2
			)
		`, workspaceID, labelID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return errInvalidLabel
		}
	}

	return nil
}

func verifyActiveWorkspaceMember(ctx context.Context, tx pgx.Tx, workspaceID string, userID string) error {
	if userID == "" {
		return nil
	}

	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM workspace_members wm
			JOIN users u ON u.id = wm.user_id
			WHERE wm.workspace_id = $1
				AND wm.user_id = $2
				AND u.is_active = true
		)
	`, workspaceID, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errInvalidAssignee
	}

	return nil
}

func verifyActiveProjectMember(
	ctx context.Context,
	tx pgx.Tx,
	user auth.CurrentUser,
	projectID string,
	userID string,
) error {
	if userID == "" {
		return nil
	}

	var exists bool
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
				AND project_member.user_id = workspace_member.user_id
			WHERE project.id = $1
				AND project.workspace_id = $3
				AND (workspace_member.role = 'admin' OR project_member.user_id IS NOT NULL)
		)
	`, projectID, userID, user.WorkspaceID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errInvalidAssignee
	}
	return nil
}

func verifyIssueParent(ctx context.Context, tx pgx.Tx, user auth.CurrentUser, issueID string, parentIssueID string) error {
	if parentIssueID == "" {
		return nil
	}
	if issueID != "" && parentIssueID == issueID {
		return errIssueParentCycle
	}
	if _, err := projectaccess.RequireIssueWrite(ctx, tx, user, parentIssueID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errInvalidIssueParent
		}
		return err
	}

	var parentExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM issues i
			JOIN projects p ON p.id = i.project_id
			WHERE i.id = $1
				AND p.workspace_id = $2
				AND p.archived_at IS NULL
				AND i.archived_at IS NULL
		)
	`, parentIssueID, user.WorkspaceID).Scan(&parentExists); err != nil {
		return err
	}
	if !parentExists {
		return errInvalidIssueParent
	}

	if issueID == "" {
		return nil
	}

	var createsCycle bool
	if err := tx.QueryRow(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT i.id, i.parent_issue_id
			FROM issues i
			JOIN projects p ON p.id = i.project_id
			WHERE i.id = $1
				AND p.workspace_id = $2
				AND p.archived_at IS NULL
				AND i.archived_at IS NULL

			UNION ALL

			SELECT parent.id, parent.parent_issue_id
			FROM issues parent
			JOIN ancestors child ON child.parent_issue_id = parent.id
			JOIN projects p ON p.id = parent.project_id
			WHERE p.workspace_id = $2
				AND p.archived_at IS NULL
				AND parent.archived_at IS NULL
		)
		SELECT EXISTS (
			SELECT 1
			FROM ancestors
			WHERE id = $3
		)
	`, parentIssueID, user.WorkspaceID, issueID).Scan(&createsCycle); err != nil {
		return err
	}
	if createsCycle {
		return errIssueParentCycle
	}

	return nil
}

func getIssueInTx(ctx context.Context, tx pgx.Tx, workspaceID string, issueID string) (issueResponse, error) {
	return scanIssue(tx.QueryRow(ctx, `
		SELECT
			i.id::text,
			i.project_id::text,
			p.key,
			i.number,
			i.issue_key,
			i.title,
			i.description,
			i.issue_type,
			i.status,
			i.priority,
			i.story_points,
			i.reporter_id::text,
			i.assignee_id::text,
			i.parent_issue_id::text,
			i.sprint_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at,
			(
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'id', l.id::text,
							'name', l.name,
							'color', l.color
						)
						ORDER BY l.name
					),
					'[]'::jsonb
				)
				FROM issue_labels il
				JOIN labels l ON l.id = il.label_id
				WHERE il.issue_id = i.id
			)
		FROM issues i
		JOIN projects p ON p.id = i.project_id
	WHERE i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
	`, issueID, workspaceID))
}

func issueLinkExists(ctx context.Context, tx pgx.Tx, sourceIssueID string, targetIssueID string, linkType string) (bool, error) {
	var exists bool
	if linkType == "relates" {
		err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM issue_links
				WHERE link_type = 'relates'
					AND (
						(source_issue_id = $1 AND target_issue_id = $2)
						OR (source_issue_id = $2 AND target_issue_id = $1)
					)
			)
		`, sourceIssueID, targetIssueID).Scan(&exists)

		return exists, err
	}

	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM issue_links
			WHERE source_issue_id = $1
				AND target_issue_id = $2
				AND link_type = $3
		)
	`, sourceIssueID, targetIssueID, linkType).Scan(&exists)

	return exists, err
}

func getIssueLinkInTx(ctx context.Context, tx pgx.Tx, workspaceID string, linkID string) (issueLinkResponse, error) {
	return scanIssueLink(tx.QueryRow(ctx, `
		SELECT
			il.id::text,
			il.source_issue_id::text,
			il.target_issue_id::text,
			il.link_type,
			il.created_by::text,
			il.created_at,
			source_issue.id::text,
			source_issue.issue_key,
			source_issue.title,
			source_issue.issue_type,
			source_issue.status,
			source_issue.priority,
			target_issue.id::text,
			target_issue.issue_key,
			target_issue.title,
			target_issue.issue_type,
			target_issue.status,
			target_issue.priority
		FROM issue_links il
		JOIN issues source_issue ON source_issue.id = il.source_issue_id
		JOIN projects source_project ON source_project.id = source_issue.project_id
		JOIN issues target_issue ON target_issue.id = il.target_issue_id
		JOIN projects target_project ON target_project.id = target_issue.project_id
		WHERE il.id = $1
			AND source_project.workspace_id = $2
			AND target_project.workspace_id = $2
			AND source_project.archived_at IS NULL
			AND target_project.archived_at IS NULL
			AND source_issue.archived_at IS NULL
			AND target_issue.archived_at IS NULL
	`, linkID, workspaceID))
}

func getIssueLinkForIssueInTx(ctx context.Context, tx pgx.Tx, workspaceID string, issueID string, linkID string) (issueLinkResponse, error) {
	return scanIssueLink(tx.QueryRow(ctx, `
		SELECT
			il.id::text,
			il.source_issue_id::text,
			il.target_issue_id::text,
			il.link_type,
			il.created_by::text,
			il.created_at,
			source_issue.id::text,
			source_issue.issue_key,
			source_issue.title,
			source_issue.issue_type,
			source_issue.status,
			source_issue.priority,
			target_issue.id::text,
			target_issue.issue_key,
			target_issue.title,
			target_issue.issue_type,
			target_issue.status,
			target_issue.priority
		FROM issue_links il
		JOIN issues source_issue ON source_issue.id = il.source_issue_id
		JOIN projects source_project ON source_project.id = source_issue.project_id
		JOIN issues target_issue ON target_issue.id = il.target_issue_id
		JOIN projects target_project ON target_project.id = target_issue.project_id
		WHERE il.id = $1
			AND (il.source_issue_id = $2 OR il.target_issue_id = $2)
			AND source_project.workspace_id = $3
			AND target_project.workspace_id = $3
			AND source_project.archived_at IS NULL
			AND target_project.archived_at IS NULL
			AND source_issue.archived_at IS NULL
			AND target_issue.archived_at IS NULL
		FOR UPDATE OF il
	`, linkID, issueID, workspaceID))
}

func insertIssueLinkActivity(ctx context.Context, tx pgx.Tx, link issueLinkResponse, actorID string, action string) error {
	payload := map[string]string{
		"link_id":          link.ID,
		"link_type":        link.LinkType,
		"source_issue_id":  link.SourceIssueID,
		"source_issue_key": link.SourceIssue.IssueKey,
		"target_issue_id":  link.TargetIssueID,
		"target_issue_key": link.TargetIssue.IssueKey,
	}

	if err := insertIssueActivity(ctx, tx, link.SourceIssueID, actorID, action, payload); err != nil {
		return err
	}
	if link.TargetIssueID != link.SourceIssueID {
		if err := insertIssueActivity(ctx, tx, link.TargetIssueID, actorID, action, payload); err != nil {
			return err
		}
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}

func changedIssueFields(previous issueResponse, current issueResponse) []string {
	fields := make([]string, 0, 6)

	if previous.Title != current.Title {
		fields = append(fields, "title")
	}
	if previous.Description != current.Description {
		fields = append(fields, "description")
	}
	if previous.IssueType != current.IssueType {
		fields = append(fields, "issue_type")
	}
	if previous.Priority != current.Priority {
		fields = append(fields, "priority")
	}
	if previous.StoryPoints != current.StoryPoints {
		fields = append(fields, "story_points")
	}
	if stringOrEmpty(previous.DueDate) != stringOrEmpty(current.DueDate) {
		fields = append(fields, "due_date")
	}

	return fields
}

type rowScanner interface {
	Scan(dest ...any) error
}
