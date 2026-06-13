import assert from "node:assert/strict";
import { fireEvent, render, screen } from "@testing-library/react";
import { test, vi } from "vitest";

import type {
  Issue,
  Project,
  ProjectWorkflow,
  ProjectWorkflowStatus,
} from "../../lib/api-types";
import { BoardSection } from "./board-section";

const project: Project = {
  id: "project-1",
  key: "FLOW",
  name: "Workflow Project",
  description: "",
  created_by: "admin-1",
  created_at: "2026-06-13T00:00:00Z",
  archived_at: null,
  project_role: "lead",
  can_write: true,
  can_manage: true,
};

const todo = status("todo", "Todo", 100);
const review = status("review", "Ready for review", 200);
const done = status("done", "Done", 300, "done");
const archived = {
  ...status("archived", "Archived", 50),
  archived_at: "2026-06-13T00:00:00Z",
};
const workflow: ProjectWorkflow = {
  project_id: project.id,
  statuses: [done, archived, review, todo],
  transitions: [
    {
      from_status_id: todo.id,
      to_status_id: review.id,
      created_at: "2026-06-13T00:00:00Z",
    },
  ],
};
const issue: Issue = {
  id: "issue-1",
  project_id: project.id,
  project_key: project.key,
  number: 1,
  issue_key: "FLOW-1",
  title: "Move workflow card",
  description: "",
  issue_type: "task",
  status: todo.key,
  workflow_status: todo,
  priority: "medium",
  story_points: 3,
  reporter_id: "admin-1",
  assignee_id: null,
  parent_issue_id: null,
  sprint_id: null,
  due_date: null,
  labels: [],
  created_at: "2026-06-13T00:00:00Z",
  updated_at: "2026-06-13T00:00:00Z",
};

test("project board renders active workflow columns and allowed transitions", () => {
  const onIssueDrop = vi.fn();
  render(
    <BoardSection
      {...boardProps()}
      onIssueDrop={onIssueDrop}
    />,
  );

  assert.ok(screen.getByRole("heading", { name: "Todo" }));
  assert.ok(screen.getByRole("heading", { name: "Ready for review" }));
  assert.equal(screen.queryByRole("heading", { name: "Archived" }), null);

  const statusSelect = screen.getByLabelText("Status for FLOW-1");
  assert.ok(statusSelect.querySelector(`option[value="${todo.id}"]`));
  assert.ok(statusSelect.querySelector(`option[value="${review.id}"]`));
  assert.equal(statusSelect.querySelector(`option[value="${done.id}"]`), null);

  const dataTransfer = dragDataTransfer(issue.id);
  fireEvent.dragStart(screen.getByText(issue.title).closest("article")!, {
    dataTransfer,
  });

  const reviewColumn = screen
    .getByRole("heading", { name: "Ready for review" })
    .closest("article")!;
  const doneColumn = screen.getByRole("heading", { name: "Done" }).closest("article")!;
  assert.equal(reviewColumn.classList.contains("board-column-drop-allowed"), true);
  assert.equal(doneColumn.classList.contains("board-column-drop-disabled"), true);

  fireEvent.drop(reviewColumn, { dataTransfer });
  assert.equal(onIssueDrop.mock.calls[0]?.[1], review.id);
});

test("project board keeps viewer controls read-only", () => {
  render(
    <BoardSection
      {...boardProps()}
      projects={[{ ...project, project_role: "viewer", can_write: false }]}
    />,
  );

  assert.ok(screen.getByText("This board is read-only for your project role."));
  assert.equal(screen.queryByRole("button", { name: "Archive" }), null);
  assert.equal(screen.getByLabelText("Status for FLOW-1").hasAttribute("disabled"), true);
  assert.equal(
    screen.getByText(issue.title).closest("article")?.getAttribute("draggable"),
    "false",
  );
});

test("project board exposes empty loading and error states", () => {
  const { rerender } = render(<BoardSection {...boardProps()} projectId="" />);
  assert.ok(screen.getByText("Select a project to open its board"));

  rerender(<BoardSection {...boardProps()} isLoading workflow={undefined} />);
  assert.ok(screen.getByText("Loading project board"));

  rerender(<BoardSection {...boardProps()} error="Could not load project board." />);
  assert.ok(screen.getByText("Could not load project board."));
});

function boardProps() {
  return {
    archivingIssueIds: [],
    error: "",
    isActive: true,
    isLoading: false,
    issues: [issue],
    onArchiveIssue: vi.fn(),
    onIssueDrop: vi.fn(),
    onOpenIssue: vi.fn(),
    onProjectChange: vi.fn(),
    onTransitionIssue: vi.fn(),
    projectId: project.id,
    projects: [project],
    teamMembers: [],
    today: new Date(2026, 5, 13),
    transitioningIssueIds: [],
    workflow,
    workflowError: "",
  };
}

function status(
  key: string,
  name: string,
  position: number,
  category: ProjectWorkflowStatus["category"] = "in_progress",
): ProjectWorkflowStatus {
  return {
    id: `status-${key}`,
    project_id: project.id,
    key,
    name,
    color: "#0ea5e9",
    category,
    position,
    created_at: "2026-06-13T00:00:00Z",
    updated_at: "2026-06-13T00:00:00Z",
    archived_at: null,
  };
}

function dragDataTransfer(issueId: string) {
  const values = new Map<string, string>();
  values.set("text/plain", issueId);
  return {
    dropEffect: "none",
    effectAllowed: "all",
    getData: (type: string) => values.get(type) ?? "",
    setData: (type: string, value: string) => values.set(type, value),
  };
}
