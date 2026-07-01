// Package audit provides a minimal helper for recording administrative actions
// into the audit_log table. It works with either a connection pool or a
// transaction so callers can record within the same transaction as the action.
package audit

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgconn"
)

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Entry describes a single administrative action. Empty id fields are stored as
// NULL; Metadata may be nil.
type Entry struct {
	OrganizationID string
	ActorID        string
	Action         string
	TargetType     string
	TargetID       string
	Metadata       map[string]any
}

// Record inserts an audit entry. The metadata is serialized to JSON.
func Record(ctx context.Context, q Querier, entry Entry) error {
	metadata := entry.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return err
	}

	_, err = q.Exec(ctx, `
		INSERT INTO audit_log (organization_id, actor_id, action, target_type, target_id, metadata)
		VALUES (
			nullif($1, '')::uuid,
			nullif($2, '')::uuid,
			$3,
			$4,
			nullif($5, '')::uuid,
			$6
		)
	`, entry.OrganizationID, entry.ActorID, entry.Action, entry.TargetType, entry.TargetID, encoded)
	return err
}
