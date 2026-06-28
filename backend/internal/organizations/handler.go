package organizations

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

var (
	errInactiveUser          = errors.New("inactive user")
	errLastOrganizationAdmin = errors.New("last organization admin")
)

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

type organizationResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
	Role   string `json:"role"`
}

type listOrganizationsResponse struct {
	Organizations []organizationResponse `json:"organizations"`
}

type createOrganizationRequest struct {
	Name string `json:"name"`
}

type updateOrganizationRequest struct {
	Name   *string `json:"name"`
	Status *string `json:"status"`
}

type organizationMemberResponse struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	JoinedAt    time.Time `json:"joined_at"`
}

type listOrganizationMembersResponse struct {
	Members []organizationMemberResponse `json:"members"`
}

type addOrganizationMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{db: db, auth: authHandler}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/organizations", h.list)
	mux.HandleFunc("POST /api/v1/organizations", h.create)
	mux.HandleFunc("PATCH /api/v1/organizations/{id}", h.update)
	mux.HandleFunc("GET /api/v1/organizations/{id}/members", h.listMembers)
	mux.HandleFunc("POST /api/v1/organizations/{id}/members", h.addMember)
	mux.HandleFunc("DELETE /api/v1/organizations/{id}/members/{userId}", h.removeMember)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	query := `
		SELECT o.id::text, o.name, o.slug, o.status, om.role
		FROM organizations o
		JOIN organization_members om
			ON om.organization_id = o.id AND om.user_id = $1
		ORDER BY o.name ASC
	`
	if user.IsSiteAdmin {
		query = `
			SELECT o.id::text, o.name, o.slug, o.status, COALESCE(om.role, '')
			FROM organizations o
			LEFT JOIN organization_members om
				ON om.organization_id = o.id AND om.user_id = $1
			ORDER BY o.name ASC
		`
	}

	rows, err := h.db.Query(ctx, query, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list organizations")
		return
	}
	defer rows.Close()

	organizations := make([]organizationResponse, 0)
	for rows.Next() {
		var organization organizationResponse
		if err := rows.Scan(&organization.ID, &organization.Name, &organization.Slug, &organization.Status, &organization.Role); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not read organization")
			return
		}
		organizations = append(organizations, organization)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list organizations")
		return
	}

	writeJSON(w, http.StatusOK, listOrganizationsResponse{Organizations: organizations})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	if !user.IsSiteAdmin {
		writeError(w, http.StatusForbidden, "forbidden", "site administrator role is required")
		return
	}

	var req createOrganizationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	name, slug, err := normalizeName(req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	organization, err := h.createOrganization(ctx, name, slug, user.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "organization_exists", "an organization with a similar name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not create organization")
		return
	}

	writeJSON(w, http.StatusCreated, organization)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	organizationID, err := normalizeID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !user.IsSiteAdmin {
		isOrgAdmin, err := h.isOrganizationAdmin(ctx, organizationID, user.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not authorize organization")
			return
		}
		if !isOrgAdmin {
			writeError(w, http.StatusForbidden, "forbidden", "organization administrator role is required")
			return
		}
	}

	var req updateOrganizationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var name *string
	if req.Name != nil {
		normalized, _, normalizeErr := normalizeName(*req.Name)
		if normalizeErr != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", normalizeErr.Error())
			return
		}
		name = &normalized
	}

	var status *string
	if req.Status != nil {
		value := strings.ToLower(strings.TrimSpace(*req.Status))
		if value != "active" && value != "archived" {
			writeError(w, http.StatusBadRequest, "invalid_request", "status must be active or archived")
			return
		}
		status = &value
	}

	if name == nil && status == nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "name or status is required")
		return
	}

	organization, err := h.updateOrganization(ctx, organizationID, name, status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "organization_not_found", "organization was not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not update organization")
		return
	}

	writeJSON(w, http.StatusOK, organization)
}

func (h *Handler) createOrganization(ctx context.Context, name string, slug string, creatorID string) (organizationResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return organizationResponse{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var organization organizationResponse
	if err := tx.QueryRow(ctx, `
		INSERT INTO organizations (name, slug, created_by)
		VALUES ($1, $2, $3)
		RETURNING id::text, name, slug, status
	`, name, slug, creatorID).Scan(&organization.ID, &organization.Name, &organization.Slug, &organization.Status); err != nil {
		return organizationResponse{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, 'org_admin')
	`, organization.ID, creatorID); err != nil {
		return organizationResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return organizationResponse{}, err
	}

	organization.Role = "org_admin"
	return organization, nil
}

func (h *Handler) updateOrganization(ctx context.Context, organizationID string, name *string, status *string) (organizationResponse, error) {
	var organization organizationResponse
	err := h.db.QueryRow(ctx, `
		UPDATE organizations
		SET
			name = COALESCE($2, name),
			status = COALESCE($3, status),
			updated_at = now()
		WHERE id = $1
		RETURNING id::text, name, slug, status
	`, organizationID, name, status).Scan(&organization.ID, &organization.Name, &organization.Slug, &organization.Status)
	if err != nil {
		return organizationResponse{}, err
	}
	return organization, nil
}

func (h *Handler) isOrganizationAdmin(ctx context.Context, organizationID string, userID string) (bool, error) {
	var isAdmin bool
	if err := h.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM organization_members
			WHERE organization_id = $1 AND user_id = $2 AND role = 'org_admin'
		)
	`, organizationID, userID).Scan(&isAdmin); err != nil {
		return false, err
	}
	return isAdmin, nil
}

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	organizationID, err := normalizeID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !h.authorizeOrganizationRead(ctx, w, user, organizationID) {
		return
	}

	members, err := h.listOrganizationMembers(ctx, organizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list organization members")
		return
	}
	writeJSON(w, http.StatusOK, listOrganizationMembersResponse{Members: members})
}

func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	organizationID, err := normalizeID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !h.authorizeOrganizationAdmin(ctx, w, user, organizationID) {
		return
	}

	var req addOrganizationMemberRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	targetID, err := normalizeID(req.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role != "org_admin" && role != "org_member" {
		writeError(w, http.StatusBadRequest, "invalid_request", "role must be org_admin or org_member")
		return
	}

	member, err := h.addOrganizationMember(ctx, organizationID, targetID, role)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			writeError(w, http.StatusNotFound, "user_not_found", "user was not found")
		case errors.Is(err, errInactiveUser):
			writeError(w, http.StatusBadRequest, "inactive_user", "user is not active")
		case errors.Is(err, errLastOrganizationAdmin):
			writeError(w, http.StatusConflict, "last_org_admin", "an organization must keep at least one administrator")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "could not update organization member")
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
	organizationID, err := normalizeID(r.PathValue("id"))
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

	if !h.authorizeOrganizationAdmin(ctx, w, user, organizationID) {
		return
	}

	if err := h.removeOrganizationMember(ctx, organizationID, targetID); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			writeError(w, http.StatusNotFound, "member_not_found", "organization member was not found")
		case errors.Is(err, errLastOrganizationAdmin):
			writeError(w, http.StatusConflict, "last_org_admin", "an organization must keep at least one administrator")
		default:
			writeError(w, http.StatusInternalServerError, "internal_error", "could not remove organization member")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) authorizeOrganizationRead(ctx context.Context, w http.ResponseWriter, user auth.CurrentUser, organizationID string) bool {
	exists, err := h.organizationExists(ctx, organizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load organization")
		return false
	}
	if !exists {
		writeError(w, http.StatusNotFound, "organization_not_found", "organization was not found")
		return false
	}
	if user.IsSiteAdmin {
		return true
	}
	isMember, err := h.isOrganizationMember(ctx, organizationID, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not authorize organization")
		return false
	}
	if !isMember {
		writeError(w, http.StatusForbidden, "forbidden", "organization membership is required")
		return false
	}
	return true
}

func (h *Handler) authorizeOrganizationAdmin(ctx context.Context, w http.ResponseWriter, user auth.CurrentUser, organizationID string) bool {
	exists, err := h.organizationExists(ctx, organizationID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load organization")
		return false
	}
	if !exists {
		writeError(w, http.StatusNotFound, "organization_not_found", "organization was not found")
		return false
	}
	if user.IsSiteAdmin {
		return true
	}
	isOrgAdmin, err := h.isOrganizationAdmin(ctx, organizationID, user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not authorize organization")
		return false
	}
	if !isOrgAdmin {
		writeError(w, http.StatusForbidden, "forbidden", "organization administrator role is required")
		return false
	}
	return true
}

func (h *Handler) organizationExists(ctx context.Context, organizationID string) (bool, error) {
	var exists bool
	if err := h.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM organizations WHERE id = $1)
	`, organizationID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (h *Handler) isOrganizationMember(ctx context.Context, organizationID string, userID string) (bool, error) {
	var isMember bool
	if err := h.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM organization_members
			WHERE organization_id = $1 AND user_id = $2
		)
	`, organizationID, userID).Scan(&isMember); err != nil {
		return false, err
	}
	return isMember, nil
}

func (h *Handler) listOrganizationMembers(ctx context.Context, organizationID string) ([]organizationMemberResponse, error) {
	rows, err := h.db.Query(ctx, `
		SELECT u.id::text, u.username, u.display_name, u.email, om.role, om.joined_at
		FROM organization_members om
		JOIN users u ON u.id = om.user_id
		WHERE om.organization_id = $1
		ORDER BY (om.role = 'org_admin') DESC, u.display_name ASC
	`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]organizationMemberResponse, 0)
	for rows.Next() {
		var member organizationMemberResponse
		if err := rows.Scan(&member.UserID, &member.Username, &member.DisplayName, &member.Email, &member.Role, &member.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, member)
	}
	return members, rows.Err()
}

func (h *Handler) addOrganizationMember(ctx context.Context, organizationID string, userID string, role string) (organizationMemberResponse, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return organizationMemberResponse{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var member organizationMemberResponse
	var isActive bool
	if err := tx.QueryRow(ctx, `
		SELECT id::text, username, display_name, email, is_active
		FROM users WHERE id = $1
	`, userID).Scan(&member.UserID, &member.Username, &member.DisplayName, &member.Email, &isActive); err != nil {
		return organizationMemberResponse{}, err
	}
	if !isActive {
		return organizationMemberResponse{}, errInactiveUser
	}

	if role == "org_member" {
		demotesLastAdmin, err := lastOrganizationAdmin(ctx, tx, organizationID, userID)
		if err != nil {
			return organizationMemberResponse{}, err
		}
		if demotesLastAdmin {
			return organizationMemberResponse{}, errLastOrganizationAdmin
		}
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (organization_id, user_id) DO UPDATE SET role = EXCLUDED.role
		RETURNING role, joined_at
	`, organizationID, userID, role).Scan(&member.Role, &member.JoinedAt); err != nil {
		return organizationMemberResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return organizationMemberResponse{}, err
	}
	return member, nil
}

func (h *Handler) removeOrganizationMember(ctx context.Context, organizationID string, userID string) error {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var role string
	if err := tx.QueryRow(ctx, `
		SELECT role FROM organization_members
		WHERE organization_id = $1 AND user_id = $2
	`, organizationID, userID).Scan(&role); err != nil {
		return err
	}

	if role == "org_admin" {
		isLast, err := lastOrganizationAdmin(ctx, tx, organizationID, userID)
		if err != nil {
			return err
		}
		if isLast {
			return errLastOrganizationAdmin
		}
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM organization_members
		WHERE organization_id = $1 AND user_id = $2
	`, organizationID, userID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func lastOrganizationAdmin(ctx context.Context, tx pgx.Tx, organizationID string, userID string) (bool, error) {
	var targetIsAdmin bool
	var adminCount int
	if err := tx.QueryRow(ctx, `
		SELECT
			EXISTS (
				SELECT 1 FROM organization_members
				WHERE organization_id = $1 AND user_id = $2 AND role = 'org_admin'
			),
			(SELECT count(*)::int FROM organization_members WHERE organization_id = $1 AND role = 'org_admin')
	`, organizationID, userID).Scan(&targetIsAdmin, &adminCount); err != nil {
		return false, err
	}
	return targetIsAdmin && adminCount <= 1, nil
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
		return "", errors.New("organization id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("organization id is invalid")
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
