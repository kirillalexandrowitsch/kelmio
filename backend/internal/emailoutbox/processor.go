package emailoutbox

import (
	"context"

	"kelmio/backend/internal/mailer"
)

type StateStore interface {
	MarkSent(ctx context.Context, id string) error
	MarkRetry(ctx context.Context, email Email, cause error, maxAttempts int) error
	MarkFailed(ctx context.Context, id string, cause error) error
}

type ProcessResult struct {
	Status string
}

func ProcessRecord(ctx context.Context, store StateStore, client mailer.Client, email Email, maxAttempts int) (ProcessResult, error) {
	message, err := Render(email)
	if err != nil {
		return ProcessResult{Status: StatusFailed}, store.MarkFailed(ctx, email.ID, err)
	}
	if err := client.Send(ctx, message); err != nil {
		status := StatusPending
		if maxAttempts <= 0 || email.AttemptCount >= maxAttempts {
			status = StatusFailed
		}
		return ProcessResult{Status: status}, store.MarkRetry(ctx, email, err, maxAttempts)
	}
	return ProcessResult{Status: StatusSent}, store.MarkSent(ctx, email.ID)
}
