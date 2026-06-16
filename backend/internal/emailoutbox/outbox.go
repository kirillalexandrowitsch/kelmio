package emailoutbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusSent       = "sent"
	StatusFailed     = "failed"

	TypeSystemTest    = "system_test"
	TypePasswordReset = "password_reset"
)

var ErrInvalidInput = errors.New("email outbox input is invalid")

var sensitiveErrorPattern = regexp.MustCompile(`(?i)(password|token|secret)\s*[:=]\s*[^ ]+`)

type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Store struct {
	db  *pgxpool.Pool
	now func() time.Time
}

type EnqueueInput struct {
	WorkspaceID      *string
	EmailType        string
	RecipientEmail   string
	TemplateData     map[string]any
	DeduplicationKey string
	NextAttemptAt    time.Time
}

type Email struct {
	ID                  string
	WorkspaceID         *string
	EmailType           string
	RecipientEmail      string
	TemplateData        map[string]any
	Status              string
	AttemptCount        int
	NextAttemptAt       time.Time
	LastError           *string
	DeduplicationKey    string
	ProcessingStartedAt *time.Time
	SentAt              *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db, now: time.Now}
}

func Enqueue(ctx context.Context, db DBTX, input EnqueueInput) (Email, error) {
	normalized, err := normalizeEnqueueInput(input)
	if err != nil {
		return Email{}, err
	}
	templateData, err := json.Marshal(normalized.TemplateData)
	if err != nil {
		return Email{}, fmt.Errorf("marshal template data: %w", err)
	}
	nextAttemptAt := normalized.NextAttemptAt
	if nextAttemptAt.IsZero() {
		nextAttemptAt = time.Now()
	}

	return scanEmail(db.QueryRow(ctx, `
		INSERT INTO email_outbox (
			workspace_id,
			email_type,
			recipient_email,
			template_data,
			status,
			next_attempt_at,
			deduplication_key
		)
		VALUES ($1, $2, $3, $4::jsonb, 'pending', $5, $6)
		ON CONFLICT (deduplication_key) WHERE deduplication_key <> ''
		DO UPDATE SET deduplication_key = email_outbox.deduplication_key
		RETURNING
			id::text,
			workspace_id::text,
			email_type,
			recipient_email,
			template_data::text,
			status,
			attempt_count,
			next_attempt_at,
			last_error,
			deduplication_key,
			processing_started_at,
			sent_at,
			created_at,
			updated_at
	`, normalized.WorkspaceID, normalized.EmailType, normalized.RecipientEmail, string(templateData), nextAttemptAt, normalized.DeduplicationKey))
}

func (s *Store) Enqueue(ctx context.Context, input EnqueueInput) (Email, error) {
	if input.NextAttemptAt.IsZero() {
		input.NextAttemptAt = s.now().UTC()
	}
	return Enqueue(ctx, s.db, input)
}

func (s *Store) ClaimBatch(ctx context.Context, limit int, staleAfter time.Duration) ([]Email, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%w: limit must be greater than 0", ErrInvalidInput)
	}
	if staleAfter <= 0 {
		return nil, fmt.Errorf("%w: staleAfter must be greater than 0", ErrInvalidInput)
	}

	now := s.now().UTC()
	staleBefore := now.Add(-staleAfter)
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		WITH candidate AS (
			SELECT id
			FROM email_outbox
			WHERE (
					status = 'pending'
					AND next_attempt_at <= $1
				)
				OR (
					status = 'processing'
					AND processing_started_at <= $2
				)
			ORDER BY next_attempt_at, created_at, id
			LIMIT $3
			FOR UPDATE SKIP LOCKED
		)
		UPDATE email_outbox email
		SET status = 'processing',
			attempt_count = email.attempt_count + 1,
			processing_started_at = $1,
			updated_at = $1
		FROM candidate
		WHERE email.id = candidate.id
		RETURNING
			email.id::text,
			email.workspace_id::text,
			email.email_type,
			email.recipient_email,
			email.template_data::text,
			email.status,
			email.attempt_count,
			email.next_attempt_at,
			email.last_error,
			email.deduplication_key,
			email.processing_started_at,
			email.sent_at,
			email.created_at,
			email.updated_at
	`, now, staleBefore, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emails := make([]Email, 0, limit)
	for rows.Next() {
		email, err := scanEmail(rows)
		if err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return emails, nil
}

func (s *Store) MarkSent(ctx context.Context, id string) error {
	now := s.now().UTC()
	_, err := s.db.Exec(ctx, `
		UPDATE email_outbox
		SET status = 'sent',
			last_error = NULL,
			processing_started_at = NULL,
			sent_at = $2,
			updated_at = $2
		WHERE id = $1
	`, id, now)
	return err
}

func (s *Store) MarkRetry(ctx context.Context, email Email, cause error, maxAttempts int) error {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	if email.AttemptCount >= maxAttempts {
		return s.MarkFailed(ctx, email.ID, cause)
	}
	now := s.now().UTC()
	_, err := s.db.Exec(ctx, `
		UPDATE email_outbox
		SET status = 'pending',
			next_attempt_at = $2,
			last_error = $3,
			processing_started_at = NULL,
			updated_at = $4
		WHERE id = $1
	`, email.ID, now.Add(RetryDelay(email.AttemptCount)), SanitizeError(cause), now)
	return err
}

func (s *Store) MarkFailed(ctx context.Context, id string, cause error) error {
	now := s.now().UTC()
	_, err := s.db.Exec(ctx, `
		UPDATE email_outbox
		SET status = 'failed',
			last_error = $2,
			processing_started_at = NULL,
			updated_at = $3
		WHERE id = $1
	`, id, SanitizeError(cause), now)
	return err
}

func RetryDelay(attemptCount int) time.Duration {
	switch {
	case attemptCount <= 1:
		return time.Minute
	case attemptCount == 2:
		return 5 * time.Minute
	case attemptCount == 3:
		return 15 * time.Minute
	default:
		return time.Hour
	}
}

func SanitizeError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	message = strings.ReplaceAll(message, "\r", " ")
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.Join(strings.Fields(message), " ")
	message = sensitiveErrorPattern.ReplaceAllString(message, "$1=[redacted]")
	if len(message) > 240 {
		message = message[:240]
	}
	if message == "" {
		return "email delivery failed"
	}
	return message
}

func normalizeEnqueueInput(input EnqueueInput) (EnqueueInput, error) {
	input.EmailType = strings.TrimSpace(input.EmailType)
	input.RecipientEmail = strings.ToLower(strings.TrimSpace(input.RecipientEmail))
	input.DeduplicationKey = strings.TrimSpace(input.DeduplicationKey)
	if input.TemplateData == nil {
		input.TemplateData = map[string]any{}
	}
	if input.EmailType == "" {
		return input, fmt.Errorf("%w: email type is required", ErrInvalidInput)
	}
	parsedAddress, err := mail.ParseAddress(input.RecipientEmail)
	if err != nil || parsedAddress.Address != input.RecipientEmail || !strings.Contains(parsedAddress.Address, "@") {
		return input, fmt.Errorf("%w: recipient email is invalid", ErrInvalidInput)
	}
	if len(input.DeduplicationKey) > 160 {
		return input, fmt.Errorf("%w: deduplication key is too long", ErrInvalidInput)
	}
	return input, nil
}

func scanEmail(row interface{ Scan(...any) error }) (Email, error) {
	var email Email
	var workspaceID pgtype.Text
	var templateData string
	var lastError pgtype.Text
	var processingStartedAt pgtype.Timestamptz
	var sentAt pgtype.Timestamptz
	if err := row.Scan(
		&email.ID,
		&workspaceID,
		&email.EmailType,
		&email.RecipientEmail,
		&templateData,
		&email.Status,
		&email.AttemptCount,
		&email.NextAttemptAt,
		&lastError,
		&email.DeduplicationKey,
		&processingStartedAt,
		&sentAt,
		&email.CreatedAt,
		&email.UpdatedAt,
	); err != nil {
		return Email{}, err
	}
	if workspaceID.Valid {
		email.WorkspaceID = &workspaceID.String
	}
	if lastError.Valid {
		email.LastError = &lastError.String
	}
	if processingStartedAt.Valid {
		email.ProcessingStartedAt = &processingStartedAt.Time
	}
	if sentAt.Valid {
		email.SentAt = &sentAt.Time
	}
	if err := json.Unmarshal([]byte(templateData), &email.TemplateData); err != nil {
		return Email{}, fmt.Errorf("unmarshal template data: %w", err)
	}
	return email, nil
}
