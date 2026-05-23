package projects

import (
	"strings"
	"testing"
)

func TestNormalizeProjectKey(t *testing.T) {
	t.Parallel()

	got := normalizeProjectKey(" core ")
	if got != "CORE" {
		t.Fatalf("normalizeProjectKey() = %q, want %q", got, "CORE")
	}
}

func TestNormalizeProjectID(t *testing.T) {
	t.Parallel()

	got, err := normalizeProjectID(" 6D5257D4-002E-44DA-8925-D9108699C504 ")
	if err != nil {
		t.Fatalf("normalizeProjectID() error = %v", err)
	}

	want := "6d5257d4-002e-44da-8925-d9108699c504"
	if got != want {
		t.Fatalf("normalizeProjectID() = %q, want %q", got, want)
	}
}

func TestNormalizeProjectIDValidation(t *testing.T) {
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

			if _, err := normalizeProjectID(tt.id); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidateProjectInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		project string
		wantErr bool
	}{
		{name: "valid", key: "CORE", project: "Core Platform"},
		{name: "too short", key: "C", project: "Core Platform", wantErr: true},
		{name: "too long", key: "CORETRACKER", project: "Core Platform", wantErr: true},
		{name: "invalid chars", key: "CORE-1", project: "Core Platform", wantErr: true},
		{name: "missing name", key: "CORE", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateProjectInput(tt.key, tt.project)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateProjectDetails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		project string
		wantErr bool
	}{
		{name: "valid", project: "Core Platform"},
		{name: "max length", project: strings.Repeat("a", 120)},
		{name: "missing name", wantErr: true},
		{name: "too long", project: "x" + strings.Repeat("a", 120), wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateProjectDetails(tt.project)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
