package sprints

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"team-task-tracker/backend/internal/notifications"
)

func getSprintForUpdate(ctx context.Context, querier sprintQuerier, workspaceID string, sprintID string, forUpdate bool) (sprintResponse, error) {
	lockClause := ""
	if forUpdate {
		lockClause = " FOR UPDATE OF s"
	}

	return scanSprint(querier.QueryRow(ctx, `
		SELECT
			s.id::text,
			s.workspace_id::text,
			s.project_id::text,
			p.key,
			p.name,
			s.name,
			s.goal,
			s.status,
			s.start_date::text,
			s.end_date::text,
			s.created_by::text,
			s.created_at,
			s.completed_at,
			(
				SELECT COUNT(*)::int
				FROM issues i
				WHERE i.sprint_id = s.id
					AND i.archived_at IS NULL
			),
			(
				SELECT COUNT(*)::int
				FROM issues i
				WHERE i.sprint_id = s.id
					AND i.archived_at IS NULL
					AND i.status = 'done'
			),
			(
				SELECT COALESCE(SUM(i.story_points), 0)::int
				FROM issues i
				WHERE i.sprint_id = s.id
					AND i.archived_at IS NULL
			),
			(
				SELECT COALESCE(SUM(i.story_points), 0)::int
				FROM issues i
				WHERE i.sprint_id = s.id
					AND i.archived_at IS NULL
					AND i.status = 'done'
			),
			(
				SELECT COALESCE(SUM(i.story_points), 0)::int
				FROM issues i
				WHERE i.sprint_id = s.id
					AND i.archived_at IS NULL
					AND i.status <> 'done'
			)
		FROM sprints s
		JOIN projects p ON p.id = s.project_id
		WHERE s.id = $1
			AND s.workspace_id = $2
			AND p.archived_at IS NULL
	`+lockClause, sprintID, workspaceID))
}

func dateOrNil(value string) any {
	if value == "" {
		return nil
	}

	return value
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func insertIssueActivity(ctx context.Context, tx pgx.Tx, issueID string, actorID string, action string, payload map[string]string) error {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO activity_log (entity_type, entity_id, action, actor_id, payload)
		VALUES ('issue', $1, $2, $3, $4::jsonb)
	`, issueID, action, actorID, string(encodedPayload))

	return err
}

func scanSprint(row rowScanner) (sprintResponse, error) {
	var sprint sprintResponse
	var startDate pgtype.Text
	var endDate pgtype.Text
	if err := row.Scan(
		&sprint.ID,
		&sprint.WorkspaceID,
		&sprint.ProjectID,
		&sprint.ProjectKey,
		&sprint.ProjectName,
		&sprint.Name,
		&sprint.Goal,
		&sprint.Status,
		&startDate,
		&endDate,
		&sprint.CreatedBy,
		&sprint.CreatedAt,
		&sprint.CompletedAt,
		&sprint.IssueCount,
		&sprint.DoneCount,
		&sprint.PointsTotal,
		&sprint.PointsDone,
		&sprint.PointsOpen,
	); err != nil {
		return sprintResponse{}, err
	}

	sprint.StartDate = nullableText(startDate)
	sprint.EndDate = nullableText(endDate)

	return sprint, nil
}

func notificationSprintContext(sprint sprintResponse) notifications.SprintContext {
	return notifications.SprintContext{
		ID:         sprint.ID,
		Name:       sprint.Name,
		ProjectKey: sprint.ProjectKey,
	}
}
