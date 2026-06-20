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
