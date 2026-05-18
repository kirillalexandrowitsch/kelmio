package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const SessionCookieName = "team_task_tracker_session"

var errInvalidCredentials = errors.New("invalid credentials")

type Handler struct {
	db         *pgxpool.Pool
	sessionTTL time.Duration
}

type loginRequest struct {
	Login    string `json:"login"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	User      userResponse `json:"user"`
	ExpiresAt time.Time    `json:"expires_at"`
}

type meResponse struct {
	User userResponse `json:"user"`
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

func NewHandler(db *pgxpool.Pool, sessionTTL time.Duration) *Handler {
	return &Handler{
		db:         db,
		sessionTTL: sessionTTL,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("POST /api/v1/auth/logout", h.logout)
	mux.HandleFunc("GET /api/v1/auth/me", h.me)
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

	http.SetCookie(w, sessionCookie(token, expiresAt, int(h.sessionTTL.Seconds())))
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

	http.SetCookie(w, expiredSessionCookie())
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
			http.SetCookie(w, expiredSessionCookie())
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

func (req loginRequest) identifier() string {
	if strings.TrimSpace(req.Login) != "" {
		return strings.TrimSpace(req.Login)
	}
	if strings.TrimSpace(req.Email) != "" {
		return strings.TrimSpace(req.Email)
	}
	return strings.TrimSpace(req.Username)
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

func sessionCookie(token string, expiresAt time.Time, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func expiredSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dest)
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
