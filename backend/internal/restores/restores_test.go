package restores

import (
	"compress/gzip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeExecutor struct {
	resetErr     error
	restoreErr   error
	verifyErr    error
	cleanupErr   error
	verification Verification
	calls        []string
}

func (e *fakeExecutor) Reset(context.Context) error {
	e.calls = append(e.calls, "reset")
	return e.resetErr
}
func (e *fakeExecutor) Restore(context.Context, string) error {
	e.calls = append(e.calls, "restore")
	return e.restoreErr
}
func (e *fakeExecutor) Verify(context.Context) (Verification, error) {
	e.calls = append(e.calls, "verify")
	return e.verification, e.verifyErr
}
func (e *fakeExecutor) Cleanup(context.Context) error {
	e.calls = append(e.calls, "cleanup")
	return e.cleanupErr
}

func TestRunnerRecordsSuccessfulVerifiedRestore(t *testing.T) {
	dir := t.TempDir()
	artifact := writeGzipArtifact(t, dir, "team-task-tracker-scheduled-20260620-120000.sql.gz", "SELECT 1;")
	executor := &fakeExecutor{verification: validVerification(17)}
	times := []time.Time{
		time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 20, 12, 0, 2, 0, time.UTC),
	}
	runner := Runner{Executor: executor, StatePath: StatePath(dir), ExpectedMigrationVersion: 17, Now: nextTime(times)}

	result, err := runner.Run(context.Background(), artifact)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Duration != 2*time.Second || result.MigrationVersion != 17 || len(result.BackupSHA256) != 64 {
		t.Fatalf("result = %#v", result)
	}
	if got := stringsJoin(executor.calls); got != "reset,restore,verify,cleanup" {
		t.Fatalf("calls = %s", got)
	}
	state, err := LoadState(StatePath(dir))
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if state.LastResult != "success" || state.LastSuccessAt == nil || state.LastSuccessMigrationVersion != 17 {
		t.Fatalf("state = %#v", state)
	}
	info, err := os.Stat(StatePath(dir))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("state mode = %v, err = %v", info.Mode().Perm(), err)
	}
}

func TestRunnerFailurePreservesPreviousSuccessAndCleansTarget(t *testing.T) {
	dir := t.TempDir()
	good := writeGzipArtifact(t, dir, "team-task-tracker-scheduled-20260619-120000.sql.gz", "SELECT 1;")
	goodExecutor := &fakeExecutor{verification: validVerification(17)}
	goodRunner := Runner{Executor: goodExecutor, StatePath: StatePath(dir), ExpectedMigrationVersion: 17}
	if _, err := goodRunner.Run(context.Background(), good); err != nil {
		t.Fatalf("initial Run() error = %v", err)
	}

	failed := writeGzipArtifact(t, dir, "team-task-tracker-scheduled-20260620-120000.sql.gz", "SELECT 2;")
	failedExecutor := &fakeExecutor{restoreErr: errors.New("provider included a secret")}
	failedRunner := Runner{Executor: failedExecutor, StatePath: StatePath(dir), ExpectedMigrationVersion: 17}
	if _, err := failedRunner.Run(context.Background(), failed); ErrorCode(err) != ErrorRestoreFailed {
		t.Fatalf("Run() error = %v", err)
	}
	if got := stringsJoin(failedExecutor.calls); got != "reset,restore,cleanup" {
		t.Fatalf("calls = %s", got)
	}
	state, err := LoadState(StatePath(dir))
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if state.LastResult != "failure" || state.LastErrorCode != ErrorRestoreFailed {
		t.Fatalf("state = %#v", state)
	}
	if state.LastSuccessBackupFile != filepath.Base(good) || state.LastSuccessAt == nil {
		t.Fatalf("previous success was lost: %#v", state)
	}
}

func TestRunnerRejectsInvalidArtifactBeforeReset(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "team-task-tracker-scheduled-broken.sql.gz")
	if err := os.WriteFile(artifact, []byte("not gzip"), 0o600); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{}
	runner := Runner{Executor: executor, StatePath: StatePath(dir), ExpectedMigrationVersion: 17}
	if _, err := runner.Run(context.Background(), artifact); ErrorCode(err) != ErrorArtifactInvalid {
		t.Fatalf("Run() error = %v", err)
	}
	if len(executor.calls) != 0 {
		t.Fatalf("executor calls = %#v", executor.calls)
	}
}

func TestRunnerRejectsIncompleteVerification(t *testing.T) {
	dir := t.TempDir()
	artifact := writeGzipArtifact(t, dir, "team-task-tracker-scheduled-20260620-120000.sql.gz", "SELECT 1;")
	executor := &fakeExecutor{verification: Verification{MigrationVersion: 16, CoreTableCount: 6, WorkspaceCount: 1, UserCount: 1, MembershipCount: 1}}
	runner := Runner{Executor: executor, StatePath: StatePath(dir), ExpectedMigrationVersion: 17}
	if _, err := runner.Run(context.Background(), artifact); ErrorCode(err) != ErrorVerification {
		t.Fatalf("Run() error = %v", err)
	}
	if got := stringsJoin(executor.calls); got != "reset,restore,verify,cleanup" {
		t.Fatalf("calls = %s", got)
	}
}

func TestLatestMigrationVersion(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"000001_init.sql", "000017_reset.sql", "README.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("migration"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	version, err := LatestMigrationVersion(dir)
	if err != nil || version != 17 {
		t.Fatalf("LatestMigrationVersion() = %d, %v", version, err)
	}
}

func validVerification(version int) Verification {
	return Verification{MigrationVersion: version, CoreTableCount: 6, WorkspaceCount: 1, UserCount: 1, MembershipCount: 1}
}

func writeGzipArtifact(t *testing.T, dir string, name string, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := gzip.NewWriter(file)
	if _, err := writer.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func nextTime(values []time.Time) func() time.Time {
	index := 0
	return func() time.Time {
		value := values[index]
		if index < len(values)-1 {
			index++
		}
		return value
	}
}

func stringsJoin(values []string) string {
	result := ""
	for index, value := range values {
		if index > 0 {
			result += ","
		}
		result += value
	}
	return result
}
