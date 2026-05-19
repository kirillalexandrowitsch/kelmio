package labels

import "testing"

func TestNormalizeCreateLabel(t *testing.T) {
	t.Parallel()

	name, color, err := normalizeCreateLabel(createLabelRequest{
		Name:  " Frontend Bug ",
		Color: " #CCE8D4 ",
	})
	if err != nil {
		t.Fatalf("normalize create label: %v", err)
	}

	if name != "frontend-bug" {
		t.Fatalf("name = %q, want %q", name, "frontend-bug")
	}
	if color != "#cce8d4" {
		t.Fatalf("color = %q, want %q", color, "#cce8d4")
	}
}

func TestNormalizeCreateLabelDefaultColor(t *testing.T) {
	t.Parallel()

	_, color, err := normalizeCreateLabel(createLabelRequest{Name: "backend"})
	if err != nil {
		t.Fatalf("normalize create label: %v", err)
	}

	if color != "#4e795d" {
		t.Fatalf("color = %q, want %q", color, "#4e795d")
	}
}

func TestNormalizeCreateLabelValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  createLabelRequest
	}{
		{
			name: "missing name",
			req:  createLabelRequest{},
		},
		{
			name: "bad color",
			req: createLabelRequest{
				Name:  "backend",
				Color: "green",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, _, err := normalizeCreateLabel(tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
