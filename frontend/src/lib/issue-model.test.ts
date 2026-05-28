import assert from "node:assert/strict";
import test from "node:test";

import {
  issueDueInfo,
  issueLabelIds,
  issueMatchesFilters,
  editableIssueTypeOptions,
  rootIssueTypeOptions,
  statusLabel,
} from "./issue-model.ts";
import { type Issue } from "./api-types.ts";

function makeIssue(overrides: Partial<Issue> = {}): Issue {
  return {
    id: "issue-1",
    project_id: "project-1",
    project_key: "CORE",
    number: 1,
    issue_key: "CORE-1",
    title: "Build routing foundation",
    description: "Add direct app routes",
    issue_type: "task",
    status: "todo",
    priority: "medium",
    reporter_id: "user-1",
    assignee_id: "user-2",
    parent_issue_id: null,
    due_date: "2026-05-28",
    labels: [{ id: "label-1", name: "frontend", color: "#4e795d" }],
    created_at: "2026-05-26T10:00:00Z",
    updated_at: "2026-05-26T10:00:00Z",
    ...overrides,
  };
}

test("labels known and unknown statuses for display", () => {
  assert.equal(statusLabel("in_progress"), "In progress");
  assert.equal(statusLabel("custom_status"), "custom_status");
});

test("extracts issue label ids in display order", () => {
  assert.deepEqual(
    issueLabelIds(
      makeIssue({
        labels: [
          { id: "backend", name: "backend", color: "#3662a1" },
          { id: "bug", name: "bug", color: "#923c2d" },
        ],
      }),
    ),
    ["backend", "bug"],
  );
});

test("keeps safe issue type options for root and child issues", () => {
  assert.deepEqual(rootIssueTypeOptions, ["task", "bug", "story", "epic"]);
  assert.deepEqual(editableIssueTypeOptions(makeIssue()), [
    "task",
    "bug",
    "story",
    "epic",
  ]);
  assert.deepEqual(
    editableIssueTypeOptions(makeIssue({ parent_issue_id: "parent-1" })),
    ["task", "bug", "story", "subtask"],
  );
});

test("describes due state relative to today", () => {
  const today = new Date(2026, 4, 26);

  assert.deepEqual(issueDueInfo(makeIssue({ due_date: "2026-05-26" }), today), {
    label: "Due today",
    tone: "due-soon",
  });
  assert.deepEqual(issueDueInfo(makeIssue({ due_date: "2026-05-25" }), today), {
    label: "Overdue by 1 day",
    tone: "overdue",
  });
  assert.deepEqual(issueDueInfo(makeIssue({ status: "done" }), today), {
    label: "Done, due 2026-05-28",
    tone: "done",
  });
});

test("matches combined issue filters", () => {
  const today = new Date(2026, 4, 26);
  const issue = makeIssue();

  assert.equal(
    issueMatchesFilters(
      issue,
      "project-1",
      "todo",
      "medium",
      "user-2",
      "label-1",
      "due_soon",
      "routing",
      today,
    ),
    true,
  );
  assert.equal(
    issueMatchesFilters(issue, "other-project", "", "", "", "", "", "", today),
    false,
  );
  assert.equal(
    issueMatchesFilters(issue, "", "", "", "unassigned", "", "", "", today),
    false,
  );
});
