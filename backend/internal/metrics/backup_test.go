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
		"team_task_tracker_backup_last_attempt_timestamp_seconds 2000",
		"team_task_tracker_backup_last_success_timestamp_seconds 2000",
		"team_task_tracker_backup_duration_seconds 3",
		"team_task_tracker_backup_artifacts 3",
		`team_task_tracker_backup_last_result{result="success"} 1`,
		`team_task_tracker_backup_last_result{result="failure"} 0`,
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
		"team_task_tracker_backup_last_attempt_timestamp_seconds 3000",
		"team_task_tracker_backup_last_success_timestamp_seconds 2000",
		"team_task_tracker_backup_failures_total 1",
		"team_task_tracker_backup_retention_failures_total 1",
		`team_task_tracker_backup_last_result{result="success"} 0`,
		`team_task_tracker_backup_last_result{result="failure"} 1`,
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
		"team_task_tracker_restore_drill_last_attempt_timestamp_seconds 5000",
		"team_task_tracker_restore_drill_last_success_timestamp_seconds 4000",
		"team_task_tracker_restore_drill_duration_seconds 2",
		"team_task_tracker_restore_drill_backup_timestamp_seconds 4900",
		"team_task_tracker_restore_drill_failures_total 1",
		`team_task_tracker_restore_drill_last_result{result="failure"} 1`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics output missing %q:\n%s", expected, body)
		}
	}
}
