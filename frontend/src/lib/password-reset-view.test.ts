import assert from "node:assert/strict";
import { test } from "vitest";

import { ApiError } from "./api.ts";
import {
  normalizedPasswordResetComplete,
  normalizedPasswordResetEmail,
  passwordResetTokenErrorMessage,
  validatePasswordResetComplete,
  validatePasswordResetEmail,
} from "./password-reset-view.ts";

test("validates and normalizes password reset email", () => {
  assert.equal(validatePasswordResetEmail("member@example.com"), "");
  assert.equal(validatePasswordResetEmail("missing-email"), "Email is invalid.");
  assert.equal(
    normalizedPasswordResetEmail(" MEMBER@Example.COM "),
    "member@example.com",
  );
});

test("validates password reset completion", () => {
  assert.equal(
    validatePasswordResetComplete({
      password: "short",
      confirmPassword: "short",
    }),
    "Password must be at least 8 characters.",
  );
  assert.equal(
    validatePasswordResetComplete({
      password: "new-password",
      confirmPassword: "other-password",
    }),
    "Password confirmation does not match.",
  );
  assert.equal(
    validatePasswordResetComplete({
      password: "new-password",
      confirmPassword: "new-password",
    }),
    "",
  );
  assert.deepEqual(
    normalizedPasswordResetComplete({
      password: " new-password ",
      confirmPassword: " new-password ",
    }),
    { password: "new-password", confirmPassword: "new-password" },
  );
});

test("maps stable password reset token errors", () => {
  assert.equal(
    passwordResetTokenErrorMessage(
      new ApiError("not found", 404, "password_reset_not_found"),
    ),
    "Password reset link was not found. Request a new link.",
  );
  assert.equal(
    passwordResetTokenErrorMessage(
      new ApiError("expired", 400, "password_reset_expired"),
    ),
    "Password reset link has expired. Request a new link.",
  );
  assert.equal(
    passwordResetTokenErrorMessage(new ApiError("used", 400, "password_reset_used")),
    "Password reset link was already used. Request a new link.",
  );
  assert.equal(
    passwordResetTokenErrorMessage(
      new ApiError("revoked", 400, "password_reset_revoked"),
    ),
    "Password reset link was revoked. Request a new link.",
  );
});
