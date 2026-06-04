package invites

import (
	"strings"
	"testing"
	"time"
)

func TestNewInviteTokenAndHash(t *testing.T) {
	t.Parallel()

	token, err := newInviteToken()
	if err != nil {
		t.Fatalf("newInviteToken: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
	if strings.ContainsAny(token, "+/=") {
		t.Fatalf("token = %q, want raw base64url token", token)
	}

	hash := hashInviteToken(token)
	if len(hash) != 64 {
		t.Fatalf("hash length = %d, want 64", len(hash))
	}
	if hash == token {
		t.Fatal("hash should not equal raw token")
	}
	if hashInviteToken(token) != hash {
		t.Fatal("hash should be deterministic")
	}
}

func TestNormalizeCreateInvite(t *testing.T) {
	t.Parallel()

	got, err := normalizeCreateInvite(createInviteRequest{
		Email: " New.Member@Example.COM ",
	})
	if err != nil {
		t.Fatalf("normalize create invite: %v", err)
	}

	if got.Email != "new.member@example.com" {
		t.Fatalf("Email = %q, want new.member@example.com", got.Email)
	}
	if got.Role != "member" {
		t.Fatalf("Role = %q, want member", got.Role)
	}
}

func TestNormalizeCreateInviteValidation(t *testing.T) {
	t.Parallel()

	tests := []createInviteRequest{
		{Email: "not-email", Role: "member"},
		{Email: "member@example.com", Role: "owner"},
	}

	for _, req := range tests {
		req := req
		t.Run(req.Email+" "+req.Role, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeCreateInvite(req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeAcceptInvite(t *testing.T) {
	t.Parallel()

	got, err := normalizeAcceptInvite(acceptInviteRequest{
		Username:    " New_Member ",
		DisplayName: " New   Member ",
		Password:    " password123 ",
	})
	if err != nil {
		t.Fatalf("normalize accept invite: %v", err)
	}

	if got.Username != "new_member" {
		t.Fatalf("Username = %q, want new_member", got.Username)
	}
	if got.DisplayName != "New Member" {
		t.Fatalf("DisplayName = %q, want New Member", got.DisplayName)
	}
	if got.Password != "password123" {
		t.Fatalf("Password = %q, want password123", got.Password)
	}
}

func TestNormalizeAcceptInviteValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  acceptInviteRequest
	}{
		{
			name: "bad username",
			req: acceptInviteRequest{
				Username:    "NO",
				DisplayName: "New Member",
				Password:    "password123",
			},
		},
		{
			name: "missing display name",
			req: acceptInviteRequest{
				Username: "new_member",
				Password: "password123",
			},
		},
		{
			name: "short password",
			req: acceptInviteRequest{
				Username:    "new_member",
				DisplayName: "New Member",
				Password:    "short",
			},
		},
		{
			name: "long password",
			req: acceptInviteRequest{
				Username:    "new_member",
				DisplayName: "New Member",
				Password:    strings.Repeat("x", 129),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeAcceptInvite(tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestInviteStatus(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 4, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name   string
		invite inviteRecord
		want   string
	}{
		{
			name: "pending",
			invite: inviteRecord{
				ExpiresAt: future,
			},
			want: "pending",
		},
		{
			name: "accepted",
			invite: inviteRecord{
				ExpiresAt:  future,
				AcceptedAt: &past,
			},
			want: "accepted",
		},
		{
			name: "revoked",
			invite: inviteRecord{
				ExpiresAt: future,
				RevokedAt: &past,
			},
			want: "revoked",
		},
		{
			name: "expired",
			invite: inviteRecord{
				ExpiresAt: past,
			},
			want: "expired",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := inviteStatus(tt.invite, now); got != tt.want {
				t.Fatalf("inviteStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}
