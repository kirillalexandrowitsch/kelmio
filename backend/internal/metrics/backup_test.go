package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBackupMetricsRecordSuccessAndFailure(t *testing.T) {
	metrics := NewBackupMetrics(time.Unix(1000, 0), 2)
	metrics.RecordSuccess(time.Unix(2000, 0), 3*time.Second, 3)

	recorder := httptest.NewRecorder()
	metrics.Handler("").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := recorder.Body.String()
	for _, expected := range []string{
		"kelmio_backup_last_attempt_timestamp_seconds 2000",
		"kelmio_backup_last_success_timestamp_seconds 2000",
		"kelmio_backup_duration_seconds 3",
		"kelmio_backup_artifacts 3",
		`kelmio_backup_last_result{result="success"} 1`,
		`kelmio_backup_last_result{result="failure"} 0`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics output missing %q:\n%s", expected, body)
		}
	}

	metrics.RecordFailure(time.Unix(3000, 0), time.Second)
	metrics.RecordRetentionFailure()
	recorder = httptest.NewRecorder()
	metrics.Handler("").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body = recorder.Body.String()
	for _, expected := range []string{
		"kelmio_backup_last_attempt_timestamp_seconds 3000",
		"kelmio_backup_last_success_timestamp_seconds 2000",
		"kelmio_backup_failures_total 1",
		"kelmio_backup_retention_failures_total 1",
		`kelmio_backup_last_result{result="success"} 0`,
		`kelmio_backup_last_result{result="failure"} 1`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics output missing %q:\n%s", expected, body)
		}
	}
}

func TestBackupMetricsRecordsRestoreDrill(t *testing.T) {
	metrics := NewBackupMetrics(time.Time{}, 0)
	metrics.RecordRestoreSuccess(time.Unix(4000, 0), 4*time.Second, time.Unix(3900, 0))
	metrics.RecordRestoreFailure(time.Unix(5000, 0), 2*time.Second, time.Unix(4900, 0))

	recorder := httptest.NewRecorder()
	metrics.Handler("").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := recorder.Body.String()
	for _, expected := range []string{
		"kelmio_restore_drill_last_attempt_timestamp_seconds 5000",
		"kelmio_restore_drill_last_success_timestamp_seconds 4000",
		"kelmio_restore_drill_duration_seconds 2",
		"kelmio_restore_drill_backup_timestamp_seconds 4900",
		"kelmio_restore_drill_failures_total 1",
		`kelmio_restore_drill_last_result{result="failure"} 1`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics output missing %q:\n%s", expected, body)
		}
	}
}
