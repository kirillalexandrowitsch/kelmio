package issues

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/automations"
	"team-task-tracker/backend/internal/notifications"
	"team-task-tracker/backend/internal/pagination"
	"team-task-tracker/backend/internal/projectaccess"
)

var validIssueTypes = map[string]bool{
	"task":    true,
	"bug":     true,
	"story":   true,
	"epic":    true,
	"subtask": true,
}

var validIssuePriorities = map[string]bool{
	"low":      true,
	"medium":   true,
	"high":     true,
	"critical": true,
}

var validIssueLinkTypes = map[string]bool{
	"blocks":  true,
	"relates": true,
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var workflowStatusKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)

var errCommentForbidden = errors.New("comment update is forbidden")

type Handler struct {
	db            *pgxpool.Pool
	auth          *auth.Handler
	automations   *automations.Engine
	notifications *notifications.Service
}

type createIssueRequest struct {
	ProjectID        string   `json:"project_id"`
	ParentIssueID    string   `json:"parent_issue_id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	IssueType        string   `json:"issue_type"`
	Status           string   `json:"status"`
	WorkflowStatusID string   `json:"workflow_status_id"`
	Priority         string   `json:"priority"`
	StoryPoints      int      `json:"story_points"`
	AssigneeID       string   `json:"assignee_id"`
	DueDate          string   `json:"due_date"`
	LabelIDs         []string `json:"label_ids"`
}

type createSubtaskRequest struct {
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Status           string   `json:"status"`
	WorkflowStatusID string   `json:"workflow_status_id"`
	Priority         string   `json:"priority"`
	StoryPoints      int      `json:"story_points"`
	AssigneeID       string   `json:"assignee_id"`
	DueDate          string   `json:"due_date"`
	LabelIDs         []string `json:"label_ids"`
}

type updateIssueRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	IssueType   string `json:"issue_type"`
	Priority    string `json:"priority"`
	StoryPoints int    `json:"story_points"`
	DueDate     string `json:"due_date"`
}

type transitionIssueRequest struct {
	Status           string `json:"status"`
	WorkflowStatusID string `json:"workflow_status_id"`
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

type createIssueLinkRequest struct {
	TargetIssueID string `json:"target_issue_id"`
	LinkType      string `json:"link_type"`
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
	ID             string                 `json:"id"`
	ProjectID      string                 `json:"project_id"`
	ProjectKey     string                 `json:"project_key"`
	Number         int                    `json:"number"`
	IssueKey       string                 `json:"issue_key"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	IssueType      string                 `json:"issue_type"`
	Status         string                 `json:"status"`
	WorkflowStatus workflowStatusResponse `json:"workflow_status"`
	Priority       string                 `json:"priority"`
	StoryPoints    int                    `json:"story_points"`
	ReporterID     string                 `json:"reporter_id"`
	AssigneeID     *string                `json:"assignee_id"`
	ParentIssueID  *string                `json:"parent_issue_id"`
	SprintID       *string                `json:"sprint_id"`
	DueDate        *string                `json:"due_date"`
	Labels         []issueLabelResponse   `json:"labels"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type listIssuesResponse struct {
	Issues []issueResponse `json:"issues"`
}

type paginatedListIssuesResponse struct {
	Issues     []issueResponse `json:"issues"`
	NextCursor *string         `json:"next_cursor"`
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
	Activity   []issueActivityResponse `json:"activity"`
	NextCursor *string                 `json:"next_cursor"`
}

type issueLinkIssueResponse struct {
	ID             string                 `json:"id"`
	IssueKey       string                 `json:"issue_key"`
	Title          string                 `json:"title"`
	IssueType      string                 `json:"issue_type"`
	Status         string                 `json:"status"`
	WorkflowStatus workflowStatusResponse `json:"workflow_status"`
	Priority       string                 `json:"priority"`
}

type issueLinkResponse struct {
	ID            string                 `json:"id"`
	SourceIssueID string                 `json:"source_issue_id"`
	TargetIssueID string                 `json:"target_issue_id"`
	LinkType      string                 `json:"link_type"`
	CreatedBy     string                 `json:"created_by"`
	CreatedAt     time.Time              `json:"created_at"`
	SourceIssue   issueLinkIssueResponse `json:"source_issue"`
	TargetIssue   issueLinkIssueResponse `json:"target_issue"`
}

type listIssueLinksResponse struct {
	Links []issueLinkResponse `json:"links"`
}

type normalizedCreateIssue struct {
	ProjectID        string
	ParentIssueID    string
	Title            string
	Description      string
	IssueType        string
	Status           string
	WorkflowStatusID string
	Priority         string
	StoryPoints      int
	AssigneeID       string
	DueDate          string
	LabelIDs         []string
}

type normalizedTransitionIssue struct {
	Status           string
	WorkflowStatusID string
}

type workflowStatusResponse struct {
	ID       string `json:"id"`
	Key      string `json:"key"`
	Name     string `json:"name"`
	Color    string `json:"color"`
	Category string `json:"category"`
}

type normalizedUpdateIssue struct {
	Title       string
	Description string
	IssueType   string
	Priority    string
	StoryPoints int
	DueDate     string
}

type normalizedCreateIssueLink struct {
	TargetIssueID string
	LinkType      string
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler, notificationServices ...*notifications.Service) *Handler {
	var notificationService *notifications.Service
	if len(notificationServices) > 0 {
		notificationService = notificationServices[0]
	}

	return &Handler{
		db:            db,
		auth:          authHandler,
		automations:   automations.NewEngine(),
		notifications: notificationService,
	}
}

func (h *Handler) executeAutomations(ctx context.Context, tx pgx.Tx, user auth.CurrentUser, issueID string, triggerType string) (issueResponse, error) {
	result, err := h.automations.Execute(ctx, tx, automations.ExecuteRequest{
		WorkspaceID: user.WorkspaceID, IssueID: issueID, TriggerType: triggerType, InitiatedByUserID: user.ID,
	})
	if err != nil {
		return issueResponse{}, err
	}

	issue, err := getIssueInTx(ctx, tx, user.WorkspaceID, issueID)
	if err != nil {
		return issueResponse{}, err
	}
	if h.notifications == nil || len(result.ChangedFields) == 0 {
		return issue, nil
	}
	if err := h.notifications.NotifyAutomationChanges(ctx, tx, user.WorkspaceID, user.ID, notificationIssueContext(issue), notifications.AutomationChanges{
		AppliedRuleNames: result.AppliedRuleNames,
		ChangedFields:    result.ChangedFields,
		FromStatus:       result.FromStatus,
		ToStatus:         result.ToStatus,
		FromAssigneeID:   result.FromAssigneeID,
		ToAssigneeID:     result.ToAssigneeID,
	}); err != nil {
		return issueResponse{}, err
	}
	return issue, nil
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/issues", h.list)
	mux.HandleFunc("POST /api/v1/issues", h.create)
	mux.HandleFunc("GET /api/v1/issues/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/issues/{id}", h.update)
	mux.HandleFunc("GET /api/v1/issues/{id}/children", h.listChildren)
	mux.HandleFunc("POST /api/v1/issues/{id}/subtasks", h.createSubtask)
	mux.HandleFunc("PATCH /api/v1/issues/{id}/parent", h.setParent)
	mux.HandleFunc("GET /api/v1/issues/{id}/links", h.listLinks)
	mux.HandleFunc("POST /api/v1/issues/{id}/links", h.createLink)
	mux.HandleFunc("DELETE /api/v1/issues/{id}/links/{linkID}", h.deleteLink)
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

	page, err := pagination.Parse(r.URL.Query(), 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	issues, nextCursor, err := h.listIssuesPage(ctx, user.WorkspaceID, r.URL.Query(), page, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list issues")
		return
	}

	writeJSON(w, http.StatusOK, paginatedListIssuesResponse{Issues: issues, NextCursor: nextCursor})
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
		if h.writeAutomationError(w, err) {
			return
		}
		if h.writeProjectAccessError(w, err) {
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "project_not_found", "project was not found")
			return
		}
		if errors.Is(err, errWorkflowStatusNotFound) {
			writeError(w, http.StatusNotFound, "workflow_status_not_found", "workflow status was not found")
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

	issue, err := h.getIssue(ctx, user.WorkspaceID, issueID, user)
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
		if h.writeAutomationError(w, err) {
			return
		}
		if h.writeProjectAccessError(w, err) {
			return
		}
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

	if _, err := h.getIssue(ctx, user.WorkspaceID, issueID, user); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load issue")
		return
	}

	children, err := h.listIssueChildren(ctx, user.WorkspaceID, issueID, user)
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

	parent, err := h.getIssue(ctx, user.WorkspaceID, parentIssueID, user)
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
		if h.writeAutomationError(w, err) {
			return
		}
		if h.writeProjectAccessError(w, err) {
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "parent issue was not found")
			return
		}
		if errors.Is(err, errWorkflowStatusNotFound) {
			writeError(w, http.StatusNotFound, "workflow_status_not_found", "workflow status was not found")
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
		if h.writeProjectAccessError(w, err) {
			return
		}
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

func (h *Handler) listLinks(w http.ResponseWriter, r *http.Request) {
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

	if _, err := h.getIssue(ctx, user.WorkspaceID, issueID, user); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load issue")
		return
	}

	links, err := h.listIssueLinks(ctx, user.WorkspaceID, issueID, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list issue links")
		return
	}

	writeJSON(w, http.StatusOK, listIssueLinksResponse{Links: links})
}

func (h *Handler) createLink(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var req createIssueLinkRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input, err := normalizeCreateIssueLink(issueID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	link, err := h.createIssueLink(ctx, user, issueID, input)
	if err != nil {
		if h.writeProjectAccessError(w, err) {
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}
		if errors.Is(err, errInvalidIssueLinkTarget) {
			writeError(w, http.StatusNotFound, "issue_not_found", "target issue was not found")
			return
		}
		if errors.Is(err, errIssueLinkSelf) {
			writeError(w, http.StatusBadRequest, "invalid_link", "issue cannot be linked to itself")
			return
		}
		if errors.Is(err, errIssueLinkDuplicate) {
			writeError(w, http.StatusConflict, "duplicate_link", "issue link already exists")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not create issue link")
		return
	}

	writeJSON(w, http.StatusCreated, link)
}

func (h *Handler) deleteLink(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	linkID, err := normalizeIssueLinkID(r.PathValue("linkID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := h.getIssue(ctx, user.WorkspaceID, issueID, user); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load issue")
		return
	}

	if err := h.deleteIssueLink(ctx, user, issueID, linkID); err != nil {
		if h.writeProjectAccessError(w, err) {
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "link_not_found", "issue link was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not delete issue link")
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
		if h.writeAutomationError(w, err) {
			return
		}
		if h.writeProjectAccessError(w, err) {
			return
		}
		if errors.Is(err, errWorkflowStatusNotFound) {
			writeError(w, http.StatusNotFound, "workflow_status_not_found", "workflow status was not found")
			return
		}
		if errors.Is(err, errTransitionNotAllowed) {
			writeError(w, http.StatusConflict, "transition_not_allowed", "workflow transition is not allowed")
			return
		}
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
		if h.writeAutomationError(w, err) {
			return
		}
		if h.writeProjectAccessError(w, err) {
			return
		}
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
		if h.writeProjectAccessError(w, err) {
			return
		}
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
		if h.writeProjectAccessError(w, err) {
			return
		}
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

	if _, err := h.getIssue(ctx, user.WorkspaceID, issueID, user); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load issue")
		return
	}
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
		if h.writeProjectAccessError(w, err) {
			return
		}
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
		if h.writeProjectAccessError(w, err) {
			return
		}
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
		if h.writeProjectAccessError(w, err) {
			return
		}
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
		if h.writeProjectAccessError(w, err) {
			return
		}
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
		if h.writeProjectAccessError(w, err) {
			return
		}
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

	page, err := pagination.Parse(r.URL.Query(), 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	issueID, err := normalizeIssueID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := h.getIssue(ctx, user.WorkspaceID, issueID, user); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "issue_not_found", "issue was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load issue")
		return
	}

	activity, nextCursor, err := h.listIssueActivityPage(ctx, user.WorkspaceID, issueID, page)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list activity")
		return
	}

	writeJSON(w, http.StatusOK, listActivityResponse{Activity: activity, NextCursor: nextCursor})
}

func (h *Handler) listIssues(ctx context.Context, workspaceID string, query map[string][]string) ([]issueResponse, error) {
	issues, _, err := h.listIssuesPage(ctx, workspaceID, query, pagination.Default(100))
	return issues, err
}

func (h *Handler) listIssuesPage(ctx context.Context, workspaceID string, query map[string][]string, page pagination.Params, users ...auth.CurrentUser) ([]issueResponse, *string, error) {
	args := []any{workspaceID}
	conditions := []string{
		"p.workspace_id = $1",
		"p.archived_at IS NULL",
		"i.archived_at IS NULL",
	}
	parentIssueIDColumn := "i.parent_issue_id::text"
	if len(users) > 0 {
		args = append(args, users[0].Role == "admin", users[0].ID)
		conditions = append(conditions, fmt.Sprintf(`
			($%d::boolean OR EXISTS (
				SELECT 1
				FROM project_members project_member
				WHERE project_member.project_id = p.id
					AND project_member.user_id = $%d
			))
		`, len(args)-1, len(args)))
		parentIssueIDColumn = fmt.Sprintf(`
			CASE
				WHEN i.parent_issue_id IS NULL THEN NULL
				WHEN $%d::boolean OR EXISTS (
					SELECT 1
					FROM issues parent_issue
					JOIN projects parent_project ON parent_project.id = parent_issue.project_id
					JOIN project_members parent_member ON parent_member.project_id = parent_project.id
						AND parent_member.user_id = $%d
					WHERE parent_issue.id = i.parent_issue_id
						AND parent_issue.archived_at IS NULL
						AND parent_project.archived_at IS NULL
				) THEN i.parent_issue_id::text
				ELSE NULL
			END
		`, len(args)-1, len(args))
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
	sprintID := strings.TrimSpace(firstQueryValue(query, "sprint_id"))
	if sprintCondition := issueSprintFilterCondition(sprintID, len(args)+1); sprintCondition != "" {
		if sprintID != "none" {
			args = append(args, sprintID)
		}
		conditions = append(conditions, sprintCondition)
	}
	addFilter("i.status", firstQueryValue(query, "status"))
	addFilter("i.workflow_status_id", firstQueryValue(query, "workflow_status_id"))
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
	args = append(args, page.Limit+1, page.Offset)
	limitPlaceholder := len(args) - 1
	offsetPlaceholder := len(args)
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
			(SELECT jsonb_build_object('id', ws.id::text, 'key', ws.key, 'name', ws.name, 'color', ws.color, 'category', ws.category) FROM project_workflow_statuses ws WHERE ws.id = i.workflow_status_id),
			i.priority,
			i.story_points,
			i.reporter_id::text,
			i.assignee_id::text,
			%s,
			i.sprint_id::text,
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
		LIMIT $%d OFFSET $%d
	`, parentIssueIDColumn, strings.Join(conditions, " AND "), orderClause, limitPlaceholder, offsetPlaceholder)

	rows, err := h.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	issues := make([]issueResponse, 0)
	for rows.Next() {
		issue, err := scanIssue(rows)
		if err != nil {
			return nil, nil, err
		}

		issues = append(issues, issue)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return pagination.Window(issues, page)
}

var errInvalidAssignee = errors.New("invalid assignee")
var errInvalidLabel = errors.New("invalid label")
var errInvalidIssueParent = errors.New("invalid issue parent")
var errIssueParentCycle = errors.New("issue parent cycle")
var errIssueParentRequired = errors.New("issue parent required")
var errIssueParentForbidden = errors.New("issue parent forbidden")
var errInvalidIssueLinkTarget = errors.New("invalid issue link target")
var errIssueLinkSelf = errors.New("issue link self")
var errIssueLinkDuplicate = errors.New("issue link duplicate")
var errWorkflowStatusNotFound = errors.New("workflow status not found")
var errTransitionNotAllowed = errors.New("transition not allowed")

func (h *Handler) getIssue(ctx context.Context, workspaceID string, issueID string, users ...auth.CurrentUser) (issueResponse, error) {
	if len(users) > 0 {
		if _, err := projectaccess.RequireIssueRead(ctx, h.db, users[0], issueID); err != nil {
			return issueResponse{}, err
		}
	}
	issue, err := scanIssue(h.db.QueryRow(ctx, `
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
			(SELECT jsonb_build_object('id', ws.id::text, 'key', ws.key, 'name', ws.name, 'color', ws.color, 'category', ws.category) FROM project_workflow_statuses ws WHERE ws.id = i.workflow_status_id),
			i.priority,
			i.story_points,
			i.reporter_id::text,
			i.assignee_id::text,
			i.parent_issue_id::text,
			i.sprint_id::text,
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
	if err != nil || len(users) == 0 || issue.ParentIssueID == nil {
		return issue, err
	}
	if _, err := projectaccess.RequireIssueRead(ctx, h.db, users[0], *issue.ParentIssueID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			issue.ParentIssueID = nil
			return issue, nil
		}
		return issueResponse{}, err
	}
	return issue, nil
}

func (h *Handler) createIssue(ctx context.Context, user auth.CurrentUser, input normalizedCreateIssue) (issueResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return issueResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := projectaccess.RequireWriteForUpdate(ctx, tx, user, input.ProjectID); err != nil {
		return issueResponse{}, err
	}
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

	if err := verifyActiveProjectMember(ctx, tx, user, input.ProjectID, input.AssigneeID); err != nil {
		return issueResponse{}, err
	}

	if err := verifyWorkspaceLabels(ctx, tx, user.WorkspaceID, input.LabelIDs); err != nil {
		return issueResponse{}, err
	}

	if err := verifyIssueParent(ctx, tx, user, "", input.ParentIssueID); err != nil {
		return issueResponse{}, err
	}
	workflowStatus, err := resolveActiveWorkflowStatus(ctx, tx, input.ProjectID, input.WorkflowStatusID, input.Status)
	if err != nil {
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
			workflow_status_id,
			priority,
			story_points,
			reporter_id,
			assignee_id,
			parent_issue_id,
			due_date
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING
			id::text,
			project_id::text,
			$15::text,
			number,
			issue_key,
			title,
			description,
			issue_type,
			status,
			(SELECT jsonb_build_object('id', ws.id::text, 'key', ws.key, 'name', ws.name, 'color', ws.color, 'category', ws.category) FROM project_workflow_statuses ws WHERE ws.id = workflow_status_id),
			priority,
			story_points,
			reporter_id::text,
			assignee_id::text,
			parent_issue_id::text,
			sprint_id::text,
			due_date::text,
			created_at,
			updated_at,
			'[]'::jsonb
	`, input.ProjectID, nextNumber, issueKey, input.Title, input.Description, input.IssueType, workflowStatus.Key, workflowStatus.ID, input.Priority, input.StoryPoints, user.ID, assigneeID, parentIssueID, dueDate, projectKey))
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
		"points":    fmt.Sprintf("%d", issue.StoryPoints),
	}
	if input.ParentIssueID != "" {
		activityPayload["parent_issue_id"] = input.ParentIssueID
	}
	if err := insertIssueActivity(ctx, tx, issue.ID, user.ID, "issue_created", activityPayload); err != nil {
		return issueResponse{}, err
	}
	issue, err = h.executeAutomations(ctx, tx, user, issue.ID, "issue_created")
	if err != nil {
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

	if _, err := projectaccess.RequireIssueWrite(ctx, tx, user, issueID); err != nil {
		return issueResponse{}, err
	}
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
			(SELECT jsonb_build_object('id', ws.id::text, 'key', ws.key, 'name', ws.name, 'color', ws.color, 'category', ws.category) FROM project_workflow_statuses ws WHERE ws.id = i.workflow_status_id),
			i.priority,
			i.story_points,
			i.reporter_id::text,
			i.assignee_id::text,
			i.parent_issue_id::text,
			i.sprint_id::text,
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
			story_points = $7,
			due_date = $8::date,
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
			(SELECT jsonb_build_object('id', ws.id::text, 'key', ws.key, 'name', ws.name, 'color', ws.color, 'category', ws.category) FROM project_workflow_statuses ws WHERE ws.id = i.workflow_status_id),
			i.priority,
			i.story_points,
			i.reporter_id::text,
			i.assignee_id::text,
			i.parent_issue_id::text,
			i.sprint_id::text,
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
	`, issueID, user.WorkspaceID, input.Title, input.Description, input.IssueType, input.Priority, input.StoryPoints, dueDate))
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
	if previous.Priority != issue.Priority {
		issue, err = h.executeAutomations(ctx, tx, user, issue.ID, "priority_changed")
		if err != nil {
			return issueResponse{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return issueResponse{}, err
	}

	return issue, nil
}

func (h *Handler) listIssueChildren(ctx context.Context, workspaceID string, issueID string, users ...auth.CurrentUser) ([]issueResponse, error) {
	args := []any{issueID, workspaceID}
	accessCondition := ""
	if len(users) > 0 {
		args = append(args, users[0].Role == "admin", users[0].ID)
		accessCondition = `
			AND ($3::boolean OR EXISTS (
				SELECT 1
				FROM project_members project_member
				WHERE project_member.project_id = p.id
					AND project_member.user_id = $4
			))
		`
	}
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
			(SELECT jsonb_build_object('id', ws.id::text, 'key', ws.key, 'name', ws.name, 'color', ws.color, 'category', ws.category) FROM project_workflow_statuses ws WHERE ws.id = i.workflow_status_id),
			i.priority,
			i.story_points,
			i.reporter_id::text,
			i.assignee_id::text,
			i.parent_issue_id::text,
			i.sprint_id::text,
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
			`+accessCondition+`
		ORDER BY i.number ASC
		LIMIT 100
	`, args...)
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

	if _, err := projectaccess.RequireIssueWrite(ctx, tx, user, issueID); err != nil {
		return issueResponse{}, err
	}
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
			(SELECT jsonb_build_object('id', ws.id::text, 'key', ws.key, 'name', ws.name, 'color', ws.color, 'category', ws.category) FROM project_workflow_statuses ws WHERE ws.id = i.workflow_status_id),
			i.priority,
			i.story_points,
			i.reporter_id::text,
			i.assignee_id::text,
			i.parent_issue_id::text,
			i.sprint_id::text,
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

	if err := verifyIssueParent(ctx, tx, user, issueID, parentIssueID); err != nil {
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
			(SELECT jsonb_build_object('id', ws.id::text, 'key', ws.key, 'name', ws.name, 'color', ws.color, 'category', ws.category) FROM project_workflow_statuses ws WHERE ws.id = i.workflow_status_id),
			i.priority,
			i.story_points,
			i.reporter_id::text,
			i.assignee_id::text,
			i.parent_issue_id::text,
			i.sprint_id::text,
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

func (h *Handler) listIssueLinks(ctx context.Context, workspaceID string, issueID string, users ...auth.CurrentUser) ([]issueLinkResponse, error) {
	args := []any{issueID, workspaceID}
	accessCondition := ""
	if len(users) > 0 {
		args = append(args, users[0].Role == "admin", users[0].ID)
		accessCondition = `
			AND ($3::boolean OR (
				EXISTS (
					SELECT 1 FROM project_members source_member
					WHERE source_member.project_id = source_project.id
						AND source_member.user_id = $4
				)
				AND EXISTS (
					SELECT 1 FROM project_members target_member
					WHERE target_member.project_id = target_project.id
						AND target_member.user_id = $4
				)
			))
		`
	}
	rows, err := h.db.Query(ctx, `
		SELECT
			il.id::text,
			il.source_issue_id::text,
			il.target_issue_id::text,
			il.link_type,
			il.created_by::text,
			il.created_at,
			source_issue.id::text,
			source_issue.issue_key,
			source_issue.title,
			source_issue.issue_type,
			source_issue.status,
			(SELECT jsonb_build_object('id', ws.id::text, 'key', ws.key, 'name', ws.name, 'color', ws.color, 'category', ws.category) FROM project_workflow_statuses ws WHERE ws.id = source_issue.workflow_status_id),
			source_issue.priority,
			target_issue.id::text,
			target_issue.issue_key,
			target_issue.title,
			target_issue.issue_type,
			target_issue.status,
			(SELECT jsonb_build_object('id', ws.id::text, 'key', ws.key, 'name', ws.name, 'color', ws.color, 'category', ws.category) FROM project_workflow_statuses ws WHERE ws.id = target_issue.workflow_status_id),
			target_issue.priority
		FROM issue_links il
		JOIN issues source_issue ON source_issue.id = il.source_issue_id
		JOIN projects source_project ON source_project.id = source_issue.project_id
		JOIN issues target_issue ON target_issue.id = il.target_issue_id
		JOIN projects target_project ON target_project.id = target_issue.project_id
		WHERE (il.source_issue_id = $1 OR il.target_issue_id = $1)
			AND source_project.workspace_id = $2
			AND target_project.workspace_id = $2
			AND source_project.archived_at IS NULL
			AND target_project.archived_at IS NULL
			AND source_issue.archived_at IS NULL
			AND target_issue.archived_at IS NULL
			`+accessCondition+`
		ORDER BY il.created_at DESC, il.id DESC
		LIMIT 100
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := make([]issueLinkResponse, 0)
	for rows.Next() {
		link, err := scanIssueLink(rows)
		if err != nil {
			return nil, err
		}

		links = append(links, link)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return links, nil
}

func (h *Handler) createIssueLink(ctx context.Context, user auth.CurrentUser, sourceIssueID string, input normalizedCreateIssueLink) (issueLinkResponse, error) {
	if sourceIssueID == input.TargetIssueID {
		return issueLinkResponse{}, errIssueLinkSelf
	}

	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return issueLinkResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := projectaccess.RequireIssueWrite(ctx, tx, user, sourceIssueID); err != nil {
		return issueLinkResponse{}, err
	}
	if _, err := projectaccess.RequireIssueWrite(ctx, tx, user, input.TargetIssueID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return issueLinkResponse{}, errInvalidIssueLinkTarget
		}
		return issueLinkResponse{}, err
	}
	if _, err := getIssueInTx(ctx, tx, user.WorkspaceID, sourceIssueID); err != nil {
		return issueLinkResponse{}, err
	}
	if _, err := getIssueInTx(ctx, tx, user.WorkspaceID, input.TargetIssueID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return issueLinkResponse{}, errInvalidIssueLinkTarget
		}
		return issueLinkResponse{}, err
	}

	linkExists, err := issueLinkExists(ctx, tx, sourceIssueID, input.TargetIssueID, input.LinkType)
	if err != nil {
		return issueLinkResponse{}, err
	}
	if linkExists {
		return issueLinkResponse{}, errIssueLinkDuplicate
	}

	var linkID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO issue_links (source_issue_id, target_issue_id, link_type, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, sourceIssueID, input.TargetIssueID, input.LinkType, user.ID).Scan(&linkID); err != nil {
		if isUniqueViolation(err) || isCheckViolation(err) {
			return issueLinkResponse{}, errIssueLinkDuplicate
		}
		return issueLinkResponse{}, err
	}

	link, err := getIssueLinkInTx(ctx, tx, user.WorkspaceID, linkID)
	if err != nil {
		return issueLinkResponse{}, err
	}

	if err := insertIssueLinkActivity(ctx, tx, link, user.ID, "issue_link_created"); err != nil {
		return issueLinkResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return issueLinkResponse{}, err
	}

	return link, nil
}

func (h *Handler) deleteIssueLink(ctx context.Context, user auth.CurrentUser, issueID string, linkID string) error {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := projectaccess.RequireIssueWrite(ctx, tx, user, issueID); err != nil {
		return err
	}
	link, err := getIssueLinkForIssueInTx(ctx, tx, user.WorkspaceID, issueID, linkID)
	if err != nil {
		return err
	}
	otherIssueID := link.SourceIssueID
	if otherIssueID == issueID {
		otherIssueID = link.TargetIssueID
	}
	if _, err := projectaccess.RequireIssueWrite(ctx, tx, user, otherIssueID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM issue_links
		WHERE id = $1
	`, linkID); err != nil {
		return err
	}

	if err := insertIssueLinkActivity(ctx, tx, link, user.ID, "issue_link_deleted"); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (h *Handler) transitionIssueStatus(ctx context.Context, user auth.CurrentUser, issueID string, input normalizedTransitionIssue) (issueResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return issueResponse{}, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := projectaccess.RequireIssueWrite(ctx, tx, user, issueID); err != nil {
		return issueResponse{}, err
	}
	var projectID string
	var previousStatus string
	var previousWorkflowStatusID string
	if err := tx.QueryRow(ctx, `
		SELECT i.project_id::text, i.status, i.workflow_status_id::text
		FROM issues i
		JOIN projects p ON p.id = i.project_id
	WHERE i.id = $1
		AND p.workspace_id = $2
		AND p.archived_at IS NULL
		AND i.archived_at IS NULL
	FOR UPDATE OF i
	`, issueID, user.WorkspaceID).Scan(&projectID, &previousStatus, &previousWorkflowStatusID); err != nil {
		return issueResponse{}, err
	}
	target, err := resolveActiveWorkflowStatus(ctx, tx, projectID, input.WorkflowStatusID, input.Status)
	if err != nil {
		return issueResponse{}, err
	}
	if target.ID == previousWorkflowStatusID {
		issue, err := getIssueInTx(ctx, tx, user.WorkspaceID, issueID)
		if err != nil {
			return issueResponse{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return issueResponse{}, err
		}
		return issue, nil
	}
	allowed, err := workflowTransitionExists(ctx, tx, projectID, previousWorkflowStatusID, target.ID)
	if err != nil {
		return issueResponse{}, err
	}
	if !allowed {
		return issueResponse{}, errTransitionNotAllowed
	}

	issue, err := scanIssue(tx.QueryRow(ctx, `
		UPDATE issues i
		SET workflow_status_id = $3,
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
			(SELECT jsonb_build_object('id', ws.id::text, 'key', ws.key, 'name', ws.name, 'color', ws.color, 'category', ws.category) FROM project_workflow_statuses ws WHERE ws.id = i.workflow_status_id),
			i.priority,
			i.story_points,
			i.reporter_id::text,
			i.assignee_id::text,
			i.parent_issue_id::text,
			i.sprint_id::text,
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
	`, issueID, user.WorkspaceID, target.ID))
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
		issue, err = h.executeAutomations(ctx, tx, user, issue.ID, "status_changed")
		if err != nil {
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

	access, err := projectaccess.RequireIssueWrite(ctx, tx, user, issueID)
	if err != nil {
		return issueResponse{}, err
	}
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

	if err := verifyActiveProjectMember(ctx, tx, user, access.ProjectID, assigneeID); err != nil {
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
			(SELECT jsonb_build_object('id', ws.id::text, 'key', ws.key, 'name', ws.name, 'color', ws.color, 'category', ws.category) FROM project_workflow_statuses ws WHERE ws.id = i.workflow_status_id),
			i.priority,
			i.story_points,
			i.reporter_id::text,
			i.assignee_id::text,
			i.parent_issue_id::text,
			i.sprint_id::text,
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
		issue, err = h.executeAutomations(ctx, tx, user, issue.ID, "assignee_changed")
		if err != nil {
			return issueResponse{}, err
		}
		if h.notifications != nil && current != "" && stringOrEmpty(issue.AssigneeID) == current {
			if err := h.notifications.NotifyIssueAssigned(ctx, tx, user.WorkspaceID, user.ID, notificationIssueContext(issue), current); err != nil {
				return issueResponse{}, err
			}
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

	if _, err := projectaccess.RequireIssueWrite(ctx, tx, user, issueID); err != nil {
		return issueResponse{}, err
	}
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
			(SELECT jsonb_build_object('id', ws.id::text, 'key', ws.key, 'name', ws.name, 'color', ws.color, 'category', ws.category) FROM project_workflow_statuses ws WHERE ws.id = i.workflow_status_id),
			i.priority,
			i.story_points,
			i.reporter_id::text,
			i.assignee_id::text,
			i.parent_issue_id::text,
			i.sprint_id::text,
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

	if _, err := projectaccess.RequireIssueWrite(ctx, tx, user, issueID); err != nil {
		return err
	}
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

	if _, err := projectaccess.RequireIssueWrite(ctx, tx, user, issueID); err != nil {
		return issueCommentResponse{}, err
	}
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
	if h.notifications != nil {
		if err := h.notifications.NotifyIssueComment(ctx, tx, user.WorkspaceID, user.ID, issueID, comment.ID, body); err != nil {
			return issueCommentResponse{}, err
		}
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

	if _, err := projectaccess.RequireIssueWrite(ctx, tx, user, issueID); err != nil {
		return issueCommentResponse{}, err
	}
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

	if _, err := projectaccess.RequireIssueWrite(ctx, tx, user, issueID); err != nil {
		return err
	}
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
	activity, _, err := h.listIssueActivityPage(ctx, workspaceID, issueID, pagination.Default(100))
	return activity, err
}

func (h *Handler) listIssueActivityPage(ctx context.Context, workspaceID string, issueID string, page pagination.Params) ([]issueActivityResponse, *string, error) {
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
	ORDER BY al.created_at DESC, al.id DESC
		LIMIT $3 OFFSET $4
	`, issueID, workspaceID, page.Limit+1, page.Offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	activity := make([]issueActivityResponse, 0)
	for rows.Next() {
		entry, err := scanIssueActivity(rows)
		if err != nil {
			return nil, nil, err
		}

		activity = append(activity, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return pagination.Window(activity, page)
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

func (h *Handler) writeProjectAccessError(w http.ResponseWriter, err error) bool {
	if !errors.Is(err, projectaccess.ErrForbidden) {
		return false
	}
	writeError(w, http.StatusForbidden, "forbidden", "project write access is required")
	return true
}

func (h *Handler) writeAutomationError(w http.ResponseWriter, err error) bool {
	if !errors.Is(err, automations.ErrActionFailed) {
		return false
	}
	writeError(w, http.StatusConflict, "automation_action_failed", "automation action could not be applied")
	return true
}
