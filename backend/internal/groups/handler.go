package groups

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

	"kelmio/backend/internal/auth"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

var errInactiveUser = errors.New("inactive user")

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

type groupResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MemberCount int    `json:"member_count"`
}

type listGroupsResponse struct {
	Groups []groupResponse `json:"groups"`
}

type createGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type updateGroupRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type groupMemberResponse struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	AddedAt     time.Time `json:"added_at"`
}

type listGroupMembersResponse struct {
	Members []groupMemberResponse `json:"members"`
}

type addGroupMemberRequest struct {
	UserID string `json:"user_id"`
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{db: db, auth: authHandler}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/groups", h.list)
	mux.HandleFunc("POST /api/v1/groups", h.create)
	mux.HandleFunc("PATCH /api/v1/groups/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/groups/{id}", h.delete)
	mux.HandleFunc("GET /api/v1/groups/{id}/members", h.listMembers)
	mux.HandleFunc("POST /api/v1/groups/{id}/members", h.addMember)
	mux.HandleFunc("DELETE /api/v1/groups/{id}/members/{userId}", h.removeMember)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
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

	rows, err := h.db.Query(ctx, `
		SELECT g.id::text, g.name, g.description,
			(SELECT count(*)::int FROM group_members gm WHERE gm.group_id = g.id)
		FROM groups g
		WHERE g.organization_id = $1
		ORDER BY g.name ASC
	`, user.OrganizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list groups")
		return
	}
	defer rows.Close()

	groups := make([]groupResponse, 0)
	for rows.Next() {
		var group groupResponse
		if err := rows.Scan(&group.ID, &group.Name, &group.Description, &group.MemberCount); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not list groups")
			return
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list groups")
		return
	}

	writeJSON(w, http.StatusOK, listGroupsResponse{Groups: groups})
}

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

	var req createGroupRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	name, err := normalizeGroupName(req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	description := strings.TrimSpace(req.Description)

	var group groupResponse
	if err := h.db.QueryRow(ctx, `
		INSERT INTO groups (organization_id, name, description, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, name, description, 0
	`, user.OrganizationID, name, description, user.ID).Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.MemberCount,
	); err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "group_exists", "a group with this name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create group")
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	groupID, err := normalizeID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !h.authorizeGroupAdmin(ctx, w, user, groupID) {
		return
	}

	var req updateGroupRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var name *string
	if req.Name != nil {
		normalized, err := normalizeGroupName(*req.Name)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		name = &normalized
	}
	var description *string
	if req.Description != nil {
		trimmed := strings.TrimSpace(*req.Description)
		description = &trimmed
	}

	var group groupResponse
	if err := h.db.QueryRow(ctx, `
		UPDATE groups
		SET name = COALESCE($2, name),
			description = COALESCE($3, description),
			updated_at = now()
		WHERE id = $1
		RETURNING id::text, name, description,
			(SELECT count(*)::int FROM group_members gm WHERE gm.group_id = groups.id)
	`, groupID, name, description).Scan(
		&group.ID,
		&group.Name,
		&group.Description,
		&group.MemberCount,
	); err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "group_exists", "a group with this name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update group")
		return
	}
	writeJSON(w, http.StatusOK, group)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	groupID, err := normalizeID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !h.authorizeGroupAdmin(ctx, w, user, groupID) {
		return
	}

	if _, err := h.db.Exec(ctx, `DELETE FROM groups WHERE id = $1`, groupID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not delete group")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	groupID, err := normalizeID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !h.authorizeGroupAdmin(ctx, w, user, groupID) {
		return
	}

	rows, err := h.db.Query(ctx, `
		SELECT u.id::text, u.username, u.display_name, u.email, gm.added_at
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = $1
		ORDER BY u.display_name ASC
	`, groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list group members")
		return
	}
	defer rows.Close()

	members := make([]groupMemberResponse, 0)
	for rows.Next() {
		var member groupMemberResponse
		if err := rows.Scan(&member.UserID, &member.Username, &member.DisplayName, &member.Email, &member.AddedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not list group members")
			return
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list group members")
		return
	}

	writeJSON(w, http.StatusOK, listGroupMembersResponse{Members: members})
}

func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	groupID, err := normalizeID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	organizationID, ok := h.authorizeGroupAdminWithOrg(ctx, w, user, groupID)
	if !ok {
		return
	}

	var req addGroupMemberRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	targetID, err := normalizeID(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	member, err := h.addGroupMember(ctx, organizationID, groupID, targetID)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			writeError(w, http.StatusNotFound, "user_not_found", "user is not a member of this organization")
		case errors.Is(err, errInactiveUser):
			writeError(w, http.StatusBadRequest, "inactive_user", "user is not active")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "could not add group member")
		}
		return
	}
	writeJSON(w, http.StatusOK, member)
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	groupID, err := normalizeID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	targetID, err := normalizeID(r.PathValue("userId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !h.authorizeGroupAdmin(ctx, w, user, groupID) {
		return
	}

	tag, err := h.db.Exec(ctx, `
		DELETE FROM group_members WHERE group_id = $1 AND user_id = $2
	`, groupID, targetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not remove group member")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "member_not_found", "group member was not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) addGroupMember(ctx context.Context, organizationID string, groupID string, userID string) (groupMemberResponse, error) {
	var isActive bool
	if err := h.db.QueryRow(ctx, `
		SELECT u.is_active
		FROM organization_members om
		JOIN users u ON u.id = om.user_id
		WHERE om.organization_id = $1 AND om.user_id = $2
	`, organizationID, userID).Scan(&isActive); err != nil {
		return groupMemberResponse{}, err
	}
	if !isActive {
		return groupMemberResponse{}, errInactiveUser
	}

	if _, err := h.db.Exec(ctx, `
		INSERT INTO group_members (group_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (group_id, user_id) DO NOTHING
	`, groupID, userID); err != nil {
		return groupMemberResponse{}, err
	}

	var member groupMemberResponse
	if err := h.db.QueryRow(ctx, `
		SELECT u.id::text, u.username, u.display_name, u.email, gm.added_at
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = $1 AND gm.user_id = $2
	`, groupID, userID).Scan(&member.UserID, &member.Username, &member.DisplayName, &member.Email, &member.AddedAt); err != nil {
		return groupMemberResponse{}, err
	}
	return member, nil
}

func (h *Handler) authorizeGroupAdmin(ctx context.Context, w http.ResponseWriter, user auth.CurrentUser, groupID string) bool {
	_, ok := h.authorizeGroupAdminWithOrg(ctx, w, user, groupID)
	return ok
}

func (h *Handler) authorizeGroupAdminWithOrg(ctx context.Context, w http.ResponseWriter, user auth.CurrentUser, groupID string) (string, bool) {
	var organizationID string
	if err := h.db.QueryRow(ctx, `
		SELECT organization_id::text FROM groups WHERE id = $1
	`, groupID).Scan(&organizationID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "group_not_found", "group was not found")
			return "", false
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load group")
		return "", false
	}
	if !h.authorizeOrganizationAdmin(ctx, w, user, organizationID) {
		return "", false
	}
	return organizationID, true
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

func normalizeGroupName(value string) (string, error) {
	name := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if name == "" {
		return "", errors.New("name is required")
	}
	if len([]rune(name)) > 80 {
		return "", errors.New("name must be 80 characters or fewer")
	}
	return name, nil
}

func normalizeID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New("id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("id is invalid")
	}
	return id, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
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
