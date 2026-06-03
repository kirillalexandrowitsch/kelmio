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
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"team-task-tracker/backend/internal/csrf"
)

const SessionCookieName = "team_task_tracker_session"

var errInvalidCredentials = errors.New("invalid credentials")
var ErrUnauthorized = errors.New("unauthorized")

type Handler struct {
	db                  *pgxpool.Pool
	sessionTTL          time.Duration
	sessionCookieSecure bool
	csrfManager         *csrf.Manager
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

type userRecord struct {
	ID           string
	Email        string
	Username     string
	PasswordHash string
	DisplayName  string
	WorkspaceID  string
	Role         string
}

type userResponse struct {
	ID          string            `json:"id"`
	Email       string            `json:"email"`
	Username    string            `json:"username"`
	DisplayName string            `json:"display_name"`
	Workspace   workspaceResponse `json:"workspace"`
}

type workspaceResponse struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

type CurrentUser struct {
	ID          string
	Email       string
	Username    string
	DisplayName string
	WorkspaceID string
	Role        string
}

func NewHandler(
	db *pgxpool.Pool,
	sessionTTL time.Duration,
	sessionCookieSecure bool,
	csrfManager *csrf.Manager,
) *Handler {
	return &Handler{
		db:                  db,
		sessionTTL:          sessionTTL,
		sessionCookieSecure: sessionCookieSecure,
		csrfManager:         csrfManager,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.HandleFunc("GET /api/v1/auth/me", h.me)
	mux.HandleFunc("GET /api/v1/auth/csrf-token", h.csrfToken)
	mux.HandleFunc("PATCH /api/v1/auth/profile", h.updateProfile)
	mux.HandleFunc("PATCH /api/v1/auth/password", h.changePassword)
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
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	identifier := req.identifier()
	if identifier == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "login and password are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	user, err := h.userByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid login or password")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "login failed")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "invalid login or password")
		return
	}

	token, err := newSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create session")
		return
	}

	expiresAt := time.Now().UTC().Add(h.sessionTTL)
	if err := h.createSession(ctx, user.ID, hashToken(token), expiresAt); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create session")
		return
	}

	http.SetCookie(w, sessionCookie(token, expiresAt, int(h.sessionTTL.Seconds()), h.sessionCookieSecure))
	writeJSON(w, http.StatusOK, loginResponse{
		User:      user.toResponse(),
		ExpiresAt: expiresAt,
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
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
			wm.role
		FROM users u
		JOIN workspace_members wm ON wm.user_id = u.id
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
			wm.role
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		JOIN workspace_members wm ON wm.user_id = u.id
		WHERE
			s.token_hash = $1
			AND s.expires_at > now()
			AND u.is_active = true
		ORDER BY wm.joined_at ASC
		LIMIT 1
	`, tokenHash).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.DisplayName,
		&user.WorkspaceID,
		&user.Role,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return userRecord{}, errInvalidCredentials
		}

		return userRecord{}, err
	}

	return user, nil
}

func (h *Handler) createSession(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	if _, err := h.db.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= now()`); err != nil {
		return err
	}

	_, err := h.db.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
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

func (user userRecord) toResponse() userResponse {
	return userResponse{
		ID:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Workspace: workspaceResponse{
			ID:   user.WorkspaceID,
			Role: user.Role,
		},
	}
}

func (user userRecord) toCurrentUser() CurrentUser {
	return CurrentUser{
		ID:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		WorkspaceID: user.WorkspaceID,
		Role:        user.Role,
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
