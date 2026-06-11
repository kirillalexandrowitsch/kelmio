package issues

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"team-task-tracker/backend/internal/auth"
)

func normalizeCreateIssue(req createIssueRequest) (normalizedCreateIssue, error) {
	labelIDs, err := normalizeIssueLabelIDs(req.LabelIDs)
	if err != nil {
		return normalizedCreateIssue{}, err
	}

	input := normalizedCreateIssue{
		ProjectID:        strings.TrimSpace(req.ProjectID),
		Title:            strings.TrimSpace(req.Title),
		Description:      strings.TrimSpace(req.Description),
		IssueType:        withDefault(strings.TrimSpace(req.IssueType), "task"),
		Status:           strings.TrimSpace(req.Status),
		WorkflowStatusID: strings.ToLower(strings.TrimSpace(req.WorkflowStatusID)),
		Priority:         withDefault(strings.TrimSpace(req.Priority), "medium"),
		StoryPoints:      req.StoryPoints,
		AssigneeID:       strings.TrimSpace(req.AssigneeID),
		DueDate:          strings.TrimSpace(req.DueDate),
		LabelIDs:         labelIDs,
	}

	parentIssueID, err := normalizeOptionalIssueID(req.ParentIssueID)
	if err != nil {
		return input, err
	}
	input.ParentIssueID = parentIssueID

	if input.ProjectID == "" {
		return input, errors.New("project_id is required")
	}
	if input.Title == "" {
		return input, errors.New("title is required")
	}
	if len(input.Title) > 180 {
		return input, errors.New("title must be 180 characters or fewer")
	}
	if !validIssueTypes[input.IssueType] {
		return input, errors.New("issue_type is invalid")
	}
	if input.IssueType == "subtask" {
		if input.ParentIssueID == "" {
			return input, errors.New("parent_issue_id is required for subtask")
		}
	}
	if input.IssueType == "epic" && input.ParentIssueID != "" {
		return input, errors.New("epic cannot have a parent issue")
	}
	if input.WorkflowStatusID != "" && !uuidPattern.MatchString(input.WorkflowStatusID) {
		return input, errors.New("workflow_status_id is invalid")
	}
	if input.Status == "" && input.WorkflowStatusID == "" {
		input.Status = "todo"
	}
	if input.WorkflowStatusID == "" && input.Status != "" && !workflowStatusKeyPattern.MatchString(input.Status) {
		return input, errors.New("status is invalid")
	}
	if !validIssuePriorities[input.Priority] {
		return input, errors.New("priority is invalid")
	}
	if input.StoryPoints < 0 || input.StoryPoints > 100 {
		return input, errors.New("story_points must be between 0 and 100")
	}
	if input.DueDate != "" {
		if _, err := time.Parse(time.DateOnly, input.DueDate); err != nil {
			return input, errors.New("due_date must be YYYY-MM-DD")
		}
	}

	return input, nil
}

func normalizeCreateSubtask(parent issueResponse, req createSubtaskRequest) (normalizedCreateIssue, error) {
	return normalizeCreateIssue(createIssueRequest{
		ProjectID:        parent.ProjectID,
		ParentIssueID:    parent.ID,
		Title:            req.Title,
		Description:      req.Description,
		IssueType:        "subtask",
		Status:           req.Status,
		WorkflowStatusID: req.WorkflowStatusID,
		Priority:         req.Priority,
		StoryPoints:      req.StoryPoints,
		AssigneeID:       req.AssigneeID,
		DueDate:          req.DueDate,
		LabelIDs:         req.LabelIDs,
	})
}

func normalizeTransitionIssue(req transitionIssueRequest) (normalizedTransitionIssue, error) {
	input := normalizedTransitionIssue{
		Status:           strings.TrimSpace(req.Status),
		WorkflowStatusID: strings.ToLower(strings.TrimSpace(req.WorkflowStatusID)),
	}
	if input.Status == "" && input.WorkflowStatusID == "" {
		return input, errors.New("status or workflow_status_id is required")
	}
	if input.WorkflowStatusID != "" && !uuidPattern.MatchString(input.WorkflowStatusID) {
		return input, errors.New("workflow_status_id is invalid")
	}
	if input.WorkflowStatusID == "" && input.Status != "" && !workflowStatusKeyPattern.MatchString(input.Status) {
		return input, errors.New("status is invalid")
	}
	return input, nil
}

func normalizeUpdateIssue(req updateIssueRequest) (normalizedUpdateIssue, error) {
	input := normalizedUpdateIssue{
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		IssueType:   strings.TrimSpace(req.IssueType),
		Priority:    strings.TrimSpace(req.Priority),
		StoryPoints: req.StoryPoints,
		DueDate:     strings.TrimSpace(req.DueDate),
	}

	if input.Title == "" {
		return input, errors.New("title is required")
	}
	if len(input.Title) > 180 {
		return input, errors.New("title must be 180 characters or fewer")
	}
	if input.IssueType == "" {
		return input, errors.New("issue_type is required")
	}
	if !validIssueTypes[input.IssueType] {
		return input, errors.New("issue_type is invalid")
	}
	if input.Priority == "" {
		return input, errors.New("priority is required")
	}
	if !validIssuePriorities[input.Priority] {
		return input, errors.New("priority is invalid")
	}
	if input.StoryPoints < 0 || input.StoryPoints > 100 {
		return input, errors.New("story_points must be between 0 and 100")
	}
	if input.DueDate != "" {
		if _, err := time.Parse(time.DateOnly, input.DueDate); err != nil {
			return input, errors.New("due_date must be YYYY-MM-DD")
		}
	}

	return input, nil
}

func normalizeSetIssueParent(req setIssueParentRequest) (string, error) {
	if req.ParentIssueID == nil {
		return "", nil
	}

	return normalizeOptionalIssueID(*req.ParentIssueID)
}

func normalizeCreateIssueLink(sourceIssueID string, req createIssueLinkRequest) (normalizedCreateIssueLink, error) {
	targetIssueID := strings.ToLower(strings.TrimSpace(req.TargetIssueID))
	linkType := strings.TrimSpace(req.LinkType)

	input := normalizedCreateIssueLink{
		TargetIssueID: targetIssueID,
		LinkType:      linkType,
	}

	if input.TargetIssueID == "" {
		return input, errors.New("target_issue_id is required")
	}
	if !uuidPattern.MatchString(input.TargetIssueID) {
		return input, errors.New("target_issue_id must be a valid issue id")
	}
	if input.TargetIssueID == sourceIssueID {
		return input, errors.New("target_issue_id must be different from source issue id")
	}
	if input.LinkType == "" {
		return input, errors.New("link_type is required")
	}
	if !validIssueLinkTypes[input.LinkType] {
		return input, errors.New("link_type is invalid")
	}

	return input, nil
}

func normalizeIssueID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New("issue id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("issue id is invalid")
	}

	return id, nil
}

func normalizeIssueLinkID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New("issue link id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("issue link id is invalid")
	}

	return id, nil
}

func normalizeOptionalIssueID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", nil
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("parent_issue_id must be a valid issue id")
	}

	return id, nil
}

func normalizeCommentID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New("comment id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("comment id is invalid")
	}

	return id, nil
}

func canEditComment(user auth.CurrentUser, authorID string) bool {
	return user.Role == "admin" || user.ID == authorID
}

func normalizeOptionalUserID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", nil
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("assignee_id is invalid")
	}

	return id, nil
}

func normalizeIssueLabelIDs(labelIDs []string) ([]string, error) {
	if len(labelIDs) > 20 {
		return nil, errors.New("label_ids must contain 20 labels or fewer")
	}

	normalized := make([]string, 0, len(labelIDs))
	seen := make(map[string]bool, len(labelIDs))
	for _, labelID := range labelIDs {
		labelID = strings.ToLower(strings.TrimSpace(labelID))
		if labelID == "" {
			continue
		}
		if !uuidPattern.MatchString(labelID) {
			return nil, errors.New("label_ids contains an invalid label id")
		}
		if seen[labelID] {
			continue
		}

		seen[labelID] = true
		normalized = append(normalized, labelID)
	}

	return normalized, nil
}

func normalizeCommentBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", errors.New("body is required")
	}
	if len(body) > 4000 {
		return "", errors.New("body must be 4000 characters or fewer")
	}

	return body, nil
}

func commentPreview(body string) string {
	const maxPreviewLength = 120
	body = strings.TrimSpace(body)
	runes := []rune(body)
	if len(runes) <= maxPreviewLength {
		return body
	}

	return string(runes[:maxPreviewLength])
}

func activityPayloadJSON(payload map[string]string) (string, error) {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return string(encodedPayload), nil
}
