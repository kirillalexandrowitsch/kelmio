package issues

import (
	"strings"
	"testing"

	"team-task-tracker/backend/internal/auth"
)

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
	wantLabelIDs := strings.Join([]string{
		"6d5257d4-002e-44da-8925-d9108699c504",
		"f2d59348-61a3-491a-9eb1-5aec91fbdf1e",
	}, ",")
	if strings.Join(got.LabelIDs, ",") != wantLabelIDs {
		t.Fatalf("LabelIDs = %q, want %q", strings.Join(got.LabelIDs, ","), wantLabelIDs)
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
			name: "bad date",
			req: createIssueRequest{
				ProjectID: "project-id",
				Title:     "First issue",
				DueDate:   "2026/05/18",
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

func TestNormalizeTransitionIssue(t *testing.T) {
	t.Parallel()

	status, err := normalizeTransitionIssue(transitionIssueRequest{
		Status: " in_progress ",
	})
	if err != nil {
		t.Fatalf("normalize transition issue: %v", err)
	}
	if status != "in_progress" {
		t.Fatalf("status = %q, want %q", status, "in_progress")
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
		{
			name: "bad status",
			req: transitionIssueRequest{
				Status: "review",
			},
		},
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
		IssueType:   "bug",
		Priority:    "high",
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
	if got.IssueType != "bug" {
		t.Fatalf("IssueType = %q, want %q", got.IssueType, "bug")
	}
	if got.Priority != "high" {
		t.Fatalf("Priority = %q, want %q", got.Priority, "high")
	}
	if got.DueDate != "2026-05-19" {
		t.Fatalf("DueDate = %q, want %q", got.DueDate, "2026-05-19")
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
			want:     "i.status <> 'done' AND i.due_date < CURRENT_DATE",
		},
		{
			name:     "today",
			dueValue: "today",
			want:     "i.status <> 'done' AND i.due_date = CURRENT_DATE",
		},
		{
			name:     "due soon",
			dueValue: "due_soon",
			want:     "i.status <> 'done' AND i.due_date > CURRENT_DATE AND i.due_date <= CURRENT_DATE + INTERVAL '7 days'",
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
			want:      "CASE i.priority WHEN 'critical' THEN 4 WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 ELSE 0 END DESC, i.created_at DESC",
		},
		{
			name:      "due date asc",
			sortValue: "due_date_asc",
			want:      "i.due_date ASC NULLS LAST, i.created_at DESC",
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
