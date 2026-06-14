import assert from "node:assert/strict";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { test, vi } from "vitest";

import type {
  AutomationRule,
  ProjectWorkflow,
  TeamMember,
} from "../../lib/api-types";
import { AutomationSettingsPanel } from "./automation-settings-panel";

test("creates a typed rule and preserves action order", async () => {
  const user = userEvent.setup();
  const props = panelProps();
  render(<AutomationSettingsPanel {...props} />);

  await user.type(screen.getByLabelText("Automation rule name"), "Escalate bug");
  await user.click(screen.getByRole("button", { name: "Add condition" }));
  await user.click(screen.getByRole("button", { name: "Add action" }));
  await user.selectOptions(screen.getByLabelText("Action 1 type"), "change_priority");
  await user.selectOptions(screen.getByLabelText("change priority value"), "critical");
  await user.click(screen.getByRole("button", { name: "Add action" }));
  await user.selectOptions(screen.getByLabelText("Action 2 type"), "change_workflow_status");
  await user.selectOptions(screen.getByLabelText("change workflow status value"), "done");
  await user.click(screen.getByRole("button", { name: "Create rule" }));

  assert.deepEqual(props.onCreateRule.mock.calls[0]?.[0], {
    name: "Escalate bug",
    trigger_type: "issue_created",
    conditions: [{ type: "issue_type", value: "task" }],
    actions: [
      { type: "change_priority", value: "critical" },
      { type: "change_workflow_status", workflow_status_id: "done" },
    ],
    is_enabled: true,
  });
});

test("renders disabled reason, toggles, reorders and deletes rules", async () => {
  const user = userEvent.setup();
  const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
  const props = panelProps([disabledRule, enabledRule]);
  render(<AutomationSettingsPanel {...props} />);

  assert.ok(screen.getByText("A label is unavailable."));
  const enabledCard = screen.getByText("Enabled rule").closest("article")!;
  await user.click(within(enabledCard).getByRole("button", { name: "Disable" }));
  assert.deepEqual(props.onUpdateRule.mock.calls[0]?.[1], { is_enabled: false });
  await user.click(screen.getByRole("button", { name: "Move Enabled rule up" }));
  assert.deepEqual(props.onReorderRules.mock.calls[0]?.[0], [
    enabledRule.id,
    disabledRule.id,
  ]);
  await user.click(within(enabledCard).getByRole("button", { name: "Delete" }));
  assert.equal(props.onDeleteRule.mock.calls[0]?.[0].id, enabledRule.id);
  confirm.mockRestore();
});

test("opens missing dependency in editor and blocks enabling until repaired", async () => {
  const user = userEvent.setup();
  const props = panelProps([disabledRule]);
  render(<AutomationSettingsPanel {...props} />);

  await user.click(screen.getByRole("button", { name: "Enable" }));
  assert.ok(screen.getByText("Repair missing dependencies before enabling this rule."));
  assert.ok(screen.getByRole("option", { name: "Missing label" }));
  assert.equal(props.onUpdateRule.mock.calls.length, 0);
});

function panelProps(rules: AutomationRule[] = []) {
  return {
    creatingRule: false,
    deletingRuleIds: [],
    error: "",
    isLoading: false,
    isReordering: false,
    labels: [],
    members: [],
    onCreateRule: vi.fn(async () => true),
    onDeleteRule: vi.fn(async () => true),
    onReorderRules: vi.fn(async () => true),
    onUpdateRule: vi.fn(async () => true),
    rules,
    teamMembers: [] as TeamMember[],
    updatingRuleIds: [],
    workflow,
  };
}

const workflow: ProjectWorkflow = {
  project_id: "project",
  statuses: [
    {
      id: "done",
      project_id: "project",
      key: "done",
      name: "Done",
      color: "#16a34a",
      category: "done",
      position: 100,
      created_at: "2026-06-14T00:00:00Z",
      updated_at: "2026-06-14T00:00:00Z",
      archived_at: null,
    },
  ],
  transitions: [],
};

const disabledRule: AutomationRule = {
  id: "disabled",
  project_id: "project",
  name: "Disabled rule",
  trigger_type: "issue_created",
  conditions: [{ type: "label", label_id: "missing-label" }],
  actions: [{ type: "change_priority", value: "high" }],
  position: 100,
  is_enabled: false,
  disabled_reason: "label_unavailable",
  created_by: "admin",
  created_at: "2026-06-14T00:00:00Z",
  updated_at: "2026-06-14T00:00:00Z",
};

const enabledRule: AutomationRule = {
  ...disabledRule,
  id: "enabled",
  name: "Enabled rule",
  conditions: [],
  position: 200,
  is_enabled: true,
  disabled_reason: null,
};
