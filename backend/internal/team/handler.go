package team

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

var errInvalidMemberUpdate = errors.New("invalid member update")
var errLastActiveAdmin = errors.New("last active admin")

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

type updateMemberRequest struct {
	Role     string `json:"role"`
	IsActive *bool  `json:"is_active"`
}

type resetMemberPasswordRequest struct {
	Password string `json:"password"`
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

type normalizedUpdateMember struct {
	Role        string
	IsActive    *bool
	HasChanges  bool
	RequestedID string
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
	mux.HandleFunc("PATCH /api/v1/team/members/{id}", h.updateMember)
	mux.HandleFunc("PATCH /api/v1/team/members/{id}/password", h.resetMemberPassword)
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

func (h *Handler) updateMember(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "admin role is required")
		return
	}

	memberID, err := normalizeMemberID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req updateMemberRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input, err := normalizeUpdateMember(memberID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	member, err := h.updateWorkspaceMember(ctx, user, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "member_not_found", "team member was not found")
			return
		}
		if errors.Is(err, errInvalidMemberUpdate) {
			writeError(w, http.StatusBadRequest, "invalid_member_update", "you cannot update your own membership")
			return
		}
		if errors.Is(err, errLastActiveAdmin) {
			writeError(w, http.StatusBadRequest, "last_active_admin", "workspace must keep at least one active admin")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not update team member")
		return
	}

	writeJSON(w, http.StatusOK, member)
}

func (h *Handler) resetMemberPassword(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "admin role is required")
		return
	}

	memberID, err := normalizeMemberID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req resetMemberPasswordRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	password, err := normalizeMemberPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.resetWorkspaceMemberPassword(ctx, user, memberID, password); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "member_not_found", "team member was not found")
			return
		}
		if errors.Is(err, errInvalidMemberUpdate) {
			writeError(w, http.StatusBadRequest, "invalid_member_update", "you cannot reset your own password here")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not reset team member password")
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func (h *Handler) updateWorkspaceMember(ctx context.Context, actor auth.CurrentUser, input normalizedUpdateMember) (memberResponse, error) {
	if input.RequestedID == actor.ID {
		return memberResponse{}, errInvalidMemberUpdate
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
			AND wm.user_id = $2
		FOR UPDATE OF wm, u
	`, actor.WorkspaceID, input.RequestedID).Scan(
		&member.ID,
		&member.Email,
		&member.Username,
		&member.DisplayName,
		&member.Role,
		&member.IsActive,
		&member.JoinedAt,
	); err != nil {
		return memberResponse{}, err
	}

	nextRole := member.Role
	if input.Role != "" {
		nextRole = input.Role
	}
	nextIsActive := member.IsActive
	if input.IsActive != nil {
		nextIsActive = *input.IsActive
	}

	if member.Role == "admin" && member.IsActive && (nextRole != "admin" || !nextIsActive) {
		var otherActiveAdmins int
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM workspace_members wm
			JOIN users u ON u.id = wm.user_id
			WHERE wm.workspace_id = $1
				AND wm.user_id <> $2
				AND wm.role = 'admin'
				AND u.is_active = true
		`, actor.WorkspaceID, member.ID).Scan(&otherActiveAdmins); err != nil {
			return memberResponse{}, err
		}
		if otherActiveAdmins == 0 {
			return memberResponse{}, errLastActiveAdmin
		}
	}

	if err := tx.QueryRow(ctx, `
		UPDATE workspace_members
		SET role = $3
		WHERE workspace_id = $1
			AND user_id = $2
		RETURNING role
	`, actor.WorkspaceID, member.ID, nextRole).Scan(&member.Role); err != nil {
		return memberResponse{}, err
	}

	if err := tx.QueryRow(ctx, `
		UPDATE users
		SET is_active = $2
		WHERE id = $1
		RETURNING is_active
	`, member.ID, nextIsActive).Scan(&member.IsActive); err != nil {
		return memberResponse{}, err
	}

	if !member.IsActive {
		if _, err := tx.Exec(ctx, `
			DELETE FROM sessions
			WHERE user_id = $1
		`, member.ID); err != nil {
			return memberResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return memberResponse{}, err
	}

	return member, nil
}

func (h *Handler) resetWorkspaceMemberPassword(ctx context.Context, actor auth.CurrentUser, memberID string, password string) error {
	if memberID == actor.ID {
		return errInvalidMemberUpdate
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	commandTag, err := tx.Exec(ctx, `
		UPDATE users u
		SET password_hash = $3
		FROM workspace_members wm
		WHERE wm.user_id = u.id
			AND wm.workspace_id = $1
			AND wm.user_id = $2
	`, actor.WorkspaceID, memberID, string(passwordHash))
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM sessions
		WHERE user_id = $1
	`, memberID); err != nil {
		return err
	}

	return tx.Commit(ctx)
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
	if _, err := normalizeMemberPassword(input.Password); err != nil {
		return input, err
	}
	if input.Role != "admin" && input.Role != "member" {
		return input, errors.New("role is invalid")
	}

	return input, nil
}

func normalizeMemberID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New("member id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("member id is invalid")
	}

	return id, nil
}

func normalizeUpdateMember(memberID string, req updateMemberRequest) (normalizedUpdateMember, error) {
	input := normalizedUpdateMember{
		RequestedID: memberID,
		Role:        strings.TrimSpace(req.Role),
		IsActive:    req.IsActive,
	}

	if input.Role != "" && input.Role != "admin" && input.Role != "member" {
		return input, errors.New("role is invalid")
	}
	if input.Role != "" || input.IsActive != nil {
		input.HasChanges = true
	}
	if !input.HasChanges {
		return input, errors.New("role or is_active is required")
	}

	return input, nil
}

func normalizeMemberPassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return "", errors.New("password must be 128 characters or fewer")
	}

	return password, nil
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
