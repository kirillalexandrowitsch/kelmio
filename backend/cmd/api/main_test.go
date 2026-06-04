package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/csrf"
)

func TestCORSPreflightAllowsFrontendPutRequests(t *testing.T) {
	t.Parallel()

	handler := cors([]string{"http://localhost:5173"}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/issues/issue-id/labels", nil)
	request.Header.Set("Origin", "http://localhost:5173")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}

	allowedMethods := recorder.Header().Get("Access-Control-Allow-Methods")
	if !strings.Contains(allowedMethods, "PUT") {
		t.Fatalf("Access-Control-Allow-Methods = %q, want PUT", allowedMethods)
	}
	allowedHeaders := recorder.Header().Get("Access-Control-Allow-Headers")
	if !strings.Contains(allowedHeaders, csrf.HeaderName) {
		t.Fatalf("Access-Control-Allow-Headers = %q, want %s", allowedHeaders, csrf.HeaderName)
	}

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want frontend origin", got)
	}

	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
	}
}

func TestCORSAllowsMultipleTrustedOrigins(t *testing.T) {
	t.Parallel()

	handler := cors(
		[]string{"https://tasks.example.com", "https://admin.tasks.example.com"},
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/issues", nil)
	request.Header.Set("Origin", "https://admin.tasks.example.com")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.tasks.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want trusted origin", got)
	}
	if got := recorder.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q, want Origin", got)
	}
}

func TestCORSRejectsUntrustedOrigin(t *testing.T) {
	t.Parallel()

	handler := cors([]string{"https://tasks.example.com"}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/issues", nil)
	request.Header.Set("Origin", "https://evil.example.com")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want empty", got)
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	handler := securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	assertHeader(t, recorder, "X-Content-Type-Options", "nosniff")
	assertHeader(t, recorder, "X-Frame-Options", "DENY")
	assertHeader(t, recorder, "Referrer-Policy", "no-referrer")
	assertHeader(t, recorder, "Cross-Origin-Opener-Policy", "same-origin")
	assertHeader(t, recorder, "Permissions-Policy", "camera=(), microphone=(), geolocation=()")
}

func TestRequestBodyLimitRejectsKnownLargeBody(t *testing.T) {
	t.Parallel()

	handler := requestBodyLimit(4, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("large"))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "request_too_large") {
		t.Fatalf("body = %q, want request_too_large", body)
	}
}

func TestRequestBodyLimitAllowsSmallBody(t *testing.T) {
	t.Parallel()

	handler := requestBodyLimit(4, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader("ok"))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestCSRFProtectionAllowsSafeMethods(t *testing.T) {
	t.Parallel()

	manager := newTestCSRFManager(t)
	handler := csrfProtection(manager, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestCSRFProtectionAllowsLoginWithoutToken(t *testing.T) {
	t.Parallel()

	manager := newTestCSRFManager(t)
	handler := csrfProtection(manager, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestCSRFProtectionAllowsInviteAcceptWithoutToken(t *testing.T) {
	t.Parallel()

	manager := newTestCSRFManager(t)
	handler := csrfProtection(manager, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/invites/invite-token/accept", nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestCSRFProtectionRejectsMissingTokenWithSessionCookie(t *testing.T) {
	t.Parallel()

	manager := newTestCSRFManager(t)
	handler := csrfProtection(manager, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/projects", nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "csrf_token_required") {
		t.Fatalf("body = %q, want csrf_token_required", body)
	}
}

func TestCSRFProtectionRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	manager := newTestCSRFManager(t)
	handler := csrfProtection(manager, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler should not be called")
	}))

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/project-id", nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	request.Header.Set(csrf.HeaderName, "invalid-token")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "invalid_csrf_token") {
		t.Fatalf("body = %q, want invalid_csrf_token", body)
	}
}

func TestCSRFProtectionAllowsValidToken(t *testing.T) {
	t.Parallel()

	manager := newTestCSRFManager(t)
	token, err := manager.Generate("session-token")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	handler := csrfProtection(manager, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/projects/project-id", nil)
	request.AddCookie(&http.Cookie{Name: auth.SessionCookieName, Value: "session-token"})
	request.Header.Set(csrf.HeaderName, token)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func newTestCSRFManager(t *testing.T) *csrf.Manager {
	t.Helper()

	manager, err := csrf.NewManager("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	return manager
}

func assertHeader(t *testing.T, recorder *httptest.ResponseRecorder, key string, want string) {
	t.Helper()

	if got := recorder.Header().Get(key); got != want {
		t.Fatalf("%s = %q, want %q", key, got, want)
	}
}
