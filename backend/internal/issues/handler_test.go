package issues

import "testing"

func TestNormalizeCreateIssueDefaults(t *testing.T) {
	t.Parallel()

	got, err := normalizeCreateIssue(createIssueRequest{
		ProjectID: " project-id ",
		Title:     " First issue ",
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
