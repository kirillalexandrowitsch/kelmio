#!/bin/sh
set -eu

API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
TRUSTED_ORIGIN="${TRUSTED_ORIGIN:-http://localhost:5173}"
ADMIN_LOGIN="${ADMIN_LOGIN:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin12345}"
RATE_LIMIT_LOGIN_PER_MINUTE="${RATE_LIMIT_LOGIN_PER_MINUTE:-10}"
EXPECT_SECURE_COOKIE="${EXPECT_SECURE_COOKIE:-false}"
EXPECT_HSTS="${EXPECT_HSTS:-false}"

API_BASE_URL="${API_BASE_URL%/}"
COOKIE_JAR="$(mktemp "${TMPDIR:-/tmp}/team-task-tracker-production-smoke-cookies.XXXXXX")"
HEADERS_FILE="$(mktemp "${TMPDIR:-/tmp}/team-task-tracker-production-smoke-headers.XXXXXX")"
BODY_FILE="$(mktemp "${TMPDIR:-/tmp}/team-task-tracker-production-smoke-body.XXXXXX")"
LARGE_BODY_FILE="$(mktemp "${TMPDIR:-/tmp}/team-task-tracker-production-smoke-large-body.XXXXXX")"

cleanup() {
	if [ -s "$COOKIE_JAR" ]; then
		cleanup_csrf_token="$(curl -fsS -b "$COOKIE_JAR" "$API_BASE_URL/api/v1/auth/csrf-token" 2>/dev/null | json_value 'data.csrf_token' 2>/dev/null || true)"
		if [ -n "$cleanup_csrf_token" ]; then
			curl -fsS -b "$COOKIE_JAR" \
				-X POST \
				-H "X-CSRF-Token: $cleanup_csrf_token" \
				"$API_BASE_URL/api/v1/auth/logout" >/dev/null 2>&1 || true
		fi
	fi
	rm -f "$COOKIE_JAR" "$HEADERS_FILE" "$BODY_FILE" "$LARGE_BODY_FILE"
}
trap cleanup EXIT

fail() {
	printf '%s\n' "$1" >&2
	exit 1
}

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		fail "Missing required command: $1"
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

header_value() {
	node -e '
const fs = require("node:fs");
const key = process.argv[2].toLowerCase();
const lines = fs.readFileSync(process.argv[1], "utf8").split(/\r?\n/);
const values = lines
  .map((line) => {
    const separator = line.indexOf(":");
    if (separator < 0 || line.slice(0, separator).toLowerCase() !== key) {
      return "";
    }
    return line.slice(separator + 1).trim();
  })
  .filter(Boolean);
process.stdout.write(values.at(-1) ?? "");
' "$1" "$2"
}

assert_status() {
	if [ "$1" != "$2" ]; then
		fail "Expected HTTP status $2, got $1. Body: $(cat "$BODY_FILE")"
	fi
}

assert_header_equals() {
	actual="$(header_value "$HEADERS_FILE" "$1")"
	if [ "$actual" != "$2" ]; then
		fail "Expected header $1 to equal '$2', got '$actual'"
	fi
}

assert_header_contains() {
	actual="$(header_value "$HEADERS_FILE" "$1")"
	case "$actual" in
		*"$2"*) ;;
		*) fail "Expected header $1 to contain '$2', got '$actual'" ;;
	esac
}

assert_header_empty() {
	actual="$(header_value "$HEADERS_FILE" "$1")"
	if [ -n "$actual" ]; then
		fail "Expected header $1 to be empty, got '$actual'"
	fi
}

assert_header_present() {
	actual="$(header_value "$HEADERS_FILE" "$1")"
	if [ -z "$actual" ]; then
		fail "Expected header $1 to be present"
	fi
}

assert_generated_request_id() {
	node -e '
if (!/^[0-9a-f]{32}$/.test(process.argv[1])) {
  process.exit(1);
}
' "$1" || fail "Expected generated X-Request-ID to be 32 lowercase hex characters, got '$1'"
}

require_command curl
require_command dd
require_command node
require_command tr

printf 'Checking production-sensitive API smoke at %s\n' "$API_BASE_URL"

status="$(curl -sS -D "$HEADERS_FILE" -o "$BODY_FILE" -w '%{http_code}' "$API_BASE_URL/healthz")"
assert_status "$status" "200"
json_value 'data.status === "ok"' <"$BODY_FILE" >/dev/null
assert_header_equals "X-Content-Type-Options" "nosniff"
assert_header_equals "X-Frame-Options" "DENY"
assert_header_equals "Referrer-Policy" "no-referrer"
assert_header_equals "Cross-Origin-Opener-Policy" "same-origin"
assert_header_equals "Permissions-Policy" "camera=(), microphone=(), geolocation=()"

generated_request_id="$(header_value "$HEADERS_FILE" "X-Request-ID")"
assert_generated_request_id "$generated_request_id"

status="$(curl -sS -D "$HEADERS_FILE" -o "$BODY_FILE" -w '%{http_code}' \
	-H "X-Request-ID: production-smoke-123" \
	"$API_BASE_URL/api/v1/version")"
assert_status "$status" "200"
assert_header_equals "X-Request-ID" "production-smoke-123"
json_value 'typeof data.version === "string" && typeof data.commit === "string" && typeof data.environment === "string" && Object.prototype.hasOwnProperty.call(data, "build_time")' <"$BODY_FILE" >/dev/null

status="$(curl -sS -D "$HEADERS_FILE" -o "$BODY_FILE" -w '%{http_code}' \
	-H "X-Request-ID: invalid/request?id=secret" \
	"$API_BASE_URL/healthz")"
assert_status "$status" "200"
replacement_request_id="$(header_value "$HEADERS_FILE" "X-Request-ID")"
if [ "$replacement_request_id" = "invalid/request?id=secret" ] || [ -z "$replacement_request_id" ]; then
	fail "Expected invalid inbound X-Request-ID to be replaced"
fi
assert_generated_request_id "$replacement_request_id"

if [ "$EXPECT_HSTS" = "true" ]; then
	assert_header_contains "Strict-Transport-Security" "max-age=31536000"
else
	assert_header_empty "Strict-Transport-Security"
fi

printf 'Checking trusted and untrusted CORS preflight\n'
status="$(curl -sS -D "$HEADERS_FILE" -o "$BODY_FILE" -w '%{http_code}' \
	-X OPTIONS \
	-H "Origin: $TRUSTED_ORIGIN" \
	-H "Access-Control-Request-Method: PATCH" \
	-H "Access-Control-Request-Headers: Content-Type, X-CSRF-Token, X-Request-ID" \
	"$API_BASE_URL/api/v1/projects/example")"
assert_status "$status" "204"
assert_header_equals "Access-Control-Allow-Origin" "$TRUSTED_ORIGIN"
assert_header_equals "Access-Control-Allow-Credentials" "true"
assert_header_contains "Access-Control-Allow-Headers" "X-CSRF-Token"
assert_header_contains "Access-Control-Allow-Headers" "X-Request-ID"
assert_header_contains "Access-Control-Allow-Methods" "PATCH"
assert_header_contains "Vary" "Origin"

status="$(curl -sS -D "$HEADERS_FILE" -o "$BODY_FILE" -w '%{http_code}' \
	-X OPTIONS \
	-H "Origin: https://untrusted.invalid" \
	-H "Access-Control-Request-Method: PATCH" \
	"$API_BASE_URL/api/v1/projects/example")"
assert_status "$status" "204"
assert_header_empty "Access-Control-Allow-Origin"
assert_header_empty "Access-Control-Allow-Credentials"

printf 'Checking session cookie and CSRF protection\n'
status="$(curl -sS -D "$HEADERS_FILE" -o "$BODY_FILE" -w '%{http_code}' \
	-c "$COOKIE_JAR" \
	-H "Content-Type: application/json" \
	-d "{\"login\":\"$ADMIN_LOGIN\",\"password\":\"$ADMIN_PASSWORD\"}" \
	"$API_BASE_URL/api/v1/auth/login")"
assert_status "$status" "200"
assert_header_contains "Set-Cookie" "team_task_tracker_session="
assert_header_contains "Set-Cookie" "Path=/"
assert_header_contains "Set-Cookie" "HttpOnly"
assert_header_contains "Set-Cookie" "SameSite=Lax"
session_cookie="$(header_value "$HEADERS_FILE" "Set-Cookie")"
case "$EXPECT_SECURE_COOKIE:$session_cookie" in
	true:*"; Secure"*) ;;
	true:*) fail "Expected session cookie to include Secure" ;;
	false:*"; Secure"*) fail "Expected development session cookie to omit Secure" ;;
	false:*) ;;
	*) fail "EXPECT_SECURE_COOKIE must be true or false" ;;
esac

status="$(curl -sS -D "$HEADERS_FILE" -o "$BODY_FILE" -w '%{http_code}' \
	-b "$COOKIE_JAR" \
	-X POST \
	"$API_BASE_URL/api/v1/auth/logout")"
assert_status "$status" "403"
json_value 'data.error.code === "csrf_token_required"' <"$BODY_FILE" >/dev/null

status="$(curl -sS -D "$HEADERS_FILE" -o "$BODY_FILE" -w '%{http_code}' \
	-b "$COOKIE_JAR" \
	-X POST \
	-H "X-CSRF-Token: invalid-token" \
	"$API_BASE_URL/api/v1/auth/logout")"
assert_status "$status" "403"
json_value 'data.error.code === "invalid_csrf_token"' <"$BODY_FILE" >/dev/null

csrf_token="$(curl -fsS -b "$COOKIE_JAR" "$API_BASE_URL/api/v1/auth/csrf-token" | json_value 'data.csrf_token')"

printf 'Checking request body limit\n'
dd if=/dev/zero bs=1048577 count=1 2>/dev/null | tr '\000' a >"$LARGE_BODY_FILE"
status="$(curl -sS -D "$HEADERS_FILE" -o "$BODY_FILE" -w '%{http_code}' \
	-H "Content-Type: application/json" \
	--data-binary "@$LARGE_BODY_FILE" \
	"$API_BASE_URL/api/v1/auth/login")"
assert_status "$status" "413"
json_value 'data.error.code === "request_too_large"' <"$BODY_FILE" >/dev/null

printf 'Checking login rate limiting\n'
rate_limit_login="production_smoke_missing_$(date +%s)_$$"
attempt=1
while [ "$attempt" -le "$RATE_LIMIT_LOGIN_PER_MINUTE" ]; do
	status="$(curl -sS -D "$HEADERS_FILE" -o "$BODY_FILE" -w '%{http_code}' \
		-H "Content-Type: application/json" \
		-d "{\"login\":\"$rate_limit_login\",\"password\":\"invalid-password\"}" \
		"$API_BASE_URL/api/v1/auth/login")"
	assert_status "$status" "401"
	attempt=$((attempt + 1))
done

status="$(curl -sS -D "$HEADERS_FILE" -o "$BODY_FILE" -w '%{http_code}' \
	-H "Content-Type: application/json" \
	-d "{\"login\":\"$rate_limit_login\",\"password\":\"invalid-password\"}" \
	"$API_BASE_URL/api/v1/auth/login")"
assert_status "$status" "429"
assert_header_present "Retry-After"
json_value 'data.error.code === "rate_limited"' <"$BODY_FILE" >/dev/null

printf 'Checking valid CSRF logout\n'
status="$(curl -sS -D "$HEADERS_FILE" -o "$BODY_FILE" -w '%{http_code}' \
	-b "$COOKIE_JAR" \
	-c "$COOKIE_JAR" \
	-X POST \
	-H "X-CSRF-Token: $csrf_token" \
	"$API_BASE_URL/api/v1/auth/logout")"
assert_status "$status" "204"

printf 'Production-sensitive API smoke passed\n'
