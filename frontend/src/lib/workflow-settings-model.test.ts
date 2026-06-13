import assert from "node:assert/strict";
import { test } from "vitest";

import type { ProjectWorkflowStatus } from "./api-types";
import {
  moveWorkflowStatus,
  normalizeWorkflowStatusInput,
  transitionDraftFromWorkflow,
  transitionKey,
  transitionsFromDraft,
  validateWorkflowStatusInput,
} from "./workflow-settings-model";

const todo = status("todo", 100);
const review = status("review", 200);
const done = status("done", 300);
const statuses = [todo, review, done];

test("normalizes and validates workflow status inputs", () => {
  const normalized = normalizeWorkflowStatusInput({
    key: "  READY_1 ",
    name: " Ready ",
    color: "#ABCDEF",
    category: "in_progress",
  });
  assert.deepEqual(normalized, {
    key: "ready_1",
    name: "Ready",
    color: "#abcdef",
    category: "in_progress",
  });
  assert.equal(validateWorkflowStatusInput(normalized), "");
  assert.match(
    validateWorkflowStatusInput({ ...normalized, key: "1 invalid" }),
    /lowercase identifier/,
  );
  assert.match(validateWorkflowStatusInput({ ...normalized, name: "" }), /required/);
  assert.match(
    validateWorkflowStatusInput({ ...normalized, color: "red" }),
    /#RRGGBB/,
  );
});

test("moves statuses without changing invalid boundary orders", () => {
  assert.deepEqual(moveWorkflowStatus(statuses, review.id, -1), [
    review.id,
    todo.id,
    done.id,
  ]);
  assert.deepEqual(moveWorkflowStatus(statuses, todo.id, -1), [
    todo.id,
    review.id,
    done.id,
  ]);
});

test("round-trips transition drafts and excludes invalid/self pairs", () => {
  const workflow = {
    project_id: "project-1",
    statuses,
    transitions: [
      {
        from_status_id: todo.id,
        to_status_id: review.id,
        created_at: "2026-06-13T00:00:00Z",
      },
    ],
  };
  const draft = transitionDraftFromWorkflow(workflow);
  draft.add(transitionKey(review.id, done.id));
  draft.add(transitionKey(done.id, done.id));
  draft.add(transitionKey("missing", done.id));

  assert.deepEqual(transitionsFromDraft(statuses, draft), [
    { from_status_id: todo.id, to_status_id: review.id },
    { from_status_id: review.id, to_status_id: done.id },
  ]);
});

function status(key: string, position: number): ProjectWorkflowStatus {
  return {
    id: `status-${key}`,
    project_id: "project-1",
    key,
    name: key === "review" ? "Ready for review" : key,
    color: "#0ea5e9",
    category: key === "done" ? "done" : "todo",
    position,
    created_at: "2026-06-13T00:00:00Z",
    updated_at: "2026-06-13T00:00:00Z",
    archived_at: null,
  };
}
