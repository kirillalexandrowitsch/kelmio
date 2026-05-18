package auth

import "testing"

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
