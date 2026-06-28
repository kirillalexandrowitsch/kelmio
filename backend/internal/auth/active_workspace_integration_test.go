//go:build integration

package auth

import (
	"testing"
	"time"
)

func TestUserBySessionResolvesActiveWorkspace(t *testing.T) {
	ctx, db, userID := setupAuthIntegrationWorkspace(t)
	handler := NewHandler(db, time.Hour, false, newIntegrationCSRFManager(t), nil)

	var orgID string
	if err := db.QueryRow(ctx, `SELECT id::text FROM organizations WHERE slug = 'default'`).Scan(&orgID); err != nil {
		t.Fatalf("read default organization: %v", err)
	}

	var secondWorkspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name, organization_id, slug, status)
		VALUES ('Second Workspace', $1, 'second', 'active')
		RETURNING id::text
	`, orgID).Scan(&secondWorkspaceID); err != nil {
		t.Fatalf("insert second workspace: %v", err)
	}
	if _, err := db.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'member')
	`, secondWorkspaceID, userID); err != nil {
		t.Fatalf("insert second membership: %v", err)
	}

	newSession := func(active any) string {
		token, err := newSessionToken()
		if err != nil {
			t.Fatalf("new session token: %v", err)
		}
		if _, err := db.Exec(ctx, `
			INSERT INTO sessions (user_id, token_hash, expires_at, active_workspace_id)
			VALUES ($1, $2, now() + interval '1 hour', $3)
		`, userID, hashToken(token), active); err != nil {
			t.Fatalf("insert session: %v", err)
		}
		return token
	}

	// A session pinned to the second workspace resolves to it.
	activeToken := newSession(secondWorkspaceID)
	record, err := handler.userBySession(ctx, hashToken(activeToken))
	if err != nil {
		t.Fatalf("userBySession (active): %v", err)
	}
	if record.WorkspaceID != secondWorkspaceID {
		t.Fatalf("active workspace = %q, want %q", record.WorkspaceID, secondWorkspaceID)
	}
	if record.Role != "member" {
		t.Fatalf("active workspace role = %q, want member", record.Role)
	}
	if record.OrganizationID != orgID {
		t.Fatalf("organization id = %q, want %q", record.OrganizationID, orgID)
	}

	// A session without a pinned workspace falls back to the first membership.
	fallbackToken := newSession(nil)
	record, err = handler.userBySession(ctx, hashToken(fallbackToken))
	if err != nil {
		t.Fatalf("userBySession (fallback): %v", err)
	}
	if record.WorkspaceID == secondWorkspaceID {
		t.Fatalf("fallback resolved to the pinned workspace; want the first membership")
	}
	if record.Role != "admin" {
		t.Fatalf("fallback workspace role = %q, want admin", record.Role)
	}

	// A session pinned to a workspace the user does not belong to also falls back.
	var foreignWorkspaceID string
	if err := db.QueryRow(ctx, `
		INSERT INTO workspaces (name, organization_id, slug, status)
		VALUES ('Foreign Workspace', $1, 'foreign', 'active')
		RETURNING id::text
	`, orgID).Scan(&foreignWorkspaceID); err != nil {
		t.Fatalf("insert foreign workspace: %v", err)
	}
	foreignToken := newSession(foreignWorkspaceID)
	record, err = handler.userBySession(ctx, hashToken(foreignToken))
	if err != nil {
		t.Fatalf("userBySession (foreign): %v", err)
	}
	if record.WorkspaceID == foreignWorkspaceID {
		t.Fatalf("resolved to a workspace the user is not a member of")
	}
}
