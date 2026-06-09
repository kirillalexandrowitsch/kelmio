package projectmembers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/projectaccess"
)

var errWorkspaceMemberNotFound = errors.New("workspace member not found")
var errProjectMemberNotFound = errors.New("project member not found")
var errProjectRequiresLead = errors.New("project requires lead")

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

type updateMemberRequest struct {
	Role string `json:"role"`
}

type memberResponse struct {
	ProjectID     string    `json:"project_id"`
	UserID        string    `json:"user_id"`
	Email         string    `json:"email"`
	Username      string    `json:"username"`
	DisplayName   string    `json:"display_name"`
	Role          string    `json:"role"`
	WorkspaceRole string    `json:"workspace_role"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type listMembersResponse struct {
	Members []memberResponse `json:"members"`
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{db: db, auth: authHandler}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/projects/{id}/members", h.list)
	mux.HandleFunc("PUT /api/v1/projects/{id}/members/{userID}", h.put)
	mux.HandleFunc("DELETE /api/v1/projects/{id}/members/{userID}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
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
	if err := h.requireManager(ctx, h.db, user, projectID); err != nil {
		h.writeStoreError(w, err, "could not load project members")
		return
	}
	members, err := h.listProjectMembers(ctx, projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list project members")
		return
	}
	writeJSON(w, http.StatusOK, listMembersResponse{Members: members})
}

func (h *Handler) put(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	projectID, ok := normalizePathID(w, r.PathValue("id"), "project id")
	if !ok {
		return
	}
	userID, ok := normalizePathID(w, r.PathValue("userID"), "user id")
	if !ok {
		return
	}
	var req updateMemberRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	role, err := normalizeRole(req.Role)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	member, err := h.putProjectMember(ctx, user, projectID, userID, role)
	if err != nil {
		h.writeStoreError(w, err, "could not update project member")
		return
	}
	writeJSON(w, http.StatusOK, member)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	projectID, ok := normalizePathID(w, r.PathValue("id"), "project id")
	if !ok {
		return
	}
	userID, ok := normalizePathID(w, r.PathValue("userID"), "user id")
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.deleteProjectMember(ctx, user, projectID, userID); err != nil {
		h.writeStoreError(w, err, "could not delete project member")
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func (h *Handler) requireManager(ctx context.Context, db projectaccess.Querier, user auth.CurrentUser, projectID string) error {
	_, err := projectaccess.RequireManage(ctx, db, user, projectID)
	return err
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, "project_not_found", "project was not found")
	case errors.Is(err, projectaccess.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "project lead or workspace admin role is required")
	case errors.Is(err, errWorkspaceMemberNotFound):
		writeError(w, http.StatusNotFound, "workspace_member_not_found", "active workspace member was not found")
	case errors.Is(err, errProjectMemberNotFound):
		writeError(w, http.StatusNotFound, "project_member_not_found", "project member was not found")
	case errors.Is(err, errProjectRequiresLead):
		writeError(w, http.StatusConflict, "project_requires_lead", "project requires an active lead when no active workspace admin exists")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", fallback)
	}
}

func normalizePathID(w http.ResponseWriter, value string, field string) (string, bool) {
	id, err := normalizeID(value, field)
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
