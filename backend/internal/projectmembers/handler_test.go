package projectmembers

import "testing"

const testID = "6d5257d4-002e-44da-8925-d9108699c504"

func TestNormalizeID(t *testing.T) {
	t.Parallel()

	got, err := normalizeID(" 6D5257D4-002E-44DA-8925-D9108699C504 ", "user id")
	if err != nil {
		t.Fatalf("normalize id: %v", err)
	}
	if got != testID {
		t.Fatalf("normalize id = %q, want %q", got, testID)
	}
	if _, err := normalizeID("bad-id", "user id"); err == nil {
		t.Fatal("expected malformed id error")
	}
}

func TestNormalizeRole(t *testing.T) {
	t.Parallel()

	for _, role := range []string{"lead", "contributor", "viewer"} {
		got, err := normalizeRole(" " + role + " ")
		if err != nil {
			t.Fatalf("normalize role %s: %v", role, err)
		}
		if got != role {
			t.Fatalf("normalize role = %q, want %q", got, role)
		}
	}
	if _, err := normalizeRole("admin"); err == nil {
		t.Fatal("expected invalid project role error")
	}
}
