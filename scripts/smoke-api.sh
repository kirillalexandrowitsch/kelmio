#!/bin/sh
set -eu

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
ADMIN_LOGIN="${ADMIN_LOGIN:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin12345}"
MEMBER_LOGIN="${MEMBER_LOGIN:-demo_member}"
MEMBER_PASSWORD="${MEMBER_PASSWORD:-demo12345}"
COOKIE_JAR="$(mktemp "${TMPDIR:-/tmp}/team-task-tracker-smoke.XXXXXX")"
MEMBER_COOKIE_JAR="$(mktemp "${TMPDIR:-/tmp}/team-task-tracker-smoke-member.XXXXXX")"
PROJECT_ID=""
ISSUE_ID=""
LABEL_ID=""

cleanup() {
	if [ -n "$LABEL_ID" ]; then
		curl -fsS -b "$COOKIE_JAR" -X DELETE "$API_BASE_URL/api/v1/labels/$LABEL_ID" >/dev/null 2>&1 || true
	fi
	if [ -n "$ISSUE_ID" ]; then
		curl -fsS -b "$COOKIE_JAR" -X POST "$API_BASE_URL/api/v1/issues/$ISSUE_ID/archive" >/dev/null 2>&1 || true
	fi
	if [ -n "$PROJECT_ID" ]; then
		curl -fsS -b "$COOKIE_JAR" -X POST "$API_BASE_URL/api/v1/projects/$PROJECT_ID/archive" >/dev/null 2>&1 || true
	fi
	rm -f "$COOKIE_JAR" "$MEMBER_COOKIE_JAR"
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

api_status() {
	curl -sS -o /dev/null -w '%{http_code}' -b "$COOKIE_JAR" "$API_BASE_URL$1"
}

api_post() {
	curl -fsS -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
		-H "Content-Type: application/json" \
		-d "$2" \
		"$API_BASE_URL$1"
}

api_post_status_with_jar() {
	curl -sS -o /dev/null -w '%{http_code}' -b "$1" -c "$1" \
		-H "Content-Type: application/json" \
		-d "$3" \
		"$API_BASE_URL$2"
}

api_patch_status_with_jar() {
	curl -sS -o /dev/null -w '%{http_code}' -b "$1" \
		-X PATCH \
		-H "Content-Type: application/json" \
		-d "$3" \
		"$API_BASE_URL$2"
}

api_delete_status_with_jar() {
	curl -sS -o /dev/null -w '%{http_code}' -b "$1" \
		-X DELETE \
		"$API_BASE_URL$2"
}

api_patch() {
	curl -fsS -b "$COOKIE_JAR" \
		-X PATCH \
		-H "Content-Type: application/json" \
		-d "$2" \
		"$API_BASE_URL$1"
}

api_put() {
	curl -fsS -b "$COOKIE_JAR" \
		-X PUT \
		-H "Content-Type: application/json" \
		-d "$2" \
		"$API_BASE_URL$1"
}

api_delete() {
	curl -fsS -b "$COOKIE_JAR" \
		-X DELETE \
		"$API_BASE_URL$1"
}

require_command curl
require_command node

printf 'Checking backend health at %s\n' "$API_BASE_URL"
curl -fsS "$API_BASE_URL/healthz" >/dev/null
curl -fsS "$API_BASE_URL/readyz" | json_value 'data.database === "up"' >/dev/null

printf 'Checking unauthenticated session guard\n'
if [ "$(api_status "/api/v1/auth/me")" != "401" ]; then
	printf 'Expected /api/v1/auth/me to return 401 before login\n' >&2
	exit 1
fi

printf 'Logging in as %s\n' "$ADMIN_LOGIN"
api_post "/api/v1/auth/login" "{\"login\":\"$ADMIN_LOGIN\",\"password\":\"$ADMIN_PASSWORD\"}" >/dev/null

ADMIN_USER_ID="$(api_get "/api/v1/auth/me" | json_value 'data.user.id')"

printf 'Checking team members\n'
api_get "/api/v1/team/members" | json_value "data.members.some((member) => member.id === \"$ADMIN_USER_ID\" && member.role === \"admin\")" >/dev/null

printf 'Checking member access guards\n'
MEMBER_LOGIN_BODY="$(printf '{"login":"%s","password":"%s"}' "$MEMBER_LOGIN" "$MEMBER_PASSWORD")"
if [ "$(api_post_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/auth/login" "$MEMBER_LOGIN_BODY")" != "200" ]; then
	printf 'Expected member login to succeed for %s\n' "$MEMBER_LOGIN" >&2
	exit 1
fi
if [ "$(api_post_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/projects" '{"key":"MEMBERTRY","name":"Member Project"}')" != "403" ]; then
	printf 'Expected member project creation to return 403\n' >&2
	exit 1
fi
if [ "$(api_post_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/team/members" '{"email":"blocked@example.com","username":"blocked_member","display_name":"Blocked Member","password":"blocked12345","role":"member"}')" != "403" ]; then
	printf 'Expected member team creation to return 403\n' >&2
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

RUN_ID="$(date +%M%S)$$"
PROJECT_KEY="$(printf 'S%s' "$RUN_ID" | cut -c1-10)"
PROJECT_NAME="Smoke Project $RUN_ID"
ISSUE_TITLE="Smoke issue $RUN_ID"

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
api_patch "/api/v1/issues/$ISSUE_ID/comments/$COMMENT_ID" '{"body":"Smoke test comment updated."}' | json_value 'data.body === "Smoke test comment updated."' >/dev/null
printf 'Checking member comment access guards\n'
if [ "$(api_patch_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/issues/$ISSUE_ID/comments/$COMMENT_ID" '{"body":"Blocked member edit."}')" != "403" ]; then
	printf 'Expected member comment update to return 403\n' >&2
	exit 1
fi
if [ "$(api_delete_status_with_jar "$MEMBER_COOKIE_JAR" "/api/v1/issues/$ISSUE_ID/comments/$COMMENT_ID")" != "403" ]; then
	printf 'Expected member comment delete to return 403\n' >&2
	exit 1
fi
DELETE_COMMENT_ID="$(api_post "/api/v1/issues/$ISSUE_ID/comments" '{"body":"Smoke test delete comment."}' | json_value 'data.id')"
api_delete "/api/v1/issues/$ISSUE_ID/comments/$DELETE_COMMENT_ID" >/dev/null

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

printf 'API smoke test passed\n'
