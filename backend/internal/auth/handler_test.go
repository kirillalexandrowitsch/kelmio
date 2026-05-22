package auth

import (
	"strings"
	"testing"
)

func TestLoginRequestIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  loginRequest
		want string
	}{
		{
			name: "login wins",
			req: loginRequest{
				Login:    " admin ",
				Email:    "admin@example.com",
				Username: "root",
			},
			want: "admin",
		},
		{
			name: "email fallback",
			req: loginRequest{
				Email: " admin@example.com ",
			},
			want: "admin@example.com",
		},
		{
			name: "username fallback",
			req: loginRequest{
				Username: " admin ",
			},
			want: "admin",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.req.identifier(); got != tt.want {
				t.Fatalf("identifier() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHashToken(t *testing.T) {
	t.Parallel()

	first := hashToken("session-token")
	second := hashToken("session-token")
	third := hashToken("other-token")

	if first == "" {
		t.Fatal("hash should not be empty")
	}
	if first != second {
		t.Fatal("same token should produce stable hash")
	}
	if first == third {
		t.Fatal("different tokens should produce different hashes")
	}
}

func TestNormalizeDisplayName(t *testing.T) {
	t.Parallel()

	got, err := normalizeDisplayName("  Team Member  ")
	if err != nil {
		t.Fatalf("normalize display name: %v", err)
	}

	if got != "Team Member" {
		t.Fatalf("displayName = %q, want %q", got, "Team Member")
	}
}

func TestNormalizeDisplayNameValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		displayName string
	}{
		{name: "missing", displayName: "   "},
		{name: "too long", displayName: strings.Repeat("a", 81)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeDisplayName(tt.displayName); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeChangePassword(t *testing.T) {
	t.Parallel()

	currentPassword, newPassword, err := normalizeChangePassword(changePasswordRequest{
		CurrentPassword: " old-password ",
		NewPassword:     " new-password ",
	})
	if err != nil {
		t.Fatalf("normalize change password: %v", err)
	}

	if currentPassword != "old-password" {
		t.Fatalf("currentPassword = %q, want %q", currentPassword, "old-password")
	}
	if newPassword != "new-password" {
		t.Fatalf("newPassword = %q, want %q", newPassword, "new-password")
	}
}

func TestNormalizeChangePasswordValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  changePasswordRequest
	}{
		{
			name: "missing current password",
			req: changePasswordRequest{
				NewPassword: "new-password",
			},
		},
		{
			name: "short new password",
			req: changePasswordRequest{
				CurrentPassword: "old-password",
				NewPassword:     "short",
			},
		},
		{
			name: "same password",
			req: changePasswordRequest{
				CurrentPassword: "same-password",
				NewPassword:     "same-password",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, _, err := normalizeChangePassword(tt.req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNewSessionToken(t *testing.T) {
	t.Parallel()

	token, err := newSessionToken()
	if err != nil {
		t.Fatalf("new session token: %v", err)
	}

	if len(token) < 32 {
		t.Fatalf("token is too short: %d", len(token))
	}
}
