#!/bin/sh
set -eu

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
ADMIN_LOGIN="${ADMIN_LOGIN:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin12345}"
MEMBER_LOGIN="${MEMBER_LOGIN:-demo_member}"
MEMBER_PASSWORD="${MEMBER_PASSWORD:-demo12345}"
COOKIE_JAR="$(mktemp "${TMPDIR:-/tmp}/kelmio-smoke.XXXXXX")"
MEMBER_COOKIE_JAR="$(mktemp "${TMPDIR:-/tmp}/kelmio-smoke-member.XXXXXX")"
TEMP_MEMBER_COOKIE_JAR="$(mktemp "${TMPDIR:-/tmp}/kelmio-smoke-v2-member.XXXXXX")"
PROJECT_ID=""
ISSUE_ID=""
LABEL_ID=""
SAVED_FILTER_ID=""
TEAM_INVITE_ID=""
SMOKE_MEMBER_ID=""
CSRF_TOKEN=""
MEMBER_CSRF_TOKEN=""
TEMP_MEMBER_CSRF_TOKEN=""

cleanup() {
	if [ -n "$TEAM_INVITE_ID" ]; then
		curl -fsS -b "$COOKIE_JAR" -H "X-CSRF-Token: $CSRF_TOKEN" -H "Content-Type: application/json" -d '{}' \
			"$API_BASE_URL/api/v1/team/invites/$TEAM_INVITE_ID/revoke" >/dev/null 2>&1 || true
	fi
	if [ -n "$SMOKE_MEMBER_ID" ]; then
		curl -fsS -b "$COOKIE_JAR" \
			-X PATCH \
			-H "Content-Type: application/json" \
			-H "X-CSRF-Token: $CSRF_TOKEN" \
			-d '{"role":"member","is_active":false}' \
			"$API_BASE_URL/api/v1/team/members/$SMOKE_MEMBER_ID" >/dev/null 2>&1 || true
	fi
	if [ -n "$SAVED_FILTER_ID" ]; then
		curl -fsS -b "$COOKIE_JAR" -H "X-CSRF-Token: $CSRF_TOKEN" -X DELETE "$API_BASE_URL/api/v1/saved-filters/$SAVED_FILTER_ID" >/dev/null 2>&1 || true
	fi
	if [ -n "$LABEL_ID" ]; then
		curl -fsS -b "$COOKIE_JAR" -H "X-CSRF-Token: $CSRF_TOKEN" -X DELETE "$API_BASE_URL/api/v1/labels/$LABEL_ID" >/dev/null 2>&1 || true
	fi
	if [ -n "$ISSUE_ID" ]; then
		curl -fsS -b "$COOKIE_JAR" -H "X-CSRF-Token: $CSRF_TOKEN" -X POST "$API_BASE_URL/api/v1/issues/$ISSUE_ID/archive" >/dev/null 2>&1 || true
	fi
	if [ -n "$PROJECT_ID" ]; then
		curl -fsS -b "$COOKIE_JAR" -H "X-CSRF-Token: $CSRF_TOKEN" -X POST "$API_BASE_URL/api/v1/projects/$PROJECT_ID/archive" >/dev/null 2>&1 || true
	fi
	rm -f "$COOKIE_JAR" "$MEMBER_COOKIE_JAR" "$TEMP_MEMBER_COOKIE_JAR"
}
trap cleanup EXIT

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		printf 'Missing required command: %s\n' "$1" >&2
		exit 1
	fi
}

json_value() {
	node -e '
const expression = process.argv[1];
let input = "";
process.stdin.setEncoding("utf8");
process.stdin.on("data", (chunk) => input += chunk);
process.stdin.on("end", () => {
  const data = JSON.parse(input);
  const value = Function("data", `return ${expression}`)(data);
  if (value === undefined || value === null || value === false) {
    process.exit(1);
  }
  process.stdout.write(typeof value === "string" ? value : JSON.stringify(value));
});
' "$1"
}

api_get() {
	curl -fsS -b "$COOKIE_JAR" "$API_BASE_URL$1"
}

api_get_with_jar() {
	curl -fsS -b "$1" "$API_BASE_URL$2"
}

api_get_status_with_jar() {
	curl -sS -o /dev/null -w '%{http_code}' -b "$1" "$API_BASE_URL$2"
}

api_status() {
	curl -sS -o /dev/null -w '%{http_code}' -b "$COOKIE_JAR" "$API_BASE_URL$1"
}

api_csrf_token_with_jar() {
	curl -fsS -b "$1" "$API_BASE_URL/api/v1/auth/csrf-token" | json_value 'data.csrf_token'
}

csrf_token_for_jar() {
	if [ "$1" = "$COOKIE_JAR" ]; then
		printf '%s' "$CSRF_TOKEN"
	elif [ "$1" = "$MEMBER_COOKIE_JAR" ]; then
		printf '%s' "$MEMBER_CSRF_TOKEN"
	elif [ "$1" = "$TEMP_MEMBER_COOKIE_JAR" ]; then
		printf '%s' "$TEMP_MEMBER_CSRF_TOKEN"
	fi
}

api_post() {
	curl -fsS -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
		-H "Content-Type: application/json" \
		-H "X-CSRF-Token: $CSRF_TOKEN" \
		-d "$2" \
		"$API_BASE_URL$1"
}

api_post_with_jar() {
	curl -fsS -b "$1" -c "$1" \
		-H "Content-Type: application/json" \
		-H "X-CSRF-Token: $(csrf_token_for_jar "$1")" \
		-d "$3" \
		"$API_BASE_URL$2"
}

api_post_status_with_jar() {
	curl -sS -o /dev/null -w '%{http_code}' -b "$1" -c "$1" \
		-H "Content-Type: application/json" \
		-H "X-CSRF-Token: $(csrf_token_for_jar "$1")" \
		-d "$3" \
		"$API_BASE_URL$2"
}

api_put_with_jar() {
	curl -fsS -b "$1" \
		-X PUT \
		-H "Content-Type: application/json" \
		-H "X-CSRF-Token: $(csrf_token_for_jar "$1")" \
		-d "$3" \
		"$API_BASE_URL$2"
}

api_patch_status_with_jar() {
	curl -sS -o /dev/null -w '%{http_code}' -b "$1" \
		-X PATCH \
		-H "Content-Type: application/json" \
		-H "X-CSRF-Token: $(csrf_token_for_jar "$1")" \
		-d "$3" \
		"$API_BASE_URL$2"
}

api_delete_status_with_jar() {
	curl -sS -o /dev/null -w '%{http_code}' -b "$1" \
		-X DELETE \
		-H "X-CSRF-Token: $(csrf_token_for_jar "$1")" \
		"$API_BASE_URL$2"
}

api_patch() {
	curl -fsS -b "$COOKIE_JAR" \
		-X PATCH \
		-H "Content-Type: application/json" \
		-H "X-CSRF-Token: $CSRF_TOKEN" \
		-d "$2" \
		"$API_BASE_URL$1"
}

api_put() {
	curl -fsS -b "$COOKIE_JAR" \
		-X PUT \
		-H "Content-Type: application/json" \
		-H "X-CSRF-Token: $CSRF_TOKEN" \
		-d "$2" \
		"$API_BASE_URL$1"
}

api_delete() {
	curl -fsS -b "$COOKIE_JAR" \
		-X DELETE \
		-H "X-CSRF-Token: $CSRF_TOKEN" \
		"$API_BASE_URL$1"
}

require_command curl
require_command node
require_command grep

printf 'Checking backend health at %s\n' "$API_BASE_URL"
curl -fsS "$API_BASE_URL/healthz" >/dev/null
curl -fsS "$API_BASE_URL/readyz" | json_value 'data.database === "up"' >/dev/null
curl -fsS "$API_BASE_URL/api/v1/version" | json_value 'typeof data.version === "string" && typeof data.commit === "string" && typeof data.environment === "string" && Object.prototype.hasOwnProperty.call(data, "build_time")' >/dev/null

printf 'Checking Prometheus metrics safety\n'
METRICS_BODY="$(curl -fsS "$API_BASE_URL/metrics")"
printf '%s' "$METRICS_BODY" | grep -q 'kelmio_http_requests_total'
printf '%s' "$METRICS_BODY" | grep -q 'kelmio_database_ready'
if printf '%s' "$METRICS_BODY" | grep -Fq "$ADMIN_PASSWORD"; then
	printf 'Metrics leaked the admin password\n' >&2
	exit 1
fi

printf 'Checking unauthenticated session guard\n'
if [ "$(api_status "/api/v1/auth/me")" != "401" ]; then
	printf 'Expected /api/v1/auth/me to return 401 before login\n' >&2
	exit 1
fi

printf 'Logging in as %s\n' "$ADMIN_LOGIN"
api_post "/api/v1/auth/login" "{\"login\":\"$ADMIN_LOGIN\",\"password\":\"$ADMIN_PASSWORD\"}" >/dev/null
CSRF_TOKEN="$(api_csrf_token_with_jar "$COOKIE_JAR")"

ADMIN_USER_ID="$(api_get "/api/v1/auth/me" | json_value 'data.user.id')"
ADMIN_EMAIL="$(api_get "/api/v1/auth/me" | json_value 'data.user.email')"

printf 'Checking team members\n'
api_get "/api/v1/team/members" | json_value "data.members.some((member) => member.id === \"$ADMIN_USER_ID\" && member.role === \"admin\")" >/dev/null
api_get "/api/v1/users" | json_value "data.users.some((user) => user.id === \"$ADMIN_USER_ID\" && user.role === \"admin\")" >/dev/null
MEMBER_USER_ID="$(api_get "/api/v1/team/members" | json_value "data.members.find((member) => member.username === \"$MEMBER_LOGIN\").id")"

printf 'Checking V4 seeded workflow roles automation and issues\n'
DEMO_PROJECT_ID="$(api_get "/api/v1/projects" | json_value 'data.projects.find((project) => project.key === "DEMO").id')"
api_get "/api/v1/projects/$DEMO_PROJECT_ID/workflow" | json_value 'data.statuses.filter((status) => status.key === "review" && status.archived_at === null).length === 1' >/dev/null
api_get "/api/v1/projects/$DEMO_PROJECT_ID/members" | json_value "data.members.some((member) => member.user_id === \"$ADMIN_USER_ID\" && member.role === \"lead\") && data.members.some((member) => member.user_id === \"$MEMBER_USER_ID\" && member.role === \"contributor\")" >/dev/null
api_get "/api/v1/projects/$DEMO_PROJECT_ID/automation-rules" | json_value 'data.automation_rules.filter((rule) => rule.name === "Critical bugs move to blocked").length === 1 && data.automation_rules.filter((rule) => rule.name === "Review work assigns demo member").length === 1' >/dev/null
api_get "/api/v1/issues?project_id=$DEMO_PROJECT_ID" | json_value 'data.issues.filter((issue) => issue.issue_key === "DEMO-11").length === 1 && data.issues.filter((issue) => issue.issue_key === "DEMO-12").length === 1' >/dev/null

printf 'Checking member access guards\n'
MEMBER_LOGIN_BODY="$(printf '{"login":"%s","password":"%s"}' "$MEMBER_LOGIN" "$MEMBER_PASSWORD")"
if [ "$(api_post_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/auth/login" "$MEMBER_LOGIN_BODY")" != "200" ]; then
	printf 'Expected member login to succeed for %s\n' "$MEMBER_LOGIN" >&2
	exit 1
fi
MEMBER_CSRF_TOKEN="$(api_csrf_token_with_jar "$MEMBER_COOKIE_JAR")"
if [ "$(api_post_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/projects" '{"key":"MEMBERTRY","name":"Member Project"}')" != "403" ]; then
	printf 'Expected member project creation to return 403\n' >&2
	exit 1
fi
if [ "$(api_post_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/team/members" '{"email":"blocked@example.com","username":"blocked_member","display_name":"Blocked Member","password":"blocked12345","role":"member"}')" != "403" ]; then
	printf 'Expected member team creation to return 403\n' >&2
	exit 1
fi
if [ "$(api_post_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/users" '{"email":"blocked-user@example.com","username":"blocked_user","display_name":"Blocked User","password":"blocked12345","role":"member"}')" != "403" ]; then
	printf 'Expected member user creation to return 403\n' >&2
	exit 1
fi
if [ "$(api_patch_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/team/members/$ADMIN_USER_ID" '{"role":"member","is_active":true}')" != "403" ]; then
	printf 'Expected member team update to return 403\n' >&2
	exit 1
fi
if [ "$(api_patch_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/team/members/$ADMIN_USER_ID/password" '{"password":"blocked12345"}')" != "403" ]; then
	printf 'Expected member password reset to return 403\n' >&2
	exit 1
fi
if [ "$(api_get_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/email/diagnostics")" != "403" ]; then
	printf 'Expected member email diagnostics to return 403\n' >&2
	exit 1
fi

RUN_ID="$(date +%M%S)$$"
PROJECT_KEY="$(printf 'S%s' "$RUN_ID" | cut -c1-10)"
PROJECT_NAME="Smoke Project $RUN_ID"
ISSUE_TITLE="Smoke issue $RUN_ID"

printf 'Checking V5 password reset privacy and email diagnostics\n'
RESET_MESSAGE_KNOWN="$(curl -fsS -H 'Content-Type: application/json' -d "{\"email\":\"$ADMIN_EMAIL\"}" "$API_BASE_URL/api/v1/auth/password-reset/request" | json_value 'data.message')"
RESET_MESSAGE_UNKNOWN="$(curl -fsS -H 'Content-Type: application/json' -d "{\"email\":\"missing-$RUN_ID@example.com\"}" "$API_BASE_URL/api/v1/auth/password-reset/request" | json_value 'data.message')"
if [ "$RESET_MESSAGE_KNOWN" != "$RESET_MESSAGE_UNKNOWN" ]; then
	printf 'Password reset response disclosed account existence\n' >&2
	exit 1
fi
api_get "/api/v1/email/diagnostics" | json_value 'typeof data.total === "number" && typeof data.counts.pending === "number" && Array.isArray(data.recent_terminal_failures) && data.recent_terminal_failures.every((failure) => !Object.prototype.hasOwnProperty.call(failure, "template_data"))' >/dev/null

printf 'Checking V5 invite delivery metadata\n'
TEAM_INVITE_EMAIL="smoke-invite-$RUN_ID@example.com"
TEAM_INVITE_ID="$(api_post "/api/v1/team/invites" "{\"email\":\"$TEAM_INVITE_EMAIL\",\"role\":\"member\"}" | json_value 'data.email_delivery_status === "pending" && typeof data.email_queued_at === "string" && data.email_queued_at.length > 0 && data.id')"
api_get "/api/v1/team/invites" | json_value "data.invites.some((invite) => invite.id === \"$TEAM_INVITE_ID\" && invite.email_delivery_status !== \"not_sent\" && !Object.prototype.hasOwnProperty.call(invite, \"accept_token\"))" >/dev/null
api_post "/api/v1/team/invites/$TEAM_INVITE_ID/revoke" '{}' | json_value 'data.status === "revoked"' >/dev/null
TEAM_INVITE_ID=""

printf 'Creating project %s\n' "$PROJECT_KEY"
PROJECT_ID="$(api_post "/api/v1/projects" "{\"key\":\"$PROJECT_KEY\",\"name\":\"$PROJECT_NAME\",\"description\":\"Created by API smoke test.\"}" | json_value 'data.id')"
api_get "/api/v1/projects/$PROJECT_ID" | json_value "data.id === \"$PROJECT_ID\" && data.key === \"$PROJECT_KEY\"" >/dev/null

printf 'Checking member project access guards\n'
if [ "$(api_patch_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/projects/$PROJECT_ID" '{"name":"Member Update","description":"Blocked by smoke test."}')" != "403" ]; then
	printf 'Expected member project update to return 403\n' >&2
	exit 1
fi
if [ "$(api_post_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/projects/$PROJECT_ID/archive" '{}')" != "403" ]; then
	printf 'Expected member project archive to return 403\n' >&2
	exit 1
fi

printf 'Creating issue in project %s\n' "$PROJECT_KEY"
ISSUE_ID="$(api_post "/api/v1/issues" "{\"project_id\":\"$PROJECT_ID\",\"title\":\"$ISSUE_TITLE\",\"description\":\"Created by API smoke test.\",\"issue_type\":\"task\",\"status\":\"todo\",\"priority\":\"high\"}" | json_value 'data.id')"

printf 'Creating and attaching label\n'
LABEL_ID="$(api_post "/api/v1/labels" "{\"name\":\"smoke-$RUN_ID\",\"color\":\"#3662a1\"}" | json_value 'data.id')"
api_put "/api/v1/issues/$ISSUE_ID/labels" "{\"label_ids\":[\"$LABEL_ID\"]}" | json_value "data.labels.some((label) => label.id === \"$LABEL_ID\")" >/dev/null

printf 'Moving issue to in_progress\n'
api_post "/api/v1/issues/$ISSUE_ID/transition" '{"status":"in_progress"}' | json_value 'data.status === "in_progress"' >/dev/null

printf 'Adding issue comment\n'
COMMENT_ID="$(api_post "/api/v1/issues/$ISSUE_ID/comments" '{"body":"Smoke test comment."}' | json_value 'data.id')"
api_patch "/api/v1/comments/$COMMENT_ID" '{"body":"Smoke test comment updated."}' | json_value 'data.body === "Smoke test comment updated."' >/dev/null
printf 'Checking member comment access guards\n'
if [ "$(api_patch_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/comments/$COMMENT_ID" '{"body":"Blocked member edit."}')" != "403" ]; then
	printf 'Expected member comment update to return 403\n' >&2
	exit 1
fi
if [ "$(api_delete_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/comments/$COMMENT_ID")" != "403" ]; then
	printf 'Expected member comment delete to return 403\n' >&2
	exit 1
fi
DELETE_COMMENT_ID="$(api_post "/api/v1/issues/$ISSUE_ID/comments" '{"body":"Smoke test delete comment."}' | json_value 'data.id')"
api_delete "/api/v1/comments/$DELETE_COMMENT_ID" >/dev/null

printf 'Checking issue filters\n'
api_get "/api/v1/issues?project_id=$PROJECT_ID&status=in_progress&q=Smoke" | json_value "data.issues.some((issue) => issue.id === \"$ISSUE_ID\")" >/dev/null
api_get "/api/v1/issues?label_id=$LABEL_ID" | json_value "data.issues.some((issue) => issue.id === \"$ISSUE_ID\")" >/dev/null

printf 'Checking comments\n'
api_get "/api/v1/issues/$ISSUE_ID/comments" | json_value "data.comments.some((comment) => comment.id === \"$COMMENT_ID\" && comment.body === \"Smoke test comment updated.\")" >/dev/null
api_get "/api/v1/issues/$ISSUE_ID/comments" | json_value "data.comments.every((comment) => comment.id !== \"$DELETE_COMMENT_ID\")" >/dev/null

printf 'Checking activity log\n'
api_get "/api/v1/issues/$ISSUE_ID/activity" | json_value 'data.activity.some((entry) => entry.action === "labels_changed")' >/dev/null
api_get "/api/v1/issues/$ISSUE_ID/activity" | json_value 'data.activity.some((entry) => entry.action === "status_changed")' >/dev/null
api_get "/api/v1/issues/$ISSUE_ID/activity" | json_value 'data.activity.some((entry) => entry.action === "comment_added")' >/dev/null
api_get "/api/v1/issues/$ISSUE_ID/activity" | json_value 'data.activity.some((entry) => entry.action === "comment_updated")' >/dev/null
api_get "/api/v1/issues/$ISSUE_ID/activity" | json_value 'data.activity.some((entry) => entry.action === "comment_deleted")' >/dev/null

printf 'Checking V2 hierarchy\n'
EPIC_ID="$(api_post "/api/v1/issues" "{\"project_id\":\"$PROJECT_ID\",\"title\":\"Smoke V2 epic $RUN_ID\",\"description\":\"Created by API smoke test.\",\"issue_type\":\"epic\",\"status\":\"todo\",\"priority\":\"high\",\"story_points\":13}" | json_value 'data.id')"
CHILD_ID="$(api_post "/api/v1/issues" "{\"project_id\":\"$PROJECT_ID\",\"parent_issue_id\":\"$EPIC_ID\",\"title\":\"Smoke V2 child story $RUN_ID\",\"description\":\"Created by API smoke test.\",\"issue_type\":\"story\",\"status\":\"todo\",\"priority\":\"high\",\"story_points\":8}" | json_value 'data.id')"
SUBTASK_ID="$(api_post "/api/v1/issues/$CHILD_ID/subtasks" "{\"title\":\"Smoke V2 subtask $RUN_ID\",\"description\":\"Created by API smoke test.\",\"status\":\"todo\",\"priority\":\"medium\",\"story_points\":2,\"assignee_id\":\"\",\"due_date\":\"\",\"label_ids\":[]}" | json_value 'data.id')"
api_get "/api/v1/issues/$EPIC_ID/children" | json_value "data.issues.some((issue) => issue.id === \"$CHILD_ID\" && issue.parent_issue_id === \"$EPIC_ID\")" >/dev/null
api_get "/api/v1/issues/$CHILD_ID/children" | json_value "data.issues.some((issue) => issue.id === \"$SUBTASK_ID\" && issue.parent_issue_id === \"$CHILD_ID\" && issue.issue_type === \"subtask\")" >/dev/null

printf 'Checking V2 issue links\n'
BLOCKER_ID="$(api_post "/api/v1/issues" "{\"project_id\":\"$PROJECT_ID\",\"title\":\"Smoke V2 blocker $RUN_ID\",\"description\":\"Created by API smoke test.\",\"issue_type\":\"bug\",\"status\":\"blocked\",\"priority\":\"critical\",\"story_points\":3}" | json_value 'data.id')"
api_post "/api/v1/issues/$BLOCKER_ID/links" "{\"target_issue_id\":\"$CHILD_ID\",\"link_type\":\"blocks\"}" | json_value 'data.link_type === "blocks"' >/dev/null
api_post "/api/v1/issues/$EPIC_ID/links" "{\"target_issue_id\":\"$BLOCKER_ID\",\"link_type\":\"relates\"}" | json_value 'data.link_type === "relates"' >/dev/null
api_get "/api/v1/issues/$BLOCKER_ID/links" | json_value "data.links.some((link) => link.link_type === \"blocks\" && link.target_issue.id === \"$CHILD_ID\")" >/dev/null
api_get "/api/v1/issues/$EPIC_ID/links" | json_value "data.links.some((link) => link.link_type === \"relates\" && link.target_issue.id === \"$BLOCKER_ID\")" >/dev/null

printf 'Checking V3 issue pagination\n'
ISSUE_PAGE_ONE="$(api_get "/api/v1/issues?project_id=$PROJECT_ID&limit=1")"
ISSUE_PAGE_ONE_ID="$(printf '%s' "$ISSUE_PAGE_ONE" | json_value 'data.issues.length === 1 && data.issues[0].id')"
ISSUE_NEXT_CURSOR="$(printf '%s' "$ISSUE_PAGE_ONE" | json_value 'typeof data.next_cursor === "string" && data.next_cursor.length > 0 && data.next_cursor')"
api_get "/api/v1/issues?project_id=$PROJECT_ID&limit=1&cursor=$ISSUE_NEXT_CURSOR" | json_value "data.issues.length === 1 && data.issues[0].id !== \"$ISSUE_PAGE_ONE_ID\" && Object.prototype.hasOwnProperty.call(data, \"next_cursor\")" >/dev/null

printf 'Checking V2 sprints\n'
SPRINT_ID="$(api_post "/api/v1/sprints" "{\"project_id\":\"$PROJECT_ID\",\"name\":\"Smoke Sprint $RUN_ID\",\"goal\":\"Created by API smoke test.\",\"start_date\":\"\",\"end_date\":\"\"}" | json_value 'data.id')"
api_post "/api/v1/sprints/$SPRINT_ID/issues" "{\"issue_id\":\"$ISSUE_ID\"}" | json_value 'data.issue_count >= 1' >/dev/null
api_post "/api/v1/sprints/$SPRINT_ID/issues" "{\"issue_id\":\"$CHILD_ID\"}" | json_value 'data.issue_count >= 2 && data.points_total >= 8' >/dev/null
api_post "/api/v1/sprints/$SPRINT_ID/issues" "{\"issue_id\":\"$BLOCKER_ID\"}" | json_value 'data.issue_count >= 3 && data.points_total >= 11' >/dev/null
api_post "/api/v1/sprints/$SPRINT_ID/start" '{}' | json_value 'data.status === "active"' >/dev/null
api_post "/api/v1/issues/$CHILD_ID/transition" '{"status":"done"}' | json_value 'data.status === "done"' >/dev/null
api_post "/api/v1/sprints/$SPRINT_ID/complete" '{}' | json_value 'data.status === "completed" && data.points_total >= 11 && data.points_done >= 8' >/dev/null
api_get "/api/v1/sprints/$SPRINT_ID" | json_value 'data.status === "completed" && data.issue_count >= 3' >/dev/null
api_get "/api/v1/issues?sprint_id=$SPRINT_ID&status=done" | json_value "data.issues.some((issue) => issue.id === \"$CHILD_ID\")" >/dev/null

printf 'Checking V2 saved filters\n'
SAVED_FILTER_ID="$(api_post "/api/v1/saved-filters" "{\"name\":\"Smoke active blockers $RUN_ID\",\"filters\":{\"sort\":\"priority_desc\",\"sprintId\":\"$SPRINT_ID\",\"status\":\"blocked\"}}" | json_value 'data.id')"
api_get "/api/v1/saved-filters" | json_value "data.saved_filters.some((filter) => filter.id === \"$SAVED_FILTER_ID\" && filter.filters.sprintId === \"$SPRINT_ID\")" >/dev/null
api_patch "/api/v1/saved-filters/$SAVED_FILTER_ID" "{\"name\":\"Smoke done sprint $RUN_ID\",\"filters\":{\"sort\":\"created_desc\",\"sprintId\":\"$SPRINT_ID\",\"status\":\"done\"}}" | json_value 'data.filters.status === "done"' >/dev/null
api_delete "/api/v1/saved-filters/$SAVED_FILTER_ID" >/dev/null
api_get "/api/v1/saved-filters" | json_value "data.saved_filters.every((filter) => filter.id !== \"$SAVED_FILTER_ID\")" >/dev/null
SAVED_FILTER_ID=""

printf 'Checking V2 notifications\n'
SMOKE_MEMBER_USERNAME="smoke_member_$RUN_ID"
SMOKE_MEMBER_PASSWORD="smoke12345"
SMOKE_MEMBER_ID="$(api_post "/api/v1/team/members" "{\"email\":\"$SMOKE_MEMBER_USERNAME@example.com\",\"username\":\"$SMOKE_MEMBER_USERNAME\",\"display_name\":\"Smoke Member $RUN_ID\",\"password\":\"$SMOKE_MEMBER_PASSWORD\",\"role\":\"member\"}" | json_value 'data.id')"
api_put "/api/v1/projects/$PROJECT_ID/members/$SMOKE_MEMBER_ID" '{"role":"contributor"}' | json_value 'data.role === "contributor"' >/dev/null
api_post "/api/v1/issues/$BLOCKER_ID/assign" "{\"assignee_id\":\"$SMOKE_MEMBER_ID\"}" | json_value "data.assignee_id === \"$SMOKE_MEMBER_ID\"" >/dev/null
api_post "/api/v1/issues/$BLOCKER_ID/comments" "{\"body\":\"@$SMOKE_MEMBER_USERNAME Please check this smoke notification.\"}" | json_value "data.body.includes(\"@$SMOKE_MEMBER_USERNAME\")" >/dev/null

api_post_with_jar "$TEMP_MEMBER_COOKIE_JAR" "/api/v1/auth/login" "{\"login\":\"$SMOKE_MEMBER_USERNAME\",\"password\":\"$SMOKE_MEMBER_PASSWORD\"}" >/dev/null
TEMP_MEMBER_CSRF_TOKEN="$(api_csrf_token_with_jar "$TEMP_MEMBER_COOKIE_JAR")"
api_get_with_jar "$TEMP_MEMBER_COOKIE_JAR" "/api/v1/notifications/unread-count" | json_value 'data.unread_count >= 2' >/dev/null
api_get_with_jar "$TEMP_MEMBER_COOKIE_JAR" "/api/v1/notifications" | json_value 'data.notifications.some((notification) => notification.notification_type === "issue_assigned" && notification.read_at === null)' >/dev/null
api_get_with_jar "$TEMP_MEMBER_COOKIE_JAR" "/api/v1/notifications" | json_value 'data.notifications.some((notification) => notification.notification_type === "issue_mentioned" && notification.read_at === null)' >/dev/null
NOTIFICATION_ID="$(api_get_with_jar "$TEMP_MEMBER_COOKIE_JAR" "/api/v1/notifications" | json_value 'data.notifications.find((notification) => notification.notification_type === "issue_assigned").id')"
api_post_with_jar "$TEMP_MEMBER_COOKIE_JAR" "/api/v1/notifications/$NOTIFICATION_ID/read" '{}' | json_value 'data.read_at !== null' >/dev/null
api_post_with_jar "$TEMP_MEMBER_COOKIE_JAR" "/api/v1/notifications/read-all" '{}' >/dev/null
api_get_with_jar "$TEMP_MEMBER_COOKIE_JAR" "/api/v1/notifications/unread-count" | json_value 'data.unread_count === 0' >/dev/null

printf 'Checking V4 workflow roles transitions automation and replacement\n'
WORKFLOW="$(api_get "/api/v1/projects/$PROJECT_ID/workflow")"
TODO_STATUS_ID="$(printf '%s' "$WORKFLOW" | json_value 'data.statuses.find((status) => status.key === "todo" && status.archived_at === null).id')"
IN_PROGRESS_STATUS_ID="$(printf '%s' "$WORKFLOW" | json_value 'data.statuses.find((status) => status.key === "in_progress" && status.archived_at === null).id')"
DONE_STATUS_ID="$(printf '%s' "$WORKFLOW" | json_value 'data.statuses.find((status) => status.key === "done" && status.archived_at === null).id')"

CONTRIBUTOR_ISSUE_ID="$(api_post_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/issues" "{\"project_id\":\"$PROJECT_ID\",\"title\":\"Smoke V4 contributor issue $RUN_ID\",\"description\":\"Created by contributor API smoke flow.\",\"issue_type\":\"task\",\"workflow_status_id\":\"$TODO_STATUS_ID\",\"priority\":\"medium\"}" | json_value 'data.id')"
api_post_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/issues/$CONTRIBUTOR_ISSUE_ID/transition" "{\"workflow_status_id\":\"$IN_PROGRESS_STATUS_ID\"}" | json_value "data.workflow_status.id === \"$IN_PROGRESS_STATUS_ID\"" >/dev/null

api_put "/api/v1/projects/$PROJECT_ID/members/$MEMBER_USER_ID" '{"role":"lead"}' | json_value 'data.role === "lead"' >/dev/null
REVIEW_STATUS_ID="$(api_post_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/projects/$PROJECT_ID/workflow/statuses" "{\"key\":\"review_$RUN_ID\",\"name\":\"Smoke review $RUN_ID\",\"color\":\"#0ea5e9\",\"category\":\"in_progress\"}" | json_value 'data.id')"
api_put_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/projects/$PROJECT_ID/workflow/transitions" "{\"transitions\":[{\"from_status_id\":\"$TODO_STATUS_ID\",\"to_status_id\":\"$REVIEW_STATUS_ID\"},{\"from_status_id\":\"$IN_PROGRESS_STATUS_ID\",\"to_status_id\":\"$REVIEW_STATUS_ID\"},{\"from_status_id\":\"$REVIEW_STATUS_ID\",\"to_status_id\":\"$DONE_STATUS_ID\"}]}" | json_value 'data.transitions.length === 3' >/dev/null

api_put "/api/v1/projects/$PROJECT_ID/members/$MEMBER_USER_ID" '{"role":"contributor"}' | json_value 'data.role === "contributor"' >/dev/null
V4_TRANSITION_ISSUE_ID="$(api_post_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/issues" "{\"project_id\":\"$PROJECT_ID\",\"title\":\"Smoke V4 transition issue $RUN_ID\",\"description\":\"Created by contributor API smoke flow.\",\"issue_type\":\"task\",\"workflow_status_id\":\"$TODO_STATUS_ID\",\"priority\":\"medium\"}" | json_value 'data.id')"
api_post_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/issues/$V4_TRANSITION_ISSUE_ID/transition" "{\"workflow_status_id\":\"$REVIEW_STATUS_ID\"}" | json_value "data.workflow_status.id === \"$REVIEW_STATUS_ID\"" >/dev/null
if [ "$(api_post_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/issues/$V4_TRANSITION_ISSUE_ID/transition" "{\"workflow_status_id\":\"$IN_PROGRESS_STATUS_ID\"}")" != "409" ]; then
	printf 'Expected forbidden V4 transition to return 409\n' >&2
	exit 1
fi

AUTOMATION_RULE_ID="$(api_post "/api/v1/projects/$PROJECT_ID/automation-rules" "{\"name\":\"Smoke assign review $RUN_ID\",\"trigger_type\":\"issue_created\",\"conditions\":[],\"actions\":[{\"type\":\"change_assignee\",\"user_id\":\"$SMOKE_MEMBER_ID\"},{\"type\":\"change_workflow_status\",\"workflow_status_id\":\"$REVIEW_STATUS_ID\"}],\"is_enabled\":true}" | json_value 'data.id')"
AUTOMATED_ISSUE_ID="$(api_post "/api/v1/issues" "{\"project_id\":\"$PROJECT_ID\",\"title\":\"Smoke V4 automated issue $RUN_ID\",\"description\":\"Created by automation API smoke flow.\",\"issue_type\":\"task\",\"workflow_status_id\":\"$TODO_STATUS_ID\",\"priority\":\"medium\"}" | json_value "data.workflow_status.id === \"$REVIEW_STATUS_ID\" && data.assignee_id === \"$SMOKE_MEMBER_ID\" && data.id")"
api_get "/api/v1/issues/$AUTOMATED_ISSUE_ID/activity" | json_value "data.activity.some((entry) => entry.action === \"automation_applied\" && entry.actor_id === null && entry.payload.rule_id === \"$AUTOMATION_RULE_ID\")" >/dev/null
api_get_with_jar "$TEMP_MEMBER_COOKIE_JAR" "/api/v1/notifications" | json_value "data.notifications.some((notification) => notification.issue_id === \"$AUTOMATED_ISSUE_ID\" && notification.notification_type === \"issue_automation_assigned\")" >/dev/null

api_post "/api/v1/projects/$PROJECT_ID/workflow/statuses/$REVIEW_STATUS_ID/archive" "{\"replacement_status_id\":\"$DONE_STATUS_ID\"}" | json_value 'data.archived_at !== null' >/dev/null
api_get "/api/v1/issues/$AUTOMATED_ISSUE_ID" | json_value "data.workflow_status.id === \"$DONE_STATUS_ID\" && data.status === \"done\"" >/dev/null
api_get "/api/v1/issues/$V4_TRANSITION_ISSUE_ID" | json_value "data.workflow_status.id === \"$DONE_STATUS_ID\" && data.status === \"done\"" >/dev/null

api_put "/api/v1/projects/$PROJECT_ID/members/$MEMBER_USER_ID" '{"role":"viewer"}' | json_value 'data.role === "viewer"' >/dev/null
api_get_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/projects/$PROJECT_ID" | json_value 'data.project_role === "viewer" && data.can_write === false' >/dev/null
VIEWER_CREATE_STATUS="$(api_post_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/issues" "{\"project_id\":\"$PROJECT_ID\",\"title\":\"Viewer write $RUN_ID\",\"issue_type\":\"task\",\"workflow_status_id\":\"$TODO_STATUS_ID\",\"priority\":\"medium\"}")"
if [ "$VIEWER_CREATE_STATUS" != "403" ]; then
	printf 'Expected viewer issue creation to return 403\n' >&2
	exit 1
fi
api_delete "/api/v1/projects/$PROJECT_ID/members/$MEMBER_USER_ID" >/dev/null
if [ "$(api_get_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/projects/$PROJECT_ID")" != "404" ]; then
	printf 'Expected removed project member read to return 404\n' >&2
	exit 1
fi

printf 'API smoke test passed\n'
