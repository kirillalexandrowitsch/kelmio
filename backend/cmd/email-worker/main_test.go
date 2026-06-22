package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	appmetrics "kelmio/backend/internal/metrics"
)

func TestEmailWorkerMetricsServerExposesAndShutsDown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("local sandbox does not allow opening a test listener: %v", err)
		}
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	metricsRecorder := appmetrics.NewAppMetrics()
	metricsRecorder.RecordEmailWorkerHeartbeat(time.Unix(1234, 0))
	startMetricsServer(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), addr, "", metricsRecorder)
	client := &http.Client{Timeout: 200 * time.Millisecond}

	var body string
	for i := 0; i < 20; i++ {
		response, err := client.Get("http://" + addr + "/metrics")
		if err == nil {
			rawBody, readErr := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if readErr != nil {
				t.Fatalf("read metrics body: %v", readErr)
			}
			if response.StatusCode == http.StatusOK {
				body = string(rawBody)
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !strings.Contains(body, "kelmio_email_worker_heartbeat_timestamp_seconds 1234") {
		t.Fatalf("metrics body missing heartbeat:\n%s", body)
	}

	cancel()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		response, err := client.Get("http://" + addr + "/metrics")
		if err != nil {
			return
		}
		_ = response.Body.Close()
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("metrics server still accepted requests after context cancellation")
}
