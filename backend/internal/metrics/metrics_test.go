package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNormalizeRouteUsesServeMuxPattern(t *testing.T) {
	t.Parallel()

	got := NormalizeRoute(http.MethodGet, "/api/v1/issues/123", "GET /api/v1/issues/{id}")
	if got != "/api/v1/issues/{id}" {
		t.Fatalf("NormalizeRoute() = %q, want pattern path", got)
	}
}

func TestNormalizeRouteRemovesHighCardinalityFallbackValues(t *testing.T) {
	t.Parallel()

	got := NormalizeRoute(
		http.MethodPost,
		"/api/v1/issues/4fdc482f-61b6-47f0-a1a8-ff9f34ca5c35/comments/42",
		"",
	)
	if got != "/api/v1/issues/{id}/comments/{number}" {
		t.Fatalf("NormalizeRoute() = %q, want normalized route", got)
	}
}

func TestProtectHandlerAllowsNoToken(t *testing.T) {
	t.Parallel()

	handler := ProtectHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), "")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
}

func TestProtectHandlerRequiresBearerToken(t *testing.T) {
	t.Parallel()

	handler := ProtectHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), "0123456789abcdef0123456789abcdef")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d, want 401", recorder.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	request.Header.Set("Authorization", "Bearer 0123456789abcdef0123456789abcdef")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("valid token status = %d, want 204", recorder.Code)
	}
}

func TestHTTPMiddlewareEmitsMetricsWithoutSensitiveLabels(t *testing.T) {
	t.Parallel()

	appMetrics := NewAppMetrics()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/issues/{id}/comments", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	handler := appMetrics.HTTPMiddleware(mux)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/issues/4fdc482f-61b6-47f0-a1a8-ff9f34ca5c35/comments?token=secret",
		strings.NewReader("password=secret"),
	)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	metricsRecorder := httptest.NewRecorder()
	appMetrics.Handler("").ServeHTTP(metricsRecorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := metricsRecorder.Body.String()

	if !strings.Contains(body, `kelmio_http_requests_total{method="POST",route="/api/v1/issues/{id}/comments",status="201"} 1`) {
		t.Fatalf("metrics output missing normalized request metric:\n%s", body)
	}
	for _, secret := range []string{"4fdc482f-61b6-47f0-a1a8-ff9f34ca5c35", "token=secret", "password=secret"} {
		if strings.Contains(body, secret) {
			t.Fatalf("metrics output leaked %q:\n%s", secret, body)
		}
	}
}

func TestAuthLoginOutcomeMetrics(t *testing.T) {
	t.Parallel()

	appMetrics := NewAppMetrics()
	appMetrics.RecordAuthLoginOutcome("success")
	appMetrics.RecordAuthLoginOutcome("invalid")
	appMetrics.RecordAuthLoginOutcome("rate_limited")
	appMetrics.RecordAuthLoginOutcome("unexpected")

	recorder := httptest.NewRecorder()
	appMetrics.Handler("").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := recorder.Body.String()

	for _, expected := range []string{
		`kelmio_auth_login_attempts_total{outcome="success"} 1`,
		`kelmio_auth_login_attempts_total{outcome="invalid"} 1`,
		`kelmio_auth_login_attempts_total{outcome="rate_limited"} 1`,
		`kelmio_auth_login_attempts_total{outcome="error"} 1`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics output missing %q:\n%s", expected, body)
		}
	}
}

func TestEmailWorkerMetrics(t *testing.T) {
	t.Parallel()

	appMetrics := NewAppMetrics()
	appMetrics.RecordEmailWorkerHeartbeat(time.Unix(1234, 0))
	appMetrics.RecordEmailWorkerDeliveryResult("sent")
	appMetrics.RecordEmailWorkerDeliveryResult("pending")
	appMetrics.RecordEmailWorkerDeliveryResult("failed")
	appMetrics.RecordEmailWorkerDeliveryResult("unknown")
	appMetrics.RecordEmailWorkerBatchError()

	recorder := httptest.NewRecorder()
	appMetrics.Handler("").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := recorder.Body.String()

	for _, expected := range []string{
		"kelmio_email_worker_heartbeat_timestamp_seconds 1234",
		`kelmio_email_worker_delivery_attempts_total{result="sent"} 1`,
		`kelmio_email_worker_delivery_attempts_total{result="pending"} 1`,
		`kelmio_email_worker_delivery_attempts_total{result="failed"} 1`,
		`kelmio_email_worker_delivery_attempts_total{result="error"} 1`,
		"kelmio_email_worker_batch_errors_total 1",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics output missing %q:\n%s", expected, body)
		}
	}
}
