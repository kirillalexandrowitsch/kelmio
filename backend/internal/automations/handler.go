package automations

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kelmio/backend/internal/auth"
	"kelmio/backend/internal/projectaccess"
)

var errRuleNotFound = errors.New("automation rule not found")
var errRuleOrderMismatch = errors.New("automation rule order mismatch")
var errInvalidDependency = errors.New("automation rule dependency is invalid")

type Handler struct {
	db   *pgxpool.Pool
	auth *auth.Handler
}

type createRuleRequest struct {
	Name        string          `json:"name"`
	TriggerType string          `json:"trigger_type"`
	Conditions  json.RawMessage `json:"conditions"`
	Actions     json.RawMessage `json:"actions"`
	IsEnabled   *bool           `json:"is_enabled"`
}

type updateRuleRequest struct {
	Name        *string         `json:"name"`
	TriggerType *string         `json:"trigger_type"`
	Conditions  json.RawMessage `json:"conditions"`
	Actions     json.RawMessage `json:"actions"`
	IsEnabled   *bool           `json:"is_enabled"`
}

type ruleOrderRequest struct {
	RuleIDs []string `json:"rule_ids"`
}

type normalizedCreateRule struct {
	Name        string
	TriggerType string
	Conditions  json.RawMessage
	Actions     json.RawMessage
	IsEnabled   bool
	Definition  normalizedDefinition
}

type normalizedUpdateRule struct {
	Name           string
	HasName        bool
	TriggerType    string
	HasTriggerType bool
	Conditions     json.RawMessage
	HasConditions  bool
	Actions        json.RawMessage
	HasActions     bool
	IsEnabled      bool
	HasIsEnabled   bool
	Definition     normalizedDefinition
}

type ruleResponse struct {
	ID             string          `json:"id"`
	ProjectID      string          `json:"project_id"`
	Name           string          `json:"name"`
	TriggerType    string          `json:"trigger_type"`
	Conditions     json.RawMessage `json:"conditions"`
	Actions        json.RawMessage `json:"actions"`
	Position       int             `json:"position"`
	IsEnabled      bool            `json:"is_enabled"`
	DisabledReason *string         `json:"disabled_reason"`
	CreatedBy      string          `json:"created_by"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type listRulesResponse struct {
	AutomationRules []ruleResponse `json:"automation_rules"`
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{db: db, auth: authHandler}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/projects/{id}/automation-rules", h.list)
	mux.HandleFunc("POST /api/v1/projects/{id}/automation-rules", h.create)
	mux.HandleFunc("PATCH /api/v1/projects/{id}/automation-rules/{ruleID}", h.update)
	mux.HandleFunc("DELETE /api/v1/projects/{id}/automation-rules/{ruleID}", h.delete)
	mux.HandleFunc("PUT /api/v1/projects/{id}/automation-rules/order", h.reorder)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, projectID, ok := h.requireManagerRequest(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if _, err := projectaccess.RequireManage(ctx, h.db, user, projectID); err != nil {
		h.writeStoreError(w, err, "could not list automation rules")
		return
	}
	rules, err := h.listRules(ctx, projectID)
	if err != nil {
		h.writeStoreError(w, err, "could not list automation rules")
		return
	}
	writeJSON(w, http.StatusOK, listRulesResponse{AutomationRules: rules})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	user, projectID, ok := h.requireManagerRequest(w, r)
	if !ok {
		return
	}
	var req createRuleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input, err := normalizeCreateRule(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rule, err := h.createRule(ctx, user, projectID, input)
	if err != nil {
		h.writeStoreError(w, err, "could not create automation rule")
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	user, projectID, ok := h.requireManagerRequest(w, r)
	if !ok {
		return
	}
	ruleID, ok := normalizePathID(w, r.PathValue("ruleID"), "rule id")
	if !ok {
		return
	}
	var req updateRuleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	input, err := normalizeUpdateRule(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rule, err := h.updateRule(ctx, user, projectID, ruleID, input)
	if err != nil {
		h.writeStoreError(w, err, "could not update automation rule")
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	user, projectID, ok := h.requireManagerRequest(w, r)
	if !ok {
		return
	}
	ruleID, ok := normalizePathID(w, r.PathValue("ruleID"), "rule id")
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.deleteRule(ctx, user, projectID, ruleID); err != nil {
		h.writeStoreError(w, err, "could not delete automation rule")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) reorder(w http.ResponseWriter, r *http.Request) {
	user, projectID, ok := h.requireManagerRequest(w, r)
	if !ok {
		return
	}
	var req ruleOrderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	ruleIDs, err := normalizeRuleOrder(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := h.reorderRules(ctx, user, projectID, ruleIDs); err != nil {
		h.writeStoreError(w, err, "could not reorder automation rules")
		return
	}
	rules, err := h.listRules(ctx, projectID)
	if err != nil {
		h.writeStoreError(w, err, "could not list automation rules")
		return
	}
	writeJSON(w, http.StatusOK, listRulesResponse{AutomationRules: rules})
}

func (h *Handler) requireManagerRequest(w http.ResponseWriter, r *http.Request) (auth.CurrentUser, string, bool) {
	user, err := h.auth.CurrentUser(r)
	if err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "session is required")
			return auth.CurrentUser{}, "", false
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load session")
		return auth.CurrentUser{}, "", false
	}
	projectID, ok := normalizePathID(w, r.PathValue("id"), "project id")
	return user, projectID, ok
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, "project_not_found", "project was not found")
	case errors.Is(err, projectaccess.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "project lead or workspace admin role is required")
	case errors.Is(err, errRuleNotFound):
		writeError(w, http.StatusNotFound, "automation_rule_not_found", "automation rule was not found")
	case errors.Is(err, errRuleOrderMismatch):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	case errors.Is(err, errInvalidDependency):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
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
