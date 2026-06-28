package workspaces

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kelmio/backend/internal/auth"
)

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

type workspaceResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Status   string `json:"status"`
	Role     string `json:"role"`
	IsActive bool   `json:"is_active"`
}

type listWorkspacesResponse struct {
	Workspaces []workspaceResponse `json:"workspaces"`
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{db: db, auth: authHandler}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/workspaces", h.list)
}

// list returns the active workspaces the current user belongs to within their
// active organization, flagging the one resolved as the active workspace. The
// switcher only surfaces active workspaces, so archived ones are excluded.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT w.id::text, w.name, COALESCE(w.slug, ''), w.status, wm.role
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id = w.id AND wm.user_id = $1
		WHERE w.status = 'active'
			AND ($2 = '' OR w.organization_id = $2::uuid)
		ORDER BY w.name ASC
	`, user.ID, user.OrganizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list workspaces")
		return
	}
	defer rows.Close()

	workspaces := make([]workspaceResponse, 0)
	for rows.Next() {
		var workspace workspaceResponse
		if err := rows.Scan(
			&workspace.ID,
			&workspace.Name,
			&workspace.Slug,
			&workspace.Status,
			&workspace.Role,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not list workspaces")
			return
		}
		workspace.IsActive = workspace.ID == user.WorkspaceID
		workspaces = append(workspaces, workspace)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list workspaces")
		return
	}

	writeJSON(w, http.StatusOK, listWorkspacesResponse{Workspaces: workspaces})
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
