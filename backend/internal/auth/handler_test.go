package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestNormalizeLoginRateLimitKey(t *testing.T) {
	t.Parallel()

	if got := normalizeLoginRateLimitKey(" Admin@Example.COM "); got != "admin@example.com" {
		t.Fatalf("normalizeLoginRateLimitKey() = %q, want admin@example.com", got)
	}
}

func TestRetryAfterSeconds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		want     int
	}{
		{name: "zero", duration: 0, want: 1},
		{name: "sub second", duration: 500 * time.Millisecond, want: 1},
		{name: "whole seconds", duration: 3 * time.Second, want: 3},
		{name: "round up", duration: 3*time.Second + time.Millisecond, want: 4},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := retryAfterSeconds(tt.duration); got != tt.want {
				t.Fatalf("retryAfterSeconds() = %d, want %d", got, tt.want)
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
		{
			name: "current password too long",
			req: changePasswordRequest{
				CurrentPassword: "x" + strings.Repeat("a", 128),
				NewPassword:     "new-password",
			},
		},
		{
			name: "new password too long",
			req: changePasswordRequest{
				CurrentPassword: "old-password",
				NewPassword:     "x" + strings.Repeat("a", 128),
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

func TestNormalizePasswordResetEmail(t *testing.T) {
	t.Parallel()

	got, err := normalizePasswordResetEmail(" MEMBER@Example.COM ")
	if err != nil {
		t.Fatalf("normalizePasswordResetEmail() error = %v", err)
	}
	if got != "member@example.com" {
		t.Fatalf("email = %q, want normalized email", got)
	}
}

func TestNormalizePasswordResetEmailValidation(t *testing.T) {
	t.Parallel()

	tests := []string{"", "bad", strings.Repeat("a", 321)}
	for _, email := range tests {
		email := email
		t.Run(email, func(t *testing.T) {
			t.Parallel()
			if _, err := normalizePasswordResetEmail(email); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeCompletePasswordReset(t *testing.T) {
	t.Parallel()

	password, err := normalizeCompletePasswordReset(completePasswordResetRequest{
		Password:        " new-password ",
		ConfirmPassword: " new-password ",
	})
	if err != nil {
		t.Fatalf("normalizeCompletePasswordReset() error = %v", err)
	}
	if password != "new-password" {
		t.Fatalf("password = %q, want new-password", password)
	}
}

func TestNormalizeCompletePasswordResetValidation(t *testing.T) {
	t.Parallel()

	tests := []completePasswordResetRequest{
		{Password: "short", ConfirmPassword: "short"},
		{Password: "new-password", ConfirmPassword: ""},
		{Password: "new-password", ConfirmPassword: "other-password"},
		{Password: "x" + strings.Repeat("a", 128), ConfirmPassword: "x" + strings.Repeat("a", 128)},
	}
	for _, req := range tests {
		req := req
		t.Run(req.Password, func(t *testing.T) {
			t.Parallel()
			if _, err := normalizeCompletePasswordReset(req); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidatePasswordResetTokenState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 16, 12, 0, 0, 0, time.UTC)
	usedAt := now.Add(-time.Minute)
	revokedAt := now.Add(-2 * time.Minute)
	tests := []struct {
		name   string
		record passwordResetTokenRecord
		want   error
	}{
		{name: "valid", record: passwordResetTokenRecord{ExpiresAt: now.Add(time.Minute)}},
		{name: "revoked wins", record: passwordResetTokenRecord{ExpiresAt: now.Add(time.Minute), RevokedAt: &revokedAt, UsedAt: &usedAt}, want: errPasswordResetRevoked},
		{name: "used", record: passwordResetTokenRecord{ExpiresAt: now.Add(time.Minute), UsedAt: &usedAt}, want: errPasswordResetUsed},
		{name: "expired", record: passwordResetTokenRecord{ExpiresAt: now}, want: errPasswordResetExpired},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validatePasswordResetTokenState(tt.record, now)
			if !errors.Is(err, tt.want) {
				t.Fatalf("validatePasswordResetTokenState() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestPasswordResetRateLimitKey(t *testing.T) {
	t.Parallel()

	if got := passwordResetRateLimitKey(" MEMBER@Example.COM ", "127.0.0.1:1234"); got != "email:member@example.com" {
		t.Fatalf("passwordResetRateLimitKey(valid email) = %q", got)
	}
	got := passwordResetRateLimitKey("bad", "127.0.0.1:1234")
	if !strings.HasPrefix(got, "ip:") || strings.Contains(got, "127.0.0.1") {
		t.Fatalf("passwordResetRateLimitKey(invalid email) = %q, want hashed ip fallback", got)
	}
}

func TestSessionCookie(t *testing.T) {
	t.Parallel()

	expiresAt := time.Date(2026, time.May, 23, 12, 0, 0, 0, time.UTC)

	for _, secure := range []bool{false, true} {
		secure := secure
		t.Run("secure_"+boolName(secure), func(t *testing.T) {
			t.Parallel()

			cookie := sessionCookie("session-token", expiresAt, 3600, secure)

			if cookie.Name != SessionCookieName {
				t.Fatalf("Name = %q, want %q", cookie.Name, SessionCookieName)
			}
			if cookie.Value != "session-token" {
				t.Fatalf("Value = %q, want session-token", cookie.Value)
			}
			if cookie.Path != "/" {
				t.Fatalf("Path = %q, want /", cookie.Path)
			}
			if !cookie.Expires.Equal(expiresAt) {
				t.Fatalf("Expires = %s, want %s", cookie.Expires, expiresAt)
			}
			if cookie.MaxAge != 3600 {
				t.Fatalf("MaxAge = %d, want 3600", cookie.MaxAge)
			}
			if !cookie.HttpOnly {
				t.Fatal("HttpOnly = false, want true")
			}
			if cookie.Secure != secure {
				t.Fatalf("Secure = %t, want %t", cookie.Secure, secure)
			}
			if cookie.SameSite != http.SameSiteLaxMode {
				t.Fatalf("SameSite = %v, want %v", cookie.SameSite, http.SameSiteLaxMode)
			}
		})
	}
}

func TestExpiredSessionCookie(t *testing.T) {
	t.Parallel()

	for _, secure := range []bool{false, true} {
		secure := secure
		t.Run("secure_"+boolName(secure), func(t *testing.T) {
			t.Parallel()

			cookie := expiredSessionCookie(secure)

			if cookie.Name != SessionCookieName {
				t.Fatalf("Name = %q, want %q", cookie.Name, SessionCookieName)
			}
			if cookie.Value != "" {
				t.Fatalf("Value = %q, want empty", cookie.Value)
			}
			if cookie.Path != "/" {
				t.Fatalf("Path = %q, want /", cookie.Path)
			}
			if cookie.MaxAge != -1 {
				t.Fatalf("MaxAge = %d, want -1", cookie.MaxAge)
			}
			if !cookie.HttpOnly {
				t.Fatal("HttpOnly = false, want true")
			}
			if cookie.Secure != secure {
				t.Fatalf("Secure = %t, want %t", cookie.Secure, secure)
			}
			if cookie.SameSite != http.SameSiteLaxMode {
				t.Fatalf("SameSite = %v, want %v", cookie.SameSite, http.SameSiteLaxMode)
			}
		})
	}
}

func boolName(value bool) string {
	if value {
		return "true"
	}
	return "false"
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

func TestDecodeJSONRejectsTrailingPayload(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		strings.NewReader(`{"login":"admin","password":"admin12345"}{"login":"other"}`),
	)
	recorder := httptest.NewRecorder()

	var req loginRequest
	if err := decodeJSON(recorder, request, &req); err == nil {
		t.Fatal("expected trailing JSON payload error")
	}
}
