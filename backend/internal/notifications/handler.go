package notifications

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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"team-task-tracker/backend/internal/auth"
	"team-task-tracker/backend/internal/pagination"
)

const (
	TypeIssueAssigned                = "issue_assigned"
	TypeIssueMentioned               = "issue_mentioned"
	TypeIssueCommented               = "issue_commented"
	TypeIssueAutomationAssigned      = "issue_automation_assigned"
	TypeIssueAutomationStatusChanged = "issue_automation_status_changed"
	TypeSprintStarted                = "sprint_started"
	TypeSprintCompleted              = "sprint_completed"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var mentionPattern = regexp.MustCompile(`@([A-Za-z0-9_][A-Za-z0-9_-]{0,39})`)

type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Handler struct {
	db      *pgxpool.Pool
	auth    *auth.Handler
	service *Service
}

type Service struct{}

type Notification struct {
	ID               string            `json:"id"`
	WorkspaceID      string            `json:"workspace_id"`
	UserID           string            `json:"user_id"`
	ActorID          *string           `json:"actor_id"`
	ActorDisplayName *string           `json:"actor_display_name"`
	IssueID          *string           `json:"issue_id"`
	IssueKey         *string           `json:"issue_key"`
	IssueTitle       *string           `json:"issue_title"`
	NotificationType string            `json:"notification_type"`
	Payload          map[string]string `json:"payload"`
	ReadAt           *time.Time        `json:"read_at"`
	CreatedAt        time.Time         `json:"created_at"`
}

type IssueContext struct {
	ID         string
	ProjectID  string
	IssueKey   string
	Title      string
	ReporterID string
	AssigneeID string
}

type SprintContext struct {
	ID         string
	ProjectID  string
	Name       string
	ProjectKey string
}

type AutomationChanges struct {
	AppliedRuleNames []string
	ChangedFields    []string
	FromStatus       string
	ToStatus         string
	FromAssigneeID   string
	ToAssigneeID     string
}

type listNotificationsResponse struct {
	Notifications []Notification `json:"notifications"`
	NextCursor    *string        `json:"next_cursor"`
}

type unreadCountResponse struct {
	UnreadCount int `json:"unread_count"`
}

type notificationInput struct {
	WorkspaceID      string
	UserID           string
	ActorID          string
	IssueID          string
	NotificationType string
	Payload          map[string]string
}

type commentRecipient struct {
	UserID           string
	NotificationType string
}

type automationRecipient struct {
	UserID           string
	NotificationType string
}

func NewService() *Service {
	return &Service{}
}

func NewHandler(db *pgxpool.Pool, authHandler *auth.Handler) *Handler {
	return &Handler{
		db:      db,
		auth:    authHandler,
		service: NewService(),
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/notifications", h.list)
	mux.HandleFunc("GET /api/v1/notifications/unread-count", h.unreadCount)
	mux.HandleFunc("POST /api/v1/notifications/{id}/read", h.markRead)
	mux.HandleFunc("POST /api/v1/notifications/read-all", h.markAllRead)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	page, err := pagination.Parse(r.URL.Query(), 50)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	notifications, nextCursor, err := h.service.ListPage(ctx, h.db, user, page)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not list notifications")
		return
	}

	writeJSON(w, http.StatusOK, listNotificationsResponse{Notifications: notifications, NextCursor: nextCursor})
}

func (h *Handler) unreadCount(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	count, err := h.service.UnreadCount(ctx, h.db, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not load unread count")
		return
	}

	writeJSON(w, http.StatusOK, unreadCountResponse{UnreadCount: count})
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	notificationID, err := normalizeNotificationID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	notification, err := h.service.MarkRead(ctx, h.db, user, notificationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "notification_not_found", "notification was not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "internal_error", "could not mark notification read")
		return
	}

	writeJSON(w, http.StatusOK, notification)
}

func (h *Handler) markAllRead(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.service.MarkAllRead(ctx, h.db, user); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "could not mark notifications read")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) List(ctx context.Context, db DBTX, user auth.CurrentUser) ([]Notification, error) {
	notifications, _, err := s.ListPage(ctx, db, user, pagination.Default(50))
	return notifications, err
}

func (s *Service) ListPage(ctx context.Context, db DBTX, user auth.CurrentUser, page pagination.Params) ([]Notification, *string, error) {
	accessCondition := notificationAccessCondition("$2", "$3")
	rows, err := db.Query(ctx, `
		SELECT
			n.id::text,
			n.workspace_id::text,
			n.user_id::text,
			n.actor_id::text,
			actor.display_name,
			n.issue_id::text,
			i.issue_key,
			i.title,
			n.notification_type,
			n.payload,
			n.read_at,
			n.created_at
		FROM notifications n
		LEFT JOIN users actor ON actor.id = n.actor_id
		LEFT JOIN issues i ON i.id = n.issue_id
		WHERE n.workspace_id = $1
			AND n.user_id = $2
			AND `+accessCondition+`
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $4 OFFSET $5
	`, user.WorkspaceID, user.ID, user.Role == "admin", page.Limit+1, page.Offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	notifications := make([]Notification, 0)
	for rows.Next() {
		notification, err := scanNotification(rows)
		if err != nil {
			return nil, nil, err
		}
		notifications = append(notifications, notification)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return pagination.Window(notifications, page)
}

func (s *Service) UnreadCount(ctx context.Context, db DBTX, user auth.CurrentUser) (int, error) {
	var count int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM notifications n
		WHERE n.workspace_id = $1
			AND n.user_id = $2
			AND n.read_at IS NULL
			AND `+notificationAccessCondition("$2", "$3")+`
	`, user.WorkspaceID, user.ID, user.Role == "admin").Scan(&count)
	return count, err
}

func (s *Service) MarkRead(ctx context.Context, db DBTX, user auth.CurrentUser, notificationID string) (Notification, error) {
	return scanNotification(db.QueryRow(ctx, `
		WITH updated AS (
			UPDATE notifications n
			SET read_at = COALESCE(read_at, now())
			WHERE n.id = $1
				AND n.workspace_id = $2
				AND n.user_id = $3
				AND `+notificationAccessCondition("$3", "$4")+`
			RETURNING *
		)
		SELECT
			updated.id::text,
			updated.workspace_id::text,
			updated.user_id::text,
			updated.actor_id::text,
			actor.display_name,
			updated.issue_id::text,
			i.issue_key,
			i.title,
			updated.notification_type,
			updated.payload,
			updated.read_at,
			updated.created_at
		FROM updated
		LEFT JOIN users actor ON actor.id = updated.actor_id
		LEFT JOIN issues i ON i.id = updated.issue_id
	`, notificationID, user.WorkspaceID, user.ID, user.Role == "admin"))
}

func (s *Service) MarkAllRead(ctx context.Context, db DBTX, user auth.CurrentUser) error {
	_, err := db.Exec(ctx, `
		UPDATE notifications n
		SET read_at = COALESCE(read_at, now())
		WHERE n.workspace_id = $1
			AND n.user_id = $2
			AND n.read_at IS NULL
			AND `+notificationAccessCondition("$2", "$3")+`
	`, user.WorkspaceID, user.ID, user.Role == "admin")
	return err
}

func (s *Service) NotifyIssueAssigned(ctx context.Context, db DBTX, workspaceID string, actorID string, issue IssueContext, assigneeID string) error {
	if assigneeID == "" || assigneeID == actorID {
		return nil
	}
	canRead, err := userCanReadProject(ctx, db, workspaceID, issue.ProjectID, assigneeID)
	if err != nil {
		return err
	}
	if !canRead {
		return nil
	}

	return s.insert(ctx, db, notificationInput{
		WorkspaceID:      workspaceID,
		UserID:           assigneeID,
		ActorID:          actorID,
		IssueID:          issue.ID,
		NotificationType: TypeIssueAssigned,
		Payload: issuePayload(issue, map[string]string{
			"message": "assigned you to an issue",
		}),
	})
}

func (s *Service) NotifyAutomationChanges(ctx context.Context, db DBTX, workspaceID string, initiatedByUserID string, issue IssueContext, changes AutomationChanges) error {
	recipients := automationNotificationRecipients(initiatedByUserID, issue.ReporterID, issue.AssigneeID, changes)
	for _, recipient := range recipients {
		canRead, err := userCanReadProject(ctx, db, workspaceID, issue.ProjectID, recipient.UserID)
		if err != nil {
			return err
		}
		if !canRead {
			continue
		}

		payload := issuePayload(issue, map[string]string{
			"automation_rule_names": strings.Join(changes.AppliedRuleNames, ", "),
			"changed_fields":        strings.Join(changes.ChangedFields, ","),
		})
		switch recipient.NotificationType {
		case TypeIssueAutomationAssigned:
			payload["message"] = "automation assigned you to an issue"
			payload["from_assignee_id"] = changes.FromAssigneeID
			payload["to_assignee_id"] = changes.ToAssigneeID
		case TypeIssueAutomationStatusChanged:
			payload["message"] = "automation changed issue status"
			payload["from_status"] = changes.FromStatus
			payload["to_status"] = changes.ToStatus
		}

		if err := s.insert(ctx, db, notificationInput{
			WorkspaceID:      workspaceID,
			UserID:           recipient.UserID,
			IssueID:          issue.ID,
			NotificationType: recipient.NotificationType,
			Payload:          payload,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) NotifyIssueComment(ctx context.Context, db DBTX, workspaceID string, actorID string, issueID string, commentID string, body string) error {
	issue, err := issueContext(ctx, db, workspaceID, issueID)
	if err != nil {
		return err
	}

	mentionedUserIDs, err := mentionedWorkspaceUserIDs(ctx, db, workspaceID, issue.ProjectID, body)
	if err != nil {
		return err
	}

	recipients := commentNotificationRecipients(actorID, issue.ReporterID, issue.AssigneeID, mentionedUserIDs)
	for _, recipient := range recipients {
		canRead, err := userCanReadProject(ctx, db, workspaceID, issue.ProjectID, recipient.UserID)
		if err != nil {
			return err
		}
		if !canRead {
			continue
		}
		payload := issuePayload(issue, map[string]string{
			"comment_id": commentID,
			"preview":    commentPreview(body),
		})
		if recipient.NotificationType == TypeIssueMentioned {
			payload["message"] = "mentioned you in a comment"
		} else {
			payload["message"] = "commented on your issue"
		}
		if err := s.insert(ctx, db, notificationInput{
			WorkspaceID:      workspaceID,
			UserID:           recipient.UserID,
			ActorID:          actorID,
			IssueID:          issue.ID,
			NotificationType: recipient.NotificationType,
			Payload:          payload,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) NotifySprintEvent(ctx context.Context, db DBTX, workspaceID string, actorID string, sprint SprintContext, notificationType string) error {
	if notificationType != TypeSprintStarted && notificationType != TypeSprintCompleted {
		return errors.New("notification type is invalid")
	}

	rows, err := db.Query(ctx, `
		SELECT u.id::text
		FROM workspace_members wm
		JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = $1
			AND u.is_active = true
			AND u.id <> $2
			AND (
				wm.role = 'admin'
				OR EXISTS (
					SELECT 1
					FROM project_members project_member
					WHERE project_member.project_id = $3
						AND project_member.user_id = wm.user_id
				)
			)
		ORDER BY u.id
	`, workspaceID, actorID, sprint.ProjectID)
	if err != nil {
		return err
	}

	recipientIDs := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			rows.Close()
			return err
		}
		recipientIDs = append(recipientIDs, userID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	payload := map[string]string{
		"sprint_id":   sprint.ID,
		"sprint_name": sprint.Name,
		"project_id":  sprint.ProjectID,
		"project_key": sprint.ProjectKey,
	}
	if notificationType == TypeSprintStarted {
		payload["message"] = "started a sprint"
	} else {
		payload["message"] = "completed a sprint"
	}

	for _, userID := range recipientIDs {
		if err := s.insert(ctx, db, notificationInput{
			WorkspaceID:      workspaceID,
			UserID:           userID,
			ActorID:          actorID,
			NotificationType: notificationType,
			Payload:          payload,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) insert(ctx context.Context, db DBTX, input notificationInput) error {
	payload := input.Payload
	if payload == nil {
		payload = map[string]string{}
	}

	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var issueID any
	if input.IssueID != "" {
		issueID = input.IssueID
	}
	var actorID any
	if input.ActorID != "" {
		actorID = input.ActorID
	}

	_, err = db.Exec(ctx, `
		INSERT INTO notifications (
			workspace_id,
			user_id,
			actor_id,
			issue_id,
			notification_type,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
	`, input.WorkspaceID, input.UserID, actorID, issueID, input.NotificationType, string(encodedPayload))
	return err
}

func mentionUsernames(body string) []string {
	matches := mentionPattern.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(matches))
	usernames := make([]string, 0, len(matches))
	for _, match := range matches {
		username := strings.ToLower(match[1])
		if seen[username] {
			continue
		}
		seen[username] = true
		usernames = append(usernames, username)
	}
	sort.Strings(usernames)

	return usernames
}

func commentNotificationRecipients(actorID string, reporterID string, assigneeID string, mentionedUserIDs []string) []commentRecipient {
	recipients := make([]commentRecipient, 0, len(mentionedUserIDs)+2)
	seen := make(map[string]bool, len(mentionedUserIDs)+2)

	for _, userID := range mentionedUserIDs {
		if userID == "" || userID == actorID || seen[userID] {
			continue
		}
		seen[userID] = true
		recipients = append(recipients, commentRecipient{
			UserID:           userID,
			NotificationType: TypeIssueMentioned,
		})
	}

	for _, userID := range []string{reporterID, assigneeID} {
		if userID == "" || userID == actorID || seen[userID] {
			continue
		}
		seen[userID] = true
		recipients = append(recipients, commentRecipient{
			UserID:           userID,
			NotificationType: TypeIssueCommented,
		})
	}

	return recipients
}

func automationNotificationRecipients(initiatedByUserID string, reporterID string, finalAssigneeID string, changes AutomationChanges) []automationRecipient {
	recipients := make([]automationRecipient, 0, 2)
	seen := make(map[string]bool, 2)

	if changes.FromAssigneeID != changes.ToAssigneeID && finalAssigneeID != "" && finalAssigneeID != initiatedByUserID {
		seen[finalAssigneeID] = true
		recipients = append(recipients, automationRecipient{
			UserID:           finalAssigneeID,
			NotificationType: TypeIssueAutomationAssigned,
		})
	}

	if changes.FromStatus == changes.ToStatus {
		return recipients
	}
	for _, userID := range []string{reporterID, finalAssigneeID} {
		if userID == "" || userID == initiatedByUserID || seen[userID] {
			continue
		}
		seen[userID] = true
		recipients = append(recipients, automationRecipient{
			UserID:           userID,
			NotificationType: TypeIssueAutomationStatusChanged,
		})
	}
	return recipients
}

func mentionedWorkspaceUserIDs(ctx context.Context, db DBTX, workspaceID string, projectID string, body string) ([]string, error) {
	usernames := mentionUsernames(body)
	if len(usernames) == 0 {
		return nil, nil
	}

	rows, err := db.Query(ctx, `
		SELECT u.id::text
		FROM workspace_members wm
		JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = $1
			AND u.is_active = true
			AND lower(u.username) = ANY($2)
			AND (
				wm.role = 'admin'
				OR EXISTS (
					SELECT 1
					FROM project_members project_member
					WHERE project_member.project_id = $3
						AND project_member.user_id = wm.user_id
				)
			)
		ORDER BY u.id
	`, workspaceID, usernames, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userIDs := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	return userIDs, rows.Err()
}

func issueContext(ctx context.Context, db DBTX, workspaceID string, issueID string) (IssueContext, error) {
	var issue IssueContext
	var assigneeID pgtype.Text
	if err := db.QueryRow(ctx, `
		SELECT
			i.id::text,
			i.project_id::text,
			i.issue_key,
			i.title,
			i.reporter_id::text,
			i.assignee_id::text
		FROM issues i
		JOIN projects p ON p.id = i.project_id
		WHERE i.id = $1
			AND p.workspace_id = $2
			AND p.archived_at IS NULL
			AND i.archived_at IS NULL
	`, issueID, workspaceID).Scan(
		&issue.ID,
		&issue.ProjectID,
		&issue.IssueKey,
		&issue.Title,
		&issue.ReporterID,
		&assigneeID,
	); err != nil {
		return IssueContext{}, err
	}
	if assigneeID.Valid {
		issue.AssigneeID = assigneeID.String
	}

	return issue, nil
}

func issuePayload(issue IssueContext, values map[string]string) map[string]string {
	payload := map[string]string{
		"project_id":  issue.ProjectID,
		"issue_key":   issue.IssueKey,
		"issue_title": issue.Title,
	}
	for key, value := range values {
		payload[key] = value
	}
	return payload
}

func notificationAccessCondition(userPlaceholder string, adminPlaceholder string) string {
	return fmt.Sprintf(`
		(
			%s::boolean
			OR (
				n.issue_id IS NOT NULL
				AND EXISTS (
					SELECT 1
					FROM issues access_issue
					JOIN projects access_project ON access_project.id = access_issue.project_id
					JOIN project_members access_member ON access_member.project_id = access_project.id
					WHERE access_issue.id = n.issue_id
						AND access_issue.archived_at IS NULL
						AND access_project.archived_at IS NULL
						AND access_member.user_id = %s
				)
			)
			OR (
				n.notification_type IN ('sprint_started', 'sprint_completed')
				AND EXISTS (
					SELECT 1
					FROM projects access_project
					JOIN project_members access_member ON access_member.project_id = access_project.id
					WHERE access_project.workspace_id = n.workspace_id
						AND access_project.archived_at IS NULL
						AND access_member.user_id = %s
						AND (
							access_project.id::text = n.payload->>'project_id'
							OR access_project.key = n.payload->>'project_key'
						)
				)
			)
		)
	`, adminPlaceholder, userPlaceholder, userPlaceholder)
}

func userCanReadProject(
	ctx context.Context,
	db DBTX,
	workspaceID string,
	projectID string,
	userID string,
) (bool, error) {
	var canRead bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM projects project
			JOIN workspace_members workspace_member
				ON workspace_member.workspace_id = project.workspace_id
				AND workspace_member.user_id = $3
			JOIN users app_user
				ON app_user.id = workspace_member.user_id
				AND app_user.is_active = true
			LEFT JOIN project_members project_member
				ON project_member.project_id = project.id
				AND project_member.user_id = workspace_member.user_id
			WHERE project.id = $2
				AND project.workspace_id = $1
				AND project.archived_at IS NULL
				AND (workspace_member.role = 'admin' OR project_member.user_id IS NOT NULL)
		)
	`, workspaceID, projectID, userID).Scan(&canRead)
	return canRead, err
}

func commentPreview(body string) string {
	body = strings.Join(strings.Fields(strings.TrimSpace(body)), " ")
	if len([]rune(body)) <= 80 {
		return body
	}

	runes := []rune(body)
	return string(runes[:80]) + "..."
}

func normalizeNotificationID(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "", errors.New("notification id is required")
	}
	if !uuidPattern.MatchString(id) {
		return "", errors.New("notification id is invalid")
	}
	return id, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNotification(row rowScanner) (Notification, error) {
	var notification Notification
	var actorID pgtype.Text
	var actorDisplayName pgtype.Text
	var issueID pgtype.Text
	var issueKey pgtype.Text
	var issueTitle pgtype.Text
	var payloadJSON []byte
	if err := row.Scan(
		&notification.ID,
		&notification.WorkspaceID,
		&notification.UserID,
		&actorID,
		&actorDisplayName,
		&issueID,
		&issueKey,
		&issueTitle,
		&notification.NotificationType,
		&payloadJSON,
		&notification.ReadAt,
		&notification.CreatedAt,
	); err != nil {
		return Notification{}, err
	}

	notification.ActorID = nullableText(actorID)
	notification.ActorDisplayName = nullableText(actorDisplayName)
	notification.IssueID = nullableText(issueID)
	notification.IssueKey = nullableText(issueKey)
	notification.IssueTitle = nullableText(issueTitle)
	if err := json.Unmarshal(payloadJSON, &notification.Payload); err != nil {
		return Notification{}, err
	}
	if notification.Payload == nil {
		notification.Payload = map[string]string{}
	}

	return notification, nil
}

func nullableText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
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
