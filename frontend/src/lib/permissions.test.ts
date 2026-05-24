import assert from "node:assert/strict";
import test from "node:test";

import {
  PROJECT_PERMISSION_NOTE,
  TEAM_PERMISSION_NOTE,
} from "./permissions.ts";

test("keeps member permission notes consistently read-only", () => {
  assert.equal(PROJECT_PERMISSION_NOTE.eyebrow, "Read-only");
  assert.equal(TEAM_PERMISSION_NOTE.eyebrow, "Read-only");
  assert.equal(PROJECT_PERMISSION_NOTE.title, "Project management");
  assert.equal(TEAM_PERMISSION_NOTE.title, "Team management");
});

test("documents admin-only project and team actions", () => {
  assert.match(PROJECT_PERMISSION_NOTE.body, /Creating, editing, and archiving/);
  assert.match(TEAM_PERMISSION_NOTE.body, /Creating members, editing roles/);
  assert.match(TEAM_PERMISSION_NOTE.body, /resetting passwords/);
});
