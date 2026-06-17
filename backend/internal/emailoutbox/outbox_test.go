package emailoutbox

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"team-task-tracker/backend/internal/mailer"
)

func TestNormalizeEnqueueInput(t *testing.T) {
	t.Parallel()
	input, err := normalizeEnqueueInput(EnqueueInput{
		EmailType:      "  system_test ",
		RecipientEmail: " MEMBER@Example.COM ",
		TemplateData:   nil,
	})
	if err != nil {
		t.Fatalf("normalizeEnqueueInput() error = %v", err)
	}
	if input.EmailType != TypeSystemTest {
		t.Fatalf("EmailType = %q, want %q", input.EmailType, TypeSystemTest)
	}
	if input.RecipientEmail != "member@example.com" {
		t.Fatalf("RecipientEmail = %q, want normalized email", input.RecipientEmail)
	}
	if input.TemplateData == nil {
		t.Fatal("TemplateData = nil, want empty object")
	}
}

func TestNormalizeEnqueueInputValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   EnqueueInput
		want string
	}{
		{name: "missing type", in: EnqueueInput{RecipientEmail: "member@example.com"}, want: "email type is required"},
		{name: "invalid recipient", in: EnqueueInput{EmailType: TypeSystemTest, RecipientEmail: "bad"}, want: "recipient email is invalid"},
		{name: "long dedup", in: EnqueueInput{EmailType: TypeSystemTest, RecipientEmail: "member@example.com", DeduplicationKey: strings.Repeat("d", 161)}, want: "deduplication key is too long"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := normalizeEnqueueInput(tt.in)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("normalizeEnqueueInput() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestRenderSystemTestTemplate(t *testing.T) {
	t.Parallel()
	msg, err := Render(Email{
		EmailType:      TypeSystemTest,
		RecipientEmail: "member@example.com",
		TemplateData: map[string]any{
			"subject":   " Smoke ",
			"text_body": " Plain body ",
			"html_body": "<p>HTML body</p>",
		},
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if msg.Subject != "Smoke" || msg.TextBody != "Plain body" || msg.HTMLBody != "<p>HTML body</p>" {
		t.Fatalf("rendered message = %#v", msg)
	}
	if len(msg.To) != 1 || msg.To[0] != "member@example.com" {
		t.Fatalf("To = %#v", msg.To)
	}
}

func TestRenderPasswordResetTemplate(t *testing.T) {
	t.Parallel()
	msg, err := Render(Email{
		EmailType:      TypePasswordReset,
		RecipientEmail: "member@example.com",
		TemplateData: map[string]any{
			"display_name": "Member",
			"reset_url":    "http://localhost:5173/reset-password?token=secret-token",
		},
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if msg.Subject != "Reset your Team Task Tracker password" {
		t.Fatalf("Subject = %q", msg.Subject)
	}
	if !strings.Contains(msg.TextBody, "secret-token") || !strings.Contains(msg.HTMLBody, "Reset password") {
		t.Fatalf("message = %#v, want reset link in text/html", msg)
	}
}

func TestRenderTeamInviteTemplate(t *testing.T) {
	t.Parallel()
	msg, err := Render(Email{
		EmailType:      TypeTeamInvite,
		RecipientEmail: "member@example.com",
		TemplateData: map[string]any{
			"invite_url":           "http://localhost:5173/accept-invite?token=secret-token",
			"workspace_name":       "Demo Workspace",
			"role":                 "member",
			"inviter_display_name": "Admin User",
		},
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if msg.Subject != "You're invited to Demo Workspace" {
		t.Fatalf("Subject = %q", msg.Subject)
	}
	if !strings.Contains(msg.TextBody, "secret-token") || !strings.Contains(msg.HTMLBody, "Accept invite") {
		t.Fatalf("message = %#v, want invite link in text/html", msg)
	}
}

func TestRenderRejectsUnknownOrInvalidTemplate(t *testing.T) {
	t.Parallel()
	tests := []Email{
		{EmailType: "unknown", RecipientEmail: "member@example.com", TemplateData: map[string]any{}},
		{EmailType: TypeSystemTest, RecipientEmail: "member@example.com", TemplateData: map[string]any{"text_body": "body"}},
		{EmailType: TypeSystemTest, RecipientEmail: "member@example.com", TemplateData: map[string]any{"subject": "subject"}},
		{EmailType: TypePasswordReset, RecipientEmail: "member@example.com", TemplateData: map[string]any{}},
		{EmailType: TypeTeamInvite, RecipientEmail: "member@example.com", TemplateData: map[string]any{}},
	}
	for _, email := range tests {
		if _, err := Render(email); err == nil {
			t.Fatalf("Render(%#v) expected error", email)
		}
	}
}

func TestRetryDelay(t *testing.T) {
	t.Parallel()
	tests := map[int]time.Duration{
		0: time.Minute,
		1: time.Minute,
		2: 5 * time.Minute,
		3: 15 * time.Minute,
		4: time.Hour,
		9: time.Hour,
	}
	for attempt, want := range tests {
		if got := RetryDelay(attempt); got != want {
			t.Fatalf("RetryDelay(%d) = %s, want %s", attempt, got, want)
		}
	}
}

func TestSanitizeError(t *testing.T) {
	t.Parallel()
	err := errors.New("smtp failed password=super-secret token=raw-token\n" + strings.Repeat("x", 300))
	got := SanitizeError(err)
	if strings.Contains(got, "super-secret") || strings.Contains(got, "raw-token") {
		t.Fatalf("SanitizeError leaked sensitive value: %q", got)
	}
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("SanitizeError kept newlines: %q", got)
	}
	if len(got) > 240 {
		t.Fatalf("SanitizeError length = %d, want <= 240", len(got))
	}
}

func TestProcessRecordMarksSent(t *testing.T) {
	t.Parallel()
	store := &fakeStateStore{}
	client := fakeMailer{}
	email := validEmailForProcessing()
	if err := ProcessRecord(context.Background(), store, client, email, 5); err != nil {
		t.Fatalf("ProcessRecord() error = %v", err)
	}
	if store.sentID != email.ID {
		t.Fatalf("sentID = %q, want %q", store.sentID, email.ID)
	}
}

func TestProcessRecordMarksRetryAndFailed(t *testing.T) {
	t.Parallel()
	sendErr := errors.New("smtp failed")
	store := &fakeStateStore{}
	client := fakeMailer{err: sendErr}
	email := validEmailForProcessing()
	email.AttemptCount = 1
	if err := ProcessRecord(context.Background(), store, client, email, 5); err != nil {
		t.Fatalf("ProcessRecord retry error = %v", err)
	}
	if store.retryID != email.ID {
		t.Fatalf("retryID = %q, want %q", store.retryID, email.ID)
	}

	store = &fakeStateStore{}
	email.AttemptCount = 5
	if err := ProcessRecord(context.Background(), store, client, email, 5); err != nil {
		t.Fatalf("ProcessRecord failed error = %v", err)
	}
	if store.failedID != email.ID {
		t.Fatalf("failedID = %q, want %q", store.failedID, email.ID)
	}
}

func TestProcessRecordMarksFailedForTemplateError(t *testing.T) {
	t.Parallel()
	store := &fakeStateStore{}
	email := validEmailForProcessing()
	email.EmailType = "unknown"
	if err := ProcessRecord(context.Background(), store, fakeMailer{}, email, 5); err != nil {
		t.Fatalf("ProcessRecord() error = %v", err)
	}
	if store.failedID != email.ID {
		t.Fatalf("failedID = %q, want %q", store.failedID, email.ID)
	}
}

func validEmailForProcessing() Email {
	return Email{
		ID:             "email-1",
		EmailType:      TypeSystemTest,
		RecipientEmail: "member@example.com",
		TemplateData: map[string]any{
			"subject":   "Subject",
			"text_body": "Body",
		},
	}
}

type fakeMailer struct {
	err error
}

func (f fakeMailer) Send(context.Context, mailer.Message) error {
	return f.err
}

type fakeStateStore struct {
	sentID   string
	retryID  string
	failedID string
}

func (f *fakeStateStore) MarkSent(_ context.Context, id string) error {
	f.sentID = id
	return nil
}

func (f *fakeStateStore) MarkRetry(_ context.Context, email Email, _ error, maxAttempts int) error {
	if email.AttemptCount >= maxAttempts {
		return f.MarkFailed(context.Background(), email.ID, errors.New("terminal"))
	}
	f.retryID = email.ID
	return nil
}

func (f *fakeStateStore) MarkFailed(_ context.Context, id string, _ error) error {
	f.failedID = id
	return nil
}
