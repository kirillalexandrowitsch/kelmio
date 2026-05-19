package team

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"team-task-tracker/backend/internal/auth"
)

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
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

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{
		db:   db,
		auth: authHandler,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/team/members", h.listMembers)
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
