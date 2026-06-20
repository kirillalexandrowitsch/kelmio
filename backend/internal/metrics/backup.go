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
	return builder.String()
}
