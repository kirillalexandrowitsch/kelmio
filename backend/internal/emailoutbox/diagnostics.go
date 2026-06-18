package emailoutbox

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const diagnosticsFailureLimit = 10

type DiagnosticsQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Diagnostics struct {
	Total                     int                  `json:"total"`
	Counts                    DiagnosticsCounts    `json:"counts"`
	OldestPendingAt           *time.Time           `json:"oldest_pending_at"`
	OldestProcessingStartedAt *time.Time           `json:"oldest_processing_started_at"`
	RecentTerminalFailures    []DiagnosticsFailure `json:"recent_terminal_failures"`
}

type DiagnosticsCounts struct {
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Sent       int `json:"sent"`
	Failed     int `json:"failed"`
}

type DiagnosticsFailure struct {
	ID             string     `json:"id"`
	EmailType      string     `json:"email_type"`
	RecipientEmail string     `json:"recipient_email"`
	AttemptCount   int        `json:"attempt_count"`
	LastError      string     `json:"last_error"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	NextAttemptAt  time.Time  `json:"next_attempt_at"`
	SentAt         *time.Time `json:"sent_at"`
}

func LoadDiagnostics(ctx context.Context, db DiagnosticsQueryer, workspaceID string) (Diagnostics, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return Diagnostics{}, errors.New("workspace id is required")
	}

	var diagnostics Diagnostics
	var oldestPendingAt pgtype.Timestamptz
	var oldestProcessingStartedAt pgtype.Timestamptz
	if err := db.QueryRow(ctx, `
		SELECT
			count(*)::int,
			count(*) FILTER (WHERE status = 'pending')::int,
			count(*) FILTER (WHERE status = 'processing')::int,
			count(*) FILTER (WHERE status = 'sent')::int,
			count(*) FILTER (WHERE status = 'failed')::int,
			min(created_at) FILTER (WHERE status = 'pending'),
			min(processing_started_at) FILTER (WHERE status = 'processing')
		FROM email_outbox
		WHERE workspace_id = $1
	`, workspaceID).Scan(
		&diagnostics.Total,
		&diagnostics.Counts.Pending,
		&diagnostics.Counts.Processing,
		&diagnostics.Counts.Sent,
		&diagnostics.Counts.Failed,
		&oldestPendingAt,
		&oldestProcessingStartedAt,
	); err != nil {
		return Diagnostics{}, err
	}
	if oldestPendingAt.Valid {
		diagnostics.OldestPendingAt = &oldestPendingAt.Time
	}
	if oldestProcessingStartedAt.Valid {
		diagnostics.OldestProcessingStartedAt = &oldestProcessingStartedAt.Time
	}

	failures, err := loadRecentTerminalFailures(ctx, db, workspaceID)
	if err != nil {
		return Diagnostics{}, err
	}
	diagnostics.RecentTerminalFailures = failures

	return diagnostics, nil
}

func loadRecentTerminalFailures(ctx context.Context, db DiagnosticsQueryer, workspaceID string) ([]DiagnosticsFailure, error) {
	rows, err := db.Query(ctx, `
		SELECT
			id::text,
			email_type,
			recipient_email,
			attempt_count,
			COALESCE(last_error, ''),
			created_at,
			updated_at,
			next_attempt_at,
			sent_at
		FROM email_outbox
		WHERE workspace_id = $1
			AND status = 'failed'
		ORDER BY updated_at DESC, id DESC
		LIMIT $2
	`, workspaceID, diagnosticsFailureLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	failures := make([]DiagnosticsFailure, 0)
	for rows.Next() {
		var failure DiagnosticsFailure
		var lastError string
		var sentAt pgtype.Timestamptz
		if err := rows.Scan(
			&failure.ID,
			&failure.EmailType,
			&failure.RecipientEmail,
			&failure.AttemptCount,
			&lastError,
			&failure.CreatedAt,
			&failure.UpdatedAt,
			&failure.NextAttemptAt,
			&sentAt,
		); err != nil {
			return nil, err
		}
		failure.RecipientEmail = MaskRecipientEmail(failure.RecipientEmail)
		if lastError != "" {
			failure.LastError = SanitizeError(errors.New(lastError))
		}
		if sentAt.Valid {
			failure.SentAt = &sentAt.Time
		}
		failures = append(failures, failure)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return failures, nil
}

func MaskRecipientEmail(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	localPart, domain, ok := strings.Cut(email, "@")
	if !ok || localPart == "" || domain == "" {
		return "[masked]"
	}
	first := localPart[:1]
	return first + "***@" + domain
}
