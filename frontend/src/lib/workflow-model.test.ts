import assert from "node:assert/strict";
import { test } from "vitest";

import type { ProjectWorkflow } from "./api-types";
import {
  activeWorkflowStatuses,
  allowedTransitionStatuses,
  defaultWorkflowStatus,
  workflowStatusLabel,
} from "./workflow-model";

const workflow: ProjectWorkflow = {
  project_id: "project-1",
  statuses: [
    status("done", "Done", 300),
    status("review", "Ready for review", 200),
    status("todo", "Todo", 100),
    { ...status("archived", "Archived", 50), archived_at: "2026-06-12T00:00:00Z" },
  ],
  transitions: [
    {
      from_status_id: "todo",
      to_status_id: "review",
      created_at: "2026-06-12T00:00:00Z",
    },
  ],
};

test("sorts active workflow statuses and chooses todo as default", () => {
  assert.deepEqual(
    activeWorkflowStatuses(workflow).map((item) => item.id),
    ["todo", "review", "done"],
  );
  assert.equal(defaultWorkflowStatus(workflow)?.id, "todo");
  assert.equal(defaultWorkflowStatus({ ...workflow, statuses: [status("review", "Review", 1)] })?.id, "review");
});

test("returns current and allowed transition statuses only", () => {
  assert.deepEqual(
    allowedTransitionStatuses(workflow, "todo").map((item) => item.id),
    ["todo", "review"],
  );
});

test("uses workflow display name with legacy fallback", () => {
  assert.equal(
    workflowStatusLabel({
      status: "review",
      workflow_status: {
        id: "review",
        key: "review",
        name: "Ready for review",
        color: "#0ea5e9",
        category: "in_progress",
      },
    }),
    "Ready for review",
  );
  assert.equal(
    workflowStatusLabel({
      status: "blocked",
      workflow_status: undefined as never,
    }),
    "Blocked",
  );
});

function status(id: string, name: string, position: number) {
  return {
    id,
    project_id: "project-1",
    key: id,
    name,
    color: "#0ea5e9",
    category: id === "done" ? ("done" as const) : ("in_progress" as const),
    position,
    created_at: "2026-06-12T00:00:00Z",
    updated_at: "2026-06-12T00:00:00Z",
    archived_at: null,
  };
}
