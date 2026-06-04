import assert from "node:assert/strict";
import test from "node:test";

import {
  buildInviteAcceptURL,
  inviteStatusLabel,
  normalizedInviteAcceptInput,
  normalizedInviteEmail,
  validateInviteAcceptForm,
  validateInviteEmail,
} from "./invite-view.ts";

test("formats invite statuses", () => {
  assert.equal(inviteStatusLabel("pending"), "Pending");
  assert.equal(inviteStatusLabel("accepted"), "Accepted");
  assert.equal(inviteStatusLabel("revoked"), "Revoked");
  assert.equal(inviteStatusLabel("expired"), "Expired");
});

test("builds invite accept URLs from origin and path", () => {
  assert.equal(
    buildInviteAcceptURL("/accept-invite?token=abc", "http://localhost:5173"),
    "http://localhost:5173/accept-invite?token=abc",
  );
  assert.equal(
    buildInviteAcceptURL("/accept-invite?token=abc", "http://localhost:5173/"),
    "http://localhost:5173/accept-invite?token=abc",
  );
});

test("validates invite create email", () => {
  assert.equal(normalizedInviteEmail(" New.Member@Example.COM "), "new.member@example.com");
  assert.equal(validateInviteEmail("new.member@example.com"), "");
  assert.match(validateInviteEmail("not-email"), /Email is invalid/);
});

test("validates invite accept form", () => {
  assert.equal(
    validateInviteAcceptForm({
      username: "new_member",
      displayName: "New Member",
      password: "password123",
      confirmPassword: "password123",
    }),
    "",
  );
  assert.match(
    validateInviteAcceptForm({
      username: "NO",
      displayName: "New Member",
      password: "password123",
      confirmPassword: "password123",
    }),
    /Username must be/,
  );
  assert.match(
    validateInviteAcceptForm({
      username: "new_member",
      displayName: "",
      password: "password123",
      confirmPassword: "password123",
    }),
    /Display name is required/,
  );
  assert.match(
    validateInviteAcceptForm({
      username: "new_member",
      displayName: "New Member",
      password: "short",
      confirmPassword: "short",
    }),
    /Password must be/,
  );
  assert.match(
    validateInviteAcceptForm({
      username: "new_member",
      displayName: "New Member",
      password: "password123",
      confirmPassword: "password456",
    }),
    /confirmation does not match/,
  );
});

test("normalizes invite accept payload", () => {
  assert.deepEqual(
    normalizedInviteAcceptInput({
      username: " New_Member ",
      displayName: " New Member ",
      password: " password123 ",
      confirmPassword: " password123 ",
    }),
    {
      username: "new_member",
      display_name: "New Member",
      password: "password123",
    },
  );
});
