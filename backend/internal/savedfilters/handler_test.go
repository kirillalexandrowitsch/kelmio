package savedfilters

import (
	"strings"
	"testing"
)

const testUUID = "6d5257d4-002e-44da-8925-d9108699c504"

func TestNormalizeCreateSavedFilter(t *testing.T) {
	t.Parallel()

	got, err := normalizeCreateSavedFilter(createSavedFilterRequest{
		Name: "  My   Work  ",
		Filters: map[string]string{
			"query":            " routing ",
			"sort":             "priority_desc",
			"projectId":        " 6D5257D4-002E-44DA-8925-D9108699C504 ",
			"sprintId":         "none",
			"status":           "todo",
			"workflowStatusId": testUUID,
			"priority":         "high",
			"assigneeId":       "unassigned",
			"labelId":          testUUID,
			"due":              "due_soon",
		},
	})
	if err != nil {
		t.Fatalf("normalize saved filter: %v", err)
	}

	if got.Name != "My Work" {
		t.Fatalf("Name = %q, want My Work", got.Name)
	}
	if got.Filters["query"] != "routing" {
		t.Fatalf("query = %q, want routing", got.Filters["query"])
	}
	if got.Filters["projectId"] != testUUID {
		t.Fatalf("projectId = %q, want %q", got.Filters["projectId"], testUUID)
	}
	if got.Filters["sort"] != "priority_desc" {
		t.Fatalf("sort = %q, want priority_desc", got.Filters["sort"])
	}
	if got.Filters["workflowStatusId"] != testUUID {
		t.Fatalf("workflowStatusId = %q, want %q", got.Filters["workflowStatusId"], testUUID)
	}
}

func TestNormalizeCreateSavedFilterDefaultsSort(t *testing.T) {
	t.Parallel()

	got, err := normalizeCreateSavedFilter(createSavedFilterRequest{
		Name:    "Default sort",
		Filters: map[string]string{},
	})
	if err != nil {
		t.Fatalf("normalize saved filter: %v", err)
	}

	if got.Filters["sort"] != "created_desc" {
		t.Fatalf("sort = %q, want created_desc", got.Filters["sort"])
	}
}

func TestNormalizeCreateSavedFilterValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  createSavedFilterRequest
	}{
		{
			name: "missing name",
			req: createSavedFilterRequest{
				Filters: map[string]string{},
			},
		},
		{
			name: "name too long",
			req: createSavedFilterRequest{
				Name:    strings.Repeat("x", 61),
				Filters: map[string]string{},
			},
		},
		{
			name: "unknown key",
			req: createSavedFilterRequest{
				Name:    "Bad filter",
				Filters: map[string]string{"owner": "me"},
			},
		},
		{
			name: "bad status",
			req: createSavedFilterRequest{
				Name:    "Bad status",
				Filters: map[string]string{"status": "Ready for review"},
			},
		},
		{
			name: "bad workflow status id",
			req: createSavedFilterRequest{
				Name:    "Bad workflow status",
				Filters: map[string]string{"workflowStatusId": "not-a-uuid"},
			},
		},
		{
			name: "bad project id",
			req: createSavedFilterRequest{
				Name:    "Bad project",
				Filters: map[string]string{"projectId": "not-a-uuid"},
			},
		},
		{
			name: "bad sprint id",
			req: createSavedFilterRequest{
				Name:    "Bad sprint",
				Filters: map[string]string{"sprintId": "not-a-uuid"},
			},
		},
		{
			name: "bad assignee id",
			req: createSavedFilterRequest{
				Name:    "Bad assignee",
				Filters: map[string]string{"assigneeId": "not-a-uuid"},
			},
		},
		{
			name: "bad sort",
			req: createSavedFilterRequest{
				Name:    "Bad sort",
				Filters: map[string]string{"sort": "title_desc"},
			},
		},
		{
			name: "query too long",
			req: createSavedFilterRequest{
				Name:    "Bad query",
				Filters: map[string]string{"query": strings.Repeat("x", 201)},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeCreateSavedFilter(tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeUpdateSavedFilter(t *testing.T) {
	t.Parallel()

	name := " Updated "
	filters := map[string]string{"priority": "critical"}
	got, err := normalizeUpdateSavedFilter(updateSavedFilterRequest{
		Name:    &name,
		Filters: &filters,
	})
	if err != nil {
		t.Fatalf("normalize update saved filter: %v", err)
	}

	if !got.HasName || got.Name != "Updated" {
		t.Fatalf("name = %q has=%t, want Updated true", got.Name, got.HasName)
	}
	if !got.HasFilters || got.Filters["priority"] != "critical" {
		t.Fatalf("filters = %#v has=%t, want priority critical true", got.Filters, got.HasFilters)
	}
}

func TestNormalizeUpdateSavedFilterRequiresChanges(t *testing.T) {
	t.Parallel()

	if _, err := normalizeUpdateSavedFilter(updateSavedFilterRequest{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeSavedFilterID(t *testing.T) {
	t.Parallel()

	got, err := normalizeSavedFilterID(" 6D5257D4-002E-44DA-8925-D9108699C504 ")
	if err != nil {
		t.Fatalf("normalize id: %v", err)
	}
	if got != testUUID {
		t.Fatalf("id = %q, want %q", got, testUUID)
	}
}
