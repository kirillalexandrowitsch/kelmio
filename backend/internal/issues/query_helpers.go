package issues

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func insertIssueActivity(ctx context.Context, tx pgx.Tx, issueID string, actorID string, action string, payload map[string]string) error {
	encodedPayload, err := activityPayloadJSON(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO activity_log (entity_type, entity_id, action, actor_id, payload)
		VALUES ('issue', $1, $2, $3, $4::jsonb)
	`, issueID, action, actorID, encodedPayload)

	return err
}

func withDefault(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func firstQueryValue(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func issueSearchPattern(query string) string {
	escapedQuery := strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	).Replace(query)

	return "%" + escapedQuery + "%"
}

func issueDueFilterCondition(dueValue string) string {
	switch strings.TrimSpace(dueValue) {
	case "overdue":
		return "EXISTS (SELECT 1 FROM project_workflow_statuses ws_due WHERE ws_due.id = i.workflow_status_id AND ws_due.category <> 'done') AND i.due_date < CURRENT_DATE"
	case "today":
		return "EXISTS (SELECT 1 FROM project_workflow_statuses ws_due WHERE ws_due.id = i.workflow_status_id AND ws_due.category <> 'done') AND i.due_date = CURRENT_DATE"
	case "due_soon":
		return "EXISTS (SELECT 1 FROM project_workflow_statuses ws_due WHERE ws_due.id = i.workflow_status_id AND ws_due.category <> 'done') AND i.due_date > CURRENT_DATE AND i.due_date <= CURRENT_DATE + INTERVAL '7 days'"
	case "no_due":
		return "i.due_date IS NULL"
	default:
		return ""
	}
}

func issueSprintFilterCondition(sprintValue string, placeholder int) string {
	switch strings.TrimSpace(sprintValue) {
	case "":
		return ""
	case "none":
		return "i.sprint_id IS NULL"
	default:
		return fmt.Sprintf("i.sprint_id = $%d", placeholder)
	}
}

func issueListOrderClause(sortValue string) string {
	switch strings.TrimSpace(sortValue) {
	case "created_asc":
		return "i.created_at ASC, i.id ASC"
	case "priority_desc":
		return "CASE i.priority WHEN 'critical' THEN 4 WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 ELSE 0 END DESC, i.created_at DESC, i.id DESC"
	case "due_date_asc":
		return "i.due_date ASC NULLS LAST, i.created_at DESC, i.id DESC"
	default:
		return "i.created_at DESC, i.id DESC"
	}
}

func textOrEmpty(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}

	return value.String
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func sortedStrings(values []string) []string {
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	return sorted
}
