package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSPreflightAllowsFrontendPutRequests(t *testing.T) {
	t.Parallel()

	handler := cors("http://localhost:5173", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want frontend origin", got)
	}

	if got := recorder.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
	}
}
