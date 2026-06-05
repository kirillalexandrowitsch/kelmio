package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/csrf"
)

func TestProductionLoggerEmitsJSON(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger := newLogger("production", &buffer)
	logger.Info("json log", "request_id", "req-12345678")

	fields := decodeJSONLogLine(t, buffer.String())
	if got := fields["msg"]; got != "json log" {
		t.Fatalf("msg = %v, want json log", got)
	}
	if got := fields["request_id"]; got != "req-12345678" {
		t.Fatalf("request_id = %v, want req-12345678", got)
	}
}

func TestDevelopmentLoggerEmitsText(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger := newLogger("development", &buffer)
	logger.Info("text log", "request_id", "req-12345678")

	line := strings.TrimSpace(buffer.String())
	if json.Valid([]byte(line)) {
		t.Fatalf("development log = %q, want text format", line)
	}
	if !strings.Contains(line, `msg="text log"`) {
		t.Fatalf("development log = %q, want text message", line)
	}
	if !strings.Contains(line, "request_id=req-12345678") {
		t.Fatalf("development log = %q, want request_id", line)
	}
}

func TestRequestIDGeneratesMissingID(t *testing.T) {
	t.Parallel()

	var contextRequestID string
	handler := requestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextRequestID = requestIDFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	headerRequestID := recorder.Header().Get(requestIDHeader)
	if !isValidRequestID(headerRequestID) {
		t.Fatalf("response request id = %q, want valid generated id", headerRequestID)
	}
	if contextRequestID != headerRequestID {
		t.Fatalf("context request id = %q, want response id %q", contextRequestID, headerRequestID)
	}
}

func TestRequestIDPreservesValidInboundID(t *testing.T) {
	t.Parallel()

	handler := requestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := requestIDFromContext(r.Context()); got != "client.req-123_ABC" {
			t.Fatalf("context request id = %q, want inbound id", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	request.Header.Set(requestIDHeader, "client.req-123_ABC")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if got := recorder.Header().Get(requestIDHeader); got != "client.req-123_ABC" {
		t.Fatalf("response request id = %q, want inbound id", got)
	}
}

func TestRequestIDReplacesInvalidInboundID(t *testing.T) {
	t.Parallel()

	handler := requestID(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	request.Header.Set(requestIDHeader, "bad/token?password=secret")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	got := recorder.Header().Get(requestIDHeader)
	if got == "bad/token?password=secret" {
		t.Fatal("invalid inbound request id was preserved")
	}
	if !isValidRequestID(got) {
		t.Fatalf("replacement request id = %q, want valid id", got)
	}
}

func TestRequestLoggerIncludesResponseMetadataAndExcludesSensitiveData(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger := newLogger("production", &buffer)
	handler := requestID(requestLogger(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	})))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/issues?password=secret", strings.NewReader("super-secret-password"))
	request.Header.Set(requestIDHeader, "req-12345678")
	request.Header.Set("Cookie", "team_task_tracker_session=session-token")
	request.Header.Set(csrf.HeaderName, "csrf-secret")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	fields := decodeJSONLogLine(t, buffer.String())
	if got := fields["request_id"]; got != "req-12345678" {
		t.Fatalf("request_id = %v, want req-12345678", got)
	}
	if got := fields["method"]; got != http.MethodPost {
		t.Fatalf("method = %v, want POST", got)
	}
	if got := fields["path"]; got != "/api/v1/issues" {
		t.Fatalf("path = %v, want path without query", got)
	}
	if got := fields["status"]; got != float64(http.StatusCreated) {
		t.Fatalf("status = %v, want %d", got, http.StatusCreated)
	}
	if got := fields["response_bytes"]; got != float64(len("created")) {
		t.Fatalf("response_bytes = %v, want %d", got, len("created"))
	}
	if _, ok := fields["duration_ms"]; !ok {
		t.Fatal("duration_ms missing from log fields")
	}

	rawLog := buffer.String()
	for _, secret := range []string{"password=secret", "session-token", "csrf-secret", "super-secret-password"} {
		if strings.Contains(rawLog, secret) {
			t.Fatalf("request log leaked sensitive value %q: %s", secret, rawLog)
		}
	}
}

func TestRequestLoggerDefaultsStatusOKWhenHandlerDoesNotWrite(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger := newLogger("production", &buffer)
	handler := requestID(requestLogger(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	fields := decodeJSONLogLine(t, buffer.String())
	if got := fields["status"]; got != float64(http.StatusOK) {
		t.Fatalf("status = %v, want %d", got, http.StatusOK)
	}
	if got := fields["response_bytes"]; got != float64(0) {
		t.Fatalf("response_bytes = %v, want 0", got)
	}
}

func TestRecoverPanicReturnsJSONErrorAndLogsRequestID(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	logger := newLogger("production", &buffer)
	handler := requestID(recoverPanic(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/issues?token=secret", nil)
	request.Header.Set(requestIDHeader, "panic-12345678")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if got := recorder.Header().Get(requestIDHeader); got != "panic-12345678" {
		t.Fatalf("response request id = %q, want panic-12345678", got)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "internal_server_error") {
		t.Fatalf("body = %q, want internal_server_error", body)
	}

	fields := decodeJSONLogLine(t, buffer.String())
	if got := fields["request_id"]; got != "panic-12345678" {
		t.Fatalf("panic log request_id = %v, want panic-12345678", got)
	}
	if got := fields["path"]; got != "/api/v1/issues" {
		t.Fatalf("panic log path = %v, want path without query", got)
	}
	if strings.Contains(buffer.String(), "token=secret") {
		t.Fatalf("panic log leaked query string: %s", buffer.String())
	}
	if strings.Contains(buffer.String(), "boom") {
		t.Fatalf("panic log leaked panic value: %s", buffer.String())
	}
}

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
	if !strings.Contains(allowedHeaders, requestIDHeader) {
		t.Fatalf("Access-Control-Allow-Headers = %q, want %s", allowedHeaders, requestIDHeader)
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

func decodeJSONLogLine(t *testing.T, rawLog string) map[string]any {
	t.Helper()

	line := strings.TrimSpace(rawLog)
	if !json.Valid([]byte(line)) {
		t.Fatalf("log line is not valid JSON: %q", line)
	}

	var fields map[string]any
	if err := json.Unmarshal([]byte(line), &fields); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return fields
}
