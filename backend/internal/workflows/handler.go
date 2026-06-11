package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/projectaccess"
)

var errStatusKeyExists = errors.New("workflow status key exists")
var errStatusNameExists = errors.New("workflow status name exists")
var errStatusNotFound = errors.New("workflow status not found")
var errStatusArchived = errors.New("workflow status is archived")
var errRequiresDoneStatus = errors.New("workflow requires a done status")
var errStatusOrderMismatch = errors.New("workflow status order mismatch")
var errInvalidTransitionStatus = errors.New("invalid workflow transition status")

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

type createStatusRequest struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Category string `json:"category"`
}

type updateStatusRequest struct {
	Name     *string `json:"name"`
	Color    *string `json:"color"`
	Category *string `json:"category"`
}

type statusOrderRequest struct {
	StatusIDs []string `json:"status_ids"`
}

type archiveStatusRequest struct {
	ReplacementStatusID string `json:"replacement_status_id"`
}

type transitionRequest struct {
	FromStatusID string `json:"from_status_id"`
	ToStatusID   string `json:"to_status_id"`
}

type replaceTransitionsRequest struct {
	Transitions []transitionRequest `json:"transitions"`
}

type normalizedCreateStatus struct {
	Key      string
	Name     string
	Color    string
	Category string
}

type normalizedUpdateStatus struct {
	Name        string
	HasName     bool
	Color       string
	HasColor    bool
	Category    string
	HasCategory bool
}

type normalizedTransition struct {
	FromStatusID string
	ToStatusID   string
}

type statusResponse struct {
	ID         string     `json:"id"`
	ProjectID  string     `json:"project_id"`
	Key        string     `json:"key"`
	Name       string     `json:"name"`
	Color      string     `json:"color"`
	Category   string     `json:"category"`
	Position   int        `json:"position"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ArchivedAt *time.Time `json:"archived_at"`
}

type transitionResponse struct {
	FromStatusID string    `json:"from_status_id"`
	ToStatusID   string    `json:"to_status_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type workflowResponse struct {
	ProjectID   string               `json:"project_id"`
	Statuses    []statusResponse     `json:"statuses"`
	Transitions []transitionResponse `json:"transitions"`
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{db: db, auth: authHandler}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/projects/{id}/workflow", h.get)
	mux.HandleFunc("POST /api/v1/projects/{id}/workflow/statuses", h.createStatus)
	mux.HandleFunc("PATCH /api/v1/projects/{id}/workflow/statuses/{statusID}", h.updateStatus)
	mux.HandleFunc("PUT /api/v1/projects/{id}/workflow/statuses/order", h.reorderStatuses)
	mux.HandleFunc("POST /api/v1/projects/{id}/workflow/statuses/{statusID}/archive", h.archiveStatus)
	mux.HandleFunc("PUT /api/v1/projects/{id}/workflow/transitions", h.replaceTransitions)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	projectID, ok := normalizePathID(w, r.PathValue("id"), "project id")
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	workflow, err := h.getWorkflow(ctx, user.WorkspaceID, projectID, user)
	if err != nil {
		h.writeStoreError(w, err, "could not load workflow")
		return
	}
	writeJSON(w, http.StatusOK, workflow)
}

func (h *Handler) createStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	projectID, ok := normalizePathID(w, r.PathValue("id"), "project id")
	if !ok {
		return
	}
	var req createStatusRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input, err := normalizeCreateStatus(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	status, err := h.createWorkflowStatus(ctx, user.WorkspaceID, projectID, input, user)
	if err != nil {
		h.writeStoreError(w, err, "could not create workflow status")
		return
	}
	writeJSON(w, http.StatusCreated, status)
}

func (h *Handler) updateStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	projectID, ok := normalizePathID(w, r.PathValue("id"), "project id")
	if !ok {
		return
	}
	statusID, ok := normalizePathID(w, r.PathValue("statusID"), "status id")
	if !ok {
		return
	}
	var req updateStatusRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input, err := normalizeUpdateStatus(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	status, err := h.updateWorkflowStatus(ctx, user.WorkspaceID, projectID, statusID, input, user)
	if err != nil {
		h.writeStoreError(w, err, "could not update workflow status")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) reorderStatuses(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	projectID, ok := normalizePathID(w, r.PathValue("id"), "project id")
	if !ok {
		return
	}
	var req statusOrderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	statusIDs, err := normalizeStatusOrder(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.reorderWorkflowStatuses(ctx, user.WorkspaceID, projectID, statusIDs, user); err != nil {
		h.writeStoreError(w, err, "could not reorder workflow statuses")
		return
	}
	workflow, err := h.getWorkflow(ctx, user.WorkspaceID, projectID, user)
	if err != nil {
		h.writeStoreError(w, err, "could not load workflow")
		return
	}
	writeJSON(w, http.StatusOK, workflow)
}

func (h *Handler) archiveStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	projectID, ok := normalizePathID(w, r.PathValue("id"), "project id")
	if !ok {
		return
	}
	statusID, ok := normalizePathID(w, r.PathValue("statusID"), "status id")
	if !ok {
		return
	}
	var req archiveStatusRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	replacementID, err := normalizeWorkflowID(req.ReplacementStatusID, "replacement_status_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if statusID == replacementID {
		writeError(w, http.StatusBadRequest, "invalid_request", "replacement_status_id must be different from status id")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	status, err := h.archiveWorkflowStatus(ctx, user, projectID, statusID, replacementID)
	if err != nil {
		h.writeStoreError(w, err, "could not archive workflow status")
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) replaceTransitions(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	projectID, ok := normalizePathID(w, r.PathValue("id"), "project id")
	if !ok {
		return
	}
	var req replaceTransitionsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	transitions, err := normalizeTransitions(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.replaceWorkflowTransitions(ctx, user.WorkspaceID, projectID, transitions, user); err != nil {
		h.writeStoreError(w, err, "could not replace workflow transitions")
		return
	}
	workflow, err := h.getWorkflow(ctx, user.WorkspaceID, projectID, user)
	if err != nil {
		h.writeStoreError(w, err, "could not load workflow")
		return
	}
	writeJSON(w, http.StatusOK, workflow)
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

func (h *Handler) writeStoreError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, "project_not_found", "project was not found")
	case errors.Is(err, projectaccess.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "project lead or workspace admin role is required")
	case errors.Is(err, errStatusNotFound):
		writeError(w, http.StatusNotFound, "workflow_status_not_found", "workflow status was not found")
	case errors.Is(err, errStatusArchived):
		writeError(w, http.StatusConflict, "workflow_status_archived", "workflow status is archived")
	case errors.Is(err, errStatusKeyExists):
		writeError(w, http.StatusConflict, "workflow_status_key_exists", "workflow status key already exists")
	case errors.Is(err, errStatusNameExists):
		writeError(w, http.StatusConflict, "workflow_status_name_exists", "workflow status name already exists")
	case errors.Is(err, errRequiresDoneStatus):
		writeError(w, http.StatusConflict, "workflow_requires_done_status", "workflow requires at least one active done status")
	case errors.Is(err, errStatusOrderMismatch), errors.Is(err, errInvalidTransitionStatus):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			writeError(w, http.StatusNotFound, "workflow_status_not_found", "workflow status was not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", fallback)
	}
}

func normalizePathID(w http.ResponseWriter, value string, field string) (string, bool) {
	id, err := normalizeWorkflowID(value, field)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return "", false
	}
	return id, true
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
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
