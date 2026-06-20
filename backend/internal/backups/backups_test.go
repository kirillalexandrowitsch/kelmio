package backups

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeDumper struct {
	content string
	err     error
}

func (d fakeDumper) Dump(_ context.Context, _ string, output io.Writer) error {
	if d.err != nil {
		return d.err
	}
	_, err := io.WriteString(output, d.content)
	return err
}

func TestRunnerCreatesAtomicCompressedBackup(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	runner := NewRunner("postgres://user:password@postgres:5432/app", dir, 7)
	runner.Dumper = fakeDumper{content: "CREATE TABLE test (id integer);\n"}
	runner.Now = func() time.Time { return now }

	result, err := runner.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if filepath.Base(result.Artifact.Path) != "team-task-tracker-scheduled-20260620-120000.sql.gz" {
		t.Fatalf("artifact = %q", result.Artifact.Path)
	}
	if result.ArtifactCount != 1 || result.RemovedCount != 0 || result.RetentionError != nil {
		t.Fatalf("result = %#v", result)
	}
	info, err := os.Stat(result.Artifact.Path)
	if err != nil {
		t.Fatalf("stat artifact: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("artifact mode = %o, want 600", info.Mode().Perm())
	}

	file, err := os.Open(result.Artifact.Path)
	if err != nil {
		t.Fatalf("open artifact: %v", err)
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("open gzip: %v", err)
	}
	body, err := io.ReadAll(compressed)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	if string(body) != "CREATE TABLE test (id integer);\n" {
		t.Fatalf("body = %q", body)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Fatalf("temporary file remained: %s", entry.Name())
		}
	}
}

func TestRunnerFailureDoesNotPruneOrLeaveTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	createArtifact(t, dir, "team-task-tracker-scheduled-20260618-120000.sql.gz", time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	createArtifact(t, dir, "team-task-tracker-scheduled-20260619-120000.sql.gz", time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC))

	runner := NewRunner("postgres://postgres/app", dir, 1)
	runner.Dumper = fakeDumper{err: errors.New("dump failed")}
	if _, err := runner.RunOnce(context.Background()); err == nil {
		t.Fatal("RunOnce() error = nil, want failure")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want two existing backups", len(entries))
	}
}

func TestPruneKeepsNewestScheduledArtifactsOnly(t *testing.T) {
	dir := t.TempDir()
	for day := 17; day <= 20; day++ {
		name := time.Date(2026, 6, day, 12, 0, 0, 0, time.UTC).Format("team-task-tracker-scheduled-20060102-150405.sql.gz")
		createArtifact(t, dir, name, time.Date(2026, 6, day, 12, 0, 0, 0, time.UTC))
	}
	createArtifact(t, dir, "team-task-tracker-20260601-120000.sql.gz", time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))

	removed, err := Prune(dir, 2)
	if err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}
	artifacts, err := List(dir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(artifacts) != 2 || !strings.Contains(artifacts[0].Path, "20260620") || !strings.Contains(artifacts[1].Path, "20260619") {
		t.Fatalf("scheduled artifacts = %#v", artifacts)
	}
	if _, err := os.Stat(filepath.Join(dir, "team-task-tracker-20260601-120000.sql.gz")); err != nil {
		t.Fatalf("manual backup was removed: %v", err)
	}
}

func TestPruneRetentionOneKeepsNewest(t *testing.T) {
	dir := t.TempDir()
	createArtifact(t, dir, "team-task-tracker-scheduled-20260619-120000.sql.gz", time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC))
	createArtifact(t, dir, "team-task-tracker-scheduled-20260620-120000.sql.gz", time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC))

	if _, err := Prune(dir, 1); err != nil {
		t.Fatalf("Prune() error = %v", err)
	}
	artifacts, err := List(dir)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(artifacts) != 1 || !strings.Contains(artifacts[0].Path, "20260620") {
		t.Fatalf("artifacts = %#v", artifacts)
	}
}

func TestSanitizeDumpErrorRemovesCredentials(t *testing.T) {
	message := sanitizeDumpError(
		"provider postgres://admin:super-secret@postgres:5432/app rejected password super-secret",
		"postgres://admin:super-secret@postgres:5432/app",
	)
	if strings.Contains(message, "super-secret") {
		t.Fatalf("sanitized message leaked password: %q", message)
	}
}

func createArtifact(t *testing.T, dir string, name string, modifiedAt time.Time) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("backup"), 0o600); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	if err := os.Chtimes(path, modifiedAt, modifiedAt); err != nil {
		t.Fatalf("set artifact time: %v", err)
	}
}
