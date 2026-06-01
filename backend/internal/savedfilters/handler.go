package savedfilters

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

	"team-task-tracker/backend/internal/auth"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

var validIssueStatuses = map[string]bool{
	"backlog":     true,
	"todo":        true,
	"in_progress": true,
	"blocked":     true,
	"done":        true,
}

var validIssuePriorities = map[string]bool{
	"low":      true,
	"medium":   true,
	"high":     true,
	"critical": true,
}

var validIssueSorts = map[string]bool{
	"created_desc":  true,
	"created_asc":   true,
	"priority_desc": true,
	"due_date_asc":  true,
}

var validIssueDueFilters = map[string]bool{
	"overdue":  true,
	"today":    true,
	"due_soon": true,
	"no_due":   true,
}

var validSavedFilterKeys = map[string]bool{
	"query":      true,
	"sort":       true,
	"projectId":  true,
	"sprintId":   true,
	"status":     true,
	"priority":   true,
	"assigneeId": true,
	"labelId":    true,
	"due":        true,
}

var errSavedFilterExists = errors.New("saved filter exists")

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

type createSavedFilterRequest struct {
	Name    string            `json:"name"`
	Filters map[string]string `json:"filters"`
}

type updateSavedFilterRequest struct {
	Name    *string            `json:"name"`
	Filters *map[string]string `json:"filters"`
}

type savedFilterResponse struct {
	ID          string            `json:"id"`
	WorkspaceID string            `json:"workspace_id"`
	UserID      string            `json:"user_id"`
	Name        string            `json:"name"`
	Filters     map[string]string `json:"filters"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type listSavedFiltersResponse struct {
	SavedFilters []savedFilterResponse `json:"saved_filters"`
}

type normalizedCreateSavedFilter struct {
	Name    string
	Filters map[string]string
}

type normalizedUpdateSavedFilter struct {
	Name       string
	HasName    bool
	Filters    map[string]string
	HasFilters bool
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{
		db:   db,
		auth: authHandler,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/saved-filters", h.list)
	mux.HandleFunc("POST /api/v1/saved-filters", h.create)
	mux.HandleFunc("PATCH /api/v1/saved-filters/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/saved-filters/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	filters, err := h.listSavedFilters(ctx, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list saved filters")
		return
	}

	writeJSON(w, http.StatusOK, listSavedFiltersResponse{SavedFilters: filters})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	var req createSavedFilterRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input, err := normalizeCreateSavedFilter(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	filter, err := h.createSavedFilter(ctx, user, input)
	if err != nil {
		if errors.Is(err, errSavedFilterExists) {
			writeError(w, http.StatusConflict, "saved_filter_exists", "saved filter already exists")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not create saved filter")
		return
	}

	writeJSON(w, http.StatusCreated, filter)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	filterID, err := normalizeSavedFilterID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req updateSavedFilterRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input, err := normalizeUpdateSavedFilter(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	filter, err := h.updateSavedFilter(ctx, user, filterID, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "saved_filter_not_found", "saved filter was not found")
			return
		}
		if errors.Is(err, errSavedFilterExists) {
			writeError(w, http.StatusConflict, "saved_filter_exists", "saved filter already exists")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not update saved filter")
		return
	}

	writeJSON(w, http.StatusOK, filter)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	filterID, err := normalizeSavedFilterID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.deleteSavedFilter(ctx, user, filterID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "saved_filter_not_found", "saved filter was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not delete saved filter")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listSavedFilters(ctx context.Context, user auth.CurrentUser) ([]savedFilterResponse, error) {
	rows, err := h.db.Query(ctx, `
		SELECT id::text, workspace_id::text, user_id::text, name, filters, created_at, updated_at
		FROM saved_filters
		WHERE workspace_id = $1
			AND user_id = $2
		ORDER BY updated_at DESC, name ASC
	`, user.WorkspaceID, user.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	filters := make([]savedFilterResponse, 0)
	for rows.Next() {
		filter, err := scanSavedFilter(rows)
		if err != nil {
			return nil, err
		}

		filters = append(filters, filter)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return filters, nil
}

func (h *Handler) createSavedFilter(ctx context.Context, user auth.CurrentUser, input normalizedCreateSavedFilter) (savedFilterResponse, error) {
	filtersJSON, err := json.Marshal(input.Filters)
	if err != nil {
		return savedFilterResponse{}, err
	}

	filter, err := scanSavedFilter(h.db.QueryRow(ctx, `
		INSERT INTO saved_filters (workspace_id, user_id, name, filters)
		VALUES ($1, $2, $3, $4::jsonb)
		RETURNING id::text, workspace_id::text, user_id::text, name, filters, created_at, updated_at
	`, user.WorkspaceID, user.ID, input.Name, string(filtersJSON)))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return savedFilterResponse{}, errSavedFilterExists
		}

		return savedFilterResponse{}, err
	}

	return filter, nil
}

func (h *Handler) updateSavedFilter(ctx context.Context, user auth.CurrentUser, filterID string, input normalizedUpdateSavedFilter) (savedFilterResponse, error) {
	var nameArg any
	if input.HasName {
		nameArg = input.Name
	}

	var filtersArg any
	if input.HasFilters {
		filtersJSON, err := json.Marshal(input.Filters)
		if err != nil {
			return savedFilterResponse{}, err
		}
		filtersArg = string(filtersJSON)
	}

	filter, err := scanSavedFilter(h.db.QueryRow(ctx, `
		UPDATE saved_filters
		SET name = COALESCE($4::text, name),
			filters = COALESCE($5::jsonb, filters),
			updated_at = now()
		WHERE id = $1
			AND workspace_id = $2
			AND user_id = $3
		RETURNING id::text, workspace_id::text, user_id::text, name, filters, created_at, updated_at
	`, filterID, user.WorkspaceID, user.ID, nameArg, filtersArg))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return savedFilterResponse{}, errSavedFilterExists
		}

		return savedFilterResponse{}, err
	}

	return filter, nil
}

func (h *Handler) deleteSavedFilter(ctx context.Context, user auth.CurrentUser, filterID string) error {
	var deletedID string
	return h.db.QueryRow(ctx, `
		DELETE FROM saved_filters
		WHERE id = $1
			AND workspace_id = $2
			AND user_id = $3
		RETURNING id::text
	`, filterID, user.WorkspaceID, user.ID).Scan(&deletedID)
}

func normalizeCreateSavedFilter(req createSavedFilterRequest) (normalizedCreateSavedFilter, error) {
	name, err := normalizeSavedFilterName(req.Name)
	if err != nil {
		return normalizedCreateSavedFilter{}, err
	}

	filters, err := normalizeSavedFilterValues(req.Filters)
	if err != nil {
		return normalizedCreateSavedFilter{}, err
	}

	return normalizedCreateSavedFilter{
		Name:    name,
		Filters: filters,
	}, nil
}

func normalizeUpdateSavedFilter(req updateSavedFilterRequest) (normalizedUpdateSavedFilter, error) {
	var input normalizedUpdateSavedFilter
	if req.Name != nil {
		name, err := normalizeSavedFilterName(*req.Name)
		if err != nil {
			return normalizedUpdateSavedFilter{}, err
		}
		input.Name = name
		input.HasName = true
	}
	if req.Filters != nil {
		filters, err := normalizeSavedFilterValues(*req.Filters)
		if err != nil {
			return normalizedUpdateSavedFilter{}, err
		}
		input.Filters = filters
		input.HasFilters = true
	}
	if !input.HasName && !input.HasFilters {
		return normalizedUpdateSavedFilter{}, errors.New("name or filters is required")
	}

	return input, nil
}

func normalizeSavedFilterName(name string) (string, error) {
	name = strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	if name == "" {
		return "", errors.New("name is required")
	}
	if len([]rune(name)) > 60 {
		return "", errors.New("name must be 60 characters or fewer")
	}

	return name, nil
}

func normalizeSavedFilterValues(filters map[string]string) (map[string]string, error) {
	normalized := make(map[string]string)
	for key, rawValue := range filters {
		if !validSavedFilterKeys[key] {
			return nil, errors.New("filters contains an unknown key")
		}

		value := strings.TrimSpace(rawValue)
		if value == "" {
			continue
		}

		switch key {
		case "query":
			if len([]rune(value)) > 200 {
				return nil, errors.New("query must be 200 characters or fewer")
			}
			normalized[key] = value
		case "sort":
			if !validIssueSorts[value] {
				return nil, errors.New("sort is invalid")
			}
			normalized[key] = value
		case "projectId", "labelId":
			if !uuidPattern.MatchString(value) {
				return nil, errors.New(key + " is invalid")
			}
			normalized[key] = strings.ToLower(value)
		case "sprintId":
			if value != "none" && !uuidPattern.MatchString(value) {
				return nil, errors.New("sprintId is invalid")
			}
			normalized[key] = strings.ToLower(value)
		case "status":
			if !validIssueStatuses[value] {
				return nil, errors.New("status is invalid")
			}
			normalized[key] = value
		case "priority":
			if !validIssuePriorities[value] {
				return nil, errors.New("priority is invalid")
			}
			normalized[key] = value
		case "assigneeId":
			if value != "unassigned" && !uuidPattern.MatchString(value) {
				return nil, errors.New("assigneeId is invalid")
			}
			normalized[key] = strings.ToLower(value)
		case "due":
			if !validIssueDueFilters[value] {
				return nil, errors.New("due is invalid")
			}
			normalized[key] = value
		}
	}

	if normalized["sort"] == "" {
		normalized["sort"] = "created_desc"
	}

	return normalized, nil
}

func normalizeSavedFilterID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New("saved filter id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("saved filter id is invalid")
	}

	return id, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSavedFilter(row rowScanner) (savedFilterResponse, error) {
	var filter savedFilterResponse
	var filtersJSON []byte
	if err := row.Scan(
		&filter.ID,
		&filter.WorkspaceID,
		&filter.UserID,
		&filter.Name,
		&filtersJSON,
		&filter.CreatedAt,
		&filter.UpdatedAt,
	); err != nil {
		return savedFilterResponse{}, err
	}

	if err := json.Unmarshal(filtersJSON, &filter.Filters); err != nil {
		return savedFilterResponse{}, err
	}
	if filter.Filters == nil {
		filter.Filters = map[string]string{}
	}

	return filter, nil
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
