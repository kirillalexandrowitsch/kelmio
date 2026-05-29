package sprints

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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"team-task-tracker/backend/internal/auth"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

var validSprintStatuses = map[string]bool{
	"planned":   true,
	"active":    true,
	"completed": true,
}

var errActiveSprintExists = errors.New("active sprint exists")
var errSprintCompleted = errors.New("sprint completed")
var errSprintNotActive = errors.New("sprint not active")
var errSprintNotPlanned = errors.New("sprint not planned")
var errSprintIssueProjectMismatch = errors.New("sprint issue project mismatch")

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

type createSprintRequest struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Goal      string `json:"goal"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type updateSprintRequest struct {
	Name      string `json:"name"`
	Goal      string `json:"goal"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type addSprintIssueRequest struct {
	IssueID string `json:"issue_id"`
}

type sprintResponse struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspace_id"`
	ProjectID   string     `json:"project_id"`
	ProjectKey  string     `json:"project_key"`
	ProjectName string     `json:"project_name"`
	Name        string     `json:"name"`
	Goal        string     `json:"goal"`
	Status      string     `json:"status"`
	StartDate   *string    `json:"start_date"`
	EndDate     *string    `json:"end_date"`
	CreatedBy   string     `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`
	IssueCount  int        `json:"issue_count"`
}

type listSprintsResponse struct {
	Sprints []sprintResponse `json:"sprints"`
}

type normalizedCreateSprint struct {
	ProjectID string
	Name      string
	Goal      string
	StartDate string
	EndDate   string
}

type normalizedUpdateSprint struct {
	Name      string
	Goal      string
	StartDate string
	EndDate   string
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{
		db:   db,
		auth: authHandler,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/sprints", h.list)
	mux.HandleFunc("POST /api/v1/sprints", h.create)
	mux.HandleFunc("GET /api/v1/sprints/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/sprints/{id}", h.update)
	mux.HandleFunc("POST /api/v1/sprints/{id}/start", h.start)
	mux.HandleFunc("POST /api/v1/sprints/{id}/complete", h.complete)
	mux.HandleFunc("POST /api/v1/sprints/{id}/issues", h.addIssue)
	mux.HandleFunc("DELETE /api/v1/sprints/{id}/issues/{issueId}", h.removeIssue)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	projectID, err := normalizeOptionalID(r.URL.Query().Get("project_id"), "project_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status != "" && !validSprintStatuses[status] {
		writeError(w, http.StatusBadRequest, "invalid_request", "status is invalid")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sprints, err := h.listSprints(ctx, user.WorkspaceID, projectID, status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list sprints")
		return
	}

	writeJSON(w, http.StatusOK, listSprintsResponse{Sprints: sprints})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	var req createSprintRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input, err := normalizeCreateSprint(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sprint, err := h.createSprint(ctx, user, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "project_not_found", "project was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not create sprint")
		return
	}

	writeJSON(w, http.StatusCreated, sprint)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	sprintID, err := normalizeID(r.PathValue("id"), "sprint id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sprint, err := h.getSprint(ctx, user.WorkspaceID, sprintID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sprint_not_found", "sprint was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load sprint")
		return
	}

	writeJSON(w, http.StatusOK, sprint)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	sprintID, err := normalizeID(r.PathValue("id"), "sprint id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req updateSprintRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input, err := normalizeUpdateSprint(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sprint, err := h.updateSprint(ctx, user.WorkspaceID, sprintID, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sprint_not_found", "sprint was not found")
			return
		}
		if errors.Is(err, errSprintCompleted) {
			writeError(w, http.StatusBadRequest, "sprint_completed", "completed sprint cannot be updated")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not update sprint")
		return
	}

	writeJSON(w, http.StatusOK, sprint)
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	sprintID, err := normalizeID(r.PathValue("id"), "sprint id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sprint, err := h.startSprint(ctx, user.WorkspaceID, sprintID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sprint_not_found", "sprint was not found")
			return
		}
		if errors.Is(err, errSprintNotPlanned) {
			writeError(w, http.StatusBadRequest, "invalid_status", "only planned sprint can be started")
			return
		}
		if errors.Is(err, errActiveSprintExists) {
			writeError(w, http.StatusConflict, "active_sprint_exists", "project already has an active sprint")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not start sprint")
		return
	}

	writeJSON(w, http.StatusOK, sprint)
}

func (h *Handler) complete(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	sprintID, err := normalizeID(r.PathValue("id"), "sprint id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sprint, err := h.completeSprint(ctx, user.WorkspaceID, sprintID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sprint_not_found", "sprint was not found")
			return
		}
		if errors.Is(err, errSprintNotActive) {
			writeError(w, http.StatusBadRequest, "invalid_status", "only active sprint can be completed")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not complete sprint")
		return
	}

	writeJSON(w, http.StatusOK, sprint)
}

func (h *Handler) addIssue(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	sprintID, err := normalizeID(r.PathValue("id"), "sprint id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req addSprintIssueRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	issueID, err := normalizeID(req.IssueID, "issue id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	sprint, err := h.addIssueToSprint(ctx, user, sprintID, issueID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sprint_or_issue_not_found", "sprint or issue was not found")
			return
		}
		if errors.Is(err, errSprintCompleted) {
			writeError(w, http.StatusBadRequest, "sprint_completed", "completed sprint cannot accept issues")
			return
		}
		if errors.Is(err, errSprintIssueProjectMismatch) {
			writeError(w, http.StatusBadRequest, "project_mismatch", "issue must belong to sprint project")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not add issue to sprint")
		return
	}

	writeJSON(w, http.StatusOK, sprint)
}

func (h *Handler) removeIssue(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	sprintID, err := normalizeID(r.PathValue("id"), "sprint id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	issueID, err := normalizeID(r.PathValue("issueId"), "issue id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.removeIssueFromSprint(ctx, user, sprintID, issueID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "sprint_issue_not_found", "sprint issue was not found")
			return
		}
		if errors.Is(err, errSprintCompleted) {
			writeError(w, http.StatusBadRequest, "sprint_completed", "completed sprint cannot be changed")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not remove issue from sprint")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listSprints(ctx context.Context, workspaceID string, projectID string, status string) ([]sprintResponse, error) {
	args := []any{workspaceID}
	conditions := []string{
		"s.workspace_id = $1",
		"p.archived_at IS NULL",
	}

	if projectID != "" {
		args = append(args, projectID)
		conditions = append(conditions, "s.project_id = $2")
	}
	if status != "" {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("s.status = $%d", len(args)))
	}

	rows, err := h.db.Query(ctx, `
		SELECT
			s.id::text,
			s.workspace_id::text,
			s.project_id::text,
			p.key,
			p.name,
			s.name,
			s.goal,
			s.status,
			s.start_date::text,
			s.end_date::text,
			s.created_by::text,
			s.created_at,
			s.completed_at,
			(
				SELECT COUNT(*)::int
				FROM issues i
				WHERE i.sprint_id = s.id
					AND i.archived_at IS NULL
			)
		FROM sprints s
		JOIN projects p ON p.id = s.project_id
		WHERE `+strings.Join(conditions, " AND ")+`
		ORDER BY
			CASE s.status
				WHEN 'active' THEN 1
				WHEN 'planned' THEN 2
				ELSE 3
			END,
			s.created_at DESC,
			s.id DESC
		LIMIT 100
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sprints := make([]sprintResponse, 0)
	for rows.Next() {
		sprint, err := scanSprint(rows)
		if err != nil {
			return nil, err
		}

		sprints = append(sprints, sprint)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sprints, nil
}

func (h *Handler) createSprint(ctx context.Context, user auth.CurrentUser, input normalizedCreateSprint) (sprintResponse, error) {
	sprint, err := scanSprint(h.db.QueryRow(ctx, `
		WITH target_project AS (
			SELECT id, workspace_id
			FROM projects
			WHERE id = $1
				AND workspace_id = $2
				AND archived_at IS NULL
		),
		inserted AS (
			INSERT INTO sprints (
				workspace_id,
				project_id,
				name,
				goal,
				start_date,
				end_date,
				created_by
			)
			SELECT workspace_id, id, $3, $4, $5::date, $6::date, $7
			FROM target_project
			RETURNING *
		)
		SELECT
			inserted.id::text,
			inserted.workspace_id::text,
			inserted.project_id::text,
			p.key,
			p.name,
			inserted.name,
			inserted.goal,
			inserted.status,
			inserted.start_date::text,
			inserted.end_date::text,
			inserted.created_by::text,
			inserted.created_at,
			inserted.completed_at,
			0::int
		FROM inserted
		JOIN projects p ON p.id = inserted.project_id
	`, input.ProjectID, user.WorkspaceID, input.Name, input.Goal, dateOrNil(input.StartDate), dateOrNil(input.EndDate), user.ID))
	if err != nil {
		return sprintResponse{}, err
	}

	return sprint, nil
}

func (h *Handler) getSprint(ctx context.Context, workspaceID string, sprintID string) (sprintResponse, error) {
	return getSprintForUpdate(ctx, h.db, workspaceID, sprintID, false)
}

func (h *Handler) updateSprint(ctx context.Context, workspaceID string, sprintID string, input normalizedUpdateSprint) (sprintResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return sprintResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := getSprintForUpdate(ctx, tx, workspaceID, sprintID, true)
	if err != nil {
		return sprintResponse{}, err
	}
	if current.Status == "completed" {
		return sprintResponse{}, errSprintCompleted
	}

	sprint, err := scanSprint(tx.QueryRow(ctx, `
		UPDATE sprints s
		SET name = $3,
			goal = $4,
			start_date = $5::date,
			end_date = $6::date
		FROM projects p
		WHERE s.project_id = p.id
			AND s.id = $1
			AND s.workspace_id = $2
			AND p.archived_at IS NULL
		RETURNING
			s.id::text,
			s.workspace_id::text,
			s.project_id::text,
			p.key,
			p.name,
			s.name,
			s.goal,
			s.status,
			s.start_date::text,
			s.end_date::text,
			s.created_by::text,
			s.created_at,
			s.completed_at,
			(
				SELECT COUNT(*)::int
				FROM issues i
				WHERE i.sprint_id = s.id
					AND i.archived_at IS NULL
			)
	`, sprintID, workspaceID, input.Name, input.Goal, dateOrNil(input.StartDate), dateOrNil(input.EndDate)))
	if err != nil {
		return sprintResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return sprintResponse{}, err
	}

	return sprint, nil
}

func (h *Handler) startSprint(ctx context.Context, workspaceID string, sprintID string) (sprintResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return sprintResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := getSprintForUpdate(ctx, tx, workspaceID, sprintID, true)
	if err != nil {
		return sprintResponse{}, err
	}
	if current.Status != "planned" {
		return sprintResponse{}, errSprintNotPlanned
	}

	sprint, err := scanSprint(tx.QueryRow(ctx, `
		UPDATE sprints s
		SET status = 'active',
			completed_at = NULL
		FROM projects p
		WHERE s.project_id = p.id
			AND s.id = $1
			AND s.workspace_id = $2
			AND p.archived_at IS NULL
		RETURNING
			s.id::text,
			s.workspace_id::text,
			s.project_id::text,
			p.key,
			p.name,
			s.name,
			s.goal,
			s.status,
			s.start_date::text,
			s.end_date::text,
			s.created_by::text,
			s.created_at,
			s.completed_at,
			(
				SELECT COUNT(*)::int
				FROM issues i
				WHERE i.sprint_id = s.id
					AND i.archived_at IS NULL
			)
	`, sprintID, workspaceID))
	if err != nil {
		if isUniqueViolation(err) {
			return sprintResponse{}, errActiveSprintExists
		}
		return sprintResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return sprintResponse{}, err
	}

	return sprint, nil
}

func (h *Handler) completeSprint(ctx context.Context, workspaceID string, sprintID string) (sprintResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return sprintResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	current, err := getSprintForUpdate(ctx, tx, workspaceID, sprintID, true)
	if err != nil {
		return sprintResponse{}, err
	}
	if current.Status != "active" {
		return sprintResponse{}, errSprintNotActive
	}

	sprint, err := scanSprint(tx.QueryRow(ctx, `
		UPDATE sprints s
		SET status = 'completed',
			completed_at = now()
		FROM projects p
		WHERE s.project_id = p.id
			AND s.id = $1
			AND s.workspace_id = $2
			AND p.archived_at IS NULL
		RETURNING
			s.id::text,
			s.workspace_id::text,
			s.project_id::text,
			p.key,
			p.name,
			s.name,
			s.goal,
			s.status,
			s.start_date::text,
			s.end_date::text,
			s.created_by::text,
			s.created_at,
			s.completed_at,
			(
				SELECT COUNT(*)::int
				FROM issues i
				WHERE i.sprint_id = s.id
					AND i.archived_at IS NULL
			)
	`, sprintID, workspaceID))
	if err != nil {
		return sprintResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return sprintResponse{}, err
	}

	return sprint, nil
}

func (h *Handler) addIssueToSprint(ctx context.Context, user auth.CurrentUser, sprintID string, issueID string) (sprintResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return sprintResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	sprint, err := getSprintForUpdate(ctx, tx, user.WorkspaceID, sprintID, true)
	if err != nil {
		return sprintResponse{}, err
	}
	if sprint.Status == "completed" {
		return sprintResponse{}, errSprintCompleted
	}

	var issueProjectID string
	var issueKey string
	if err := tx.QueryRow(ctx, `
		SELECT i.project_id::text, i.issue_key
		FROM issues i
		JOIN projects p ON p.id = i.project_id
		WHERE i.id = $1
			AND p.workspace_id = $2
			AND p.archived_at IS NULL
			AND i.archived_at IS NULL
		FOR UPDATE OF i
	`, issueID, user.WorkspaceID).Scan(&issueProjectID, &issueKey); err != nil {
		return sprintResponse{}, err
	}
	if issueProjectID != sprint.ProjectID {
		return sprintResponse{}, errSprintIssueProjectMismatch
	}

	if _, err := tx.Exec(ctx, `
		UPDATE issues
		SET sprint_id = $1,
			updated_at = now()
		WHERE id = $2
	`, sprintID, issueID); err != nil {
		return sprintResponse{}, err
	}

	if err := insertIssueActivity(ctx, tx, issueID, user.ID, "issue_added_to_sprint", map[string]string{
		"sprint_id":   sprint.ID,
		"sprint_name": sprint.Name,
		"issue_key":   issueKey,
	}); err != nil {
		return sprintResponse{}, err
	}

	updatedSprint, err := getSprintForUpdate(ctx, tx, user.WorkspaceID, sprintID, false)
	if err != nil {
		return sprintResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return sprintResponse{}, err
	}

	return updatedSprint, nil
}

func (h *Handler) removeIssueFromSprint(ctx context.Context, user auth.CurrentUser, sprintID string, issueID string) error {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	sprint, err := getSprintForUpdate(ctx, tx, user.WorkspaceID, sprintID, true)
	if err != nil {
		return err
	}
	if sprint.Status == "completed" {
		return errSprintCompleted
	}

	var issueKey string
	if err := tx.QueryRow(ctx, `
		SELECT i.issue_key
		FROM issues i
		JOIN projects p ON p.id = i.project_id
		WHERE i.id = $1
			AND i.sprint_id = $2
			AND p.workspace_id = $3
			AND p.archived_at IS NULL
			AND i.archived_at IS NULL
		FOR UPDATE OF i
	`, issueID, sprintID, user.WorkspaceID).Scan(&issueKey); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		UPDATE issues
		SET sprint_id = NULL,
			updated_at = now()
		WHERE id = $1
	`, issueID); err != nil {
		return err
	}

	if err := insertIssueActivity(ctx, tx, issueID, user.ID, "issue_removed_from_sprint", map[string]string{
		"sprint_id":   sprint.ID,
		"sprint_name": sprint.Name,
		"issue_key":   issueKey,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func normalizeCreateSprint(req createSprintRequest) (normalizedCreateSprint, error) {
	projectID, err := normalizeID(req.ProjectID, "project_id")
	if err != nil {
		return normalizedCreateSprint{}, err
	}

	input := normalizedCreateSprint{
		ProjectID: projectID,
		Name:      strings.TrimSpace(req.Name),
		Goal:      strings.TrimSpace(req.Goal),
		StartDate: strings.TrimSpace(req.StartDate),
		EndDate:   strings.TrimSpace(req.EndDate),
	}

	if err := validateSprintDetails(input.Name, input.Goal, input.StartDate, input.EndDate); err != nil {
		return input, err
	}

	return input, nil
}

func normalizeUpdateSprint(req updateSprintRequest) (normalizedUpdateSprint, error) {
	input := normalizedUpdateSprint{
		Name:      strings.TrimSpace(req.Name),
		Goal:      strings.TrimSpace(req.Goal),
		StartDate: strings.TrimSpace(req.StartDate),
		EndDate:   strings.TrimSpace(req.EndDate),
	}

	if err := validateSprintDetails(input.Name, input.Goal, input.StartDate, input.EndDate); err != nil {
		return input, err
	}

	return input, nil
}

func validateSprintDetails(name string, goal string, startDate string, endDate string) error {
	if name == "" {
		return errors.New("name is required")
	}
	if len(name) > 120 {
		return errors.New("name must be 120 characters or fewer")
	}
	if len(goal) > 1000 {
		return errors.New("goal must be 1000 characters or fewer")
	}

	var parsedStartDate time.Time
	var parsedEndDate time.Time
	if startDate != "" {
		date, err := time.Parse(time.DateOnly, startDate)
		if err != nil {
			return errors.New("start_date must be YYYY-MM-DD")
		}
		parsedStartDate = date
	}
	if endDate != "" {
		date, err := time.Parse(time.DateOnly, endDate)
		if err != nil {
			return errors.New("end_date must be YYYY-MM-DD")
		}
		parsedEndDate = date
	}
	if !parsedStartDate.IsZero() && !parsedEndDate.IsZero() && parsedStartDate.After(parsedEndDate) {
		return errors.New("start_date must be before or equal to end_date")
	}

	return nil
}

func normalizeID(id string, field string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New(field + " is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New(field + " is invalid")
	}

	return id, nil
}

func normalizeOptionalID(id string, field string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", nil
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New(field + " is invalid")
	}

	return id, nil
}

func getSprintForUpdate(ctx context.Context, querier sprintQuerier, workspaceID string, sprintID string, forUpdate bool) (sprintResponse, error) {
	lockClause := ""
	if forUpdate {
		lockClause = " FOR UPDATE OF s"
	}

	return scanSprint(querier.QueryRow(ctx, `
		SELECT
			s.id::text,
			s.workspace_id::text,
			s.project_id::text,
			p.key,
			p.name,
			s.name,
			s.goal,
			s.status,
			s.start_date::text,
			s.end_date::text,
			s.created_by::text,
			s.created_at,
			s.completed_at,
			(
				SELECT COUNT(*)::int
				FROM issues i
				WHERE i.sprint_id = s.id
					AND i.archived_at IS NULL
			)
		FROM sprints s
		JOIN projects p ON p.id = s.project_id
		WHERE s.id = $1
			AND s.workspace_id = $2
			AND p.archived_at IS NULL
	`+lockClause, sprintID, workspaceID))
}

func dateOrNil(value string) any {
	if value == "" {
		return nil
	}

	return value
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func insertIssueActivity(ctx context.Context, tx pgx.Tx, issueID string, actorID string, action string, payload map[string]string) error {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO activity_log (entity_type, entity_id, action, actor_id, payload)
		VALUES ('issue', $1, $2, $3, $4::jsonb)
	`, issueID, action, actorID, string(encodedPayload))

	return err
}

func scanSprint(row rowScanner) (sprintResponse, error) {
	var sprint sprintResponse
	var startDate pgtype.Text
	var endDate pgtype.Text
	if err := row.Scan(
		&sprint.ID,
		&sprint.WorkspaceID,
		&sprint.ProjectID,
		&sprint.ProjectKey,
		&sprint.ProjectName,
		&sprint.Name,
		&sprint.Goal,
		&sprint.Status,
		&startDate,
		&endDate,
		&sprint.CreatedBy,
		&sprint.CreatedAt,
		&sprint.CompletedAt,
		&sprint.IssueCount,
	); err != nil {
		return sprintResponse{}, err
	}

	sprint.StartDate = nullableText(startDate)
	sprint.EndDate = nullableText(endDate)

	return sprint, nil
}

func nullableText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

type rowScanner interface {
	Scan(dest ...any) error
}

type sprintQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
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
