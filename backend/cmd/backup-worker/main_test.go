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

	"team-task-tracker/backend/internal/backups"
	appmetrics "team-task-tracker/backend/internal/metrics"
)

type stubRunner struct {
	result backups.Result
	err    error
	calls  chan time.Time
}

func (r *stubRunner) RunOnce(context.Context) (backups.Result, error) {
	if r.calls != nil {
		r.calls <- time.Now()
	}
	return r.result, r.err
}

func TestPerformBackupRecordsSuccess(t *testing.T) {
	completedAt := time.Unix(2000, 0)
	runner := &stubRunner{result: backups.Result{
		Artifact:      backups.Artifact{Path: "/backups/team-task-tracker-scheduled.sql.gz", Size: 42},
		CompletedAt:   completedAt,
		Duration:      2 * time.Second,
		ArtifactCount: 3,
	}}
	metricsRecorder := appmetrics.NewBackupMetrics(time.Time{}, 0)
	success, retentionOK := performBackup(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)), runner, metricsRecorder)
	if !success || !retentionOK {
		t.Fatalf("performBackup() = %v, %v", success, retentionOK)
	}
}

func TestRunUsesRetryIntervalAfterFailure(t *testing.T) {
	calls := make(chan time.Time, 2)
	runner := &stubRunner{err: context.DeadlineExceeded, calls: calls}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- run(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), runner, appmetrics.NewBackupMetrics(time.Time{}, 0), time.Hour, 20*time.Millisecond)
	}()
	first := <-calls
	second := <-calls
	if elapsed := second.Sub(first); elapsed < 15*time.Millisecond || elapsed > 500*time.Millisecond {
		t.Fatalf("retry elapsed = %s", elapsed)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("run did not stop after cancellation")
	}
}

func TestBackupWorkerMetricsServerExposesHealthAndShutsDown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("local sandbox does not allow opening a test listener: %v", err)
		}
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	ctx, cancel := context.WithCancel(context.Background())
	metricsRecorder := appmetrics.NewBackupMetrics(time.Unix(1234, 0), 1)
	startMetricsServer(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), addr, "", true, metricsRecorder)
	client := &http.Client{Timeout: 200 * time.Millisecond}

	for i := 0; i < 20; i++ {
		response, requestErr := client.Get("http://" + addr + "/healthz")
		if requestErr == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusNoContent {
				break
			}
		}
		if i == 19 {
			t.Fatal("health endpoint did not become ready")
		}
		time.Sleep(25 * time.Millisecond)
	}
	response, err := client.Get("http://" + addr + "/metrics")
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("metrics status = %d", response.StatusCode)
	}

	cancel()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		response, requestErr := client.Get("http://" + addr + "/healthz")
		if requestErr != nil {
			return
		}
		_ = response.Body.Close()
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("metrics server still accepted requests after cancellation")
}
