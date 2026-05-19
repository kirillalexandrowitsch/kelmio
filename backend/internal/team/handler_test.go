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

func TestNormalizeMemberID(t *testing.T) {
	t.Parallel()

	got, err := normalizeMemberID(" F2D59348-61A3-491A-9EB1-5AEC91FBDF1E ")
	if err != nil {
		t.Fatalf("normalize member id: %v", err)
	}

	want := "f2d59348-61a3-491a-9eb1-5aec91fbdf1e"
	if got != want {
		t.Fatalf("member id = %q, want %q", got, want)
	}
}

func TestNormalizeMemberIDValidation(t *testing.T) {
	t.Parallel()

	if _, err := normalizeMemberID("not-a-uuid"); err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeUpdateMember(t *testing.T) {
	t.Parallel()

	isActive := false
	got, err := normalizeUpdateMember("f2d59348-61a3-491a-9eb1-5aec91fbdf1e", updateMemberRequest{
		Role:     "admin",
		IsActive: &isActive,
	})
	if err != nil {
		t.Fatalf("normalize update member: %v", err)
	}

	if got.RequestedID != "f2d59348-61a3-491a-9eb1-5aec91fbdf1e" {
		t.Fatalf("RequestedID = %q", got.RequestedID)
	}
	if got.Role != "admin" {
		t.Fatalf("Role = %q, want %q", got.Role, "admin")
	}
	if got.IsActive == nil || *got.IsActive {
		t.Fatalf("IsActive = %v, want false pointer", got.IsActive)
	}
	if !got.HasChanges {
		t.Fatal("HasChanges = false, want true")
	}
}

func TestNormalizeUpdateMemberValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  updateMemberRequest
	}{
		{
			name: "empty",
			req:  updateMemberRequest{},
		},
		{
			name: "bad role",
			req: updateMemberRequest{
				Role: "owner",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeUpdateMember("f2d59348-61a3-491a-9eb1-5aec91fbdf1e", tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
