package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"team-task-tracker/backend/internal/backups"
	appmetrics "team-task-tracker/backend/internal/metrics"
	"team-task-tracker/backend/internal/restores"
)

type stubBackupRunner struct {
	result         backups.Result
	createErr      error
	retentionErr   error
	createCalls    int
	retentionCalls int
	mu             sync.Mutex
}

func (r *stubBackupRunner) Create(context.Context) (backups.Result, error) {
	r.mu.Lock()
	r.createCalls++
	r.mu.Unlock()
	return r.result, r.createErr
}

func (r *stubBackupRunner) ApplyRetention() (int, int, error) {
	r.mu.Lock()
	r.retentionCalls++
	r.mu.Unlock()
	return 1, 2, r.retentionErr
}

type stubRestoreRunner struct {
	mu      sync.Mutex
	results []restores.Result
	errors  []error
	paths   []string
}

func (r *stubRestoreRunner) Run(_ context.Context, path string) (restores.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.paths = append(r.paths, path)
	index := len(r.paths) - 1
	result := restores.Result{BackupPath: path, CompletedAt: time.Now(), Duration: time.Millisecond, MigrationVersion: 17}
	if index < len(r.results) {
		result = r.results[index]
	}
	if index < len(r.errors) {
		return result, r.errors[index]
	}
	return result, nil
}

func TestPerformCycleVerifiesBeforeRetention(t *testing.T) {
	backupRunner := &stubBackupRunner{result: backups.Result{Artifact: backups.Artifact{Path: "/backups/team-task-tracker-scheduled.sql.gz", Size: 42}}}
	drillRunner := &stubRestoreRunner{}
	outcome := performCycle(context.Background(), discardLogger(), backupRunner, drillRunner, appmetrics.NewBackupMetrics(time.Time{}, 0), "", time.Second)
	if !outcome.success || outcome.pendingPath != "" {
		t.Fatalf("performCycle() = %#v", outcome)
	}
	if backupRunner.createCalls != 1 || backupRunner.retentionCalls != 1 || len(drillRunner.paths) != 1 {
		t.Fatalf("calls create=%d restore=%d retention=%d", backupRunner.createCalls, len(drillRunner.paths), backupRunner.retentionCalls)
	}
}

func TestPerformCycleDoesNotPruneFailedRestore(t *testing.T) {
	backupRunner := &stubBackupRunner{result: backups.Result{Artifact: backups.Artifact{Path: "/backups/pending.sql.gz"}}}
	drillRunner := &stubRestoreRunner{errors: []error{&restores.DrillError{Code: restores.ErrorRestoreFailed, Err: errors.New("failed")}}}
	outcome := performCycle(context.Background(), discardLogger(), backupRunner, drillRunner, appmetrics.NewBackupMetrics(time.Time{}, 0), "", time.Second)
	if outcome.success || outcome.pendingPath != "/backups/pending.sql.gz" {
		t.Fatalf("performCycle() = %#v", outcome)
	}
	if backupRunner.retentionCalls != 0 {
		t.Fatalf("retention calls = %d, want 0", backupRunner.retentionCalls)
	}
}

func TestRunRetriesSameArtifactWithoutCreatingAnotherBackup(t *testing.T) {
	backupRunner := &stubBackupRunner{result: backups.Result{Artifact: backups.Artifact{Path: "/backups/pending.sql.gz"}}}
	drillRunner := &stubRestoreRunner{errors: []error{&restores.DrillError{Code: restores.ErrorRestoreFailed, Err: errors.New("failed")}, nil}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- run(ctx, discardLogger(), backupRunner, drillRunner, appmetrics.NewBackupMetrics(time.Time{}, 0), "", time.Hour, 20*time.Millisecond, time.Second)
	}()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		drillRunner.mu.Lock()
		calls := len(drillRunner.paths)
		drillRunner.mu.Unlock()
		if calls >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("run did not stop after cancellation")
	}
	if backupRunner.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", backupRunner.createCalls)
	}
	drillRunner.mu.Lock()
	defer drillRunner.mu.Unlock()
	if len(drillRunner.paths) < 2 || drillRunner.paths[0] != drillRunner.paths[1] {
		t.Fatalf("restore paths = %#v", drillRunner.paths)
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
	startMetricsServer(ctx, discardLogger(), addr, "", true, metricsRecorder)
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

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
