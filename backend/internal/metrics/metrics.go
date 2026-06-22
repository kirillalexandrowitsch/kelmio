package metrics

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

const Namespace = "kelmio"

var (
	uuidPattern   = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	numberPattern = regexp.MustCompile(`\b\d+\b`)
)

type DatabasePinger interface {
	Ping(context.Context) error
}

type OutboxQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type AppMetrics struct {
	mu sync.Mutex

	databasePinger DatabasePinger
	outboxQueryer  OutboxQueryer

	httpRequests      map[httpMetricKey]float64
	httpDurationSums  map[httpMetricKey]float64
	httpDurationCount map[httpMetricKey]float64
	authLogins        map[string]float64
	workerAttempts    map[string]float64

	workerBatchErrors        float64
	workerHeartbeatTimestamp float64
}

type httpMetricKey struct {
	method string
	route  string
	status string
}

func NewAppMetrics() *AppMetrics {
	return &AppMetrics{
		httpRequests:      map[httpMetricKey]float64{},
		httpDurationSums:  map[httpMetricKey]float64{},
		httpDurationCount: map[httpMetricKey]float64{},
		authLogins:        map[string]float64{},
		workerAttempts:    map[string]float64{},
	}
}

func (m *AppMetrics) RegisterDatabaseReadyCollector(pinger DatabasePinger) {
	if m == nil {
		return
	}
	m.databasePinger = pinger
}

func (m *AppMetrics) RegisterEmailOutboxCollector(queryer OutboxQueryer) {
	if m == nil {
		return
	}
	m.outboxQueryer = queryer
}

func (m *AppMetrics) Handler(authToken string) http.Handler {
	return ProtectHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(m.render()))
	}), authToken)
}

func ProtectHandler(next http.Handler, authToken string) http.Handler {
	authToken = strings.TrimSpace(authToken)
	if authToken == "" {
		return next
	}

	expected := "Bearer " + authToken
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte(expected)) != 1 {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("WWW-Authenticate", `Bearer realm="metrics"`)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("metrics authentication required\n"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *AppMetrics) HTTPMiddleware(next http.Handler) http.Handler {
	if m == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := newStatusRecorder(w)

		next.ServeHTTP(recorder, r)

		key := httpMetricKey{
			method: r.Method,
			route:  NormalizeRoute(r.Method, r.URL.Path, r.Pattern),
			status: strconv.Itoa(recorder.Status()),
		}
		duration := time.Since(start).Seconds()

		m.mu.Lock()
		m.httpRequests[key]++
		m.httpDurationSums[key] += duration
		m.httpDurationCount[key]++
		m.mu.Unlock()
	})
}

func (m *AppMetrics) RecordAuthLoginOutcome(outcome string) {
	if m == nil {
		return
	}
	switch outcome {
	case "success", "invalid", "rate_limited", "error":
	default:
		outcome = "error"
	}
	m.mu.Lock()
	m.authLogins[outcome]++
	m.mu.Unlock()
}

func (m *AppMetrics) RecordEmailWorkerHeartbeat(now time.Time) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.workerHeartbeatTimestamp = float64(now.Unix())
	m.mu.Unlock()
}

func (m *AppMetrics) RecordEmailWorkerDeliveryResult(result string) {
	if m == nil {
		return
	}
	switch result {
	case "sent", "pending", "failed", "error":
	default:
		result = "error"
	}
	m.mu.Lock()
	m.workerAttempts[result]++
	m.mu.Unlock()
}

func (m *AppMetrics) RecordEmailWorkerBatchError() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.workerBatchErrors++
	m.mu.Unlock()
}

func NormalizeRoute(method string, path string, pattern string) string {
	if pattern != "" {
		fields := strings.Fields(pattern)
		if len(fields) == 2 && strings.EqualFold(fields[0], method) {
			return fields[1]
		}
		return pattern
	}
	if path == "" {
		return "unmatched"
	}
	path = uuidPattern.ReplaceAllString(path, "{id}")
	path = numberPattern.ReplaceAllString(path, "{number}")
	if path == "" {
		return "unmatched"
	}
	return path
}

func (m *AppMetrics) render() string {
	var builder strings.Builder

	m.mu.Lock()
	httpRequests := cloneHTTPMap(m.httpRequests)
	httpDurationSums := cloneHTTPMap(m.httpDurationSums)
	httpDurationCount := cloneHTTPMap(m.httpDurationCount)
	authLogins := cloneStringMap(m.authLogins)
	workerAttempts := cloneStringMap(m.workerAttempts)
	workerBatchErrors := m.workerBatchErrors
	workerHeartbeat := m.workerHeartbeatTimestamp
	m.mu.Unlock()

	writeHelpType(&builder, "http_requests_total", "Total HTTP requests by method, normalized route, and status code.", "counter")
	for _, key := range sortedHTTPKeys(httpRequests) {
		writeMetric(&builder, "http_requests_total", map[string]string{"method": key.method, "route": key.route, "status": key.status}, httpRequests[key])
	}

	writeHelpType(&builder, "http_request_duration_seconds", "HTTP request duration in seconds by method, normalized route, and status code.", "summary")
	for _, key := range sortedHTTPKeys(httpDurationSums) {
		labels := map[string]string{"method": key.method, "route": key.route, "status": key.status}
		writeMetric(&builder, "http_request_duration_seconds_sum", labels, httpDurationSums[key])
		writeMetric(&builder, "http_request_duration_seconds_count", labels, httpDurationCount[key])
	}

	writeHelpType(&builder, "database_ready", "Database readiness status from a bounded ping. 1 means ready, 0 means unavailable.", "gauge")
	writeMetric(&builder, "database_ready", nil, m.databaseReady())

	writeHelpType(&builder, "auth_login_attempts_total", "Authentication login attempts by outcome.", "counter")
	for _, outcome := range sortedStringKeys(authLogins) {
		writeMetric(&builder, "auth_login_attempts_total", map[string]string{"outcome": outcome}, authLogins[outcome])
	}

	emailOutboxCounts := m.emailOutboxCounts()
	writeHelpType(&builder, "email_outbox_records", "Email outbox records by delivery status.", "gauge")
	for _, status := range []string{"pending", "processing", "sent", "failed"} {
		writeMetric(&builder, "email_outbox_records", map[string]string{"status": status}, emailOutboxCounts[status])
	}

	writeHelpType(&builder, "email_worker_delivery_attempts_total", "Email worker delivery attempts by result.", "counter")
	for _, result := range sortedStringKeys(workerAttempts) {
		writeMetric(&builder, "email_worker_delivery_attempts_total", map[string]string{"result": result}, workerAttempts[result])
	}

	writeHelpType(&builder, "email_worker_batch_errors_total", "Email worker batch claim or processing loop errors.", "counter")
	writeMetric(&builder, "email_worker_batch_errors_total", nil, workerBatchErrors)

	writeHelpType(&builder, "email_worker_heartbeat_timestamp_seconds", "Unix timestamp of the latest email worker poll loop heartbeat.", "gauge")
	writeMetric(&builder, "email_worker_heartbeat_timestamp_seconds", nil, workerHeartbeat)

	return builder.String()
}

func (m *AppMetrics) databaseReady() float64 {
	if m.databasePinger == nil {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := m.databasePinger.Ping(ctx); err != nil {
		return 0
	}
	return 1
}

func (m *AppMetrics) emailOutboxCounts() map[string]float64 {
	counts := map[string]float64{"pending": 0, "processing": 0, "sent": 0, "failed": 0}
	if m.outboxQueryer == nil {
		return counts
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	rows, err := m.outboxQueryer.Query(ctx, `
		SELECT status, count(*)::float8
		FROM email_outbox
		GROUP BY status
	`)
	if err != nil {
		return counts
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count float64
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		if _, ok := counts[status]; ok {
			counts[status] = count
		}
	}
	return counts
}

func writeHelpType(builder *strings.Builder, name string, help string, metricType string) {
	fullName := metricName(name)
	builder.WriteString("# HELP ")
	builder.WriteString(fullName)
	builder.WriteByte(' ')
	builder.WriteString(escapeHelp(help))
	builder.WriteByte('\n')
	builder.WriteString("# TYPE ")
	builder.WriteString(fullName)
	builder.WriteByte(' ')
	builder.WriteString(metricType)
	builder.WriteByte('\n')
}

func writeMetric(builder *strings.Builder, name string, labels map[string]string, value float64) {
	builder.WriteString(metricName(name))
	if len(labels) > 0 {
		builder.WriteByte('{')
		keys := sortedStringKeys(labels)
		for index, key := range keys {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(key)
			builder.WriteString("=\"")
			builder.WriteString(escapeLabelValue(labels[key]))
			builder.WriteByte('"')
		}
		builder.WriteByte('}')
	}
	builder.WriteByte(' ')
	builder.WriteString(strconv.FormatFloat(value, 'g', -1, 64))
	builder.WriteByte('\n')
}

func metricName(name string) string {
	return Namespace + "_" + name
}

func escapeHelp(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, "\n", `\n`)
}

func escapeLabelValue(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func sortedHTTPKeys(values map[httpMetricKey]float64) []httpMetricKey {
	keys := make([]httpMetricKey, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left := fmt.Sprintf("%s\x00%s\x00%s", keys[i].method, keys[i].route, keys[i].status)
		right := fmt.Sprintf("%s\x00%s\x00%s", keys[j].method, keys[j].route, keys[j].status)
		return left < right
	})
	return keys
}

func sortedStringKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cloneHTTPMap(values map[httpMetricKey]float64) map[httpMetricKey]float64 {
	cloned := make(map[httpMetricKey]float64, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneStringMap(values map[string]float64) map[string]float64 {
	cloned := make(map[string]float64, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func newStatusRecorder(w http.ResponseWriter) *statusRecorder {
	return &statusRecorder{ResponseWriter: w}
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(data)
}

func (r *statusRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}
