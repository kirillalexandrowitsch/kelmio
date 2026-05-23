import assert from "node:assert/strict";
import test from "node:test";

import {
  hasMinTrimmedLength,
  hasText,
  isValidEmail,
  isValidLabelColor,
  isValidUsername,
  normalizeEmail,
  normalizeLabelColor,
  normalizeText,
  normalizeUsername,
} from "./validation.ts";

test("normalizes common form values", () => {
  assert.equal(normalizeText("  Core Platform  "), "Core Platform");
  assert.equal(normalizeEmail(" Admin@Example.COM "), "admin@example.com");
  assert.equal(normalizeUsername(" Demo_User "), "demo_user");
  assert.equal(normalizeLabelColor(" #CCE8D4 "), "#cce8d4");
});

test("validates required text and minimum trimmed length", () => {
  assert.equal(hasText(" issue title "), true);
  assert.equal(hasText("   "), false);
  assert.equal(hasMinTrimmedLength(" password123 ", 8), true);
  assert.equal(hasMinTrimmedLength(" short ", 8), false);
});

test("validates user identity fields", () => {
  assert.equal(isValidEmail("member@example.com"), true);
  assert.equal(isValidEmail("not-email"), false);
  assert.equal(isValidUsername("member_name"), true);
  assert.equal(isValidUsername("NO"), false);
});

test("validates label colors", () => {
  assert.equal(isValidLabelColor("#4e795d"), true);
  assert.equal(isValidLabelColor("#CCE8D4"), true);
  assert.equal(isValidLabelColor("green"), false);
});
