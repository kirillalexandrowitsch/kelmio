package invites

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"kelmio/backend/internal/auth"
	"kelmio/backend/internal/emailoutbox"
)

const inviteTTL = 7 * 24 * time.Hour
const inviteResendCooldown = time.Minute

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{2,31}$`)
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

var errInviteExists = errors.New("invite exists")
var errInviteExpired = errors.New("invite expired")
var errInviteRevoked = errors.New("invite revoked")
var errInviteAccepted = errors.New("invite already accepted")
var errAlreadyMember = errors.New("already member")
var errUserExists = errors.New("user exists")
var errInviteEmailUnavailable = errors.New("invite email unavailable")

type Handler struct {
	db            *pgxpool.Pool
	auth          *auth.Handler
	inviteBaseURL string
	now           func() time.Time
}

type HandlerOption func(*Handler)

func WithInviteBaseURL(baseURL string) HandlerOption {
	return func(handler *Handler) {
		handler.inviteBaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	}
}

type inviteResendCooldownError struct {
	retryAfter time.Duration
}

func (err inviteResendCooldownError) Error() string {
	return "invite resend cooldown"
}

type createInviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type acceptInviteRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

type inviteResponse struct {
	ID                  string     `json:"id"`
	WorkspaceID         string     `json:"workspace_id"`
	Email               string     `json:"email"`
	Role                string     `json:"role"`
	Status              string     `json:"status"`
	CreatedBy           string     `json:"created_by"`
	CreatedAt           time.Time  `json:"created_at"`
	ExpiresAt           time.Time  `json:"expires_at"`
	AcceptedAt          *time.Time `json:"accepted_at"`
	RevokedAt           *time.Time `json:"revoked_at"`
	EmailDeliveryStatus string     `json:"email_delivery_status"`
	EmailQueuedAt       *time.Time `json:"email_queued_at"`
	EmailSentAt         *time.Time `json:"email_sent_at"`
}

type createInviteResponse struct {
	inviteResponse
	AcceptToken   string `json:"accept_token"`
	AcceptURLPath string `json:"accept_url_path"`
}

type listInvitesResponse struct {
	Invites []inviteResponse `json:"invites"`
}

type invitePreviewResponse struct {
	WorkspaceID   string    `json:"workspace_id"`
	WorkspaceName string    `json:"workspace_name"`
	Email         string    `json:"email"`
	Role          string    `json:"role"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type acceptInviteResponse struct {
	Accepted    bool   `json:"accepted"`
	WorkspaceID string `json:"workspace_id"`
	Email       string `json:"email"`
	Username    string `json:"username"`
	Role        string `json:"role"`
}

type normalizedCreateInvite struct {
	Email string
	Role  string
}

type normalizedAcceptInvite struct {
	Username    string
	DisplayName string
	Password    string
}

type inviteRecord struct {
	ID                  string
	WorkspaceID         string
	WorkspaceName       string
	Email               string
	Role                string
	CreatedBy           string
	CreatedByName       string
	CreatedAt           time.Time
	ExpiresAt           time.Time
	AcceptedAt          *time.Time
	RevokedAt           *time.Time
	EmailDeliveryStatus string
	EmailQueuedAt       *time.Time
	EmailSentAt         *time.Time
}

type dbtx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler, options ...HandlerOption) *Handler {
	handler := &Handler{
		db:   db,
		auth: authHandler,
		now:  func() time.Time { return time.Now().UTC() },
	}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/team/invites", h.list)
	mux.HandleFunc("POST /api/v1/team/invites", h.create)
	mux.HandleFunc("POST /api/v1/team/invites/{id}/revoke", h.revoke)
	mux.HandleFunc("POST /api/v1/team/invites/{id}/resend", h.resend)
	mux.HandleFunc("GET /api/v1/auth/invites/{token}", h.preview)
	mux.HandleFunc("POST /api/v1/auth/invites/{token}/accept", h.accept)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	invites, err := h.listInvites(ctx, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list invites")
		return
	}

	writeJSON(w, http.StatusOK, listInvitesResponse{Invites: invites})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}

	var req createInviteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input, err := normalizeCreateInvite(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	invite, token, err := h.createInvite(ctx, user, input)
	if err != nil {
		if errors.Is(err, errInviteExists) {
			writeError(w, http.StatusConflict, "invite_exists", "pending invite already exists")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not create invite")
		return
	}

	writeJSON(w, http.StatusCreated, createInviteResponse{
		inviteResponse: invite,
		AcceptToken:    token,
		AcceptURLPath:  "/accept-invite?token=" + token,
	})
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}

	inviteID, err := normalizeInviteID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	invite, err := h.revokeInvite(ctx, user, inviteID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "invite_not_found", "invite was not found")
			return
		}
		if errors.Is(err, errInviteAccepted) {
			writeError(w, http.StatusBadRequest, "invite_already_accepted", "invite was already accepted")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not revoke invite")
		return
	}

	writeJSON(w, http.StatusOK, invite)
}

func (h *Handler) resend(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}

	inviteID, err := normalizeInviteID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	invite, err := h.resendInvite(ctx, user, inviteID)
	if err != nil {
		var cooldownErr inviteResendCooldownError
		if errors.As(err, &cooldownErr) {
			retryAfter := int(cooldownErr.retryAfter.Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
			writeError(w, http.StatusTooManyRequests, "rate_limited", "invite email was resent recently")
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "invite_not_found", "invite was not found")
			return
		}
		if errors.Is(err, errInviteExpired) {
			writeError(w, http.StatusBadRequest, "invite_expired", "invite has expired")
			return
		}
		if errors.Is(err, errInviteRevoked) {
			writeError(w, http.StatusBadRequest, "invite_revoked", "invite was revoked")
			return
		}
		if errors.Is(err, errInviteAccepted) {
			writeError(w, http.StatusBadRequest, "invite_already_accepted", "invite was already accepted")
			return
		}
		if errors.Is(err, errInviteEmailUnavailable) {
			writeError(w, http.StatusConflict, "invite_email_unavailable", "invite email link is not available for resend")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not resend invite")
		return
	}

	writeJSON(w, http.StatusOK, invite)
}

func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	token, err := normalizeInviteToken(r.PathValue("token"))
	if err != nil {
		writeError(w, http.StatusNotFound, "invite_not_found", "invite was not found")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	invite, err := h.inviteByToken(ctx, token)
	if err != nil {
		h.writePublicInviteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, invitePreviewResponse{
		WorkspaceID:   invite.WorkspaceID,
		WorkspaceName: invite.WorkspaceName,
		Email:         invite.Email,
		Role:          invite.Role,
		ExpiresAt:     invite.ExpiresAt,
	})
}

func (h *Handler) accept(w http.ResponseWriter, r *http.Request) {
	token, err := normalizeInviteToken(r.PathValue("token"))
	if err != nil {
		writeError(w, http.StatusNotFound, "invite_not_found", "invite was not found")
		return
	}

	var req acceptInviteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input, err := normalizeAcceptInvite(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response, err := h.acceptInvite(ctx, token, input)
	if err != nil {
		h.writeAcceptInviteError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) listInvites(ctx context.Context, user auth.CurrentUser) ([]inviteResponse, error) {
	rows, err := h.db.Query(ctx, `
		SELECT
			ti.id::text,
			ti.workspace_id::text,
			w.name,
			ti.email,
			ti.role,
			ti.created_by::text,
			COALESCE(u.display_name, ''),
			ti.created_at,
			ti.expires_at,
			ti.accepted_at,
			ti.revoked_at,
			COALESCE(delivery.status, ''),
			delivery.created_at,
			delivery.sent_at
		FROM team_invites ti
		JOIN workspaces w ON w.id = ti.workspace_id
		LEFT JOIN users u ON u.id = ti.created_by
		LEFT JOIN LATERAL (
			SELECT eo.status, eo.created_at, eo.sent_at
			FROM email_outbox eo
			WHERE eo.email_type = $2
				AND eo.template_data->>'invite_id' = ti.id::text
			ORDER BY eo.created_at DESC, eo.id DESC
			LIMIT 1
		) delivery ON true
		WHERE ti.workspace_id = $1
		ORDER BY ti.created_at DESC, ti.id DESC
		LIMIT 100
	`, user.WorkspaceID, emailoutbox.TypeTeamInvite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now().UTC()
	invites := make([]inviteResponse, 0)
	for rows.Next() {
		invite, err := scanInvite(rows)
		if err != nil {
			return nil, err
		}

		invites = append(invites, invite.toResponse(now))
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return invites, nil
}

func (h *Handler) createInvite(ctx context.Context, user auth.CurrentUser, input normalizedCreateInvite) (inviteResponse, string, error) {
	token, err := newInviteToken()
	if err != nil {
		return inviteResponse{}, "", err
	}

	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return inviteResponse{}, "", err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	now := h.now()
	expiresAt := now.Add(inviteTTL)
	invite, err := scanInvite(tx.QueryRow(ctx, `
		INSERT INTO team_invites (workspace_id, email, role, token_hash, created_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING
			id::text,
			workspace_id::text,
			(SELECT name FROM workspaces WHERE id = $1),
			email,
			role,
			created_by::text,
			$7::text,
			created_at,
			expires_at,
			accepted_at,
			revoked_at,
			''::text,
			NULL::timestamptz,
			NULL::timestamptz
	`, user.WorkspaceID, input.Email, input.Role, hashInviteToken(token), user.ID, expiresAt, user.DisplayName))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return inviteResponse{}, "", errInviteExists
		}

		return inviteResponse{}, "", err
	}

	emailRecord, err := h.enqueueInviteEmail(ctx, tx, invite, token, "create")
	if err != nil {
		return inviteResponse{}, "", err
	}
	invite.EmailDeliveryStatus = emailRecord.Status
	invite.EmailQueuedAt = &emailRecord.CreatedAt
	invite.EmailSentAt = emailRecord.SentAt

	if err := tx.Commit(ctx); err != nil {
		return inviteResponse{}, "", err
	}

	return invite.toResponse(now), token, nil
}

func (h *Handler) resendInvite(ctx context.Context, user auth.CurrentUser, inviteID string) (inviteResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return inviteResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	now := h.now()
	invite, err := h.inviteByIDForUpdate(ctx, tx, user.WorkspaceID, inviteID)
	if err != nil {
		return inviteResponse{}, err
	}
	if err := ensureInvitePending(invite, now); err != nil {
		return inviteResponse{}, err
	}

	payload, err := h.latestInviteEmailPayload(ctx, tx, invite.ID)
	if err != nil {
		return inviteResponse{}, err
	}
	if payload.InviteURLPath == "" {
		return inviteResponse{}, errInviteEmailUnavailable
	}

	retryAfter, err := h.inviteResendRetryAfter(ctx, tx, invite.ID, now)
	if err != nil {
		return inviteResponse{}, err
	}
	if retryAfter > 0 {
		return inviteResponse{}, inviteResendCooldownError{retryAfter: retryAfter}
	}

	emailRecord, err := h.enqueueInviteEmailWithPayload(ctx, tx, invite, payload.InviteURLPath, payload.InviteURL, "resend")
	if err != nil {
		return inviteResponse{}, err
	}
	invite.EmailDeliveryStatus = emailRecord.Status
	invite.EmailQueuedAt = &emailRecord.CreatedAt
	invite.EmailSentAt = emailRecord.SentAt

	if err := tx.Commit(ctx); err != nil {
		return inviteResponse{}, err
	}

	return invite.toResponse(now), nil
}

type inviteEmailPayload struct {
	InviteURLPath string
	InviteURL     string
}

func (h *Handler) latestInviteEmailPayload(ctx context.Context, tx pgx.Tx, inviteID string) (inviteEmailPayload, error) {
	var payload inviteEmailPayload
	var inviteURLPath pgtype.Text
	var inviteURL pgtype.Text
	if err := tx.QueryRow(ctx, `
		SELECT
			template_data->>'invite_url_path',
			template_data->>'invite_url'
		FROM email_outbox
		WHERE email_type = $1
			AND template_data->>'invite_id' = $2
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, emailoutbox.TypeTeamInvite, inviteID).Scan(&inviteURLPath, &inviteURL); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return inviteEmailPayload{}, errInviteEmailUnavailable
		}
		return inviteEmailPayload{}, err
	}
	if inviteURLPath.Valid {
		payload.InviteURLPath = strings.TrimSpace(inviteURLPath.String)
	}
	if inviteURL.Valid {
		payload.InviteURL = strings.TrimSpace(inviteURL.String)
	}
	return payload, nil
}

func (h *Handler) inviteResendRetryAfter(ctx context.Context, tx pgx.Tx, inviteID string, now time.Time) (time.Duration, error) {
	var latestResendAt pgtype.Timestamptz
	if err := tx.QueryRow(ctx, `
		SELECT max(created_at)
		FROM email_outbox
		WHERE email_type = $1
			AND template_data->>'invite_id' = $2
			AND deduplication_key LIKE $3
	`, emailoutbox.TypeTeamInvite, inviteID, "team_invite:resend:"+inviteID+":%").Scan(&latestResendAt); err != nil {
		return 0, err
	}
	if !latestResendAt.Valid {
		return 0, nil
	}
	nextAllowedAt := latestResendAt.Time.Add(inviteResendCooldown)
	if !now.Before(nextAllowedAt) {
		return 0, nil
	}
	return nextAllowedAt.Sub(now), nil
}

func (h *Handler) enqueueInviteEmail(ctx context.Context, tx pgx.Tx, invite inviteRecord, token string, reason string) (emailoutbox.Email, error) {
	inviteURLPath := "/accept-invite?token=" + token
	inviteURL := inviteURLPath
	if h.inviteBaseURL != "" {
		inviteURL = h.inviteBaseURL + inviteURLPath
	}
	return h.enqueueInviteEmailWithPayload(ctx, tx, invite, inviteURLPath, inviteURL, reason)
}

func (h *Handler) enqueueInviteEmailWithPayload(ctx context.Context, tx pgx.Tx, invite inviteRecord, inviteURLPath string, inviteURL string, reason string) (emailoutbox.Email, error) {
	now := h.now()
	if inviteURL == "" && h.inviteBaseURL != "" {
		inviteURL = h.inviteBaseURL + inviteURLPath
	}
	if inviteURL == "" {
		inviteURL = inviteURLPath
	}
	return emailoutbox.Enqueue(ctx, tx, emailoutbox.EnqueueInput{
		WorkspaceID:    &invite.WorkspaceID,
		EmailType:      emailoutbox.TypeTeamInvite,
		RecipientEmail: invite.Email,
		TemplateData: map[string]any{
			"invite_url":           inviteURL,
			"invite_url_path":      inviteURLPath,
			"workspace_name":       invite.WorkspaceName,
			"email":                invite.Email,
			"role":                 invite.Role,
			"expires_at":           invite.ExpiresAt.Format(time.RFC3339),
			"inviter_display_name": invite.CreatedByName,
			"invite_id":            invite.ID,
		},
		DeduplicationKey: fmt.Sprintf("team_invite:%s:%s:%d", reason, invite.ID, now.UnixNano()),
		NextAttemptAt:    now,
	})
}

func (h *Handler) revokeInvite(ctx context.Context, user auth.CurrentUser, inviteID string) (inviteResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return inviteResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	invite, err := h.inviteByIDForUpdate(ctx, tx, user.WorkspaceID, inviteID)
	if err != nil {
		return inviteResponse{}, err
	}
	if invite.AcceptedAt != nil {
		return inviteResponse{}, errInviteAccepted
	}

	if invite.RevokedAt == nil {
		invite, err = scanInvite(tx.QueryRow(ctx, `
			UPDATE team_invites
			SET revoked_at = now()
			WHERE id = $1
			RETURNING
			id::text,
			workspace_id::text,
			(SELECT name FROM workspaces WHERE workspaces.id = team_invites.workspace_id),
			email,
			role,
			created_by::text,
			COALESCE((SELECT display_name FROM users WHERE users.id = team_invites.created_by), ''),
			created_at,
			expires_at,
			accepted_at,
			revoked_at,
			''::text,
			NULL::timestamptz,
			NULL::timestamptz
		`, inviteID))
		if err != nil {
			return inviteResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return inviteResponse{}, err
	}

	return invite.toResponse(time.Now().UTC()), nil
}

func (h *Handler) inviteByToken(ctx context.Context, token string) (inviteRecord, error) {
	invite, err := h.inviteByTokenHash(ctx, h.db, hashInviteToken(token), false)
	if err != nil {
		return inviteRecord{}, err
	}
	if err := ensureInvitePending(invite, time.Now().UTC()); err != nil {
		return inviteRecord{}, err
	}

	return invite, nil
}

func (h *Handler) acceptInvite(ctx context.Context, token string, input normalizedAcceptInvite) (acceptInviteResponse, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return acceptInviteResponse{}, err
	}

	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return acceptInviteResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	invite, err := h.inviteByTokenHash(ctx, tx, hashInviteToken(token), true)
	if err != nil {
		return acceptInviteResponse{}, err
	}
	if err := ensureInvitePending(invite, time.Now().UTC()); err != nil {
		return acceptInviteResponse{}, err
	}

	userID, err := h.upsertInvitedUser(ctx, tx, invite, input, string(passwordHash))
	if err != nil {
		return acceptInviteResponse{}, err
	}
	if userID == "" {
		return acceptInviteResponse{}, errors.New("accepted user id is empty")
	}

	if _, err := tx.Exec(ctx, `
		UPDATE team_invites
		SET accepted_at = now()
		WHERE id = $1
	`, invite.ID); err != nil {
		return acceptInviteResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return acceptInviteResponse{}, err
	}

	return acceptInviteResponse{
		Accepted:    true,
		WorkspaceID: invite.WorkspaceID,
		Email:       invite.Email,
		Username:    input.Username,
		Role:        invite.Role,
	}, nil
}

func (h *Handler) upsertInvitedUser(ctx context.Context, tx pgx.Tx, invite inviteRecord, input normalizedAcceptInvite, passwordHash string) (string, error) {
	var userID string
	var isActive bool
	err := tx.QueryRow(ctx, `
		SELECT id::text, is_active
		FROM users
		WHERE lower(email) = lower($1)
		FOR UPDATE
	`, invite.Email).Scan(&userID, &isActive)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}

	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.QueryRow(ctx, `
			INSERT INTO users (email, username, password_hash, display_name, is_active)
			VALUES ($1, $2, $3, $4, true)
			RETURNING id::text
		`, invite.Email, input.Username, passwordHash, input.DisplayName).Scan(&userID); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return "", errUserExists
			}

			return "", err
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO workspace_members (workspace_id, user_id, role)
			VALUES ($1, $2, $3)
		`, invite.WorkspaceID, userID, invite.Role); err != nil {
			return "", err
		}

		return userID, nil
	}

	var existingRole string
	memberErr := tx.QueryRow(ctx, `
		SELECT role
		FROM workspace_members
		WHERE workspace_id = $1
			AND user_id = $2
		FOR UPDATE
	`, invite.WorkspaceID, userID).Scan(&existingRole)
	if memberErr != nil && !errors.Is(memberErr, pgx.ErrNoRows) {
		return "", memberErr
	}
	if memberErr == nil && isActive {
		return "", errAlreadyMember
	}

	if _, err := tx.Exec(ctx, `
		UPDATE users
		SET username = $2,
			password_hash = $3,
			display_name = $4,
			is_active = true
		WHERE id = $1
	`, userID, input.Username, passwordHash, input.DisplayName); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return "", errUserExists
		}

		return "", err
	}

	if errors.Is(memberErr, pgx.ErrNoRows) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO workspace_members (workspace_id, user_id, role)
			VALUES ($1, $2, $3)
		`, invite.WorkspaceID, userID, invite.Role); err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return "", errAlreadyMember
			}

			return "", err
		}
	} else if _, err := tx.Exec(ctx, `
		UPDATE workspace_members
		SET role = $3
		WHERE workspace_id = $1
			AND user_id = $2
	`, invite.WorkspaceID, userID, invite.Role); err != nil {
		return "", err
	}

	return userID, nil
}

func (h *Handler) inviteByTokenHash(ctx context.Context, db dbtx, tokenHash string, forUpdate bool) (inviteRecord, error) {
	query := `
		SELECT
			ti.id::text,
			ti.workspace_id::text,
			w.name,
			ti.email,
			ti.role,
			ti.created_by::text,
			COALESCE(u.display_name, ''),
			ti.created_at,
			ti.expires_at,
			ti.accepted_at,
			ti.revoked_at,
			''::text,
			NULL::timestamptz,
			NULL::timestamptz
		FROM team_invites ti
		JOIN workspaces w ON w.id = ti.workspace_id
		LEFT JOIN users u ON u.id = ti.created_by
		WHERE ti.token_hash = $1
	`
	if forUpdate {
		query += ` FOR UPDATE OF ti`
	}

	return scanInvite(db.QueryRow(ctx, query, tokenHash))
}

func (h *Handler) inviteByIDForUpdate(ctx context.Context, db dbtx, workspaceID string, inviteID string) (inviteRecord, error) {
	return scanInvite(db.QueryRow(ctx, `
		SELECT
			ti.id::text,
			ti.workspace_id::text,
			w.name,
			ti.email,
			ti.role,
			ti.created_by::text,
			COALESCE(u.display_name, ''),
			ti.created_at,
			ti.expires_at,
			ti.accepted_at,
			ti.revoked_at,
			''::text,
			NULL::timestamptz,
			NULL::timestamptz
		FROM team_invites ti
		JOIN workspaces w ON w.id = ti.workspace_id
		LEFT JOIN users u ON u.id = ti.created_by
		WHERE ti.id = $1
			AND ti.workspace_id = $2
		FOR UPDATE OF ti
	`, inviteID, workspaceID))
}

func normalizeCreateInvite(req createInviteRequest) (normalizedCreateInvite, error) {
	input := normalizedCreateInvite{
		Email: strings.ToLower(strings.TrimSpace(req.Email)),
		Role:  strings.TrimSpace(req.Role),
	}
	if input.Role == "" {
		input.Role = "member"
	}

	if !emailPattern.MatchString(input.Email) {
		return input, errors.New("email is invalid")
	}
	if input.Role != "admin" && input.Role != "member" {
		return input, errors.New("role is invalid")
	}

	return input, nil
}

func normalizeAcceptInvite(req acceptInviteRequest) (normalizedAcceptInvite, error) {
	input := normalizedAcceptInvite{
		Username:    strings.ToLower(strings.TrimSpace(req.Username)),
		DisplayName: strings.Join(strings.Fields(strings.TrimSpace(req.DisplayName)), " "),
		Password:    strings.TrimSpace(req.Password),
	}

	if !usernamePattern.MatchString(input.Username) {
		return input, errors.New("username must be 3-32 characters and contain lowercase letters, numbers, underscores, or hyphens")
	}
	if input.DisplayName == "" {
		return input, errors.New("display_name is required")
	}
	if len([]rune(input.DisplayName)) > 80 {
		return input, errors.New("display_name must be 80 characters or fewer")
	}
	if len(input.Password) < 8 {
		return input, errors.New("password must be at least 8 characters")
	}
	if len(input.Password) > 128 {
		return input, errors.New("password must be 128 characters or fewer")
	}

	return input, nil
}

func normalizeInviteID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New("invite id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("invite id is invalid")
	}

	return id, nil
}

func normalizeInviteToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("invite token is required")
	}

	return token, nil
}

func ensureInvitePending(invite inviteRecord, now time.Time) error {
	if invite.AcceptedAt != nil {
		return errInviteAccepted
	}
	if invite.RevokedAt != nil {
		return errInviteRevoked
	}
	if !now.Before(invite.ExpiresAt) {
		return errInviteExpired
	}

	return nil
}

func inviteStatus(invite inviteRecord, now time.Time) string {
	if invite.AcceptedAt != nil {
		return "accepted"
	}
	if invite.RevokedAt != nil {
		return "revoked"
	}
	if !now.Before(invite.ExpiresAt) {
		return "expired"
	}

	return "pending"
}

func (invite inviteRecord) toResponse(now time.Time) inviteResponse {
	deliveryStatus := invite.EmailDeliveryStatus
	if deliveryStatus == "" {
		deliveryStatus = "not_sent"
	}
	return inviteResponse{
		ID:                  invite.ID,
		WorkspaceID:         invite.WorkspaceID,
		Email:               invite.Email,
		Role:                invite.Role,
		Status:              inviteStatus(invite, now),
		CreatedBy:           invite.CreatedBy,
		CreatedAt:           invite.CreatedAt,
		ExpiresAt:           invite.ExpiresAt,
		AcceptedAt:          invite.AcceptedAt,
		RevokedAt:           invite.RevokedAt,
		EmailDeliveryStatus: deliveryStatus,
		EmailQueuedAt:       invite.EmailQueuedAt,
		EmailSentAt:         invite.EmailSentAt,
	}
}

func newInviteToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashInviteToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanInvite(row rowScanner) (inviteRecord, error) {
	var invite inviteRecord
	var emailDeliveryStatus pgtype.Text
	var emailQueuedAt pgtype.Timestamptz
	var emailSentAt pgtype.Timestamptz
	if err := row.Scan(
		&invite.ID,
		&invite.WorkspaceID,
		&invite.WorkspaceName,
		&invite.Email,
		&invite.Role,
		&invite.CreatedBy,
		&invite.CreatedByName,
		&invite.CreatedAt,
		&invite.ExpiresAt,
		&invite.AcceptedAt,
		&invite.RevokedAt,
		&emailDeliveryStatus,
		&emailQueuedAt,
		&emailSentAt,
	); err != nil {
		return inviteRecord{}, err
	}
	if emailDeliveryStatus.Valid {
		invite.EmailDeliveryStatus = emailDeliveryStatus.String
	}
	if emailQueuedAt.Valid {
		invite.EmailQueuedAt = &emailQueuedAt.Time
	}
	if emailSentAt.Valid {
		invite.EmailSentAt = &emailSentAt.Time
	}

	return invite, nil
}

func (h *Handler) requireAdmin(w http.ResponseWriter, r *http.Request) (auth.CurrentUser, bool) {
	user, err := h.auth.CurrentUser(r)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "session is required")
			return auth.CurrentUser{}, false
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load session")
		return auth.CurrentUser{}, false
	}
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "admin role is required")
		return auth.CurrentUser{}, false
	}

	return user, true
}

func (h *Handler) writePublicInviteError(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "invite_not_found", "invite was not found")
		return
	}
	if errors.Is(err, errInviteExpired) {
		writeError(w, http.StatusBadRequest, "invite_expired", "invite has expired")
		return
	}
	if errors.Is(err, errInviteRevoked) {
		writeError(w, http.StatusBadRequest, "invite_revoked", "invite was revoked")
		return
	}
	if errors.Is(err, errInviteAccepted) {
		writeError(w, http.StatusBadRequest, "invite_already_accepted", "invite was already accepted")
		return
	}

	writeError(w, http.StatusInternalServerError, "internal_error", "could not load invite")
}

func (h *Handler) writeAcceptInviteError(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "invite_not_found", "invite was not found")
		return
	}
	if errors.Is(err, errInviteExpired) {
		writeError(w, http.StatusBadRequest, "invite_expired", "invite has expired")
		return
	}
	if errors.Is(err, errInviteRevoked) {
		writeError(w, http.StatusBadRequest, "invite_revoked", "invite was revoked")
		return
	}
	if errors.Is(err, errInviteAccepted) {
		writeError(w, http.StatusBadRequest, "invite_already_accepted", "invite was already accepted")
		return
	}
	if errors.Is(err, errAlreadyMember) {
		writeError(w, http.StatusConflict, "already_member", "user is already an active workspace member")
		return
	}
	if errors.Is(err, errUserExists) {
		writeError(w, http.StatusConflict, "user_exists", "email or username already exists")
		return
	}

	writeError(w, http.StatusInternalServerError, "internal_error", "could not accept invite")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain a single JSON object")
		}

		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
