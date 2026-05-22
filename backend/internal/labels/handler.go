package labels

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

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

type createLabelRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type labelResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type listLabelsResponse struct {
	Labels []labelResponse `json:"labels"`
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{
		db:   db,
		auth: authHandler,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/labels", h.list)
	mux.HandleFunc("POST /api/v1/labels", h.create)
	mux.HandleFunc("DELETE /api/v1/labels/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT id::text, name, color
		FROM labels
		WHERE workspace_id = $1
		ORDER BY name ASC
	`, user.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list labels")
		return
	}
	defer rows.Close()

	labels := make([]labelResponse, 0)
	for rows.Next() {
		var label labelResponse
		if err := rows.Scan(&label.ID, &label.Name, &label.Color); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "could not read label")
			return
		}

		labels = append(labels, label)
	}

	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list labels")
		return
	}

	writeJSON(w, http.StatusOK, listLabelsResponse{Labels: labels})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	var req createLabelRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	name, color, err := normalizeCreateLabel(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var label labelResponse
	err = h.db.QueryRow(ctx, `
		INSERT INTO labels (workspace_id, name, color)
		VALUES ($1, $2, $3)
		RETURNING id::text, name, color
	`, user.WorkspaceID, name, color).Scan(&label.ID, &label.Name, &label.Color)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "label_exists", "label already exists")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not create label")
		return
	}

	writeJSON(w, http.StatusCreated, label)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	labelID, err := normalizeLabelID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.deleteLabel(ctx, user.WorkspaceID, labelID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "label_not_found", "label was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not delete label")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteLabel(ctx context.Context, workspaceID string, labelID string) error {
	var deletedLabelID string
	return h.db.QueryRow(ctx, `
		DELETE FROM labels
		WHERE id = $1
			AND workspace_id = $2
		RETURNING id::text
	`, labelID, workspaceID).Scan(&deletedLabelID)
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

func normalizeCreateLabel(req createLabelRequest) (string, string, error) {
	name := normalizeLabelName(req.Name)
	if name == "" {
		return "", "", errors.New("name is required")
	}
	if len([]rune(name)) > 40 {
		return "", "", errors.New("name must be 40 characters or fewer")
	}

	color := strings.ToLower(strings.TrimSpace(req.Color))
	if color == "" {
		color = "#4e795d"
	}
	if !colorPattern.MatchString(color) {
		return "", "", errors.New("color must be a hex color like #4e795d")
	}

	return name, color, nil
}

func normalizeLabelName(name string) string {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(name)))
	return strings.Join(parts, "-")
}

func normalizeLabelID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New("label id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("label id is invalid")
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
