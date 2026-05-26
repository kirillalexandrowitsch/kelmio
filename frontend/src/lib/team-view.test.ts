import assert from "node:assert/strict";
import test from "node:test";

import {
  activeTeamMembers,
  assignableTeamMembers,
  memberDisplayName,
  memberInitials,
  memberOptionLabel,
} from "./team-view.ts";
import { type TeamMember } from "./api-types.ts";

const members: TeamMember[] = [
  {
    id: "admin",
    email: "admin@example.com",
    username: "admin",
    display_name: "Admin User",
    role: "admin",
    is_active: true,
    joined_at: "2026-05-26T10:00:00Z",
  },
  {
    id: "inactive-member",
    email: "inactive@example.com",
    username: "inactive",
    display_name: "Inactive Member",
    role: "member",
    is_active: false,
    joined_at: "2026-05-26T10:00:00Z",
  },
];

test("formats member display helpers", () => {
  assert.equal(memberInitials("Admin User"), "AU");
  assert.equal(memberInitials(""), "TM");
  assert.equal(memberDisplayName(members, "admin"), "Admin User");
  assert.equal(memberDisplayName(members, null), "Unassigned");
  assert.equal(memberDisplayName(members, "missing"), "missing");
});

test("keeps inactive current assignee assignable", () => {
  assert.deepEqual(activeTeamMembers(members).map((member) => member.id), ["admin"]);
  assert.deepEqual(
    assignableTeamMembers(members, "inactive-member").map((member) => member.id),
    ["admin", "inactive-member"],
  );
  assert.equal(memberOptionLabel(members[1]), "Inactive Member (inactive)");
});
