package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"kelmio/backend/internal/csrf"
	"kelmio/backend/internal/emailoutbox"
	"kelmio/backend/internal/ratelimit"
)

const SessionCookieName = "kelmio_session"
const defaultPasswordResetTTL = 30 * time.Minute

var errInvalidCredentials = errors.New("invalid credentials")
var ErrUnauthorized = errors.New("unauthorized")
var errPasswordResetNotFound = errors.New("password reset token not found")
var errPasswordResetExpired = errors.New("password reset token expired")
var errPasswordResetUsed = errors.New("password reset token used")
var errPasswordResetRevoked = errors.New("password reset token revoked")

type HandlerOption func(*Handler)

type LoginMetrics interface {
	RecordAuthLoginOutcome(outcome string)
}

type Handler struct {
	db                   *pgxpool.Pool
	sessionTTL           time.Duration
	sessionCookieSecure  bool
	csrfManager          *csrf.Manager
	loginLimiter         *ratelimit.Limiter
	passwordResetLimiter *ratelimit.Limiter
	passwordResetTTL     time.Duration
	passwordResetBaseURL string
	metrics              LoginMetrics
}

type loginRequest struct {
	Login    string `json:"login"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type updateProfileRequest struct {
	DisplayName string `json:"display_name"`
}

type setActiveWorkspaceRequest struct {
	WorkspaceID string `json:"workspace_id"`
}

type passwordResetRequest struct {
	Email string `json:"email"`
}

type completePasswordResetRequest struct {
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

type loginResponse struct {
	User      userResponse `json:"user"`
	ExpiresAt time.Time    `json:"expires_at"`
}

type meResponse struct {
	User userResponse `json:"user"`
}

type csrfTokenResponse struct {
	CSRFToken string `json:"csrf_token"`
}

type passwordResetRequestResponse struct {
	Message string `json:"message"`
}

type passwordResetPreviewResponse struct {
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

type userRecord struct {
	ID             string
	Email          string
	Username       string
	PasswordHash   string
	DisplayName    string
	WorkspaceID    string
	Role           string
	OrganizationID string
	IsSiteAdmin    bool
}

type passwordResetUserRecord struct {
	ID          string
	Email       string
	DisplayName string
	WorkspaceID string
}

type passwordResetTokenRecord struct {
	ID          string
	UserID      string
	Email       string
	DisplayName string
	TokenHash   string
	ExpiresAt   time.Time
	UsedAt      *time.Time
	RevokedAt   *time.Time
}

type userResponse struct {
	ID           string               `json:"id"`
	Email        string               `json:"email"`
	Username     string               `json:"username"`
	DisplayName  string               `json:"display_name"`
	IsSiteAdmin  bool                 `json:"is_site_admin"`
	Workspace    workspaceResponse    `json:"workspace"`
	Organization organizationResponse `json:"organization"`
}

type workspaceResponse struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

type organizationResponse struct {
	ID string `json:"id"`
}

type CurrentUser struct {
	ID             string
	Email          string
	Username       string
	DisplayName    string
	WorkspaceID    string
	Role           string
	OrganizationID string
	IsSiteAdmin    bool
}

func NewHandler(
	db *pgxpool.Pool,
	sessionTTL time.Duration,
	sessionCookieSecure bool,
	csrfManager *csrf.Manager,
	loginLimiter *ratelimit.Limiter,
	options ...HandlerOption,
) *Handler {
	handler := &Handler{
		db:                  db,
		sessionTTL:          sessionTTL,
		sessionCookieSecure: sessionCookieSecure,
		csrfManager:         csrfManager,
		loginLimiter:        loginLimiter,
		passwordResetTTL:    defaultPasswordResetTTL,
	}
	for _, option := range options {
		if option != nil {
			option(handler)
		}
	}
	if handler.passwordResetTTL <= 0 {
		handler.passwordResetTTL = defaultPasswordResetTTL
	}
	return handler
}

func WithPasswordResetTTL(ttl time.Duration) HandlerOption {
	return func(handler *Handler) {
		handler.passwordResetTTL = ttl
	}
}

func WithPasswordResetLimiter(limiter *ratelimit.Limiter) HandlerOption {
	return func(handler *Handler) {
		handler.passwordResetLimiter = limiter
	}
}

func WithPasswordResetBaseURL(baseURL string) HandlerOption {
	return func(handler *Handler) {
		handler.passwordResetBaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	}
}

func WithMetrics(metrics LoginMetrics) HandlerOption {
	return func(handler *Handler) {
		handler.metrics = metrics
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.HandleFunc("GET /api/v1/auth/me", h.me)
	mux.HandleFunc("GET /api/v1/auth/csrf-token", h.csrfToken)
	mux.HandleFunc("PATCH /api/v1/auth/profile", h.updateProfile)
	mux.HandleFunc("PATCH /api/v1/auth/password", h.changePassword)
	mux.HandleFunc("POST /api/v1/session/active-workspace", h.setActiveWorkspace)
	mux.HandleFunc("POST /api/v1/auth/password-reset/request", h.requestPasswordReset)
	mux.HandleFunc("GET /api/v1/auth/password-reset/{token}", h.previewPasswordReset)
	mux.HandleFunc("POST /api/v1/auth/password-reset/{token}/complete", h.completePasswordReset)
}

func (h *Handler) CurrentUser(r *http.Request) (CurrentUser, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return CurrentUser{}, ErrUnauthorized
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := h.userBySession(ctx, hashToken(cookie.Value))
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			return CurrentUser{}, ErrUnauthorized
		}

		return CurrentUser{}, err
	}

	return user.toCurrentUser(), nil
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		h.recordLoginOutcome("invalid")
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	identifier := req.identifier()
	if identifier == "" || req.Password == "" {
		h.recordLoginOutcome("invalid")
		writeError(w, http.StatusBadRequest, "invalid_request", "login and password are required")
		return
	}

	rateLimitKey := normalizeLoginRateLimitKey(identifier)
	if h.loginLimiter != nil {
		result := h.loginLimiter.Allow(rateLimitKey)
		if !result.Allowed {
			h.recordLoginOutcome("rate_limited")
			w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(result.RetryAfter)))
			writeError(w, http.StatusTooManyRequests, "rate_limited", "too many login attempts, try again later")
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_ = h.cleanupExpiredSessions(ctx)

	user, err := h.userByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			h.recordLoginOutcome("invalid")
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid login or password")
			return
		}

		h.recordLoginOutcome("error")
		writeError(w, http.StatusInternalServerError, "internal_error", "login failed")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.recordLoginOutcome("invalid")
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid login or password")
		return
	}

	token, err := newSessionToken()
	if err != nil {
		h.recordLoginOutcome("error")
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create session")
		return
	}

	expiresAt := time.Now().UTC().Add(h.sessionTTL)
	if err := h.createSession(ctx, user.ID, hashToken(token), expiresAt); err != nil {
		h.recordLoginOutcome("error")
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create session")
		return
	}

	if h.loginLimiter != nil {
		h.loginLimiter.Reset(rateLimitKey)
	}
	h.recordLoginOutcome("success")

	http.SetCookie(w, sessionCookie(token, expiresAt, int(h.sessionTTL.Seconds()), h.sessionCookieSecure))
	writeJSON(w, http.StatusOK, loginResponse{
		User:      user.toResponse(),
		ExpiresAt: expiresAt,
	})
}

func (h *Handler) recordLoginOutcome(outcome string) {
	if h.metrics != nil {
		h.metrics.RecordAuthLoginOutcome(outcome)
	}
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_ = h.cleanupExpiredSessions(ctx)
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		_ = h.deleteSession(ctx, hashToken(cookie.Value))
	}

	http.SetCookie(w, expiredSessionCookie(h.sessionCookieSecure))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "session is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_ = h.cleanupExpiredSessions(ctx)

	user, err := h.userBySession(ctx, hashToken(cookie.Value))
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			http.SetCookie(w, expiredSessionCookie(h.sessionCookieSecure))
			writeError(w, http.StatusUnauthorized, "unauthorized", "session is invalid")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load session")
		return
	}

	writeJSON(w, http.StatusOK, meResponse{
		User: user.toResponse(),
	})
}

func (h *Handler) csrfToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "session is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := h.userBySession(ctx, hashToken(cookie.Value)); err != nil {
		if errors.Is(err, errInvalidCredentials) {
			http.SetCookie(w, expiredSessionCookie(h.sessionCookieSecure))
			writeError(w, http.StatusUnauthorized, "unauthorized", "session is invalid")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load session")
		return
	}

	if h.csrfManager == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "csrf manager is not configured")
		return
	}

	token, err := h.csrfManager.Generate(cookie.Value)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create csrf token")
		return
	}

	writeJSON(w, http.StatusOK, csrfTokenResponse{
		CSRFToken: token,
	})
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "session is required")
		return
	}

	var req updateProfileRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	displayName, err := normalizeDisplayName(req.DisplayName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tokenHash := hashToken(cookie.Value)
	currentUser, err := h.userBySession(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			http.SetCookie(w, expiredSessionCookie(h.sessionCookieSecure))
			writeError(w, http.StatusUnauthorized, "unauthorized", "session is invalid")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load session")
		return
	}

	user, err := h.updateOwnProfile(ctx, currentUser.ID, displayName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update profile")
		return
	}

	writeJSON(w, http.StatusOK, meResponse{
		User: user.toResponse(),
	})
}

func (h *Handler) setActiveWorkspace(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "session is required")
		return
	}

	var req setActiveWorkspaceRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	workspaceID, ok := normalizeWorkspaceID(req.WorkspaceID)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_request", "workspace_id is invalid")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tokenHash := hashToken(cookie.Value)
	currentUser, err := h.userBySession(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			http.SetCookie(w, expiredSessionCookie(h.sessionCookieSecure))
			writeError(w, http.StatusUnauthorized, "unauthorized", "session is invalid")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load session")
		return
	}

	var canActivate bool
	if err := h.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM workspace_members wm
			JOIN workspaces w ON w.id = wm.workspace_id
			WHERE wm.workspace_id = $1::uuid
				AND wm.user_id = $2
				AND w.status = 'active'
		)
	`, workspaceID, currentUser.ID).Scan(&canActivate); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load workspace")
		return
	}
	if !canActivate {
		writeError(w, http.StatusForbidden, "forbidden", "workspace is not accessible")
		return
	}

	if _, err := h.db.Exec(ctx, `
		UPDATE sessions SET active_workspace_id = $1::uuid WHERE token_hash = $2
	`, workspaceID, tokenHash); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not switch workspace")
		return
	}

	user, err := h.userBySession(ctx, tokenHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load session")
		return
	}

	writeJSON(w, http.StatusOK, meResponse{
		User: user.toResponse(),
	})
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "session is required")
		return
	}

	var req changePasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	currentPassword, newPassword, err := normalizeChangePassword(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tokenHash := hashToken(cookie.Value)
	user, err := h.userBySession(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			http.SetCookie(w, expiredSessionCookie(h.sessionCookieSecure))
			writeError(w, http.StatusUnauthorized, "unauthorized", "session is invalid")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load session")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_current_password", "current password is invalid")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update password")
		return
	}

	if err := h.updateOwnPassword(ctx, user.ID, tokenHash, string(passwordHash)); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update password")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req passwordResetRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	rateLimitKey := passwordResetRateLimitKey(req.Email, r.RemoteAddr)
	if h.passwordResetLimiter != nil {
		result := h.passwordResetLimiter.Allow(rateLimitKey)
		if !result.Allowed {
			w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(result.RetryAfter)))
			writeError(w, http.StatusTooManyRequests, "rate_limited", "too many password reset requests, try again later")
			return
		}
	}

	email, err := normalizePasswordResetEmail(req.Email)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.createPasswordResetRequest(ctx, email, r); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not request password reset")
		return
	}

	writeJSON(w, http.StatusAccepted, passwordResetRequestResponse{
		Message: "If an active account exists, password reset instructions will be sent.",
	})
}

func (h *Handler) previewPasswordReset(w http.ResponseWriter, r *http.Request) {
	token, err := normalizePasswordResetToken(r.PathValue("token"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	record, err := h.passwordResetTokenByHash(ctx, hashToken(token))
	if err != nil {
		writePasswordResetTokenError(w, err)
		return
	}
	if err := validatePasswordResetTokenState(record, time.Now().UTC()); err != nil {
		writePasswordResetTokenError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, passwordResetPreviewResponse{
		Email:     record.Email,
		ExpiresAt: record.ExpiresAt,
	})
}

func (h *Handler) completePasswordReset(w http.ResponseWriter, r *http.Request) {
	token, err := normalizePasswordResetToken(r.PathValue("token"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req completePasswordResetRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	password, err := normalizeCompletePasswordReset(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tokenHash := hashToken(token)
	record, err := h.passwordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		writePasswordResetTokenError(w, err)
		return
	}
	if err := validatePasswordResetTokenState(record, time.Now().UTC()); err != nil {
		writePasswordResetTokenError(w, err)
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not reset password")
		return
	}

	if err := h.completePasswordResetTransaction(ctx, tokenHash, string(passwordHash)); err != nil {
		writePasswordResetTokenError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) userByIdentifier(ctx context.Context, identifier string) (userRecord, error) {
	var user userRecord
	if err := h.db.QueryRow(ctx, `
		SELECT
			u.id::text,
			u.email,
			u.username,
			u.password_hash,
			u.display_name,
			wm.workspace_id::text,
			wm.role,
			COALESCE(w.organization_id::text, ''),
			u.is_site_admin
		FROM users u
		JOIN workspace_members wm ON wm.user_id = u.id
		JOIN workspaces w ON w.id = wm.workspace_id
		WHERE
			u.is_active = true
			AND (
				lower(u.email) = lower($1)
				OR lower(u.username) = lower($1)
			)
		ORDER BY wm.joined_at ASC
		LIMIT 1
	`, identifier).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.DisplayName,
		&user.WorkspaceID,
		&user.Role,
		&user.OrganizationID,
		&user.IsSiteAdmin,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return userRecord{}, errInvalidCredentials
		}

		return userRecord{}, err
	}

	return user, nil
}

func (h *Handler) userBySession(ctx context.Context, tokenHash string) (userRecord, error) {
	var user userRecord
	if err := h.db.QueryRow(ctx, `
		SELECT
			u.id::text,
			u.email,
			u.username,
			u.password_hash,
			u.display_name,
			wm.workspace_id::text,
			wm.role,
			COALESCE(w.organization_id::text, ''),
			u.is_site_admin
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		JOIN workspace_members wm ON wm.user_id = u.id
		JOIN workspaces w ON w.id = wm.workspace_id
		WHERE
			s.token_hash = $1
			AND s.expires_at > now()
			AND u.is_active = true
		ORDER BY
			(wm.workspace_id = s.active_workspace_id) DESC,
			wm.joined_at ASC
		LIMIT 1
	`, tokenHash).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.DisplayName,
		&user.WorkspaceID,
		&user.Role,
		&user.OrganizationID,
		&user.IsSiteAdmin,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return userRecord{}, errInvalidCredentials
		}

		return userRecord{}, err
	}

	return user, nil
}

func (h *Handler) createSession(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	_, err := h.db.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	return err
}

func (h *Handler) cleanupExpiredSessions(ctx context.Context) error {
	_, err := h.db.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= now()`)
	return err
}

func (h *Handler) deleteSession(ctx context.Context, tokenHash string) error {
	_, err := h.db.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return err
}

func (h *Handler) updateOwnPassword(ctx context.Context, userID string, currentTokenHash string, passwordHash string) error {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `
		UPDATE users
		SET password_hash = $2
		WHERE id = $1
	`, userID, passwordHash); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM sessions
		WHERE user_id = $1
			AND token_hash <> $2
	`, userID, currentTokenHash); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (h *Handler) createPasswordResetRequest(ctx context.Context, email string, r *http.Request) error {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	user, err := h.passwordResetUserByEmail(ctx, tx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return tx.Commit(ctx)
		}
		return err
	}

	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE password_reset_tokens
		SET revoked_at = $2
		WHERE user_id = $1
			AND used_at IS NULL
			AND revoked_at IS NULL
	`, user.ID, now); err != nil {
		return err
	}

	token, err := newPasswordResetToken()
	if err != nil {
		return err
	}
	tokenHash := hashToken(token)
	expiresAt := now.Add(h.passwordResetTTL)
	var resetID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO password_reset_tokens (
			user_id,
			token_hash,
			request_ip_hash,
			request_user_agent,
			created_at,
			expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text
	`, user.ID, tokenHash, requestIPHash(r), requestUserAgent(r), now, expiresAt).Scan(&resetID); err != nil {
		return err
	}

	resetURLPath := "/reset-password?token=" + token
	resetURL := resetURLPath
	if h.passwordResetBaseURL != "" {
		resetURL = h.passwordResetBaseURL + resetURLPath
	}
	if _, err := emailoutbox.Enqueue(ctx, tx, emailoutbox.EnqueueInput{
		WorkspaceID:    &user.WorkspaceID,
		EmailType:      emailoutbox.TypePasswordReset,
		RecipientEmail: user.Email,
		TemplateData: map[string]any{
			"display_name":   user.DisplayName,
			"reset_url":      resetURL,
			"reset_url_path": resetURLPath,
			"expires_at":     expiresAt.Format(time.RFC3339),
		},
		DeduplicationKey: "password_reset:" + resetID,
		NextAttemptAt:    now,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (h *Handler) passwordResetUserByEmail(ctx context.Context, tx pgx.Tx, email string) (passwordResetUserRecord, error) {
	var user passwordResetUserRecord
	if err := tx.QueryRow(ctx, `
		SELECT
			u.id::text,
			u.email,
			u.display_name,
			wm.workspace_id::text
		FROM users u
		JOIN workspace_members wm ON wm.user_id = u.id
		WHERE
			u.is_active = true
			AND lower(u.email) = lower($1)
		ORDER BY wm.joined_at ASC
		LIMIT 1
	`, email).Scan(&user.ID, &user.Email, &user.DisplayName, &user.WorkspaceID); err != nil {
		return passwordResetUserRecord{}, err
	}
	return user, nil
}

func (h *Handler) passwordResetTokenByHash(ctx context.Context, tokenHash string) (passwordResetTokenRecord, error) {
	return scanPasswordResetToken(h.db.QueryRow(ctx, `
		SELECT
			prt.id::text,
			prt.user_id::text,
			u.email,
			u.display_name,
			prt.token_hash,
			prt.expires_at,
			prt.used_at,
			prt.revoked_at
		FROM password_reset_tokens prt
		JOIN users u ON u.id = prt.user_id
		WHERE prt.token_hash = $1
			AND u.is_active = true
	`, tokenHash))
}

func (h *Handler) completePasswordResetTransaction(ctx context.Context, tokenHash string, passwordHash string) error {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	record, err := scanPasswordResetToken(tx.QueryRow(ctx, `
		SELECT
			prt.id::text,
			prt.user_id::text,
			u.email,
			u.display_name,
			prt.token_hash,
			prt.expires_at,
			prt.used_at,
			prt.revoked_at
		FROM password_reset_tokens prt
		JOIN users u ON u.id = prt.user_id
		WHERE prt.token_hash = $1
			AND u.is_active = true
		FOR UPDATE OF prt
	`, tokenHash))
	if err != nil {
		return err
	}
	if err := validatePasswordResetTokenState(record, time.Now().UTC()); err != nil {
		return err
	}

	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE users
		SET password_hash = $2
		WHERE id = $1
	`, record.UserID, passwordHash); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE password_reset_tokens
		SET used_at = $2
		WHERE id = $1
	`, record.ID, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM sessions
		WHERE user_id = $1
	`, record.UserID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (h *Handler) updateOwnProfile(ctx context.Context, userID string, displayName string) (userRecord, error) {
	var user userRecord
	if err := h.db.QueryRow(ctx, `
		WITH updated_user AS (
			UPDATE users
			SET display_name = $2
			WHERE id = $1
			RETURNING id, email, username, password_hash, display_name
		)
		SELECT
			u.id::text,
			u.email,
			u.username,
			u.password_hash,
			u.display_name,
			wm.workspace_id::text,
			wm.role
		FROM updated_user u
		JOIN workspace_members wm ON wm.user_id = u.id
		ORDER BY wm.joined_at ASC
		LIMIT 1
	`, userID, displayName).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.DisplayName,
		&user.WorkspaceID,
		&user.Role,
	); err != nil {
		return userRecord{}, err
	}

	return user, nil
}

func (req loginRequest) identifier() string {
	if strings.TrimSpace(req.Login) != "" {
		return strings.TrimSpace(req.Login)
	}
	if strings.TrimSpace(req.Email) != "" {
		return strings.TrimSpace(req.Email)
	}
	return strings.TrimSpace(req.Username)
}

func normalizeLoginRateLimitKey(identifier string) string {
	return strings.ToLower(strings.TrimSpace(identifier))
}

func retryAfterSeconds(duration time.Duration) int {
	if duration <= 0 {
		return 1
	}

	seconds := int(duration / time.Second)
	if duration%time.Second != 0 {
		seconds++
	}
	if seconds < 1 {
		return 1
	}
	return seconds
}

func normalizeDisplayName(displayName string) (string, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return "", errors.New("display_name is required")
	}
	if len([]rune(displayName)) > 80 {
		return "", errors.New("display_name must be 80 characters or fewer")
	}

	return displayName, nil
}

func normalizeWorkspaceID(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	var id pgtype.UUID
	if err := id.Scan(value); err != nil {
		return "", false
	}
	return value, true
}

func normalizeChangePassword(req changePasswordRequest) (string, string, error) {
	currentPassword := strings.TrimSpace(req.CurrentPassword)
	newPassword := strings.TrimSpace(req.NewPassword)

	if currentPassword == "" {
		return "", "", errors.New("current_password is required")
	}
	if len(currentPassword) > 128 {
		return "", "", errors.New("current_password must be 128 characters or fewer")
	}
	if len(newPassword) < 8 {
		return "", "", errors.New("new_password must be at least 8 characters")
	}
	if len(newPassword) > 128 {
		return "", "", errors.New("new_password must be 128 characters or fewer")
	}
	if newPassword == currentPassword {
		return "", "", errors.New("new_password must be different from current_password")
	}

	return currentPassword, newPassword, nil
}

func normalizePasswordResetEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", errors.New("email is required")
	}
	if len(email) > 320 {
		return "", errors.New("email must be 320 characters or fewer")
	}
	if !strings.Contains(email, "@") {
		return "", errors.New("email is invalid")
	}
	return email, nil
}

func normalizePasswordResetToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("password reset token is required")
	}
	if len(token) > 256 {
		return "", errors.New("password reset token is invalid")
	}
	return token, nil
}

func normalizeCompletePasswordReset(req completePasswordResetRequest) (string, error) {
	password := strings.TrimSpace(req.Password)
	confirmPassword := strings.TrimSpace(req.ConfirmPassword)
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return "", errors.New("password must be 128 characters or fewer")
	}
	if confirmPassword == "" {
		return "", errors.New("confirm_password is required")
	}
	if password != confirmPassword {
		return "", errors.New("confirm_password must match password")
	}
	return password, nil
}

func validatePasswordResetTokenState(record passwordResetTokenRecord, now time.Time) error {
	if record.RevokedAt != nil {
		return errPasswordResetRevoked
	}
	if record.UsedAt != nil {
		return errPasswordResetUsed
	}
	if !now.Before(record.ExpiresAt) {
		return errPasswordResetExpired
	}
	return nil
}

func writePasswordResetTokenError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, errPasswordResetNotFound):
		writeError(w, http.StatusNotFound, "password_reset_not_found", "password reset token was not found")
	case errors.Is(err, errPasswordResetExpired):
		writeError(w, http.StatusBadRequest, "password_reset_expired", "password reset token has expired")
	case errors.Is(err, errPasswordResetUsed):
		writeError(w, http.StatusBadRequest, "password_reset_used", "password reset token was already used")
	case errors.Is(err, errPasswordResetRevoked):
		writeError(w, http.StatusBadRequest, "password_reset_revoked", "password reset token was revoked")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "password reset failed")
	}
}

func passwordResetRateLimitKey(email string, remoteAddr string) string {
	if normalized, err := normalizePasswordResetEmail(email); err == nil {
		return "email:" + normalized
	}
	return "ip:" + requestIPHashFromRemoteAddr(remoteAddr)
}

func requestIPHash(r *http.Request) string {
	if r == nil {
		return ""
	}
	return requestIPHashFromRemoteAddr(r.RemoteAddr)
}

func requestIPHashFromRemoteAddr(remoteAddr string) string {
	remoteAddr = strings.TrimSpace(remoteAddr)
	if remoteAddr == "" {
		return ""
	}
	return hashToken(remoteAddr)
}

func requestUserAgent(r *http.Request) string {
	if r == nil {
		return ""
	}
	userAgent := strings.TrimSpace(r.UserAgent())
	if len(userAgent) > 240 {
		return userAgent[:240]
	}
	return userAgent
}

func newPasswordResetToken() (string, error) {
	return newSessionToken()
}

func scanPasswordResetToken(row interface{ Scan(...any) error }) (passwordResetTokenRecord, error) {
	var record passwordResetTokenRecord
	var usedAt pgtype.Timestamptz
	var revokedAt pgtype.Timestamptz
	if err := row.Scan(
		&record.ID,
		&record.UserID,
		&record.Email,
		&record.DisplayName,
		&record.TokenHash,
		&record.ExpiresAt,
		&usedAt,
		&revokedAt,
	); err != nil {
		return passwordResetTokenRecord{}, err
	}
	if usedAt.Valid {
		record.UsedAt = &usedAt.Time
	}
	if revokedAt.Valid {
		record.RevokedAt = &revokedAt.Time
	}
	return record, nil
}

func (user userRecord) toResponse() userResponse {
	return userResponse{
		ID:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		IsSiteAdmin: user.IsSiteAdmin,
		Workspace: workspaceResponse{
			ID:   user.WorkspaceID,
			Role: user.Role,
		},
		Organization: organizationResponse{
			ID: user.OrganizationID,
		},
	}
}

func (user userRecord) toCurrentUser() CurrentUser {
	return CurrentUser{
		ID:             user.ID,
		Email:          user.Email,
		Username:       user.Username,
		DisplayName:    user.DisplayName,
		WorkspaceID:    user.WorkspaceID,
		Role:           user.Role,
		OrganizationID: user.OrganizationID,
		IsSiteAdmin:    user.IsSiteAdmin,
	}
}

func newSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func sessionCookie(token string, expiresAt time.Time, maxAge int, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func expiredSessionCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
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
