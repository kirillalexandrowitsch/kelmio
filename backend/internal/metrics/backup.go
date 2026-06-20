package metrics

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type BackupMetrics struct {
	mu sync.Mutex

	lastAttemptTimestamp float64
	lastSuccessTimestamp float64
	lastDuration         float64
	artifactCount        float64
	failures             float64
	retentionFailures    float64
	lastResult           string
	restoreLastAttempt   float64
	restoreLastSuccess   float64
	restoreDuration      float64
	restoreBackupTime    float64
	restoreFailures      float64
	restoreLastResult    string
}

func (m *BackupMetrics) InitializeRestore(lastAttempt time.Time, lastSuccess time.Time, duration time.Duration, backupTime time.Time, result string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	if !lastAttempt.IsZero() {
		m.restoreLastAttempt = float64(lastAttempt.Unix())
	}
	if !lastSuccess.IsZero() {
		m.restoreLastSuccess = float64(lastSuccess.Unix())
	}
	m.restoreDuration = duration.Seconds()
	if !backupTime.IsZero() {
		m.restoreBackupTime = float64(backupTime.Unix())
	}
	m.restoreLastResult = result
	m.mu.Unlock()
}

func NewBackupMetrics(lastSuccess time.Time, artifactCount int) *BackupMetrics {
	metrics := &BackupMetrics{artifactCount: float64(artifactCount)}
	if !lastSuccess.IsZero() {
		metrics.lastSuccessTimestamp = float64(lastSuccess.Unix())
	}
	return metrics
}

func (m *BackupMetrics) RecordSuccess(at time.Time, duration time.Duration, artifactCount int) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.lastAttemptTimestamp = float64(at.Unix())
	m.lastSuccessTimestamp = float64(at.Unix())
	m.lastDuration = duration.Seconds()
	m.artifactCount = float64(artifactCount)
	m.lastResult = "success"
	m.mu.Unlock()
}

func (m *BackupMetrics) RecordFailure(at time.Time, duration time.Duration) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.lastAttemptTimestamp = float64(at.Unix())
	m.lastDuration = duration.Seconds()
	m.failures++
	m.lastResult = "failure"
	m.mu.Unlock()
}

func (m *BackupMetrics) RecordRetentionFailure() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.retentionFailures++
	m.mu.Unlock()
}

func (m *BackupMetrics) RecordRestoreSuccess(at time.Time, duration time.Duration, backupTime time.Time) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.restoreLastAttempt = float64(at.Unix())
	m.restoreLastSuccess = float64(at.Unix())
	m.restoreDuration = duration.Seconds()
	m.restoreBackupTime = float64(backupTime.Unix())
	m.restoreLastResult = "success"
	m.mu.Unlock()
}

func (m *BackupMetrics) RecordRestoreFailure(at time.Time, duration time.Duration, backupTime time.Time) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.restoreLastAttempt = float64(at.Unix())
	m.restoreDuration = duration.Seconds()
	if !backupTime.IsZero() {
		m.restoreBackupTime = float64(backupTime.Unix())
	}
	m.restoreFailures++
	m.restoreLastResult = "failure"
	m.mu.Unlock()
}

func (m *BackupMetrics) Handler(authToken string) http.Handler {
	return ProtectHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(m.render()))
	}), authToken)
}

func (m *BackupMetrics) render() string {
	m.mu.Lock()
	lastAttempt := m.lastAttemptTimestamp
	lastSuccess := m.lastSuccessTimestamp
	duration := m.lastDuration
	artifacts := m.artifactCount
	failures := m.failures
	retentionFailures := m.retentionFailures
	lastResult := m.lastResult
	restoreLastAttempt := m.restoreLastAttempt
	restoreLastSuccess := m.restoreLastSuccess
	restoreDuration := m.restoreDuration
	restoreBackupTime := m.restoreBackupTime
	restoreFailures := m.restoreFailures
	restoreLastResult := m.restoreLastResult
	m.mu.Unlock()

	var builder strings.Builder
	writeHelpType(&builder, "backup_last_attempt_timestamp_seconds", "Unix timestamp of the latest scheduled backup attempt.", "gauge")
	writeMetric(&builder, "backup_last_attempt_timestamp_seconds", nil, lastAttempt)
	writeHelpType(&builder, "backup_last_success_timestamp_seconds", "Unix timestamp of the latest successful scheduled backup.", "gauge")
	writeMetric(&builder, "backup_last_success_timestamp_seconds", nil, lastSuccess)
	writeHelpType(&builder, "backup_duration_seconds", "Duration of the latest scheduled backup attempt in seconds.", "gauge")
	writeMetric(&builder, "backup_duration_seconds", nil, duration)
	writeHelpType(&builder, "backup_artifacts", "Number of retained scheduled backup artifacts.", "gauge")
	writeMetric(&builder, "backup_artifacts", nil, artifacts)
	writeHelpType(&builder, "backup_failures_total", "Total scheduled backup failures since worker startup.", "counter")
	writeMetric(&builder, "backup_failures_total", nil, failures)
	writeHelpType(&builder, "backup_retention_failures_total", "Total scheduled backup retention failures since worker startup.", "counter")
	writeMetric(&builder, "backup_retention_failures_total", nil, retentionFailures)
	writeHelpType(&builder, "backup_last_result", "Result of the latest scheduled backup attempt.", "gauge")
	for _, result := range []string{"success", "failure"} {
		value := float64(0)
		if lastResult == result {
			value = 1
		}
		writeMetric(&builder, "backup_last_result", map[string]string{"result": result}, value)
	}
	writeHelpType(&builder, "restore_drill_last_attempt_timestamp_seconds", "Unix timestamp of the latest isolated restore drill attempt.", "gauge")
	writeMetric(&builder, "restore_drill_last_attempt_timestamp_seconds", nil, restoreLastAttempt)
	writeHelpType(&builder, "restore_drill_last_success_timestamp_seconds", "Unix timestamp of the latest successful isolated restore drill.", "gauge")
	writeMetric(&builder, "restore_drill_last_success_timestamp_seconds", nil, restoreLastSuccess)
	writeHelpType(&builder, "restore_drill_duration_seconds", "Duration of the latest isolated restore drill in seconds.", "gauge")
	writeMetric(&builder, "restore_drill_duration_seconds", nil, restoreDuration)
	writeHelpType(&builder, "restore_drill_backup_timestamp_seconds", "Modification timestamp of the backup used by the latest restore drill.", "gauge")
	writeMetric(&builder, "restore_drill_backup_timestamp_seconds", nil, restoreBackupTime)
	writeHelpType(&builder, "restore_drill_failures_total", "Total isolated restore drill failures since worker startup.", "counter")
	writeMetric(&builder, "restore_drill_failures_total", nil, restoreFailures)
	writeHelpType(&builder, "restore_drill_last_result", "Result of the latest isolated restore drill attempt.", "gauge")
	for _, result := range []string{"success", "failure"} {
		value := float64(0)
		if restoreLastResult == result {
			value = 1
		}
		writeMetric(&builder, "restore_drill_last_result", map[string]string{"result": result}, value)
	}
	return builder.String()
}
