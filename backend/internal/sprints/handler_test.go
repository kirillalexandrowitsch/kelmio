package sprints

import (
	"strings"
	"testing"
)

const testProjectID = "6d5257d4-002e-44da-8925-d9108699c504"
const testSprintID = "f2d59348-61a3-491a-9eb1-5aec91fbdf1e"

func TestNormalizeCreateSprint(t *testing.T) {
	t.Parallel()

	got, err := normalizeCreateSprint(createSprintRequest{
		ProjectID: " 6D5257D4-002E-44DA-8925-D9108699C504 ",
		Name:      " Sprint 1 ",
		Goal:      " Ship useful work ",
		StartDate: "2026-06-01",
		EndDate:   "2026-06-14",
	})
	if err != nil {
		t.Fatalf("normalize create sprint: %v", err)
	}

	if got.ProjectID != testProjectID {
		t.Fatalf("ProjectID = %q, want %q", got.ProjectID, testProjectID)
	}
	if got.Name != "Sprint 1" {
		t.Fatalf("Name = %q, want %q", got.Name, "Sprint 1")
	}
	if got.Goal != "Ship useful work" {
		t.Fatalf("Goal = %q, want %q", got.Goal, "Ship useful work")
	}
}

func TestNormalizeCreateSprintValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  createSprintRequest
	}{
		{
			name: "missing project",
			req: createSprintRequest{
				Name: "Sprint 1",
			},
		},
		{
			name: "bad project",
			req: createSprintRequest{
				ProjectID: "not-a-uuid",
				Name:      "Sprint 1",
			},
		},
		{
			name: "missing name",
			req: createSprintRequest{
				ProjectID: testProjectID,
			},
		},
		{
			name: "name too long",
			req: createSprintRequest{
				ProjectID: testProjectID,
				Name:      strings.Repeat("x", 121),
			},
		},
		{
			name: "goal too long",
			req: createSprintRequest{
				ProjectID: testProjectID,
				Name:      "Sprint 1",
				Goal:      strings.Repeat("x", 1001),
			},
		},
		{
			name: "bad start date",
			req: createSprintRequest{
				ProjectID: testProjectID,
				Name:      "Sprint 1",
				StartDate: "2026/06/01",
			},
		},
		{
			name: "bad end date",
			req: createSprintRequest{
				ProjectID: testProjectID,
				Name:      "Sprint 1",
				EndDate:   "2026/06/14",
			},
		},
		{
			name: "end before start",
			req: createSprintRequest{
				ProjectID: testProjectID,
				Name:      "Sprint 1",
				StartDate: "2026-06-14",
				EndDate:   "2026-06-01",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeCreateSprint(tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeUpdateSprint(t *testing.T) {
	t.Parallel()

	got, err := normalizeUpdateSprint(updateSprintRequest{
		Name:      " Sprint 2 ",
		Goal:      " Updated goal ",
		StartDate: "2026-06-15",
		EndDate:   "2026-06-28",
	})
	if err != nil {
		t.Fatalf("normalize update sprint: %v", err)
	}

	if got.Name != "Sprint 2" {
		t.Fatalf("Name = %q, want %q", got.Name, "Sprint 2")
	}
	if got.Goal != "Updated goal" {
		t.Fatalf("Goal = %q, want %q", got.Goal, "Updated goal")
	}
}

func TestNormalizeID(t *testing.T) {
	t.Parallel()

	got, err := normalizeID(" F2D59348-61A3-491A-9EB1-5AEC91FBDF1E ", "sprint id")
	if err != nil {
		t.Fatalf("normalize id: %v", err)
	}

	if got != testSprintID {
		t.Fatalf("id = %q, want %q", got, testSprintID)
	}
}

func TestNormalizeIDValidation(t *testing.T) {
	t.Parallel()

	if _, err := normalizeID("not-a-uuid", "sprint id"); err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeOptionalID(t *testing.T) {
	t.Parallel()

	empty, err := normalizeOptionalID(" ", "project_id")
	if err != nil {
		t.Fatalf("normalize empty optional id: %v", err)
	}
	if empty != "" {
		t.Fatalf("empty optional id = %q, want empty string", empty)
	}

	got, err := normalizeOptionalID(" F2D59348-61A3-491A-9EB1-5AEC91FBDF1E ", "project_id")
	if err != nil {
		t.Fatalf("normalize optional id: %v", err)
	}
	if got != testSprintID {
		t.Fatalf("optional id = %q, want %q", got, testSprintID)
	}
}

func TestNormalizeOptionalIDRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	if _, err := normalizeOptionalID("not-a-uuid", "project_id"); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateSprintDetailsAcceptsBoundaries(t *testing.T) {
	t.Parallel()

	if err := validateSprintDetails(
		strings.Repeat("n", 120),
		strings.Repeat("g", 1000),
		"2026-06-07",
		"2026-06-07",
	); err != nil {
		t.Fatalf("validate boundary sprint details: %v", err)
	}
}
