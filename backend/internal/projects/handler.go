package projects

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

	"team-task-tracker/backend/internal/auth"
)

var projectKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,9}$`)
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

type createProjectRequest struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type projectResponse struct {
	ID          string     `json:"id"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	ArchivedAt  *time.Time `json:"archived_at"`
}

type listProjectsResponse struct {
	Projects []projectResponse `json:"projects"`
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{
		db:   db,
		auth: authHandler,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/projects", h.list)
	mux.HandleFunc("POST /api/v1/projects", h.create)
	mux.HandleFunc("POST /api/v1/projects/{id}/archive", h.archive)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT
			id::text,
			key,
			name,
			description,
			created_by::text,
			created_at,
			archived_at
		FROM projects
		WHERE workspace_id = $1
			AND archived_at IS NULL
		ORDER BY created_at DESC
	`, user.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list projects")
		return
	}
	defer rows.Close()

	projects := make([]projectResponse, 0)
	for rows.Next() {
		var project projectResponse
		if err := rows.Scan(
			&project.ID,
			&project.Key,
			&project.Name,
			&project.Description,
			&project.CreatedBy,
			&project.CreatedAt,
			&project.ArchivedAt,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not read project")
			return
		}

		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list projects")
		return
	}

	writeJSON(w, http.StatusOK, listProjectsResponse{Projects: projects})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "admin role is required")
		return
	}

	var req createProjectRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	key := normalizeProjectKey(req.Key)
	name := strings.TrimSpace(req.Name)
	description := strings.TrimSpace(req.Description)

	if err := validateProjectInput(key, name); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var project projectResponse
	err := h.db.QueryRow(ctx, `
		INSERT INTO projects (workspace_id, key, name, description, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING
			id::text,
			key,
			name,
			description,
			created_by::text,
			created_at,
			archived_at
	`, user.WorkspaceID, key, name, description, user.ID).Scan(
		&project.ID,
		&project.Key,
		&project.Name,
		&project.Description,
		&project.CreatedBy,
		&project.CreatedAt,
		&project.ArchivedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "project_key_exists", "project key already exists")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not create project")
		return
	}

	writeJSON(w, http.StatusCreated, project)
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "admin role is required")
		return
	}

	projectID, err := normalizeProjectID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.archiveProject(ctx, user.WorkspaceID, projectID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "project_not_found", "project was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not archive project")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) archiveProject(ctx context.Context, workspaceID string, projectID string) error {
	var archivedProjectID string
	return h.db.QueryRow(ctx, `
		UPDATE projects
		SET archived_at = now()
		WHERE id = $1
			AND workspace_id = $2
			AND archived_at IS NULL
		RETURNING id::text
	`, projectID, workspaceID).Scan(&archivedProjectID)
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

func normalizeProjectKey(key string) string {
	return strings.ToUpper(strings.TrimSpace(key))
}

func normalizeProjectID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New("project id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("project id is invalid")
	}

	return id, nil
}

func validateProjectInput(key string, name string) error {
	if !projectKeyPattern.MatchString(key) {
		return errors.New("key must be 2-10 characters and contain only uppercase letters or numbers")
	}

	if name == "" {
		return errors.New("name is required")
	}

	if len(name) > 120 {
		return errors.New("name must be 120 characters or fewer")
	}

	return nil
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
