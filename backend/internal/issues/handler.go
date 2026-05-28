package issues

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"team-task-tracker/backend/internal/auth"
)

var validIssueTypes = map[string]bool{
	"task":    true,
	"bug":     true,
	"story":   true,
	"epic":    true,
	"subtask": true,
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

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

var errCommentForbidden = errors.New("comment update is forbidden")

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

type createIssueRequest struct {
	ProjectID     string   `json:"project_id"`
	ParentIssueID string   `json:"parent_issue_id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	IssueType     string   `json:"issue_type"`
	Status        string   `json:"status"`
	Priority      string   `json:"priority"`
	AssigneeID    string   `json:"assignee_id"`
	DueDate       string   `json:"due_date"`
	LabelIDs      []string `json:"label_ids"`
}

type createSubtaskRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	AssigneeID  string   `json:"assignee_id"`
	DueDate     string   `json:"due_date"`
	LabelIDs    []string `json:"label_ids"`
}

type updateIssueRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	IssueType   string `json:"issue_type"`
	Priority    string `json:"priority"`
	DueDate     string `json:"due_date"`
}

type transitionIssueRequest struct {
	Status string `json:"status"`
}

type assignIssueRequest struct {
	AssigneeID string `json:"assignee_id"`
}

type setIssueLabelsRequest struct {
	LabelIDs []string `json:"label_ids"`
}

type setIssueParentRequest struct {
	ParentIssueID *string `json:"parent_issue_id"`
}

type createCommentRequest struct {
	Body string `json:"body"`
}

type updateCommentRequest struct {
	Body string `json:"body"`
}

type issueLabelResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type issueResponse struct {
	ID            string               `json:"id"`
	ProjectID     string               `json:"project_id"`
	ProjectKey    string               `json:"project_key"`
	Number        int                  `json:"number"`
	IssueKey      string               `json:"issue_key"`
	Title         string               `json:"title"`
	Description   string               `json:"description"`
	IssueType     string               `json:"issue_type"`
	Status        string               `json:"status"`
	Priority      string               `json:"priority"`
	ReporterID    string               `json:"reporter_id"`
	AssigneeID    *string              `json:"assignee_id"`
	ParentIssueID *string              `json:"parent_issue_id"`
	DueDate       *string              `json:"due_date"`
	Labels        []issueLabelResponse `json:"labels"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

type listIssuesResponse struct {
	Issues []issueResponse `json:"issues"`
}

type issueCommentResponse struct {
	ID                string    `json:"id"`
	IssueID           string    `json:"issue_id"`
	AuthorID          string    `json:"author_id"`
	AuthorDisplayName string    `json:"author_display_name"`
	Body              string    `json:"body"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type listCommentsResponse struct {
	Comments []issueCommentResponse `json:"comments"`
}

type issueActivityResponse struct {
	ID               string            `json:"id"`
	IssueID          string            `json:"issue_id"`
	Action           string            `json:"action"`
	ActorID          *string           `json:"actor_id"`
	ActorDisplayName *string           `json:"actor_display_name"`
	Payload          map[string]string `json:"payload"`
	CreatedAt        time.Time         `json:"created_at"`
}

type listActivityResponse struct {
	Activity []issueActivityResponse `json:"activity"`
}

type normalizedCreateIssue struct {
	ProjectID     string
	ParentIssueID string
	Title         string
	Description   string
	IssueType     string
	Status        string
	Priority      string
	AssigneeID    string
	DueDate       string
	LabelIDs      []string
}

type normalizedUpdateIssue struct {
	Title       string
	Description string
	IssueType   string
	Priority    string
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
	mux.HandleFunc("GET /api/v1/issues/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/issues/{id}", h.update)
	mux.HandleFunc("GET /api/v1/issues/{id}/children", h.listChildren)
	mux.HandleFunc("POST /api/v1/issues/{id}/subtasks", h.createSubtask)
	mux.HandleFunc("PATCH /api/v1/issues/{id}/parent", h.setParent)
	mux.HandleFunc("POST /api/v1/issues/{id}/transition", h.transition)
	mux.HandleFunc("POST /api/v1/issues/{id}/assign", h.assign)
	mux.HandleFunc("PUT /api/v1/issues/{id}/labels", h.setLabels)
	mux.HandleFunc("POST /api/v1/issues/{id}/archive", h.archive)
	mux.HandleFunc("GET /api/v1/issues/{id}/comments", h.listComments)
	mux.HandleFunc("POST /api/v1/issues/{id}/comments", h.createComment)
	mux.HandleFunc("PATCH /api/v1/issues/{id}/comments/{commentID}", h.updateComment)
	mux.HandleFunc("DELETE /api/v1/issues/{id}/comments/{commentID}", h.deleteComment)
	mux.HandleFunc("PATCH /api/v1/comments/{id}", h.updateCommentByID)
	mux.HandleFunc("DELETE /api/v1/comments/{id}", h.deleteCommentByID)
	mux.HandleFunc("GET /api/v1/issues/{id}/activity", h.listActivity)
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
			writeError(w, http.StatusBadRequest, "invalid_assignee", "assignee must be an active workspace member")
			return
		}
		if errors.Is(err, errInvalidLabel) {
			writeError(w, http.StatusBadRequest, "invalid_label", "label is not in this workspace")
			return
		}
		if errors.Is(err, errInvalidIssueParent) {
			writeError(w, http.StatusBadRequest, "invalid_parent", "parent issue must be accessible in this workspace")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not create issue")
		return
	}

	writeJSON(w, http.StatusCreated, issue)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	issue, err := h.getIssue(ctx, user.WorkspaceID, issueID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load issue")
		return
	}

	writeJSON(w, http.StatusOK, issue)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req updateIssueRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input, err := normalizeUpdateIssue(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	issue, err := h.updateIssue(ctx, user, issueID, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}
		if errors.Is(err, errIssueParentRequired) {
			writeError(w, http.StatusBadRequest, "invalid_parent", "parent_issue_id is required for subtask")
			return
		}
		if errors.Is(err, errIssueParentForbidden) {
			writeError(w, http.StatusBadRequest, "invalid_parent", "epic cannot have a parent issue")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not update issue")
		return
	}

	writeJSON(w, http.StatusOK, issue)
}

func (h *Handler) listChildren(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := h.getIssue(ctx, user.WorkspaceID, issueID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load issue")
		return
	}

	children, err := h.listIssueChildren(ctx, user.WorkspaceID, issueID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list child issues")
		return
	}

	writeJSON(w, http.StatusOK, listIssuesResponse{Issues: children})
}

func (h *Handler) createSubtask(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	parentIssueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req createSubtaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	parent, err := h.getIssue(ctx, user.WorkspaceID, parentIssueID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "parent issue was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load parent issue")
		return
	}

	input, err := normalizeCreateSubtask(parent, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	issue, err := h.createIssue(ctx, user, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "parent issue was not found")
			return
		}
		if errors.Is(err, errInvalidAssignee) {
			writeError(w, http.StatusBadRequest, "invalid_assignee", "assignee must be an active workspace member")
			return
		}
		if errors.Is(err, errInvalidLabel) {
			writeError(w, http.StatusBadRequest, "invalid_label", "label is not in this workspace")
			return
		}
		if errors.Is(err, errInvalidIssueParent) {
			writeError(w, http.StatusBadRequest, "invalid_parent", "parent issue must be accessible in this workspace")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not create subtask")
		return
	}

	writeJSON(w, http.StatusCreated, issue)
}

func (h *Handler) setParent(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req setIssueParentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	parentIssueID, err := normalizeSetIssueParent(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	issue, err := h.setIssueParent(ctx, user, issueID, parentIssueID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}
		if errors.Is(err, errInvalidIssueParent) {
			writeError(w, http.StatusBadRequest, "invalid_parent", "parent issue must be accessible in this workspace")
			return
		}
		if errors.Is(err, errIssueParentCycle) {
			writeError(w, http.StatusBadRequest, "invalid_parent", "parent issue cannot create a hierarchy cycle")
			return
		}
		if errors.Is(err, errIssueParentRequired) {
			writeError(w, http.StatusBadRequest, "invalid_parent", "parent_issue_id is required for subtask")
			return
		}
		if errors.Is(err, errIssueParentForbidden) {
			writeError(w, http.StatusBadRequest, "invalid_parent", "epic cannot have a parent issue")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not update issue parent")
		return
	}

	writeJSON(w, http.StatusOK, issue)
}

func (h *Handler) transition(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
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

	issue, err := h.transitionIssueStatus(ctx, user, issueID, status)
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

func (h *Handler) assign(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req assignIssueRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	assigneeID, err := normalizeOptionalUserID(req.AssigneeID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	issue, err := h.assignIssue(ctx, user, issueID, assigneeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}
		if errors.Is(err, errInvalidAssignee) {
			writeError(w, http.StatusBadRequest, "invalid_assignee", "assignee must be an active workspace member")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not assign issue")
		return
	}

	writeJSON(w, http.StatusOK, issue)
}

func (h *Handler) setLabels(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req setIssueLabelsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	labelIDs, err := normalizeIssueLabelIDs(req.LabelIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	issue, err := h.setIssueLabels(ctx, user, issueID, labelIDs)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}
		if errors.Is(err, errInvalidLabel) {
			writeError(w, http.StatusBadRequest, "invalid_label", "label is not in this workspace")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not update issue labels")
		return
	}

	writeJSON(w, http.StatusOK, issue)
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.archiveIssue(ctx, user, issueID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not archive issue")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listComments(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	comments, err := h.listIssueComments(ctx, user.WorkspaceID, issueID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list comments")
		return
	}

	writeJSON(w, http.StatusOK, listCommentsResponse{Comments: comments})
}

func (h *Handler) createComment(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req createCommentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	body, err := normalizeCommentBody(req.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	comment, err := h.createIssueComment(ctx, user, issueID, body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not create comment")
		return
	}

	writeJSON(w, http.StatusCreated, comment)
}

func (h *Handler) updateComment(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	commentID, err := normalizeCommentID(r.PathValue("commentID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req updateCommentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	body, err := normalizeCommentBody(req.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	comment, err := h.updateIssueComment(ctx, user, issueID, commentID, body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "comment_not_found", "comment was not found")
			return
		}
		if errors.Is(err, errCommentForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "only the comment author or an admin can edit this comment")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not update comment")
		return
	}

	writeJSON(w, http.StatusOK, comment)
}

func (h *Handler) deleteComment(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	commentID, err := normalizeCommentID(r.PathValue("commentID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.deleteIssueComment(ctx, user, issueID, commentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "comment_not_found", "comment was not found")
			return
		}
		if errors.Is(err, errCommentForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "only the comment author or an admin can delete this comment")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not delete comment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) updateCommentByID(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	commentID, err := normalizeCommentID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req updateCommentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	body, err := normalizeCommentBody(req.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	issueID, err := h.issueIDForComment(ctx, user.WorkspaceID, commentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "comment_not_found", "comment was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load comment")
		return
	}

	comment, err := h.updateIssueComment(ctx, user, issueID, commentID, body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "comment_not_found", "comment was not found")
			return
		}
		if errors.Is(err, errCommentForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "only the comment author or an admin can edit this comment")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not update comment")
		return
	}

	writeJSON(w, http.StatusOK, comment)
}

func (h *Handler) deleteCommentByID(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	commentID, err := normalizeCommentID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	issueID, err := h.issueIDForComment(ctx, user.WorkspaceID, commentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "comment_not_found", "comment was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load comment")
		return
	}

	if err := h.deleteIssueComment(ctx, user, issueID, commentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "comment_not_found", "comment was not found")
			return
		}
		if errors.Is(err, errCommentForbidden) {
			writeError(w, http.StatusForbidden, "forbidden", "only the comment author or an admin can delete this comment")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not delete comment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listActivity(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := h.getIssue(ctx, user.WorkspaceID, issueID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load issue")
		return
	}

	activity, err := h.listIssueActivity(ctx, user.WorkspaceID, issueID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list activity")
		return
	}

	writeJSON(w, http.StatusOK, listActivityResponse{Activity: activity})
}

func (h *Handler) listIssues(ctx context.Context, workspaceID string, query map[string][]string) ([]issueResponse, error) {
	args := []any{workspaceID}
	conditions := []string{
		"p.workspace_id = $1",
		"p.archived_at IS NULL",
		"i.archived_at IS NULL",
	}

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
	if dueCondition := issueDueFilterCondition(firstQueryValue(query, "due")); dueCondition != "" {
		conditions = append(conditions, dueCondition)
	}

	assigneeID := strings.TrimSpace(firstQueryValue(query, "assignee_id"))
	if assigneeID == "unassigned" {
		conditions = append(conditions, "i.assignee_id IS NULL")
	} else {
		addFilter("i.assignee_id", assigneeID)
	}
	labelID := strings.TrimSpace(firstQueryValue(query, "label_id"))
	if labelID != "" {
		args = append(args, labelID)
		conditions = append(conditions, fmt.Sprintf(`
			EXISTS (
				SELECT 1
				FROM issue_labels il_filter
				WHERE il_filter.issue_id = i.id
					AND il_filter.label_id = $%d
			)
		`, len(args)))
	}
	searchQuery := strings.TrimSpace(firstQueryValue(query, "q"))
	if searchQuery != "" {
		args = append(args, issueSearchPattern(searchQuery))
		conditions = append(conditions, fmt.Sprintf(`
			(
				i.issue_key ILIKE $%d ESCAPE '\'
				OR i.title ILIKE $%d ESCAPE '\'
				OR i.description ILIKE $%d ESCAPE '\'
			)
		`, len(args), len(args), len(args)))
	}

	orderClause := issueListOrderClause(firstQueryValue(query, "sort"))
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
			i.parent_issue_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at,
			(
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'id', l.id::text,
							'name', l.name,
							'color', l.color
						)
						ORDER BY l.name
					),
					'[]'::jsonb
				)
				FROM issue_labels il
				JOIN labels l ON l.id = il.label_id
				WHERE il.issue_id = i.id
			)
		FROM issues i
		JOIN projects p ON p.id = i.project_id
		WHERE %s
		ORDER BY %s
		LIMIT 100
	`, strings.Join(conditions, " AND "), orderClause)

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
var errInvalidLabel = errors.New("invalid label")
var errInvalidIssueParent = errors.New("invalid issue parent")
var errIssueParentCycle = errors.New("issue parent cycle")
var errIssueParentRequired = errors.New("issue parent required")
var errIssueParentForbidden = errors.New("issue parent forbidden")

func (h *Handler) getIssue(ctx context.Context, workspaceID string, issueID string) (issueResponse, error) {
	return scanIssue(h.db.QueryRow(ctx, `
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
			i.parent_issue_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at,
			(
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'id', l.id::text,
							'name', l.name,
							'color', l.color
						)
						ORDER BY l.name
					),
					'[]'::jsonb
				)
				FROM issue_labels il
				JOIN labels l ON l.id = il.label_id
				WHERE il.issue_id = i.id
			)
		FROM issues i
		JOIN projects p ON p.id = i.project_id
	WHERE i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
	`, issueID, workspaceID))
}

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

	if err := verifyActiveWorkspaceMember(ctx, tx, user.WorkspaceID, input.AssigneeID); err != nil {
		return issueResponse{}, err
	}

	if err := verifyWorkspaceLabels(ctx, tx, user.WorkspaceID, input.LabelIDs); err != nil {
		return issueResponse{}, err
	}

	if err := verifyIssueParent(ctx, tx, user.WorkspaceID, "", input.ParentIssueID); err != nil {
		return issueResponse{}, err
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
	var parentIssueID any
	if input.ParentIssueID != "" {
		parentIssueID = input.ParentIssueID
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
			parent_issue_id,
			due_date
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING
			id::text,
			project_id::text,
			$13::text,
			number,
			issue_key,
			title,
			description,
			issue_type,
			status,
			priority,
			reporter_id::text,
			assignee_id::text,
			parent_issue_id::text,
			due_date::text,
			created_at,
			updated_at,
			'[]'::jsonb
	`, input.ProjectID, nextNumber, issueKey, input.Title, input.Description, input.IssueType, input.Status, input.Priority, user.ID, assigneeID, parentIssueID, dueDate, projectKey))
	if err != nil {
		return issueResponse{}, err
	}

	for _, labelID := range input.LabelIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO issue_labels (issue_id, label_id)
			VALUES ($1, $2)
		`, issue.ID, labelID); err != nil {
			return issueResponse{}, err
		}
	}

	if len(input.LabelIDs) > 0 {
		issue, err = getIssueInTx(ctx, tx, user.WorkspaceID, issue.ID)
		if err != nil {
			return issueResponse{}, err
		}
	}

	activityPayload := map[string]string{
		"issue_key": issue.IssueKey,
		"title":     issue.Title,
		"status":    issue.Status,
		"priority":  issue.Priority,
	}
	if input.ParentIssueID != "" {
		activityPayload["parent_issue_id"] = input.ParentIssueID
	}
	if err := insertIssueActivity(ctx, tx, issue.ID, user.ID, "issue_created", activityPayload); err != nil {
		return issueResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return issueResponse{}, err
	}

	return issue, nil
}

func (h *Handler) updateIssue(ctx context.Context, user auth.CurrentUser, issueID string, input normalizedUpdateIssue) (issueResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return issueResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	previous, err := scanIssue(tx.QueryRow(ctx, `
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
			i.parent_issue_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at,
			(
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'id', l.id::text,
							'name', l.name,
							'color', l.color
						)
						ORDER BY l.name
					),
					'[]'::jsonb
				)
				FROM issue_labels il
				JOIN labels l ON l.id = il.label_id
				WHERE il.issue_id = i.id
			)
		FROM issues i
		JOIN projects p ON p.id = i.project_id
	WHERE i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
	FOR UPDATE OF i
	`, issueID, user.WorkspaceID))
	if err != nil {
		return issueResponse{}, err
	}

	if input.IssueType == "subtask" && stringOrEmpty(previous.ParentIssueID) == "" {
		return issueResponse{}, errIssueParentRequired
	}
	if input.IssueType == "epic" && stringOrEmpty(previous.ParentIssueID) != "" {
		return issueResponse{}, errIssueParentForbidden
	}

	var dueDate any
	if input.DueDate != "" {
		dueDate = input.DueDate
	}

	issue, err := scanIssue(tx.QueryRow(ctx, `
		UPDATE issues i
		SET title = $3,
			description = $4,
			issue_type = $5,
			priority = $6,
			due_date = $7::date,
			updated_at = now()
		FROM projects p
		WHERE i.project_id = p.id
		AND i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
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
			i.parent_issue_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at,
			(
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'id', l.id::text,
							'name', l.name,
							'color', l.color
						)
						ORDER BY l.name
					),
					'[]'::jsonb
				)
				FROM issue_labels il
				JOIN labels l ON l.id = il.label_id
				WHERE il.issue_id = i.id
			)
	`, issueID, user.WorkspaceID, input.Title, input.Description, input.IssueType, input.Priority, dueDate))
	if err != nil {
		return issueResponse{}, err
	}

	changedFields := changedIssueFields(previous, issue)
	if len(changedFields) > 0 {
		if err := insertIssueActivity(ctx, tx, issue.ID, user.ID, "issue_updated", map[string]string{
			"fields": strings.Join(changedFields, ","),
		}); err != nil {
			return issueResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return issueResponse{}, err
	}

	return issue, nil
}

func (h *Handler) listIssueChildren(ctx context.Context, workspaceID string, issueID string) ([]issueResponse, error) {
	rows, err := h.db.Query(ctx, `
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
			i.parent_issue_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at,
			(
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'id', l.id::text,
							'name', l.name,
							'color', l.color
						)
						ORDER BY l.name
					),
					'[]'::jsonb
				)
				FROM issue_labels il
				JOIN labels l ON l.id = il.label_id
				WHERE il.issue_id = i.id
			)
		FROM issues i
		JOIN projects p ON p.id = i.project_id
		WHERE i.parent_issue_id = $1
			AND p.workspace_id = $2
			AND p.archived_at IS NULL
			AND i.archived_at IS NULL
		ORDER BY i.number ASC
		LIMIT 100
	`, issueID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	children := make([]issueResponse, 0)
	for rows.Next() {
		issue, err := scanIssue(rows)
		if err != nil {
			return nil, err
		}

		children = append(children, issue)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return children, nil
}

func (h *Handler) setIssueParent(ctx context.Context, user auth.CurrentUser, issueID string, parentIssueID string) (issueResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return issueResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	previous, err := scanIssue(tx.QueryRow(ctx, `
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
			i.parent_issue_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at,
			(
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'id', l.id::text,
							'name', l.name,
							'color', l.color
						)
						ORDER BY l.name
					),
					'[]'::jsonb
				)
				FROM issue_labels il
				JOIN labels l ON l.id = il.label_id
				WHERE il.issue_id = i.id
			)
		FROM issues i
		JOIN projects p ON p.id = i.project_id
	WHERE i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
	FOR UPDATE OF i
	`, issueID, user.WorkspaceID))
	if err != nil {
		return issueResponse{}, err
	}

	if previous.IssueType == "epic" && parentIssueID != "" {
		return issueResponse{}, errIssueParentForbidden
	}
	if previous.IssueType == "subtask" && parentIssueID == "" {
		return issueResponse{}, errIssueParentRequired
	}

	previousParentIssueID := stringOrEmpty(previous.ParentIssueID)
	if previousParentIssueID == parentIssueID {
		if err := tx.Commit(ctx); err != nil {
			return issueResponse{}, err
		}
		return previous, nil
	}

	if err := verifyIssueParent(ctx, tx, user.WorkspaceID, issueID, parentIssueID); err != nil {
		return issueResponse{}, err
	}

	var nextParentIssueID any
	if parentIssueID != "" {
		nextParentIssueID = parentIssueID
	}

	issue, err := scanIssue(tx.QueryRow(ctx, `
		UPDATE issues i
		SET parent_issue_id = $3::uuid,
			updated_at = now()
		FROM projects p
		WHERE i.project_id = p.id
			AND i.id = $1
			AND p.workspace_id = $2
			AND p.archived_at IS NULL
			AND i.archived_at IS NULL
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
			i.parent_issue_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at,
			(
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'id', l.id::text,
							'name', l.name,
							'color', l.color
						)
						ORDER BY l.name
					),
					'[]'::jsonb
				)
				FROM issue_labels il
				JOIN labels l ON l.id = il.label_id
				WHERE il.issue_id = i.id
			)
	`, issueID, user.WorkspaceID, nextParentIssueID))
	if err != nil {
		return issueResponse{}, err
	}

	if err := insertIssueActivity(ctx, tx, issue.ID, user.ID, "issue_parent_changed", map[string]string{
		"from_parent_issue_id": previousParentIssueID,
		"to_parent_issue_id":   stringOrEmpty(issue.ParentIssueID),
	}); err != nil {
		return issueResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return issueResponse{}, err
	}

	return issue, nil
}

func (h *Handler) transitionIssueStatus(ctx context.Context, user auth.CurrentUser, issueID string, status string) (issueResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return issueResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var previousStatus string
	if err := tx.QueryRow(ctx, `
		SELECT i.status
		FROM issues i
		JOIN projects p ON p.id = i.project_id
	WHERE i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
	FOR UPDATE OF i
	`, issueID, user.WorkspaceID).Scan(&previousStatus); err != nil {
		return issueResponse{}, err
	}

	issue, err := scanIssue(tx.QueryRow(ctx, `
		UPDATE issues i
		SET status = $3,
			updated_at = now()
		FROM projects p
		WHERE i.project_id = p.id
		AND i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
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
			i.parent_issue_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at,
			(
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'id', l.id::text,
							'name', l.name,
							'color', l.color
						)
						ORDER BY l.name
					),
					'[]'::jsonb
				)
				FROM issue_labels il
				JOIN labels l ON l.id = il.label_id
				WHERE il.issue_id = i.id
			)
	`, issueID, user.WorkspaceID, status))
	if err != nil {
		return issueResponse{}, err
	}

	if previousStatus != issue.Status {
		if err := insertIssueActivity(ctx, tx, issue.ID, user.ID, "status_changed", map[string]string{
			"from_status": previousStatus,
			"to_status":   issue.Status,
		}); err != nil {
			return issueResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return issueResponse{}, err
	}

	return issue, nil
}

func (h *Handler) assignIssue(ctx context.Context, user auth.CurrentUser, issueID string, assigneeID string) (issueResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return issueResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var previousAssigneeID pgtype.Text
	if err := tx.QueryRow(ctx, `
		SELECT i.assignee_id::text
		FROM issues i
		JOIN projects p ON p.id = i.project_id
	WHERE i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
	FOR UPDATE OF i
	`, issueID, user.WorkspaceID).Scan(&previousAssigneeID); err != nil {
		return issueResponse{}, err
	}

	if err := verifyActiveWorkspaceMember(ctx, tx, user.WorkspaceID, assigneeID); err != nil {
		return issueResponse{}, err
	}

	var nextAssigneeID any
	if assigneeID != "" {
		nextAssigneeID = assigneeID
	}

	issue, err := scanIssue(tx.QueryRow(ctx, `
		UPDATE issues i
		SET assignee_id = $3::uuid,
			updated_at = now()
		FROM projects p
		WHERE i.project_id = p.id
		AND i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
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
			i.parent_issue_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at,
			(
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'id', l.id::text,
							'name', l.name,
							'color', l.color
						)
						ORDER BY l.name
					),
					'[]'::jsonb
				)
				FROM issue_labels il
				JOIN labels l ON l.id = il.label_id
				WHERE il.issue_id = i.id
			)
	`, issueID, user.WorkspaceID, nextAssigneeID))
	if err != nil {
		return issueResponse{}, err
	}

	previous := textOrEmpty(previousAssigneeID)
	current := stringOrEmpty(issue.AssigneeID)
	if previous != current {
		if err := insertIssueActivity(ctx, tx, issue.ID, user.ID, "assignee_changed", map[string]string{
			"from_assignee_id": previous,
			"to_assignee_id":   current,
		}); err != nil {
			return issueResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return issueResponse{}, err
	}

	return issue, nil
}

func (h *Handler) setIssueLabels(ctx context.Context, user auth.CurrentUser, issueID string, labelIDs []string) (issueResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return issueResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var lockedIssueID string
	if err := tx.QueryRow(ctx, `
		SELECT i.id::text
		FROM issues i
		JOIN projects p ON p.id = i.project_id
	WHERE i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
	FOR UPDATE OF i
	`, issueID, user.WorkspaceID).Scan(&lockedIssueID); err != nil {
		return issueResponse{}, err
	}

	previousLabelIDs, err := listIssueLabelIDs(ctx, tx, issueID)
	if err != nil {
		return issueResponse{}, err
	}

	if err := verifyWorkspaceLabels(ctx, tx, user.WorkspaceID, labelIDs); err != nil {
		return issueResponse{}, err
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM issue_labels
		WHERE issue_id = $1
	`, issueID); err != nil {
		return issueResponse{}, err
	}

	for _, labelID := range labelIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO issue_labels (issue_id, label_id)
			VALUES ($1, $2)
		`, issueID, labelID); err != nil {
			return issueResponse{}, err
		}
	}

	previous := sortedStrings(previousLabelIDs)
	current := sortedStrings(labelIDs)
	labelsChanged := strings.Join(previous, ",") != strings.Join(current, ",")
	if labelsChanged {
		if _, err := tx.Exec(ctx, `
			UPDATE issues
			SET updated_at = now()
			WHERE id = $1
		`, issueID); err != nil {
			return issueResponse{}, err
		}
	}

	issue, err := scanIssue(tx.QueryRow(ctx, `
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
			i.parent_issue_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at,
			(
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'id', l.id::text,
							'name', l.name,
							'color', l.color
						)
						ORDER BY l.name
					),
					'[]'::jsonb
				)
				FROM issue_labels il
				JOIN labels l ON l.id = il.label_id
				WHERE il.issue_id = i.id
			)
		FROM issues i
		JOIN projects p ON p.id = i.project_id
	WHERE i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
	`, issueID, user.WorkspaceID))
	if err != nil {
		return issueResponse{}, err
	}

	if labelsChanged {
		if err := insertIssueActivity(ctx, tx, issue.ID, user.ID, "labels_changed", map[string]string{
			"from_label_ids": strings.Join(previous, ","),
			"to_label_ids":   strings.Join(current, ","),
		}); err != nil {
			return issueResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return issueResponse{}, err
	}

	return issue, nil
}

func (h *Handler) archiveIssue(ctx context.Context, user auth.CurrentUser, issueID string) error {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var archivedIssueID string
	var issueKey string
	if err := tx.QueryRow(ctx, `
		UPDATE issues i
		SET archived_at = now(),
			updated_at = now()
		FROM projects p
		WHERE i.project_id = p.id
			AND i.id = $1
			AND p.workspace_id = $2
			AND p.archived_at IS NULL
			AND i.archived_at IS NULL
		RETURNING i.id::text, i.issue_key
	`, issueID, user.WorkspaceID).Scan(&archivedIssueID, &issueKey); err != nil {
		return err
	}

	if err := insertIssueActivity(ctx, tx, archivedIssueID, user.ID, "issue_archived", map[string]string{
		"issue_key": issueKey,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (h *Handler) listIssueComments(ctx context.Context, workspaceID string, issueID string) ([]issueCommentResponse, error) {
	rows, err := h.db.Query(ctx, `
		SELECT
			c.id::text,
			c.issue_id::text,
			c.author_id::text,
			u.display_name,
			c.body,
			c.created_at,
			c.updated_at
		FROM comments c
		JOIN issues i ON i.id = c.issue_id
		JOIN projects p ON p.id = i.project_id
		JOIN users u ON u.id = c.author_id
	WHERE c.issue_id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
	ORDER BY c.created_at ASC
		LIMIT 100
	`, issueID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]issueCommentResponse, 0)
	for rows.Next() {
		comment, err := scanIssueComment(rows)
		if err != nil {
			return nil, err
		}

		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

func (h *Handler) createIssueComment(ctx context.Context, user auth.CurrentUser, issueID string, body string) (issueCommentResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return issueCommentResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	comment, err := scanIssueComment(tx.QueryRow(ctx, `
		WITH target_issue AS (
			SELECT i.id
			FROM issues i
			JOIN projects p ON p.id = i.project_id
	WHERE i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
),
		inserted AS (
			INSERT INTO comments (issue_id, author_id, body)
			SELECT id, $3, $4
			FROM target_issue
			RETURNING id, issue_id, author_id, body, created_at, updated_at
		)
		SELECT
			inserted.id::text,
			inserted.issue_id::text,
			inserted.author_id::text,
			u.display_name,
			inserted.body,
			inserted.created_at,
			inserted.updated_at
		FROM inserted
		JOIN users u ON u.id = inserted.author_id
	`, issueID, user.WorkspaceID, user.ID, body))
	if err != nil {
		return issueCommentResponse{}, err
	}

	if err := insertIssueActivity(ctx, tx, issueID, user.ID, "comment_added", map[string]string{
		"comment_id": comment.ID,
		"preview":    commentPreview(comment.Body),
	}); err != nil {
		return issueCommentResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return issueCommentResponse{}, err
	}

	return comment, nil
}

func (h *Handler) issueIDForComment(ctx context.Context, workspaceID string, commentID string) (string, error) {
	var issueID string
	err := h.db.QueryRow(ctx, `
		SELECT c.issue_id::text
		FROM comments c
		JOIN issues i ON i.id = c.issue_id
		JOIN projects p ON p.id = i.project_id
		WHERE c.id = $1
			AND p.workspace_id = $2
			AND p.archived_at IS NULL
			AND i.archived_at IS NULL
	`, commentID, workspaceID).Scan(&issueID)
	if err != nil {
		return "", err
	}

	return issueID, nil
}

func (h *Handler) updateIssueComment(ctx context.Context, user auth.CurrentUser, issueID string, commentID string, body string) (issueCommentResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return issueCommentResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var authorID string
	if err := tx.QueryRow(ctx, `
		SELECT c.author_id::text
		FROM comments c
		JOIN issues i ON i.id = c.issue_id
		JOIN projects p ON p.id = i.project_id
		WHERE c.id = $1
			AND c.issue_id = $2
			AND p.workspace_id = $3
			AND p.archived_at IS NULL
			AND i.archived_at IS NULL
	`, commentID, issueID, user.WorkspaceID).Scan(&authorID); err != nil {
		return issueCommentResponse{}, err
	}

	if !canEditComment(user, authorID) {
		return issueCommentResponse{}, errCommentForbidden
	}

	comment, err := scanIssueComment(tx.QueryRow(ctx, `
		UPDATE comments c
		SET body = $3,
			updated_at = now()
		FROM users u
		WHERE c.id = $1
			AND c.issue_id = $2
			AND u.id = c.author_id
		RETURNING
			c.id::text,
			c.issue_id::text,
			c.author_id::text,
			u.display_name,
			c.body,
			c.created_at,
			c.updated_at
	`, commentID, issueID, body))
	if err != nil {
		return issueCommentResponse{}, err
	}

	if err := insertIssueActivity(ctx, tx, issueID, user.ID, "comment_updated", map[string]string{
		"comment_id": comment.ID,
		"preview":    commentPreview(comment.Body),
	}); err != nil {
		return issueCommentResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return issueCommentResponse{}, err
	}

	return comment, nil
}

func (h *Handler) deleteIssueComment(ctx context.Context, user auth.CurrentUser, issueID string, commentID string) error {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var authorID string
	var body string
	if err := tx.QueryRow(ctx, `
		SELECT c.author_id::text, c.body
		FROM comments c
		JOIN issues i ON i.id = c.issue_id
		JOIN projects p ON p.id = i.project_id
		WHERE c.id = $1
			AND c.issue_id = $2
			AND p.workspace_id = $3
			AND p.archived_at IS NULL
			AND i.archived_at IS NULL
	`, commentID, issueID, user.WorkspaceID).Scan(&authorID, &body); err != nil {
		return err
	}

	if !canEditComment(user, authorID) {
		return errCommentForbidden
	}

	if err := insertIssueActivity(ctx, tx, issueID, user.ID, "comment_deleted", map[string]string{
		"comment_id": commentID,
		"preview":    commentPreview(body),
	}); err != nil {
		return err
	}

	commandTag, err := tx.Exec(ctx, `
		DELETE FROM comments
		WHERE id = $1
			AND issue_id = $2
	`, commentID, issueID)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return tx.Commit(ctx)
}

func (h *Handler) listIssueActivity(ctx context.Context, workspaceID string, issueID string) ([]issueActivityResponse, error) {
	rows, err := h.db.Query(ctx, `
		SELECT
			al.id::text,
			al.entity_id::text,
			al.action,
			al.actor_id::text,
			u.display_name,
			al.payload::text,
			al.created_at
		FROM activity_log al
		JOIN issues i ON i.id = al.entity_id
		JOIN projects p ON p.id = i.project_id
		LEFT JOIN users u ON u.id = al.actor_id
	WHERE al.entity_type = 'issue'
		AND al.entity_id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
	ORDER BY al.created_at DESC
		LIMIT 100
	`, issueID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activity := make([]issueActivityResponse, 0)
	for rows.Next() {
		entry, err := scanIssueActivity(rows)
		if err != nil {
			return nil, err
		}

		activity = append(activity, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return activity, nil
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
	labelIDs, err := normalizeIssueLabelIDs(req.LabelIDs)
	if err != nil {
		return normalizedCreateIssue{}, err
	}

	input := normalizedCreateIssue{
		ProjectID:   strings.TrimSpace(req.ProjectID),
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		IssueType:   withDefault(strings.TrimSpace(req.IssueType), "task"),
		Status:      withDefault(strings.TrimSpace(req.Status), "todo"),
		Priority:    withDefault(strings.TrimSpace(req.Priority), "medium"),
		AssigneeID:  strings.TrimSpace(req.AssigneeID),
		DueDate:     strings.TrimSpace(req.DueDate),
		LabelIDs:    labelIDs,
	}

	parentIssueID, err := normalizeOptionalIssueID(req.ParentIssueID)
	if err != nil {
		return input, err
	}
	input.ParentIssueID = parentIssueID

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
	if input.IssueType == "subtask" {
		if input.ParentIssueID == "" {
			return input, errors.New("parent_issue_id is required for subtask")
		}
	}
	if input.IssueType == "epic" && input.ParentIssueID != "" {
		return input, errors.New("epic cannot have a parent issue")
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

func normalizeCreateSubtask(parent issueResponse, req createSubtaskRequest) (normalizedCreateIssue, error) {
	return normalizeCreateIssue(createIssueRequest{
		ProjectID:     parent.ProjectID,
		ParentIssueID: parent.ID,
		Title:         req.Title,
		Description:   req.Description,
		IssueType:     "subtask",
		Status:        req.Status,
		Priority:      req.Priority,
		AssigneeID:    req.AssigneeID,
		DueDate:       req.DueDate,
		LabelIDs:      req.LabelIDs,
	})
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

func normalizeUpdateIssue(req updateIssueRequest) (normalizedUpdateIssue, error) {
	input := normalizedUpdateIssue{
		Title:       strings.TrimSpace(req.Title),
		Description: strings.TrimSpace(req.Description),
		IssueType:   strings.TrimSpace(req.IssueType),
		Priority:    strings.TrimSpace(req.Priority),
		DueDate:     strings.TrimSpace(req.DueDate),
	}

	if input.Title == "" {
		return input, errors.New("title is required")
	}
	if len(input.Title) > 180 {
		return input, errors.New("title must be 180 characters or fewer")
	}
	if input.IssueType == "" {
		return input, errors.New("issue_type is required")
	}
	if !validIssueTypes[input.IssueType] {
		return input, errors.New("issue_type is invalid")
	}
	if input.Priority == "" {
		return input, errors.New("priority is required")
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

func normalizeSetIssueParent(req setIssueParentRequest) (string, error) {
	if req.ParentIssueID == nil {
		return "", nil
	}

	return normalizeOptionalIssueID(*req.ParentIssueID)
}

func normalizeIssueID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New("issue id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("issue id is invalid")
	}

	return id, nil
}

func normalizeOptionalIssueID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", nil
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("parent_issue_id must be a valid issue id")
	}

	return id, nil
}

func normalizeCommentID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New("comment id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("comment id is invalid")
	}

	return id, nil
}

func canEditComment(user auth.CurrentUser, authorID string) bool {
	return user.Role == "admin" || user.ID == authorID
}

func normalizeOptionalUserID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", nil
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("assignee_id is invalid")
	}

	return id, nil
}

func normalizeIssueLabelIDs(labelIDs []string) ([]string, error) {
	if len(labelIDs) > 20 {
		return nil, errors.New("label_ids must contain 20 labels or fewer")
	}

	normalized := make([]string, 0, len(labelIDs))
	seen := make(map[string]bool, len(labelIDs))
	for _, labelID := range labelIDs {
		labelID = strings.ToLower(strings.TrimSpace(labelID))
		if labelID == "" {
			continue
		}
		if !uuidPattern.MatchString(labelID) {
			return nil, errors.New("label_ids contains an invalid label id")
		}
		if seen[labelID] {
			continue
		}

		seen[labelID] = true
		normalized = append(normalized, labelID)
	}

	return normalized, nil
}

func normalizeCommentBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", errors.New("body is required")
	}
	if len(body) > 4000 {
		return "", errors.New("body must be 4000 characters or fewer")
	}

	return body, nil
}

func commentPreview(body string) string {
	const maxPreviewLength = 120
	body = strings.TrimSpace(body)
	runes := []rune(body)
	if len(runes) <= maxPreviewLength {
		return body
	}

	return string(runes[:maxPreviewLength])
}

func activityPayloadJSON(payload map[string]string) (string, error) {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return string(encodedPayload), nil
}

func insertIssueActivity(ctx context.Context, tx pgx.Tx, issueID string, actorID string, action string, payload map[string]string) error {
	encodedPayload, err := activityPayloadJSON(payload)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO activity_log (entity_type, entity_id, action, actor_id, payload)
		VALUES ('issue', $1, $2, $3, $4::jsonb)
	`, issueID, action, actorID, encodedPayload)

	return err
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

func issueSearchPattern(query string) string {
	escapedQuery := strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	).Replace(query)

	return "%" + escapedQuery + "%"
}

func issueDueFilterCondition(dueValue string) string {
	switch strings.TrimSpace(dueValue) {
	case "overdue":
		return "i.status <> 'done' AND i.due_date < CURRENT_DATE"
	case "today":
		return "i.status <> 'done' AND i.due_date = CURRENT_DATE"
	case "due_soon":
		return "i.status <> 'done' AND i.due_date > CURRENT_DATE AND i.due_date <= CURRENT_DATE + INTERVAL '7 days'"
	case "no_due":
		return "i.due_date IS NULL"
	default:
		return ""
	}
}

func issueListOrderClause(sortValue string) string {
	switch strings.TrimSpace(sortValue) {
	case "created_asc":
		return "i.created_at ASC, i.id ASC"
	case "priority_desc":
		return "CASE i.priority WHEN 'critical' THEN 4 WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 ELSE 0 END DESC, i.created_at DESC"
	case "due_date_asc":
		return "i.due_date ASC NULLS LAST, i.created_at DESC"
	default:
		return "i.created_at DESC, i.id DESC"
	}
}

func textOrEmpty(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}

	return value.String
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func sortedStrings(values []string) []string {
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	return sorted
}

func listIssueLabelIDs(ctx context.Context, tx pgx.Tx, issueID string) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT label_id::text
		FROM issue_labels
		WHERE issue_id = $1
		ORDER BY label_id ASC
	`, issueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labelIDs := make([]string, 0)
	for rows.Next() {
		var labelID string
		if err := rows.Scan(&labelID); err != nil {
			return nil, err
		}

		labelIDs = append(labelIDs, labelID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return labelIDs, nil
}

func verifyWorkspaceLabels(ctx context.Context, tx pgx.Tx, workspaceID string, labelIDs []string) error {
	for _, labelID := range labelIDs {
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM labels
				WHERE workspace_id = $1
					AND id = $2
			)
		`, workspaceID, labelID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return errInvalidLabel
		}
	}

	return nil
}

func verifyActiveWorkspaceMember(ctx context.Context, tx pgx.Tx, workspaceID string, userID string) error {
	if userID == "" {
		return nil
	}

	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM workspace_members wm
			JOIN users u ON u.id = wm.user_id
			WHERE wm.workspace_id = $1
				AND wm.user_id = $2
				AND u.is_active = true
		)
	`, workspaceID, userID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errInvalidAssignee
	}

	return nil
}

func verifyIssueParent(ctx context.Context, tx pgx.Tx, workspaceID string, issueID string, parentIssueID string) error {
	if parentIssueID == "" {
		return nil
	}
	if issueID != "" && parentIssueID == issueID {
		return errIssueParentCycle
	}

	var parentExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM issues i
			JOIN projects p ON p.id = i.project_id
			WHERE i.id = $1
				AND p.workspace_id = $2
				AND p.archived_at IS NULL
				AND i.archived_at IS NULL
		)
	`, parentIssueID, workspaceID).Scan(&parentExists); err != nil {
		return err
	}
	if !parentExists {
		return errInvalidIssueParent
	}

	if issueID == "" {
		return nil
	}

	var createsCycle bool
	if err := tx.QueryRow(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT i.id, i.parent_issue_id
			FROM issues i
			JOIN projects p ON p.id = i.project_id
			WHERE i.id = $1
				AND p.workspace_id = $2
				AND p.archived_at IS NULL
				AND i.archived_at IS NULL

			UNION ALL

			SELECT parent.id, parent.parent_issue_id
			FROM issues parent
			JOIN ancestors child ON child.parent_issue_id = parent.id
			JOIN projects p ON p.id = parent.project_id
			WHERE p.workspace_id = $2
				AND p.archived_at IS NULL
				AND parent.archived_at IS NULL
		)
		SELECT EXISTS (
			SELECT 1
			FROM ancestors
			WHERE id = $3
		)
	`, parentIssueID, workspaceID, issueID).Scan(&createsCycle); err != nil {
		return err
	}
	if createsCycle {
		return errIssueParentCycle
	}

	return nil
}

func getIssueInTx(ctx context.Context, tx pgx.Tx, workspaceID string, issueID string) (issueResponse, error) {
	return scanIssue(tx.QueryRow(ctx, `
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
			i.parent_issue_id::text,
			i.due_date::text,
			i.created_at,
			i.updated_at,
			(
				SELECT COALESCE(
					jsonb_agg(
						jsonb_build_object(
							'id', l.id::text,
							'name', l.name,
							'color', l.color
						)
						ORDER BY l.name
					),
					'[]'::jsonb
				)
				FROM issue_labels il
				JOIN labels l ON l.id = il.label_id
				WHERE il.issue_id = i.id
			)
		FROM issues i
		JOIN projects p ON p.id = i.project_id
	WHERE i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
	`, issueID, workspaceID))
}

func changedIssueFields(previous issueResponse, current issueResponse) []string {
	fields := make([]string, 0, 5)

	if previous.Title != current.Title {
		fields = append(fields, "title")
	}
	if previous.Description != current.Description {
		fields = append(fields, "description")
	}
	if previous.IssueType != current.IssueType {
		fields = append(fields, "issue_type")
	}
	if previous.Priority != current.Priority {
		fields = append(fields, "priority")
	}
	if stringOrEmpty(previous.DueDate) != stringOrEmpty(current.DueDate) {
		fields = append(fields, "due_date")
	}

	return fields
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanIssue(row rowScanner) (issueResponse, error) {
	var issue issueResponse
	var assigneeID pgtype.Text
	var parentIssueID pgtype.Text
	var dueDate pgtype.Text
	var labelsJSON []byte

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
		&parentIssueID,
		&dueDate,
		&issue.CreatedAt,
		&issue.UpdatedAt,
		&labelsJSON,
	); err != nil {
		return issueResponse{}, err
	}

	issue.AssigneeID = nullableText(assigneeID)
	issue.ParentIssueID = nullableText(parentIssueID)
	issue.DueDate = nullableText(dueDate)
	labels, err := decodeIssueLabels(labelsJSON)
	if err != nil {
		return issueResponse{}, err
	}
	issue.Labels = labels

	return issue, nil
}

func decodeIssueLabels(labelsJSON []byte) ([]issueLabelResponse, error) {
	if len(labelsJSON) == 0 {
		return []issueLabelResponse{}, nil
	}

	var labels []issueLabelResponse
	if err := json.Unmarshal(labelsJSON, &labels); err != nil {
		return nil, err
	}
	if labels == nil {
		return []issueLabelResponse{}, nil
	}

	return labels, nil
}

func scanIssueComment(row rowScanner) (issueCommentResponse, error) {
	var comment issueCommentResponse
	if err := row.Scan(
		&comment.ID,
		&comment.IssueID,
		&comment.AuthorID,
		&comment.AuthorDisplayName,
		&comment.Body,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	); err != nil {
		return issueCommentResponse{}, err
	}

	return comment, nil
}

func scanIssueActivity(row rowScanner) (issueActivityResponse, error) {
	var entry issueActivityResponse
	var actorID pgtype.Text
	var actorDisplayName pgtype.Text
	var payloadText string

	if err := row.Scan(
		&entry.ID,
		&entry.IssueID,
		&entry.Action,
		&actorID,
		&actorDisplayName,
		&payloadText,
		&entry.CreatedAt,
	); err != nil {
		return issueActivityResponse{}, err
	}

	entry.ActorID = nullableText(actorID)
	entry.ActorDisplayName = nullableText(actorDisplayName)
	entry.Payload = map[string]string{}
	if err := json.Unmarshal([]byte(payloadText), &entry.Payload); err != nil {
		return issueActivityResponse{}, err
	}

	return entry, nil
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
