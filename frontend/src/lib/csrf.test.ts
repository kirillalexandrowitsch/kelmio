import assert from "node:assert/strict";
import { test } from "vitest";

import { isCSRFError, requestNeedsCSRF } from "./csrf.ts";

test("requires CSRF for unsafe authenticated requests", () => {
  assert.equal(requestNeedsCSRF("/api/v1/projects", "POST"), true);
  assert.equal(requestNeedsCSRF("/api/v1/projects/project-id", "PATCH"), true);
  assert.equal(requestNeedsCSRF("/api/v1/issues/issue-id/labels", "PUT"), true);
  assert.equal(requestNeedsCSRF("/api/v1/saved-filters/filter-id", "DELETE"), true);
});

test("does not require CSRF for safe requests and login", () => {
  assert.equal(requestNeedsCSRF("/api/v1/projects", "GET"), false);
  assert.equal(requestNeedsCSRF("/api/v1/auth/me"), false);
  assert.equal(requestNeedsCSRF("/api/v1/auth/login", "POST"), false);
  assert.equal(
    requestNeedsCSRF("/api/v1/auth/invites/invite-token/accept", "POST"),
    false,
  );
  assert.equal(
    requestNeedsCSRF("/api/v1/auth/password-reset/request", "POST"),
    false,
  );
  assert.equal(
    requestNeedsCSRF(
      "/api/v1/auth/password-reset/reset-token/complete",
      "POST",
    ),
    false,
  );
  assert.equal(requestNeedsCSRF("/api/v1/auth/csrf-token", "GET"), false);
});

test("detects backend CSRF error codes", () => {
  assert.equal(isCSRFError(403, "csrf_token_required"), true);
  assert.equal(isCSRFError(403, "invalid_csrf_token"), true);
  assert.equal(isCSRFError(401, "invalid_csrf_token"), false);
  assert.equal(isCSRFError(403, "forbidden"), false);
});
