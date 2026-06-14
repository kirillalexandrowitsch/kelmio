package issues

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/automations"
)

const testIssueID = "6d5257d4-002e-44da-8925-d9108699c504"
const testTargetIssueID = "f2d59348-61a3-491a-9eb1-5aec91fbdf1e"

func TestWriteAutomationError(t *testing.T) {
	t.Parallel()
	response := httptest.NewRecorder()
	if handled := (&Handler{}).writeAutomationError(response, fmt.Errorf("%w: unavailable label", automations.ErrActionFailed)); !handled {
		t.Fatal("expected automation error to be handled")
	}
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", response.Code)
	}
	if body := response.Body.String(); !strings.Contains(body, `"code":"automation_action_failed"`) {
		t.Fatalf("body = %s", body)
	}
}

func TestNormalizeCreateIssueDefaults(t *testing.T) {
	t.Parallel()

	got, err := normalizeCreateIssue(createIssueRequest{
		ProjectID: " project-id ",
		Title:     " First issue ",
		LabelIDs: []string{
			" 6D5257D4-002E-44DA-8925-D9108699C504 ",
			"6d5257d4-002e-44da-8925-d9108699c504",
			"F2D59348-61A3-491A-9EB1-5AEC91FBDF1E",
		},
	})
	if err != nil {
		t.Fatalf("normalize create issue: %v", err)
	}

	if got.ProjectID != "project-id" {
		t.Fatalf("ProjectID = %q, want %q", got.ProjectID, "project-id")
	}
	if got.Title != "First issue" {
		t.Fatalf("Title = %q, want %q", got.Title, "First issue")
	}
	if got.IssueType != "task" {
		t.Fatalf("IssueType = %q, want %q", got.IssueType, "task")
	}
	if got.Status != "todo" {
		t.Fatalf("Status = %q, want %q", got.Status, "todo")
	}
	if got.Priority != "medium" {
		t.Fatalf("Priority = %q, want %q", got.Priority, "medium")
	}
	if got.StoryPoints != 0 {
		t.Fatalf("StoryPoints = %d, want 0", got.StoryPoints)
	}
	wantLabelIDs := strings.Join([]string{
		"6d5257d4-002e-44da-8925-d9108699c504",
		"f2d59348-61a3-491a-9eb1-5aec91fbdf1e",
	}, ",")
	if strings.Join(got.LabelIDs, ",") != wantLabelIDs {
		t.Fatalf("LabelIDs = %q, want %q", strings.Join(got.LabelIDs, ","), wantLabelIDs)
	}
}

func TestNormalizeCreateIssueAcceptsEpic(t *testing.T) {
	t.Parallel()

	got, err := normalizeCreateIssue(createIssueRequest{
		ProjectID: "project-id",
		Title:     "Epic issue",
		IssueType: "epic",
	})
	if err != nil {
		t.Fatalf("normalize create epic issue: %v", err)
	}

	if got.IssueType != "epic" {
		t.Fatalf("IssueType = %q, want %q", got.IssueType, "epic")
	}
}

func TestNormalizeCreateIssueWorkflowStatusIDTakesPrecedence(t *testing.T) {
	t.Parallel()

	got, err := normalizeCreateIssue(createIssueRequest{
		ProjectID:        "project-id",
		Title:            "Workflow issue",
		Status:           "Ready for review",
		WorkflowStatusID: testIssueID,
	})
	if err != nil {
		t.Fatalf("normalize create issue with workflow status id: %v", err)
	}
	if got.WorkflowStatusID != testIssueID {
		t.Fatalf("workflow status id = %q, want %q", got.WorkflowStatusID, testIssueID)
	}
}

func TestNormalizeCreateIssueAcceptsSubtaskWithParent(t *testing.T) {
	t.Parallel()

	got, err := normalizeCreateIssue(createIssueRequest{
		ProjectID:     "project-id",
		ParentIssueID: " 6D5257D4-002E-44DA-8925-D9108699C504 ",
		Title:         "Subtask issue",
		IssueType:     "subtask",
	})
	if err != nil {
		t.Fatalf("normalize create subtask issue: %v", err)
	}

	if got.IssueType != "subtask" {
		t.Fatalf("IssueType = %q, want %q", got.IssueType, "subtask")
	}
	if got.ParentIssueID != testIssueID {
		t.Fatalf("ParentIssueID = %q, want %q", got.ParentIssueID, testIssueID)
	}
}

func TestNormalizeCreateIssueValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  createIssueRequest
	}{
		{
			name: "missing project",
			req: createIssueRequest{
				Title: "First issue",
			},
		},
		{
			name: "missing title",
			req: createIssueRequest{
				ProjectID: "project-id",
			},
		},
		{
			name: "bad type",
			req: createIssueRequest{
				ProjectID: "project-id",
				Title:     "First issue",
				IssueType: "incident",
			},
		},
		{
			name: "subtask without parent",
			req: createIssueRequest{
				ProjectID: "project-id",
				Title:     "First issue",
				IssueType: "subtask",
			},
		},
		{
			name: "epic with parent",
			req: createIssueRequest{
				ProjectID:     "project-id",
				ParentIssueID: testIssueID,
				Title:         "First issue",
				IssueType:     "epic",
			},
		},
		{
			name: "bad parent id",
			req: createIssueRequest{
				ProjectID:     "project-id",
				ParentIssueID: "not-a-uuid",
				Title:         "First issue",
			},
		},
		{
			name: "bad date",
			req: createIssueRequest{
				ProjectID: "project-id",
				Title:     "First issue",
				DueDate:   "2026/05/18",
			},
		},
		{
			name: "negative story points",
			req: createIssueRequest{
				ProjectID:   "project-id",
				Title:       "First issue",
				StoryPoints: -1,
			},
		},
		{
			name: "too many story points",
			req: createIssueRequest{
				ProjectID:   "project-id",
				Title:       "First issue",
				StoryPoints: 101,
			},
		},
		{
			name: "bad label id",
			req: createIssueRequest{
				ProjectID: "project-id",
				Title:     "First issue",
				LabelIDs:  []string{"not-a-uuid"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeCreateIssue(tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeCreateSubtask(t *testing.T) {
	t.Parallel()

	parent := issueResponse{
		ID:        testIssueID,
		ProjectID: "project-id",
	}
	got, err := normalizeCreateSubtask(parent, createSubtaskRequest{
		Title:       " Child issue ",
		Priority:    "high",
		StoryPoints: 2,
	})
	if err != nil {
		t.Fatalf("normalize create subtask: %v", err)
	}

	if got.ProjectID != "project-id" {
		t.Fatalf("ProjectID = %q, want %q", got.ProjectID, "project-id")
	}
	if got.ParentIssueID != testIssueID {
		t.Fatalf("ParentIssueID = %q, want %q", got.ParentIssueID, testIssueID)
	}
	if got.IssueType != "subtask" {
		t.Fatalf("IssueType = %q, want %q", got.IssueType, "subtask")
	}
	if got.Title != "Child issue" {
		t.Fatalf("Title = %q, want %q", got.Title, "Child issue")
	}
	if got.StoryPoints != 2 {
		t.Fatalf("StoryPoints = %d, want 2", got.StoryPoints)
	}
}

func TestNormalizeTransitionIssue(t *testing.T) {
	t.Parallel()

	input, err := normalizeTransitionIssue(transitionIssueRequest{
		Status: " in_progress ",
	})
	if err != nil {
		t.Fatalf("normalize transition issue: %v", err)
	}
	if input.Status != "in_progress" {
		t.Fatalf("status = %q, want %q", input.Status, "in_progress")
	}
}

func TestNormalizeTransitionIssueWorkflowStatusIDTakesPrecedence(t *testing.T) {
	t.Parallel()

	input, err := normalizeTransitionIssue(transitionIssueRequest{
		Status:           "Ready for review",
		WorkflowStatusID: testIssueID,
	})
	if err != nil {
		t.Fatalf("normalize transition issue with workflow status id: %v", err)
	}
	if input.WorkflowStatusID != testIssueID {
		t.Fatalf("workflow status id = %q, want %q", input.WorkflowStatusID, testIssueID)
	}
}

func TestNormalizeTransitionIssueValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  transitionIssueRequest
	}{
		{
			name: "missing status",
			req:  transitionIssueRequest{},
		},
		{name: "bad status", req: transitionIssueRequest{Status: "Ready for review"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeTransitionIssue(tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeUpdateIssue(t *testing.T) {
	t.Parallel()

	got, err := normalizeUpdateIssue(updateIssueRequest{
		Title:       "  Updated issue  ",
		Description: "  More context  ",
		IssueType:   "epic",
		Priority:    "high",
		StoryPoints: 8,
		DueDate:     "2026-05-19",
	})
	if err != nil {
		t.Fatalf("normalize update issue: %v", err)
	}

	if got.Title != "Updated issue" {
		t.Fatalf("Title = %q, want %q", got.Title, "Updated issue")
	}
	if got.Description != "More context" {
		t.Fatalf("Description = %q, want %q", got.Description, "More context")
	}
	if got.IssueType != "epic" {
		t.Fatalf("IssueType = %q, want %q", got.IssueType, "epic")
	}
	if got.Priority != "high" {
		t.Fatalf("Priority = %q, want %q", got.Priority, "high")
	}
	if got.StoryPoints != 8 {
		t.Fatalf("StoryPoints = %d, want 8", got.StoryPoints)
	}
	if got.DueDate != "2026-05-19" {
		t.Fatalf("DueDate = %q, want %q", got.DueDate, "2026-05-19")
	}
}

func TestNormalizeUpdateIssueAcceptsSubtask(t *testing.T) {
	t.Parallel()

	got, err := normalizeUpdateIssue(updateIssueRequest{
		Title:     "Updated subtask",
		IssueType: "subtask",
		Priority:  "medium",
	})
	if err != nil {
		t.Fatalf("normalize update subtask issue: %v", err)
	}

	if got.IssueType != "subtask" {
		t.Fatalf("IssueType = %q, want %q", got.IssueType, "subtask")
	}
}

func TestNormalizeUpdateIssueValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  updateIssueRequest
	}{
		{
			name: "missing title",
			req: updateIssueRequest{
				IssueType: "task",
				Priority:  "medium",
			},
		},
		{
			name: "bad type",
			req: updateIssueRequest{
				Title:     "Updated issue",
				IssueType: "incident",
				Priority:  "medium",
			},
		},
		{
			name: "missing priority",
			req: updateIssueRequest{
				Title:     "Updated issue",
				IssueType: "task",
			},
		},
		{
			name: "bad date",
			req: updateIssueRequest{
				Title:     "Updated issue",
				IssueType: "task",
				Priority:  "medium",
				DueDate:   "2026/05/19",
			},
		},
		{
			name: "negative story points",
			req: updateIssueRequest{
				Title:       "Updated issue",
				IssueType:   "task",
				Priority:    "medium",
				StoryPoints: -1,
			},
		},
		{
			name: "too many story points",
			req: updateIssueRequest{
				Title:       "Updated issue",
				IssueType:   "task",
				Priority:    "medium",
				StoryPoints: 101,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeUpdateIssue(tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeIssueID(t *testing.T) {
	t.Parallel()

	got, err := normalizeIssueID(" 6D5257D4-002E-44DA-8925-D9108699C504 ")
	if err != nil {
		t.Fatalf("normalize issue id: %v", err)
	}

	want := "6d5257d4-002e-44da-8925-d9108699c504"
	if got != want {
		t.Fatalf("issue id = %q, want %q", got, want)
	}
}

func TestNormalizeIssueIDValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
	}{
		{
			name: "missing id",
			id:   "",
		},
		{
			name: "bad id",
			id:   "not-a-uuid",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeIssueID(tt.id); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeSetIssueParent(t *testing.T) {
	t.Parallel()

	parentID := " 6D5257D4-002E-44DA-8925-D9108699C504 "
	got, err := normalizeSetIssueParent(setIssueParentRequest{
		ParentIssueID: &parentID,
	})
	if err != nil {
		t.Fatalf("normalize set issue parent: %v", err)
	}
	if got != testIssueID {
		t.Fatalf("parent issue id = %q, want %q", got, testIssueID)
	}

	cleared, err := normalizeSetIssueParent(setIssueParentRequest{})
	if err != nil {
		t.Fatalf("normalize empty set issue parent: %v", err)
	}
	if cleared != "" {
		t.Fatalf("cleared parent issue id = %q, want empty string", cleared)
	}
}

func TestNormalizeSetIssueParentValidation(t *testing.T) {
	t.Parallel()

	parentID := "not-a-uuid"
	if _, err := normalizeSetIssueParent(setIssueParentRequest{ParentIssueID: &parentID}); err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeCreateIssueLink(t *testing.T) {
	t.Parallel()

	got, err := normalizeCreateIssueLink(testIssueID, createIssueLinkRequest{
		TargetIssueID: " F2D59348-61A3-491A-9EB1-5AEC91FBDF1E ",
		LinkType:      " relates ",
	})
	if err != nil {
		t.Fatalf("normalize create issue link: %v", err)
	}

	if got.TargetIssueID != testTargetIssueID {
		t.Fatalf("TargetIssueID = %q, want %q", got.TargetIssueID, testTargetIssueID)
	}
	if got.LinkType != "relates" {
		t.Fatalf("LinkType = %q, want %q", got.LinkType, "relates")
	}
}

func TestNormalizeCreateIssueLinkValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  createIssueLinkRequest
	}{
		{
			name: "missing target",
			req: createIssueLinkRequest{
				LinkType: "relates",
			},
		},
		{
			name: "bad target",
			req: createIssueLinkRequest{
				TargetIssueID: "not-a-uuid",
				LinkType:      "relates",
			},
		},
		{
			name: "self link",
			req: createIssueLinkRequest{
				TargetIssueID: testIssueID,
				LinkType:      "relates",
			},
		},
		{
			name: "missing type",
			req: createIssueLinkRequest{
				TargetIssueID: testTargetIssueID,
			},
		},
		{
			name: "bad type",
			req: createIssueLinkRequest{
				TargetIssueID: testTargetIssueID,
				LinkType:      "duplicates",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeCreateIssueLink(testIssueID, tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeIssueLinkID(t *testing.T) {
	t.Parallel()

	got, err := normalizeIssueLinkID(" F2D59348-61A3-491A-9EB1-5AEC91FBDF1E ")
	if err != nil {
		t.Fatalf("normalize issue link id: %v", err)
	}

	if got != testTargetIssueID {
		t.Fatalf("issue link id = %q, want %q", got, testTargetIssueID)
	}
}

func TestNormalizeIssueLinkIDValidation(t *testing.T) {
	t.Parallel()

	if _, err := normalizeIssueLinkID("not-a-uuid"); err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeOptionalUserID(t *testing.T) {
	t.Parallel()

	got, err := normalizeOptionalUserID(" F2D59348-61A3-491A-9EB1-5AEC91FBDF1E ")
	if err != nil {
		t.Fatalf("normalize optional user id: %v", err)
	}

	want := "f2d59348-61a3-491a-9eb1-5aec91fbdf1e"
	if got != want {
		t.Fatalf("user id = %q, want %q", got, want)
	}

	empty, err := normalizeOptionalUserID("   ")
	if err != nil {
		t.Fatalf("normalize empty optional user id: %v", err)
	}
	if empty != "" {
		t.Fatalf("empty user id = %q, want empty string", empty)
	}
}

func TestNormalizeOptionalUserIDValidation(t *testing.T) {
	t.Parallel()

	if _, err := normalizeOptionalUserID("not-a-uuid"); err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeIssueLabelIDs(t *testing.T) {
	t.Parallel()

	got, err := normalizeIssueLabelIDs([]string{
		" 6D5257D4-002E-44DA-8925-D9108699C504 ",
		"6d5257d4-002e-44da-8925-d9108699c504",
		"",
		"F2D59348-61A3-491A-9EB1-5AEC91FBDF1E",
	})
	if err != nil {
		t.Fatalf("normalize issue label ids: %v", err)
	}

	want := strings.Join([]string{
		"6d5257d4-002e-44da-8925-d9108699c504",
		"f2d59348-61a3-491a-9eb1-5aec91fbdf1e",
	}, ",")
	if strings.Join(got, ",") != want {
		t.Fatalf("label ids = %q, want %q", strings.Join(got, ","), want)
	}
}

func TestNormalizeIssueLabelIDsValidation(t *testing.T) {
	t.Parallel()

	if _, err := normalizeIssueLabelIDs([]string{"not-a-uuid"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestIssueSearchPatternEscapesWildcards(t *testing.T) {
	t.Parallel()

	got := issueSearchPattern(`UI-1 100% ready_part \ done`)
	want := `%UI-1 100\% ready\_part \\ done%`
	if got != want {
		t.Fatalf("pattern = %q, want %q", got, want)
	}
}

func TestIssueDueFilterCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		dueValue string
		want     string
	}{
		{
			name:     "overdue",
			dueValue: "overdue",
			want:     "EXISTS (SELECT 1 FROM project_workflow_statuses ws_due WHERE ws_due.id = i.workflow_status_id AND ws_due.category <> 'done') AND i.due_date < CURRENT_DATE",
		},
		{
			name:     "today",
			dueValue: "today",
			want:     "EXISTS (SELECT 1 FROM project_workflow_statuses ws_due WHERE ws_due.id = i.workflow_status_id AND ws_due.category <> 'done') AND i.due_date = CURRENT_DATE",
		},
		{
			name:     "due soon",
			dueValue: "due_soon",
			want:     "EXISTS (SELECT 1 FROM project_workflow_statuses ws_due WHERE ws_due.id = i.workflow_status_id AND ws_due.category <> 'done') AND i.due_date > CURRENT_DATE AND i.due_date <= CURRENT_DATE + INTERVAL '7 days'",
		},
		{
			name:     "no due",
			dueValue: "no_due",
			want:     "i.due_date IS NULL",
		},
		{
			name:     "invalid ignored",
			dueValue: "drop table issues;",
			want:     "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := issueDueFilterCondition(tt.dueValue); got != tt.want {
				t.Fatalf("condition = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIssueSprintFilterCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		sprintValue string
		placeholder int
		want        string
	}{
		{
			name:        "empty ignored",
			sprintValue: "",
			placeholder: 2,
			want:        "",
		},
		{
			name:        "none",
			sprintValue: "none",
			placeholder: 2,
			want:        "i.sprint_id IS NULL",
		},
		{
			name:        "specific sprint",
			sprintValue: "6d5257d4-002e-44da-8925-d9108699c504",
			placeholder: 3,
			want:        "i.sprint_id = $3",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := issueSprintFilterCondition(tt.sprintValue, tt.placeholder); got != tt.want {
				t.Fatalf("condition = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIssueListOrderClause(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sortValue string
		want      string
	}{
		{
			name:      "default",
			sortValue: "",
			want:      "i.created_at DESC, i.id DESC",
		},
		{
			name:      "created asc",
			sortValue: "created_asc",
			want:      "i.created_at ASC, i.id ASC",
		},
		{
			name:      "priority desc",
			sortValue: "priority_desc",
			want:      "CASE i.priority WHEN 'critical' THEN 4 WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 ELSE 0 END DESC, i.created_at DESC, i.id DESC",
		},
		{
			name:      "due date asc",
			sortValue: "due_date_asc",
			want:      "i.due_date ASC NULLS LAST, i.created_at DESC, i.id DESC",
		},
		{
			name:      "invalid fallback",
			sortValue: "created_at; drop table issues;",
			want:      "i.created_at DESC, i.id DESC",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := issueListOrderClause(tt.sortValue); got != tt.want {
				t.Fatalf("order clause = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeCommentBody(t *testing.T) {
	t.Parallel()

	got, err := normalizeCommentBody("  Needs more context.  ")
	if err != nil {
		t.Fatalf("normalize comment body: %v", err)
	}

	want := "Needs more context."
	if got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestNormalizeCommentBodyValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing body",
			body: "   ",
		},
		{
			name: "too long",
			body: strings.Repeat("x", 4001),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeCommentBody(tt.body); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeCommentID(t *testing.T) {
	t.Parallel()

	got, err := normalizeCommentID(" 6D5257D4-002E-44DA-8925-D9108699C504 ")
	if err != nil {
		t.Fatalf("normalizeCommentID() error = %v", err)
	}

	want := "6d5257d4-002e-44da-8925-d9108699c504"
	if got != want {
		t.Fatalf("normalizeCommentID() = %q, want %q", got, want)
	}
}

func TestNormalizeCommentIDValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   string
	}{
		{name: "missing id", id: ""},
		{name: "bad id", id: "not-a-uuid"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeCommentID(tt.id); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestCanEditComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		user     auth.CurrentUser
		authorID string
		want     bool
	}{
		{
			name:     "author can edit",
			user:     auth.CurrentUser{ID: "user-1", Role: "member"},
			authorID: "user-1",
			want:     true,
		},
		{
			name:     "admin can edit another user comment",
			user:     auth.CurrentUser{ID: "admin-1", Role: "admin"},
			authorID: "user-1",
			want:     true,
		},
		{
			name:     "member cannot edit another user comment",
			user:     auth.CurrentUser{ID: "user-2", Role: "member"},
			authorID: "user-1",
			want:     false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := canEditComment(tt.user, tt.authorID); got != tt.want {
				t.Fatalf("canEditComment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommentPreview(t *testing.T) {
	t.Parallel()

	shortBody := "  Short comment  "
	if got := commentPreview(shortBody); got != "Short comment" {
		t.Fatalf("short preview = %q, want %q", got, "Short comment")
	}

	longBody := strings.Repeat("x", 130)
	if got := commentPreview(longBody); len(got) != 120 {
		t.Fatalf("long preview length = %d, want %d", len(got), 120)
	}
}

func TestActivityPayloadJSON(t *testing.T) {
	t.Parallel()

	got, err := activityPayloadJSON(map[string]string{
		"from_status": "todo",
		"to_status":   "done",
	})
	if err != nil {
		t.Fatalf("activity payload json: %v", err)
	}

	if !strings.Contains(got, `"from_status":"todo"`) {
		t.Fatalf("payload %q does not contain from_status", got)
	}
	if !strings.Contains(got, `"to_status":"done"`) {
		t.Fatalf("payload %q does not contain to_status", got)
	}
}

func TestChangedIssueFields(t *testing.T) {
	t.Parallel()

	oldDueDate := "2026-05-19"
	newDueDate := "2026-05-20"
	previous := issueResponse{
		Title:       "Old title",
		Description: "Old description",
		IssueType:   "task",
		Priority:    "medium",
		DueDate:     &oldDueDate,
	}
	current := issueResponse{
		Title:       "New title",
		Description: "Old description",
		IssueType:   "bug",
		Priority:    "high",
		DueDate:     &newDueDate,
	}

	got := strings.Join(changedIssueFields(previous, current), ",")
	want := "title,issue_type,priority,due_date"
	if got != want {
		t.Fatalf("changed fields = %q, want %q", got, want)
	}
}
