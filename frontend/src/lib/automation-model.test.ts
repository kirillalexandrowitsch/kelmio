import assert from "node:assert/strict";
import { test } from "vitest";

import type {
  AutomationRule,
  ProjectMember,
  ProjectWorkflow,
  TeamMember,
} from "./api-types";
import {
  automationDisabledReasonLabel,
  automationItemSummary,
  automationProjectUsers,
  emptyAutomationRuleInput,
  hasMissingAutomationDependency,
  moveAutomationRule,
  validateAutomationRuleInput,
} from "./automation-model";

test("validates automation rule limits, actions and duplicate conditions", () => {
  assert.equal(validateAutomationRuleInput(emptyAutomationRuleInput()), "Name is required.");
  assert.equal(
    validateAutomationRuleInput({
      ...emptyAutomationRuleInput(),
      name: "No action",
    }),
    "Add at least one action.",
  );
  assert.equal(
    validateAutomationRuleInput({
      ...emptyAutomationRuleInput(),
      name: "Duplicates",
      conditions: [
        { type: "priority", value: "high" },
        { type: "priority", value: "critical" },
      ],
      actions: [{ type: "change_priority", value: "low" }],
    }),
    "Scalar condition types cannot be repeated.",
  );
});

test("derives project users and identifies missing dependencies", () => {
  const users = [
    teamMember("admin", "admin"),
    teamMember("member", "member"),
    teamMember("outsider", "member"),
  ];
  const members = [projectMember("member")];
  assert.deepEqual(
    automationProjectUsers(users, members).map((user) => user.id),
    ["admin", "member"],
  );
  assert.equal(
    hasMissingAutomationDependency(
      {
        conditions: [{ type: "reporter", user_id: "outsider" }],
        actions: [{ type: "change_workflow_status", workflow_status_id: "missing" }],
      },
      workflow,
      users,
      [],
      members,
    ),
    true,
  );
});

test("keeps rule order and formats missing values and disabled reasons", () => {
  const rules = [rule("one", 100), rule("two", 200), rule("three", 300)];
  assert.deepEqual(moveAutomationRule(rules, "two", -1), ["two", "one", "three"]);
  assert.equal(
    automationItemSummary(
      { type: "change_workflow_status", workflow_status_id: "missing" },
      workflow,
      [],
      [],
      [],
    ),
    "change workflow status: Missing status",
  );
  assert.equal(
    automationDisabledReasonLabel("project_access_removed"),
    "A user no longer has project access.",
  );
});

const workflow: ProjectWorkflow = {
  project_id: "project",
  statuses: [],
  transitions: [],
};

function teamMember(id: string, role: TeamMember["role"]): TeamMember {
  return {
    id,
    email: `${id}@example.com`,
    username: id,
    display_name: id,
    role,
    is_active: true,
    joined_at: "2026-06-14T00:00:00Z",
  };
}

function projectMember(userId: string): ProjectMember {
  return {
    project_id: "project",
    user_id: userId,
    email: `${userId}@example.com`,
    username: userId,
    display_name: userId,
    role: "contributor",
    workspace_role: "member",
    is_active: true,
    created_at: "2026-06-14T00:00:00Z",
    updated_at: "2026-06-14T00:00:00Z",
  };
}

function rule(id: string, position: number): AutomationRule {
  return {
    id,
    project_id: "project",
    name: id,
    trigger_type: "issue_created",
    conditions: [],
    actions: [{ type: "change_priority", value: "high" }],
    position,
    is_enabled: true,
    disabled_reason: null,
    created_by: "admin",
    created_at: "2026-06-14T00:00:00Z",
    updated_at: "2026-06-14T00:00:00Z",
  };
}
