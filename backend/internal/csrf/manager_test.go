package csrf

import (
	"strings"
	"testing"
)

func TestManagerGeneratesValidTokenForSession(t *testing.T) {
	t.Parallel()

	manager := newTestManager(t)
	token, err := manager.Generate("session-token")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if !manager.Valid("session-token", token) {
		t.Fatal("Valid() = false, want true")
	}
}

func TestManagerRejectsTamperedToken(t *testing.T) {
	t.Parallel()

	manager := newTestManager(t)
	token, err := manager.Generate("session-token")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	replacement := "A"
	if strings.HasPrefix(token, replacement) {
		replacement = "B"
	}
	tampered := replacement + token[1:]
	if manager.Valid("session-token", tampered) {
		t.Fatal("Valid() = true, want false")
	}
}

func TestManagerRejectsWrongSession(t *testing.T) {
	t.Parallel()

	manager := newTestManager(t)
	token, err := manager.Generate("session-token")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if manager.Valid("other-session-token", token) {
		t.Fatal("Valid() = true, want false")
	}
}

func TestManagerRejectsMalformedToken(t *testing.T) {
	t.Parallel()

	manager := newTestManager(t)

	for _, token := range []string{"", "missing-separator", "bad.nonce.signature", "bad-nonce.bad-signature"} {
		token := token
		t.Run(strings.ReplaceAll(token, ".", "_"), func(t *testing.T) {
			t.Parallel()

			if manager.Valid("session-token", token) {
				t.Fatal("Valid() = true, want false")
			}
		})
	}
}

func TestManagerRejectsMissingSession(t *testing.T) {
	t.Parallel()

	manager := newTestManager(t)

	if _, err := manager.Generate(""); err == nil {
		t.Fatal("Generate() error = nil, want error")
	}
	if manager.Valid("", "token") {
		t.Fatal("Valid() = true, want false")
	}
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()

	manager, err := NewManager("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	return manager
}
