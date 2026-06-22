package emaildiagnostics

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kelmio/backend/internal/auth"
	"kelmio/backend/internal/emailoutbox"
)

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{db: db, auth: authHandler}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/email/diagnostics", h.get)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	diagnostics, err := emailoutbox.LoadDiagnostics(ctx, h.db, user.WorkspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load email diagnostics")
		return
	}

	writeJSON(w, http.StatusOK, diagnostics)
}

func (h *Handler) requireAdmin(w http.ResponseWriter, r *http.Request) (auth.CurrentUser, bool) {
	user, err := h.auth.CurrentUser(r)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "session is required")
			return auth.CurrentUser{}, false
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not load session")
		return auth.CurrentUser{}, false
	}
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "admin role is required")
		return auth.CurrentUser{}, false
	}

	return user, true
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
