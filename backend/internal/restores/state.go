package restores

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	LastAttemptAt               time.Time  `json:"last_attempt_at"`
	LastResult                  string     `json:"last_result"`
	LastDurationSeconds         float64    `json:"last_duration_seconds"`
	LastBackupFile              string     `json:"last_backup_file"`
	LastBackupSHA256            string     `json:"last_backup_sha256"`
	LastErrorCode               string     `json:"last_error_code,omitempty"`
	LastSuccessAt               *time.Time `json:"last_success_at,omitempty"`
	LastSuccessBackupFile       string     `json:"last_success_backup_file,omitempty"`
	LastSuccessBackupSHA256     string     `json:"last_success_backup_sha256,omitempty"`
	LastSuccessMigrationVersion int        `json:"last_success_migration_version,omitempty"`
}

func LoadState(path string) (State, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(content, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

func RecordSuccess(path string, result Result) error {
	completedAt := result.CompletedAt.UTC()
	state := State{
		LastAttemptAt:               completedAt,
		LastResult:                  "success",
		LastDurationSeconds:         result.Duration.Seconds(),
		LastBackupFile:              filepath.Base(result.BackupPath),
		LastBackupSHA256:            result.BackupSHA256,
		LastSuccessAt:               &completedAt,
		LastSuccessBackupFile:       filepath.Base(result.BackupPath),
		LastSuccessBackupSHA256:     result.BackupSHA256,
		LastSuccessMigrationVersion: result.MigrationVersion,
	}
	return saveState(path, state)
}

func RecordFailure(path string, result Result, code string) error {
	state, err := LoadState(path)
	if err != nil {
		return err
	}
	state.LastAttemptAt = result.CompletedAt.UTC()
	state.LastResult = "failure"
	state.LastDurationSeconds = result.Duration.Seconds()
	state.LastBackupFile = filepath.Base(result.BackupPath)
	state.LastBackupSHA256 = result.BackupSHA256
	state.LastErrorCode = code
	return saveState(path, state)
}

func saveState(path string, state State) error {
	if path == "" {
		return errors.New("restore drill state path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".restore-drill-state-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	cleanup := func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(state); err != nil {
		cleanup()
		return err
	}
	if err := temporary.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	if err := os.Chmod(temporaryPath, 0o600); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	return nil
}
