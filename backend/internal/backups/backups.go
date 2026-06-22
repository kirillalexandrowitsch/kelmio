package backups

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"kelmio/backend/internal/postgresenv"
)

const ScheduledPrefix = "kelmio-scheduled-"

var postgresURLPasswordPattern = regexp.MustCompile(`(?i)(postgres(?:ql)?://[^:/?#]+:)[^@/?#]+@`)

type Dumper interface {
	Dump(context.Context, string, io.Writer) error
}

type PGDumper struct {
	Command string
}

type Runner struct {
	DatabaseURL    string
	Dir            string
	RetentionCount int
	Dumper         Dumper
	Now            func() time.Time
}

type Artifact struct {
	Path      string
	CreatedAt time.Time
	Size      int64
}

type Result struct {
	Artifact       Artifact
	StartedAt      time.Time
	CompletedAt    time.Time
	Duration       time.Duration
	ArtifactCount  int
	RemovedCount   int
	RetentionError error
}

func NewRunner(databaseURL string, dir string, retentionCount int) *Runner {
	return &Runner{
		DatabaseURL:    databaseURL,
		Dir:            dir,
		RetentionCount: retentionCount,
		Dumper:         PGDumper{Command: "pg_dump"},
		Now:            time.Now,
	}
}

func (r *Runner) RunOnce(ctx context.Context) (Result, error) {
	result, err := r.Create(ctx)
	if err != nil {
		return result, err
	}
	removed, count, retentionErr := r.ApplyRetention()
	result.RemovedCount = removed
	result.ArtifactCount = count
	result.RetentionError = retentionErr
	return result, nil
}

func (r *Runner) Create(ctx context.Context) (Result, error) {
	startedAt := r.now().UTC()
	result := Result{StartedAt: startedAt}
	if strings.TrimSpace(r.DatabaseURL) == "" {
		return result, errors.New("database URL is required")
	}
	if strings.TrimSpace(r.Dir) == "" {
		return result, errors.New("backup directory is required")
	}
	if r.RetentionCount < 1 {
		return result, errors.New("backup retention count must be greater than 0")
	}
	if r.Dumper == nil {
		return result, errors.New("backup dumper is required")
	}

	if err := os.MkdirAll(r.Dir, 0o700); err != nil {
		return result, fmt.Errorf("create backup directory: %w", err)
	}

	finalPath := r.availablePath(startedAt)
	temporary, err := os.CreateTemp(r.Dir, ".kelmio-scheduled-*.tmp")
	if err != nil {
		return result, fmt.Errorf("create temporary backup: %w", err)
	}
	temporaryPath := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}

	compressed := gzip.NewWriter(temporary)
	if err := r.Dumper.Dump(ctx, r.DatabaseURL, compressed); err != nil {
		_ = compressed.Close()
		cleanup()
		return result, err
	}
	if err := compressed.Close(); err != nil {
		cleanup()
		return result, fmt.Errorf("finish backup compression: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		cleanup()
		return result, fmt.Errorf("sync backup artifact: %w", err)
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return result, fmt.Errorf("close backup artifact: %w", err)
	}
	if err := os.Chmod(temporaryPath, 0o600); err != nil {
		_ = os.Remove(temporaryPath)
		return result, fmt.Errorf("secure backup artifact: %w", err)
	}
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		_ = os.Remove(temporaryPath)
		return result, fmt.Errorf("publish backup artifact: %w", err)
	}

	completedAt := r.now().UTC()
	result.CompletedAt = completedAt
	result.Duration = completedAt.Sub(startedAt)
	info, err := os.Stat(finalPath)
	if err != nil {
		return result, fmt.Errorf("inspect backup artifact: %w", err)
	}
	result.Artifact = Artifact{Path: finalPath, CreatedAt: info.ModTime().UTC(), Size: info.Size()}
	artifacts, listErr := List(r.Dir)
	if listErr != nil {
		return result, listErr
	}
	result.ArtifactCount = len(artifacts)
	return result, nil
}

func (r *Runner) ApplyRetention() (int, int, error) {
	removed, err := Prune(r.Dir, r.RetentionCount)
	if err != nil {
		return removed, 0, err
	}
	artifacts, err := List(r.Dir)
	if err != nil {
		return removed, 0, err
	}
	return removed, len(artifacts), nil
}

func List(dir string) ([]Artifact, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	artifacts := make([]Artifact, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !isScheduledName(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, Artifact{
			Path:      filepath.Join(dir, entry.Name()),
			CreatedAt: info.ModTime().UTC(),
			Size:      info.Size(),
		})
	}
	sort.Slice(artifacts, func(i, j int) bool {
		if artifacts[i].CreatedAt.Equal(artifacts[j].CreatedAt) {
			return artifacts[i].Path > artifacts[j].Path
		}
		return artifacts[i].CreatedAt.After(artifacts[j].CreatedAt)
	})
	return artifacts, nil
}

func Prune(dir string, keep int) (int, error) {
	if keep < 1 {
		return 0, errors.New("backup retention count must be greater than 0")
	}
	artifacts, err := List(dir)
	if err != nil {
		return 0, err
	}
	if len(artifacts) <= keep {
		return 0, nil
	}
	removed := 0
	for _, artifact := range artifacts[keep:] {
		if err := os.Remove(artifact.Path); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func Latest(dir string) (Artifact, bool, error) {
	artifacts, err := List(dir)
	if err != nil {
		return Artifact{}, false, err
	}
	if len(artifacts) == 0 {
		return Artifact{}, false, nil
	}
	return artifacts[0], true, nil
}

func (d PGDumper) Dump(ctx context.Context, databaseURL string, output io.Writer) error {
	command := strings.TrimSpace(d.Command)
	if command == "" {
		command = "pg_dump"
	}
	connectionEnv, err := postgresenv.FromURL(databaseURL)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, command, "--no-owner", "--no-privileges", "--schema=public")
	cmd.Env = connectionEnv
	cmd.Stdout = output
	var stderr strings.Builder
	cmd.Stderr = &limitedWriter{Writer: &stderr, Remaining: 2048}
	if err := cmd.Run(); err != nil {
		message := sanitizeDumpError(stderr.String(), databaseURL)
		if message == "" {
			message = "database dump command failed"
		}
		return fmt.Errorf("pg_dump failed: %s", message)
	}
	return nil
}

func (r *Runner) availablePath(now time.Time) string {
	base := ScheduledPrefix + now.UTC().Format("20060102-150405")
	path := filepath.Join(r.Dir, base+".sql.gz")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	return filepath.Join(r.Dir, fmt.Sprintf("%s-%d.sql.gz", base, now.UnixNano()))
}

func (r *Runner) now() time.Time {
	if r.Now == nil {
		return time.Now()
	}
	return r.Now()
}

func isScheduledName(name string) bool {
	return strings.HasPrefix(name, ScheduledPrefix) && strings.HasSuffix(name, ".sql.gz")
}

func sanitizeDumpError(message string, databaseURL string) string {
	message = strings.Join(strings.Fields(message), " ")
	if parsed, err := url.Parse(databaseURL); err == nil && parsed.User != nil {
		if password, ok := parsed.User.Password(); ok && password != "" {
			message = strings.ReplaceAll(message, password, "[redacted]")
		}
	}
	message = postgresURLPasswordPattern.ReplaceAllString(message, `${1}[redacted]@`)
	if len(message) > 512 {
		message = message[:512]
	}
	return message
}

type limitedWriter struct {
	Writer    io.Writer
	Remaining int
}

func (w *limitedWriter) Write(value []byte) (int, error) {
	originalLength := len(value)
	if w.Remaining <= 0 {
		return originalLength, nil
	}
	if len(value) > w.Remaining {
		value = value[:w.Remaining]
	}
	written, err := w.Writer.Write(value)
	w.Remaining -= written
	if err != nil {
		return written, err
	}
	return originalLength, nil
}
