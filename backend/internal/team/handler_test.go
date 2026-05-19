package team

import "testing"

func TestNormalizeCreateMember(t *testing.T) {
	t.Parallel()

	got, err := normalizeCreateMember(createMemberRequest{
		Email:       " New.Member@Example.COM ",
		Username:    " New_Member ",
		DisplayName: " New Member ",
		Password:    "password123",
	})
	if err != nil {
		t.Fatalf("normalize create member: %v", err)
	}

	if got.Email != "new.member@example.com" {
		t.Fatalf("Email = %q, want %q", got.Email, "new.member@example.com")
	}
	if got.Username != "new_member" {
		t.Fatalf("Username = %q, want %q", got.Username, "new_member")
	}
	if got.DisplayName != "New Member" {
		t.Fatalf("DisplayName = %q, want %q", got.DisplayName, "New Member")
	}
	if got.Role != "member" {
		t.Fatalf("Role = %q, want %q", got.Role, "member")
	}
}

func TestNormalizeCreateMemberValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  createMemberRequest
	}{
		{
			name: "bad email",
			req: createMemberRequest{
				Email:       "not-email",
				Username:    "new_member",
				DisplayName: "New Member",
				Password:    "password123",
			},
		},
		{
			name: "bad username",
			req: createMemberRequest{
				Email:       "new@example.com",
				Username:    "NO",
				DisplayName: "New Member",
				Password:    "password123",
			},
		},
		{
			name: "missing display name",
			req: createMemberRequest{
				Email:    "new@example.com",
				Username: "new_member",
				Password: "password123",
			},
		},
		{
			name: "short password",
			req: createMemberRequest{
				Email:       "new@example.com",
				Username:    "new_member",
				DisplayName: "New Member",
				Password:    "short",
			},
		},
		{
			name: "bad role",
			req: createMemberRequest{
				Email:       "new@example.com",
				Username:    "new_member",
				DisplayName: "New Member",
				Password:    "password123",
				Role:        "owner",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeCreateMember(tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
