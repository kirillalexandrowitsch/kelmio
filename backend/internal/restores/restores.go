package restores

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"kelmio/backend/internal/postgresenv"
)

const StateFilename = "restore-drill-state.json"

const (
	ErrorArtifactInvalid   = "artifact_invalid"
	ErrorTargetUnavailable = "target_unavailable"
	ErrorRestoreFailed     = "restore_failed"
	ErrorVerification      = "verification_failed"
	ErrorCleanup           = "cleanup_failed"
	ErrorStateWrite        = "state_write_failed"
)

var migrationFilePattern = regexp.MustCompile(`^(\d+)_.*\.sql$`)

type Verification struct {
	MigrationVersion int
	CoreTableCount   int
	WorkspaceCount   int
	UserCount        int
	MembershipCount  int
}

type Result struct {
	BackupPath       string
	BackupSHA256     string
	StartedAt        time.Time
	CompletedAt      time.Time
	Duration         time.Duration
	MigrationVersion int
}

type Executor interface {
	Reset(context.Context) error
	Restore(context.Context, string) error
	Verify(context.Context) (Verification, error)
	Cleanup(context.Context) error
}

type Runner struct {
	Executor                 Executor
	StatePath                string
	ExpectedMigrationVersion int
	Now                      func() time.Time
}

type PSQLExecutor struct {
	Command     string
	DatabaseURL string
}

type DrillError struct {
	Code string
	Err  error
}

func (e *DrillError) Error() string { return e.Code }
func (e *DrillError) Unwrap() error { return e.Err }

func ErrorCode(err error) string {
	var drillErr *DrillError
	if errors.As(err, &drillErr) {
		return drillErr.Code
	}
	return ErrorRestoreFailed
}

func (r *Runner) Run(ctx context.Context, backupPath string) (Result, error) {
	startedAt := r.now().UTC()
	result := Result{BackupPath: backupPath, StartedAt: startedAt}
	checksum, err := validateArtifact(backupPath)
	if err != nil {
		return r.fail(result, ErrorArtifactInvalid, err)
	}
	result.BackupSHA256 = checksum
	if r.Executor == nil {
		return r.fail(result, ErrorTargetUnavailable, errors.New("restore executor is required"))
	}
	if r.ExpectedMigrationVersion <= 0 {
		return r.fail(result, ErrorVerification, errors.New("expected migration version is required"))
	}
	if err := r.Executor.Reset(ctx); err != nil {
		return r.fail(result, ErrorTargetUnavailable, err)
	}
	cleanupNeeded := true
	defer func() {
		if cleanupNeeded {
			_ = r.Executor.Cleanup(context.Background())
		}
	}()
	if err := r.Executor.Restore(ctx, backupPath); err != nil {
		return r.fail(result, ErrorRestoreFailed, err)
	}
	verification, err := r.Executor.Verify(ctx)
	if err != nil {
		return r.fail(result, ErrorVerification, err)
	}
	if verification.MigrationVersion != r.ExpectedMigrationVersion || verification.CoreTableCount != 6 || verification.WorkspaceCount < 1 || verification.UserCount < 1 || verification.MembershipCount < 1 {
		return r.fail(result, ErrorVerification, fmt.Errorf("restored core state is incomplete"))
	}
	result.MigrationVersion = verification.MigrationVersion
	if err := r.Executor.Cleanup(ctx); err != nil {
		return r.fail(result, ErrorCleanup, err)
	}
	cleanupNeeded = false
	result.CompletedAt = r.now().UTC()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	if err := RecordSuccess(r.StatePath, result); err != nil {
		return result, &DrillError{Code: ErrorStateWrite, Err: err}
	}
	return result, nil
}

func (r *Runner) fail(result Result, code string, cause error) (Result, error) {
	result.CompletedAt = r.now().UTC()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)
	drillErr := &DrillError{Code: code, Err: cause}
	if err := RecordFailure(r.StatePath, result, code); err != nil {
		return result, &DrillError{Code: ErrorStateWrite, Err: err}
	}
	return result, drillErr
}

func (r *Runner) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

func (e PSQLExecutor) Reset(ctx context.Context) error {
	return e.run(ctx, nil, `SET client_min_messages TO WARNING; DROP SCHEMA IF EXISTS public CASCADE;`)
}

func (e PSQLExecutor) Cleanup(ctx context.Context) error {
	return e.run(ctx, nil, `SET client_min_messages TO WARNING; DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`)
}

func (e PSQLExecutor) Restore(ctx context.Context, backupPath string) error {
	input, closeInput, err := openArtifact(backupPath)
	if err != nil {
		return err
	}
	defer closeInput()
	return e.run(ctx, input, "")
}

func (e PSQLExecutor) Verify(ctx context.Context) (Verification, error) {
	query := `SELECT concat_ws('|',
COALESCE((SELECT max(version)::text FROM schema_migrations), '0'),
(SELECT count(*)::text FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('schema_migrations','workspaces','users','workspace_members','projects','issues')),
(SELECT count(*)::text FROM workspaces),
(SELECT count(*)::text FROM users),
(SELECT count(*)::text FROM workspace_members));`
	output, err := e.output(ctx, query)
	if err != nil {
		return Verification{}, err
	}
	parts := strings.Split(strings.TrimSpace(output), "|")
	if len(parts) != 5 {
		return Verification{}, errors.New("unexpected restore verification output")
	}
	values := make([]int, len(parts))
	for index, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return Verification{}, errors.New("invalid restore verification output")
		}
		values[index] = value
	}
	return Verification{
		MigrationVersion: values[0],
		CoreTableCount:   values[1],
		WorkspaceCount:   values[2],
		UserCount:        values[3],
		MembershipCount:  values[4],
	}, nil
}

func (e PSQLExecutor) run(ctx context.Context, input io.Reader, query string) error {
	command := strings.TrimSpace(e.Command)
	if command == "" {
		command = "psql"
	}
	environment, err := postgresenv.FromURL(e.DatabaseURL)
	if err != nil {
		return err
	}
	args := []string{"-q", "-v", "ON_ERROR_STOP=1"}
	if query != "" {
		args = append(args, "-c", query)
	}
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = environment
	cmd.Stdin = input
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return errors.New("psql command failed")
	}
	return nil
}

func (e PSQLExecutor) output(ctx context.Context, query string) (string, error) {
	command := strings.TrimSpace(e.Command)
	if command == "" {
		command = "psql"
	}
	environment, err := postgresenv.FromURL(e.DatabaseURL)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, command, "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1", "-c", query)
	cmd.Env = environment
	cmd.Stderr = io.Discard
	output, err := cmd.Output()
	if err != nil {
		return "", errors.New("psql verification failed")
	}
	return string(output), nil
}

func LatestMigrationVersion(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	latest := 0
	for _, entry := range entries {
		matches := migrationFilePattern.FindStringSubmatch(entry.Name())
		if entry.IsDir() || len(matches) != 2 {
			continue
		}
		version, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0, err
		}
		if version > latest {
			latest = version
		}
	}
	if latest == 0 {
		return 0, errors.New("no migration files found")
	}
	return latest, nil
}

func validateArtifact(path string) (string, error) {
	if !strings.HasSuffix(path, ".sql") && !strings.HasSuffix(path, ".sql.gz") {
		return "", errors.New("backup artifact must be SQL or compressed SQL")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		_ = file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if strings.HasSuffix(path, ".gz") {
		compressed, err := os.Open(path)
		if err != nil {
			return "", err
		}
		reader, err := gzip.NewReader(compressed)
		if err != nil {
			_ = compressed.Close()
			return "", err
		}
		_, copyErr := io.Copy(io.Discard, reader)
		closeErr := reader.Close()
		_ = compressed.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func openArtifact(path string) (io.Reader, func(), error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, func() {}, err
	}
	if !strings.HasSuffix(path, ".gz") {
		return file, func() { _ = file.Close() }, nil
	}
	reader, err := gzip.NewReader(file)
	if err != nil {
		_ = file.Close()
		return nil, func() {}, err
	}
	return reader, func() {
		_ = reader.Close()
		_ = file.Close()
	}, nil
}

func StatePath(backupDir string) string {
	return filepath.Join(backupDir, StateFilename)
}
