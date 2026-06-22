package issues

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"kelmio/backend/internal/notifications"
)

func scanIssue(row rowScanner) (issueResponse, error) {
	var issue issueResponse
	var assigneeID pgtype.Text
	var parentIssueID pgtype.Text
	var sprintID pgtype.Text
	var dueDate pgtype.Text
	var workflowStatusJSON []byte
	var labelsJSON []byte

	if err := row.Scan(
		&issue.ID,
		&issue.ProjectID,
		&issue.ProjectKey,
		&issue.Number,
		&issue.IssueKey,
		&issue.Title,
		&issue.Description,
		&issue.IssueType,
		&issue.Status,
		&workflowStatusJSON,
		&issue.Priority,
		&issue.StoryPoints,
		&issue.ReporterID,
		&assigneeID,
		&parentIssueID,
		&sprintID,
		&dueDate,
		&issue.CreatedAt,
		&issue.UpdatedAt,
		&labelsJSON,
	); err != nil {
		return issueResponse{}, err
	}

	issue.AssigneeID = nullableText(assigneeID)
	issue.ParentIssueID = nullableText(parentIssueID)
	issue.SprintID = nullableText(sprintID)
	issue.DueDate = nullableText(dueDate)
	if err := json.Unmarshal(workflowStatusJSON, &issue.WorkflowStatus); err != nil {
		return issueResponse{}, err
	}
	labels, err := decodeIssueLabels(labelsJSON)
	if err != nil {
		return issueResponse{}, err
	}
	issue.Labels = labels

	return issue, nil
}

func notificationIssueContext(issue issueResponse) notifications.IssueContext {
	return notifications.IssueContext{
		ID:         issue.ID,
		ProjectID:  issue.ProjectID,
		IssueKey:   issue.IssueKey,
		Title:      issue.Title,
		ReporterID: issue.ReporterID,
		AssigneeID: stringOrEmpty(issue.AssigneeID),
	}
}

func decodeIssueLabels(labelsJSON []byte) ([]issueLabelResponse, error) {
	if len(labelsJSON) == 0 {
		return []issueLabelResponse{}, nil
	}

	var labels []issueLabelResponse
	if err := json.Unmarshal(labelsJSON, &labels); err != nil {
		return nil, err
	}
	if labels == nil {
		return []issueLabelResponse{}, nil
	}

	return labels, nil
}

func scanIssueLink(row rowScanner) (issueLinkResponse, error) {
	var link issueLinkResponse
	var sourceWorkflowStatusJSON []byte
	var targetWorkflowStatusJSON []byte
	if err := row.Scan(
		&link.ID,
		&link.SourceIssueID,
		&link.TargetIssueID,
		&link.LinkType,
		&link.CreatedBy,
		&link.CreatedAt,
		&link.SourceIssue.ID,
		&link.SourceIssue.IssueKey,
		&link.SourceIssue.Title,
		&link.SourceIssue.IssueType,
		&link.SourceIssue.Status,
		&sourceWorkflowStatusJSON,
		&link.SourceIssue.Priority,
		&link.TargetIssue.ID,
		&link.TargetIssue.IssueKey,
		&link.TargetIssue.Title,
		&link.TargetIssue.IssueType,
		&link.TargetIssue.Status,
		&targetWorkflowStatusJSON,
		&link.TargetIssue.Priority,
	); err != nil {
		return issueLinkResponse{}, err
	}
	if err := json.Unmarshal(sourceWorkflowStatusJSON, &link.SourceIssue.WorkflowStatus); err != nil {
		return issueLinkResponse{}, err
	}
	if err := json.Unmarshal(targetWorkflowStatusJSON, &link.TargetIssue.WorkflowStatus); err != nil {
		return issueLinkResponse{}, err
	}

	return link, nil
}

func scanIssueComment(row rowScanner) (issueCommentResponse, error) {
	var comment issueCommentResponse
	if err := row.Scan(
		&comment.ID,
		&comment.IssueID,
		&comment.AuthorID,
		&comment.AuthorDisplayName,
		&comment.Body,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	); err != nil {
		return issueCommentResponse{}, err
	}

	return comment, nil
}

func scanIssueActivity(row rowScanner) (issueActivityResponse, error) {
	var entry issueActivityResponse
	var actorID pgtype.Text
	var actorDisplayName pgtype.Text
	var payloadText string

	if err := row.Scan(
		&entry.ID,
		&entry.IssueID,
		&entry.Action,
		&actorID,
		&actorDisplayName,
		&payloadText,
		&entry.CreatedAt,
	); err != nil {
		return issueActivityResponse{}, err
	}

	entry.ActorID = nullableText(actorID)
	entry.ActorDisplayName = nullableText(actorDisplayName)
	entry.Payload = map[string]string{}
	if err := json.Unmarshal([]byte(payloadText), &entry.Payload); err != nil {
		return issueActivityResponse{}, err
	}

	return entry, nil
}

func nullableText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain a single JSON object")
		}

		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
