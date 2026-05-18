package issues

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"team-task-tracker/backend/internal/auth"
)

var validIssueTypes = map[string]bool{
	"task":  true,
	"bug":   true,
	"story": true,
}

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

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

type createIssueRequest struct {
	ProjectID   string `json:"project_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	IssueType   string `json:"issue_type"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	AssigneeID  string `json:"assignee_id"`
	DueDate     string `json:"due_date"`
}

type transitionIssueRequest struct {
	Status string `json:"status"`
}

type issueResponse struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ProjectKey  string    `json:"project_key"`
	Number      int       `json:"number"`
	IssueKey    string    `json:"issue_key"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	IssueType   string    `json:"issue_type"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	ReporterID  string    `json:"reporter_id"`
	AssigneeID  *string   `json:"assignee_id"`
	DueDate     *string   `json:"due_date"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type listIssuesResponse struct {
	Issues []issueResponse `json:"issues"`
}

type normalizedCreateIssue struct {
	ProjectID   string
	Title       string
	Description string
	IssueType   string
	Status      string
	Priority    string
	AssigneeID  string
	DueDate     string
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{
		db:   db,
		auth: authHandler,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/issues", h.list)
	mux.HandleFunc("POST /api/v1/issues", h.create)
	mux.HandleFunc("POST /api/v1/issues/{id}/transition", h.transition)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	issues, err := h.listIssues(ctx, user.WorkspaceID, r.URL.Query())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list issues")
		return
	}

	writeJSON(w, http.StatusOK, listIssuesResponse{Issues: issues})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	var req createIssueRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input, err := normalizeCreateIssue(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	issue, err := h.createIssue(ctx, user, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "project_not_found", "project was not found")
			return
		}
		if errors.Is(err, errInvalidAssignee) {
			writeError(w, http.StatusBadRequest, "invalid_assignee", "assignee is not a workspace member")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not create issue")
		return
	}

	writeJSON(w, http.StatusCreated, issue)
}

func (h *Handler) transition(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID := strings.TrimSpace(r.PathValue("id"))
	if issueID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "issue id is required")
		return
	}

	var req transitionIssueRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	status, err := normalizeTransitionIssue(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	issue, err := h.transitionIssueStatus(ctx, user.WorkspaceID, issueID, status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not update issue status")
		return
	}

	writeJSON(w, http.StatusOK, issue)
}

func (h *Handler) listIssues(ctx context.Context, workspaceID string, query map[string][]string) ([]issueResponse, error) {
	args := []any{workspaceID}
	conditions := []string{"p.workspace_id = $1"}

	addFilter := func(column string, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}

		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf("%s = $%d", column, len(args)))
	}

	addFilter("i.project_id", firstQueryValue(query, "project_id"))
	addFilter("i.status", firstQueryValue(query, "status"))
	addFilter("i.priority", firstQueryValue(query, "priority"))
	addFilter("i.assignee_id", firstQueryValue(query, "assignee_id"))

	sql := fmt.Sprintf(`
		SELECT
			i.id::text,
			i.project_id::text,
			p.key,
			i.number,
			i.issue_key,
			i.title,
			i.description,
			i.issue_type,
			i.status,
			i.priority,
			i.reporter_id::text,
			i.assignee_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at
		FROM issues i
		JOIN projects p ON p.id = i.project_id
		WHERE %s
		ORDER BY i.created_at DESC
		LIMIT 100
	`, strings.Join(conditions, " AND "))

	rows, err := h.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	issues := make([]issueResponse, 0)
	for rows.Next() {
		issue, err := scanIssue(rows)
		if err != nil {
			return nil, err
		}

		issues = append(issues, issue)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return issues, nil
}

var errInvalidAssignee = errors.New("invalid assignee")

func (h *Handler) createIssue(ctx context.Context, user auth.CurrentUser, input normalizedCreateIssue) (issueResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return issueResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var projectKey string
	if err := tx.QueryRow(ctx, `
		SELECT key
		FROM projects
		WHERE id = $1
			AND workspace_id = $2
			AND archived_at IS NULL
		FOR UPDATE
	`, input.ProjectID, user.WorkspaceID).Scan(&projectKey); err != nil {
		return issueResponse{}, err
	}

	if input.AssigneeID != "" {
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM workspace_members
				WHERE workspace_id = $1
					AND user_id = $2
			)
		`, user.WorkspaceID, input.AssigneeID).Scan(&exists); err != nil {
			return issueResponse{}, err
		}

		if !exists {
			return issueResponse{}, errInvalidAssignee
		}
	}

	var nextNumber int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(number), 0) + 1
		FROM issues
		WHERE project_id = $1
	`, input.ProjectID).Scan(&nextNumber); err != nil {
		return issueResponse{}, err
	}

	issueKey := fmt.Sprintf("%s-%d", projectKey, nextNumber)
	var assigneeID any
	if input.AssigneeID != "" {
		assigneeID = input.AssigneeID
	}

	var dueDate any
	if input.DueDate != "" {
		dueDate = input.DueDate
	}

	issue, err := scanIssue(tx.QueryRow(ctx, `
		INSERT INTO issues (
			project_id,
			number,
			issue_key,
			title,
			description,
			issue_type,
			status,
			priority,
			reporter_id,
			assignee_id,
			due_date
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING
			id::text,
			project_id::text,
			$12::text,
			number,
			issue_key,
			title,
			description,
			issue_type,
			status,
			priority,
			reporter_id::text,
			assignee_id::text,
			due_date::text,
			created_at,
			updated_at
	`, input.ProjectID, nextNumber, issueKey, input.Title, input.Description, input.IssueType, input.Status, input.Priority, user.ID, assigneeID, dueDate, projectKey))
	if err != nil {
		return issueResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return issueResponse{}, err
	}

	return issue, nil
}

func (h *Handler) transitionIssueStatus(ctx context.Context, workspaceID string, issueID string, status string) (issueResponse, error) {
	return scanIssue(h.db.QueryRow(ctx, `
		UPDATE issues i
		SET status = $3,
			updated_at = now()
		FROM projects p
		WHERE i.project_id = p.id
			AND i.id = $1
			AND p.workspace_id = $2
			AND p.archived_at IS NULL
		RETURNING
			i.id::text,
			i.project_id::text,
			p.key,
			i.number,
			i.issue_key,
			i.title,
			i.description,
			i.issue_type,
			i.status,
			i.priority,
			i.reporter_id::text,
			i.assignee_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at
	`, issueID, workspaceID, status))
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

func normalizeCreateIssue(req createIssueRequest) (normalizedCreateIssue, error) {
	input := normalizedCreateIssue{
		ProjectID:   strings.TrimSpace(req.ProjectID),
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		IssueType:   withDefault(strings.TrimSpace(req.IssueType), "task"),
		Status:      withDefault(strings.TrimSpace(req.Status), "todo"),
		Priority:    withDefault(strings.TrimSpace(req.Priority), "medium"),
		AssigneeID:  strings.TrimSpace(req.AssigneeID),
		DueDate:     strings.TrimSpace(req.DueDate),
	}

	if input.ProjectID == "" {
		return input, errors.New("project_id is required")
	}
	if input.Title == "" {
		return input, errors.New("title is required")
	}
	if len(input.Title) > 180 {
		return input, errors.New("title must be 180 characters or fewer")
	}
	if !validIssueTypes[input.IssueType] {
		return input, errors.New("issue_type is invalid")
	}
	if !validIssueStatuses[input.Status] {
		return input, errors.New("status is invalid")
	}
	if !validIssuePriorities[input.Priority] {
		return input, errors.New("priority is invalid")
	}
	if input.DueDate != "" {
		if _, err := time.Parse(time.DateOnly, input.DueDate); err != nil {
			return input, errors.New("due_date must be YYYY-MM-DD")
		}
	}

	return input, nil
}

func normalizeTransitionIssue(req transitionIssueRequest) (string, error) {
	status := strings.TrimSpace(req.Status)
	if status == "" {
		return "", errors.New("status is required")
	}
	if !validIssueStatuses[status] {
		return "", errors.New("status is invalid")
	}

	return status, nil
}

func withDefault(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func firstQueryValue(query map[string][]string, key string) string {
	values := query[key]
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanIssue(row rowScanner) (issueResponse, error) {
	var issue issueResponse
	var assigneeID pgtype.Text
	var dueDate pgtype.Text

	if err := row.Scan(
		&issue.ID,
		&issue.ProjectID,
		&issue.ProjectKey,
		&issue.Number,
		&issue.IssueKey,
		&issue.Title,
		&issue.Description,
		&issue.IssueType,
		&issue.Status,
		&issue.Priority,
		&issue.ReporterID,
		&assigneeID,
		&dueDate,
		&issue.CreatedAt,
		&issue.UpdatedAt,
	); err != nil {
		return issueResponse{}, err
	}

	issue.AssigneeID = nullableText(assigneeID)
	issue.DueDate = nullableText(dueDate)

	return issue, nil
}

func nullableText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
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
