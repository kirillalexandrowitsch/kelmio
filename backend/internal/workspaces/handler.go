package workspaces

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kelmio/backend/internal/auth"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

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

type createWorkspaceRequest struct {
	Name string `json:"name"`
}

type updateWorkspaceRequest struct {
	Name   *string `json:"name"`
	Status *string `json:"status"`
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{db: db, auth: authHandler}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/workspaces", h.list)
	mux.HandleFunc("POST /api/v1/workspaces", h.create)
	mux.HandleFunc("PATCH /api/v1/workspaces/{id}", h.update)
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

// create adds a workspace to the current user's active organization and makes
// the creator its first administrator so it can be switched into immediately.
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if user.OrganizationID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "no active organization")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !h.authorizeOrganizationAdmin(ctx, w, user, user.OrganizationID) {
		return
	}

	var req createWorkspaceRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	name, slug, err := normalizeName(req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	workspace, err := h.createWorkspace(ctx, user.OrganizationID, user.ID, name, slug)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create workspace")
		return
	}
	writeJSON(w, http.StatusCreated, workspace)
}

// update renames or archives a workspace. Only a site admin or an administrator
// of the workspace's organization may do so.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	workspaceID, err := normalizeID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var organizationID, currentName, currentStatus string
	if err := h.db.QueryRow(ctx, `
		SELECT COALESCE(organization_id::text, ''), name, status
		FROM workspaces WHERE id = $1
	`, workspaceID).Scan(&organizationID, &currentName, &currentStatus); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "workspace_not_found", "workspace was not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load workspace")
		return
	}

	if !h.authorizeOrganizationAdmin(ctx, w, user, organizationID) {
		return
	}

	var req updateWorkspaceRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	name := currentName
	if req.Name != nil {
		normalized, _, err := normalizeName(*req.Name)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		name = normalized
	}
	status := currentStatus
	if req.Status != nil {
		status = strings.ToLower(strings.TrimSpace(*req.Status))
		if status != "active" && status != "archived" {
			writeError(w, http.StatusBadRequest, "invalid_request", "status must be active or archived")
			return
		}
	}

	workspace, err := h.updateWorkspace(ctx, workspaceID, user.ID, name, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update workspace")
		return
	}
	workspace.IsActive = workspace.ID == user.WorkspaceID
	writeJSON(w, http.StatusOK, workspace)
}

func (h *Handler) authorizeOrganizationAdmin(ctx context.Context, w http.ResponseWriter, user auth.CurrentUser, organizationID string) bool {
	if user.IsSiteAdmin {
		return true
	}
	if organizationID == "" {
		writeError(w, http.StatusForbidden, "forbidden", "organization administrator role is required")
		return false
	}
	var isOrgAdmin bool
	if err := h.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM organization_members
			WHERE organization_id = $1 AND user_id = $2 AND role = 'org_admin'
		)
	`, organizationID, user.ID).Scan(&isOrgAdmin); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not authorize organization")
		return false
	}
	if !isOrgAdmin {
		writeError(w, http.StatusForbidden, "forbidden", "organization administrator role is required")
		return false
	}
	return true
}

func (h *Handler) createWorkspace(ctx context.Context, organizationID string, userID string, name string, slug string) (workspaceResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return workspaceResponse{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	uniqueSlug, err := uniqueWorkspaceSlug(ctx, tx, organizationID, slug)
	if err != nil {
		return workspaceResponse{}, err
	}

	var workspace workspaceResponse
	if err := tx.QueryRow(ctx, `
		INSERT INTO workspaces (name, organization_id, slug, status)
		VALUES ($1, $2, $3, 'active')
		RETURNING id::text, name, COALESCE(slug, ''), status
	`, name, organizationID, uniqueSlug).Scan(
		&workspace.ID,
		&workspace.Name,
		&workspace.Slug,
		&workspace.Status,
	); err != nil {
		return workspaceResponse{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, 'admin')
	`, workspace.ID, userID); err != nil {
		return workspaceResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return workspaceResponse{}, err
	}
	workspace.Role = "admin"
	return workspace, nil
}

func (h *Handler) updateWorkspace(ctx context.Context, workspaceID string, userID string, name string, status string) (workspaceResponse, error) {
	var workspace workspaceResponse
	if err := h.db.QueryRow(ctx, `
		UPDATE workspaces SET name = $2, status = $3
		WHERE id = $1
		RETURNING id::text, name, COALESCE(slug, ''), status
	`, workspaceID, name, status).Scan(
		&workspace.ID,
		&workspace.Name,
		&workspace.Slug,
		&workspace.Status,
	); err != nil {
		return workspaceResponse{}, err
	}

	if err := h.db.QueryRow(ctx, `
		SELECT COALESCE(role, '')
		FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID).Scan(&workspace.Role); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			workspace.Role = ""
		} else {
			return workspaceResponse{}, err
		}
	}
	return workspace, nil
}

func uniqueWorkspaceSlug(ctx context.Context, tx pgx.Tx, organizationID string, base string) (string, error) {
	candidate := base
	for attempt := 2; ; attempt++ {
		var taken bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM workspaces
				WHERE organization_id = $1 AND slug = $2
			)
		`, organizationID, candidate).Scan(&taken); err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, attempt)
	}
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

func normalizeName(value string) (string, string, error) {
	name := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if name == "" {
		return "", "", errors.New("name is required")
	}
	if len([]rune(name)) > 80 {
		return "", "", errors.New("name must be 80 characters or fewer")
	}
	slug := slugify(name)
	if slug == "" {
		return "", "", errors.New("name must contain letters or digits")
	}
	return name, slug, nil
}

func slugify(value string) string {
	var builder strings.Builder
	previousHyphen := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			previousHyphen = false
			continue
		}
		if !previousHyphen && builder.Len() > 0 {
			builder.WriteByte('-')
			previousHyphen = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func normalizeID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New("workspace id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("workspace id is invalid")
	}
	return id, nil
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
