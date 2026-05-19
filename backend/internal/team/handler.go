package team

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"team-task-tracker/backend/internal/auth"
)

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{2,31}$`)

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

type createMemberRequest struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Role        string `json:"role"`
}

type memberResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	IsActive    bool      `json:"is_active"`
	JoinedAt    time.Time `json:"joined_at"`
}

type listMembersResponse struct {
	Members []memberResponse `json:"members"`
}

type normalizedCreateMember struct {
	Email       string
	Username    string
	DisplayName string
	Password    string
	Role        string
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{
		db:   db,
		auth: authHandler,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/team/members", h.listMembers)
	mux.HandleFunc("POST /api/v1/team/members", h.createMember)
}

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT
			u.id::text,
			u.email,
			u.username,
			u.display_name,
			wm.role,
			u.is_active,
			wm.joined_at
		FROM workspace_members wm
		JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = $1
		ORDER BY wm.joined_at ASC
	`, user.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list team members")
		return
	}
	defer rows.Close()

	members := make([]memberResponse, 0)
	for rows.Next() {
		var member memberResponse
		if err := rows.Scan(
			&member.ID,
			&member.Email,
			&member.Username,
			&member.DisplayName,
			&member.Role,
			&member.IsActive,
			&member.JoinedAt,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not read team member")
			return
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list team members")
		return
	}

	writeJSON(w, http.StatusOK, listMembersResponse{Members: members})
}

func (h *Handler) createMember(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "admin role is required")
		return
	}

	var req createMemberRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input, err := normalizeCreateMember(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	member, err := h.createWorkspaceMember(ctx, user.WorkspaceID, input)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "user_exists", "email or username already exists")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not create team member")
		return
	}

	writeJSON(w, http.StatusCreated, member)
}

func (h *Handler) createWorkspaceMember(ctx context.Context, workspaceID string, input normalizedCreateMember) (memberResponse, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return memberResponse{}, err
	}

	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return memberResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var member memberResponse
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, display_name, is_active)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id::text, email, username, display_name, is_active
	`, input.Email, input.Username, string(passwordHash), input.DisplayName).Scan(
		&member.ID,
		&member.Email,
		&member.Username,
		&member.DisplayName,
		&member.IsActive,
	); err != nil {
		return memberResponse{}, err
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, $3)
		RETURNING role, joined_at
	`, workspaceID, member.ID, input.Role).Scan(&member.Role, &member.JoinedAt); err != nil {
		return memberResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return memberResponse{}, err
	}

	return member, nil
}

func (h *Handler) requireUser(w http.ResponseWriter, r *http.Request) (auth.CurrentUser, bool) {
	user, err := h.auth.CurrentUser(r)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "session is required")
			return auth.CurrentUser{}, false
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load session")
		return auth.CurrentUser{}, false
	}

	return user, true
}

func normalizeCreateMember(req createMemberRequest) (normalizedCreateMember, error) {
	input := normalizedCreateMember{
		Email:       strings.ToLower(strings.TrimSpace(req.Email)),
		Username:    strings.ToLower(strings.TrimSpace(req.Username)),
		DisplayName: strings.TrimSpace(req.DisplayName),
		Password:    strings.TrimSpace(req.Password),
		Role:        strings.TrimSpace(req.Role),
	}
	if input.Role == "" {
		input.Role = "member"
	}

	if !emailPattern.MatchString(input.Email) {
		return input, errors.New("email is invalid")
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
	if input.Role != "admin" && input.Role != "member" {
		return input, errors.New("role is invalid")
	}

	return input, nil
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
