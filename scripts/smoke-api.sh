#!/bin/sh
set -eu

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
ADMIN_LOGIN="${ADMIN_LOGIN:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin12345}"
COOKIE_JAR="$(mktemp "${TMPDIR:-/tmp}/team-task-tracker-smoke.XXXXXX")"
PROJECT_ID=""
ISSUE_ID=""

cleanup() {
	if [ -n "$ISSUE_ID" ]; then
		curl -fsS -b "$COOKIE_JAR" -X POST "$API_BASE_URL/api/v1/issues/$ISSUE_ID/archive" >/dev/null 2>&1 || true
	fi
	if [ -n "$PROJECT_ID" ]; then
		curl -fsS -b "$COOKIE_JAR" -X POST "$API_BASE_URL/api/v1/projects/$PROJECT_ID/archive" >/dev/null 2>&1 || true
	fi
	rm -f "$COOKIE_JAR"
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

api_post() {
	curl -fsS -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
		-H "Content-Type: application/json" \
		-d "$2" \
		"$API_BASE_URL$1"
}

require_command curl
require_command node

printf 'Checking backend health at %s\n' "$API_BASE_URL"
curl -fsS "$API_BASE_URL/healthz" >/dev/null

printf 'Logging in as %s\n' "$ADMIN_LOGIN"
api_post "/api/v1/auth/login" "{\"login\":\"$ADMIN_LOGIN\",\"password\":\"$ADMIN_PASSWORD\"}" >/dev/null

api_get "/api/v1/auth/me" | json_value 'data.user.id' >/dev/null

RUN_ID="$(date +%M%S)$$"
PROJECT_KEY="$(printf 'S%s' "$RUN_ID" | cut -c1-10)"
PROJECT_NAME="Smoke Project $RUN_ID"
ISSUE_TITLE="Smoke issue $RUN_ID"

printf 'Creating project %s\n' "$PROJECT_KEY"
PROJECT_ID="$(api_post "/api/v1/projects" "{\"key\":\"$PROJECT_KEY\",\"name\":\"$PROJECT_NAME\",\"description\":\"Created by API smoke test.\"}" | json_value 'data.id')"

printf 'Creating issue in project %s\n' "$PROJECT_KEY"
ISSUE_ID="$(api_post "/api/v1/issues" "{\"project_id\":\"$PROJECT_ID\",\"title\":\"$ISSUE_TITLE\",\"description\":\"Created by API smoke test.\",\"issue_type\":\"task\",\"status\":\"todo\",\"priority\":\"high\"}" | json_value 'data.id')"

printf 'Moving issue to in_progress\n'
api_post "/api/v1/issues/$ISSUE_ID/transition" '{"status":"in_progress"}' | json_value 'data.status === "in_progress"' >/dev/null

printf 'Adding issue comment\n'
COMMENT_ID="$(api_post "/api/v1/issues/$ISSUE_ID/comments" '{"body":"Smoke test comment."}' | json_value 'data.id')"

printf 'Checking issue filters\n'
api_get "/api/v1/issues?project_id=$PROJECT_ID&status=in_progress&q=Smoke" | json_value "data.issues.some((issue) => issue.id === \"$ISSUE_ID\")" >/dev/null

printf 'Checking comments\n'
api_get "/api/v1/issues/$ISSUE_ID/comments" | json_value "data.comments.some((comment) => comment.id === \"$COMMENT_ID\")" >/dev/null

printf 'Checking activity log\n'
api_get "/api/v1/issues/$ISSUE_ID/activity" | json_value 'data.activity.some((entry) => entry.action === "status_changed")' >/dev/null
api_get "/api/v1/issues/$ISSUE_ID/activity" | json_value 'data.activity.some((entry) => entry.action === "comment_added")' >/dev/null

printf 'API smoke test passed\n'
