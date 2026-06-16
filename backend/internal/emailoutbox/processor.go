package emailoutbox

import (
	"context"

	"team-task-tracker/backend/internal/mailer"
)

type StateStore interface {
	MarkSent(ctx context.Context, id string) error
	MarkRetry(ctx context.Context, email Email, cause error, maxAttempts int) error
	MarkFailed(ctx context.Context, id string, cause error) error
}

func ProcessRecord(ctx context.Context, store StateStore, client mailer.Client, email Email, maxAttempts int) error {
	message, err := Render(email)
	if err != nil {
		return store.MarkFailed(ctx, email.ID, err)
	}
	if err := client.Send(ctx, message); err != nil {
		return store.MarkRetry(ctx, email, err, maxAttempts)
	}
	return store.MarkSent(ctx, email.ID)
}
