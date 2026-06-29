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

type roleAssignmentResponse struct {
	ID          string `json:"id"`
	SubjectType string `json:"subject_type"`
	SubjectID   string `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	Role        string `json:"role"`
}

type listRoleAssignmentsResponse struct {
	Assignments []roleAssignmentResponse `json:"assignments"`
}

type createRoleAssignmentRequest struct {
	SubjectType string `json:"subject_type"`
	SubjectID   string `json:"subject_id"`
	Role        string `json:"role"`
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{db: db, auth: authHandler}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/workspaces", h.list)
	mux.HandleFunc("POST /api/v1/workspaces", h.create)
	mux.HandleFunc("PATCH /api/v1/workspaces/{id}", h.update)
	mux.HandleFunc("GET /api/v1/workspaces/{id}/role-assignments", h.listRoleAssignments)
	mux.HandleFunc("POST /api/v1/workspaces/{id}/role-assignments", h.createRoleAssignment)
	mux.HandleFunc("DELETE /api/v1/workspaces/{id}/role-assignments/{assignmentId}", h.deleteRoleAssignment)
}

// list returns workspaces for the current user. By default it surfaces the
// active workspaces the user belongs to within their active organization (the
// switcher source, archived workspaces excluded). With ?scope=organization an
// organization administrator gets every workspace in the active organization,
// including archived ones, for administration.
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if r.URL.Query().Get("scope") == "organization" {
		if user.OrganizationID == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "no active organization")
			return
		}
		if !h.authorizeOrganizationAdmin(ctx, w, user, user.OrganizationID) {
			return
		}
		rows, err := h.db.Query(ctx, `
			SELECT w.id::text, w.name, COALESCE(w.slug, ''), w.status, COALESCE(wm.role, '')
			FROM workspaces w
			LEFT JOIN workspace_members wm
				ON wm.workspace_id = w.id AND wm.user_id = $1
			WHERE w.organization_id = $2::uuid
			ORDER BY w.status ASC, w.name ASC
		`, user.ID, user.OrganizationID)
		writeWorkspaceRows(w, rows, err, user.WorkspaceID)
		return
	}

	rows, err := h.db.Query(ctx, `
		SELECT w.id::text, w.name, COALESCE(w.slug, ''), w.status, wm.role
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id = w.id AND wm.user_id = $1
		WHERE w.status = 'active'
			AND ($2 = '' OR w.organization_id = $2::uuid)
		ORDER BY w.name ASC
	`, user.ID, user.OrganizationID)
	writeWorkspaceRows(w, rows, err, user.WorkspaceID)
}

func writeWorkspaceRows(w http.ResponseWriter, rows pgx.Rows, queryErr error, activeWorkspaceID string) {
	if queryErr != nil {
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
		workspace.IsActive = workspace.ID == activeWorkspaceID
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

// listRoleAssignments returns the user and group role assignments for a
// workspace, resolving each subject's display name.
func (h *Handler) listRoleAssignments(w http.ResponseWriter, r *http.Request) {
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

	if _, ok := h.authorizeWorkspaceAdmin(ctx, w, user, workspaceID); !ok {
		return
	}

	rows, err := h.db.Query(ctx, `
		SELECT ra.id::text, ra.subject_type, ra.subject_id::text, ra.role,
			COALESCE(u.display_name, g.name, '')
		FROM role_assignments ra
		LEFT JOIN users u ON ra.subject_type = 'user' AND u.id = ra.subject_id
		LEFT JOIN groups g ON ra.subject_type = 'group' AND g.id = ra.subject_id
		WHERE ra.scope = 'workspace' AND ra.scope_id = $1
		ORDER BY ra.subject_type ASC, COALESCE(u.display_name, g.name, '') ASC
	`, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list role assignments")
		return
	}
	defer rows.Close()

	assignments := make([]roleAssignmentResponse, 0)
	for rows.Next() {
		var assignment roleAssignmentResponse
		if err := rows.Scan(&assignment.ID, &assignment.SubjectType, &assignment.SubjectID, &assignment.Role, &assignment.SubjectName); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not list role assignments")
			return
		}
		assignments = append(assignments, assignment)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list role assignments")
		return
	}

	writeJSON(w, http.StatusOK, listRoleAssignmentsResponse{Assignments: assignments})
}

// createRoleAssignment assigns a workspace role to a user or group from the
// workspace's organization, upserting an existing assignment for the subject.
func (h *Handler) createRoleAssignment(w http.ResponseWriter, r *http.Request) {
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

	organizationID, ok := h.authorizeWorkspaceAdmin(ctx, w, user, workspaceID)
	if !ok {
		return
	}

	var req createRoleAssignmentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	subjectType := strings.ToLower(strings.TrimSpace(req.SubjectType))
	if subjectType != "user" && subjectType != "group" {
		writeError(w, http.StatusBadRequest, "invalid_request", "subject_type must be user or group")
		return
	}
	subjectID, err := normalizeID(req.SubjectID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "subject_id is invalid")
		return
	}
	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role != "admin" && role != "member" {
		writeError(w, http.StatusBadRequest, "invalid_request", "role must be admin or member")
		return
	}

	subjectName, err := h.resolveWorkspaceSubject(ctx, organizationID, subjectType, subjectID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "subject_not_found", "subject does not belong to this organization")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not validate subject")
		return
	}

	var assignmentID string
	if err := h.db.QueryRow(ctx, `
		INSERT INTO role_assignments (scope, scope_id, subject_type, subject_id, role, created_by)
		VALUES ('workspace', $1, $2, $3, $4, $5)
		ON CONFLICT (scope, scope_id, subject_type, subject_id)
		DO UPDATE SET role = EXCLUDED.role, updated_at = now()
		RETURNING id::text
	`, workspaceID, subjectType, subjectID, role, user.ID).Scan(&assignmentID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not assign role")
		return
	}

	writeJSON(w, http.StatusOK, roleAssignmentResponse{
		ID:          assignmentID,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		SubjectName: subjectName,
		Role:        role,
	})
}

func (h *Handler) deleteRoleAssignment(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	workspaceID, err := normalizeID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	assignmentID, err := normalizeID(r.PathValue("assignmentId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, ok := h.authorizeWorkspaceAdmin(ctx, w, user, workspaceID); !ok {
		return
	}

	tag, err := h.db.Exec(ctx, `
		DELETE FROM role_assignments
		WHERE id = $1 AND scope = 'workspace' AND scope_id = $2
	`, assignmentID, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not remove role assignment")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "assignment_not_found", "role assignment was not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// resolveWorkspaceSubject confirms the subject belongs to the workspace's
// organization and returns its display name. Users must be active organization
// members; groups must belong to the organization.
func (h *Handler) resolveWorkspaceSubject(ctx context.Context, organizationID string, subjectType string, subjectID string) (string, error) {
	if subjectType == "group" {
		var name string
		if err := h.db.QueryRow(ctx, `
			SELECT name FROM groups WHERE id = $1 AND organization_id = $2
		`, subjectID, organizationID).Scan(&name); err != nil {
			return "", err
		}
		return name, nil
	}

	var name string
	if err := h.db.QueryRow(ctx, `
		SELECT u.display_name
		FROM organization_members om
		JOIN users u ON u.id = om.user_id
		WHERE om.organization_id = $1 AND om.user_id = $2 AND u.is_active = true
	`, organizationID, subjectID).Scan(&name); err != nil {
		return "", err
	}
	return name, nil
}

// authorizeWorkspaceAdmin allows site admins, organization admins of the
// workspace's organization and direct workspace admins to manage a workspace's
// role assignments. It returns the workspace's organization id.
func (h *Handler) authorizeWorkspaceAdmin(ctx context.Context, w http.ResponseWriter, user auth.CurrentUser, workspaceID string) (string, bool) {
	var organizationID string
	if err := h.db.QueryRow(ctx, `
		SELECT COALESCE(organization_id::text, '') FROM workspaces WHERE id = $1
	`, workspaceID).Scan(&organizationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "workspace_not_found", "workspace was not found")
			return "", false
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load workspace")
		return "", false
	}

	if user.IsSiteAdmin {
		return organizationID, true
	}

	var allowed bool
	if err := h.db.QueryRow(ctx, `
		SELECT
			EXISTS (
				SELECT 1 FROM organization_members
				WHERE organization_id = $1 AND user_id = $2 AND role = 'org_admin'
			)
			OR EXISTS (
				SELECT 1 FROM workspace_members
				WHERE workspace_id = $3 AND user_id = $2 AND role = 'admin'
			)
	`, organizationID, user.ID, workspaceID).Scan(&allowed); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not authorize workspace")
		return "", false
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "forbidden", "workspace administrator role is required")
		return "", false
	}
	return organizationID, true
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
