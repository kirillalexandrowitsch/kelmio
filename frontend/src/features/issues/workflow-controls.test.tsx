import assert from "node:assert/strict";
import { type ComponentProps } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { test, vi } from "vitest";

import type {
  Issue,
  Project,
  ProjectWorkflowStatus,
} from "../../lib/api-types";
import { IssueCreateForm } from "./issue-create-form";
import { IssueDetailSidebar } from "./issue-detail-sidebar";
import { IssueListPanel } from "./issue-list-panel";

const project: Project = {
  id: "project-1",
  key: "FLOW",
  name: "Workflow Project",
  description: "",
  created_by: "admin-1",
  created_at: "2026-06-12T00:00:00Z",
  archived_at: null,
  project_role: "lead",
  can_write: true,
  can_manage: true,
};

const todoStatus = workflowStatus("todo", "Todo", 100);
const reviewStatus = workflowStatus("review", "Ready for review", 200);
const doneStatus = workflowStatus("done", "Done", 300, "done");

const issue: Issue = {
  id: "issue-1",
  project_id: project.id,
  project_key: project.key,
  number: 1,
  issue_key: "FLOW-1",
  title: "Review dynamic workflow",
  description: "",
  issue_type: "task",
  status: reviewStatus.key,
  workflow_status: reviewStatus,
  priority: "medium",
  story_points: 3,
  reporter_id: "admin-1",
  assignee_id: null,
  parent_issue_id: null,
  sprint_id: null,
  due_date: null,
  labels: [],
  created_at: "2026-06-12T00:00:00Z",
  updated_at: "2026-06-12T00:00:00Z",
};

test("issue create form exposes project workflow statuses", async () => {
  const user = userEvent.setup();
  const onStatusChange = vi.fn();

  render(
    <IssueCreateForm
      {...createFormProps()}
      onStatusChange={onStatusChange}
      projectId={project.id}
      statusId={todoStatus.id}
      statuses={[todoStatus, reviewStatus]}
    />,
  );

  await user.selectOptions(screen.getByLabelText("Status"), reviewStatus.id);

  assert.equal(onStatusChange.mock.calls[0]?.[0], reviewStatus.id);
  assert.ok(screen.getByText("Ready for review", { exact: false }));
});

test("issue status filter requires a project and preserves legacy values", async () => {
  const user = userEvent.setup();
  const onWorkflowStatusFilterChange = vi.fn();
  const { rerender } = render(
    <IssueListPanel
      {...issueListProps()}
      legacyStatusFilter=""
      projectFilterId=""
      workflowStatusFilterId=""
      workflowStatuses={[]}
    />,
  );

  assert.equal(screen.getByLabelText("Status").hasAttribute("disabled"), true);

  rerender(
    <IssueListPanel
      {...issueListProps()}
      legacyStatusFilter=""
      onWorkflowStatusFilterChange={onWorkflowStatusFilterChange}
      projectFilterId={project.id}
      workflowStatusFilterId=""
      workflowStatuses={[todoStatus, reviewStatus]}
    />,
  );
  await user.selectOptions(screen.getByLabelText("Status"), reviewStatus.id);
  assert.equal(onWorkflowStatusFilterChange.mock.calls[0]?.[0], reviewStatus.id);

  rerender(
    <IssueListPanel
      {...issueListProps()}
      legacyStatusFilter="blocked"
      projectFilterId={project.id}
      workflowStatusFilterId=""
      workflowStatuses={[todoStatus, reviewStatus]}
    />,
  );
  assert.equal(
    (screen.getByLabelText("Status") as HTMLSelectElement).value,
    "legacy:blocked",
  );
  assert.ok(screen.getByRole("option", { name: "Legacy status: blocked" }));
});

test("issue detail limits transitions and disables viewer controls", () => {
  const { rerender } = render(
    <IssueDetailSidebar
      assigningIssueIds={[]}
      canWriteIssue
      issue={issue}
      labelingIssueIds={[]}
      labels={[]}
      onAssignIssue={vi.fn()}
      onSetIssueLabel={vi.fn()}
      onTransitionIssue={vi.fn()}
      teamMembers={[]}
      today={new Date(2026, 5, 12)}
      transitioningIssueIds={[]}
      transitionStatuses={[reviewStatus, doneStatus]}
    />,
  );

  assert.ok(screen.getByRole("option", { name: "Ready for review" }));
  assert.ok(screen.getByRole("option", { name: "Done" }));
  assert.equal(screen.queryByRole("option", { name: "Todo" }), null);

  rerender(
    <IssueDetailSidebar
      assigningIssueIds={[]}
      canWriteIssue={false}
      issue={issue}
      labelingIssueIds={[]}
      labels={[]}
      onAssignIssue={vi.fn()}
      onSetIssueLabel={vi.fn()}
      onTransitionIssue={vi.fn()}
      teamMembers={[]}
      today={new Date(2026, 5, 12)}
      transitioningIssueIds={[]}
      transitionStatuses={[reviewStatus, doneStatus]}
    />,
  );
  assert.equal(screen.getAllByRole("combobox")[0]?.hasAttribute("disabled"), true);
});

function createFormProps(): ComponentProps<typeof IssueCreateForm> {
  return {
    assigneeId: "",
    canCreateIssue: true,
    description: "",
    dueDate: "",
    formError: "",
    isCreatingIssue: false,
    labels: [],
    labelIds: [],
    onAssigneeChange: vi.fn(),
    onCreateIssue: vi.fn(),
    onDescriptionChange: vi.fn(),
    onDueDateChange: vi.fn(),
    onLabelChange: vi.fn(),
    onPriorityChange: vi.fn(),
    onProjectChange: vi.fn(),
    onStoryPointsChange: vi.fn(),
    onStatusChange: vi.fn(),
    onTitleChange: vi.fn(),
    onTypeChange: vi.fn(),
    priority: "medium",
    projectId: "",
    projects: [project],
    statusId: "",
    statuses: [],
    storyPoints: "0",
    teamMembers: [],
    title: "Dynamic workflow issue",
    type: "task",
  };
}

function issueListProps(): ComponentProps<typeof IssueListPanel> {
  return {
    archivingIssueIds: [],
    assigneeFilterId: "",
    dueFilter: "",
    isLoadingIssues: false,
    issues: [issue],
    issuesError: "",
    labelFilterId: "",
    labels: [],
    legacyStatusFilter: "",
    onArchiveIssue: vi.fn(),
    onAssigneeFilterChange: vi.fn(),
    onClearFilters: vi.fn(),
    onDueFilterChange: vi.fn(),
    onLabelFilterChange: vi.fn(),
    onOpenIssue: vi.fn(),
    onPriorityFilterChange: vi.fn(),
    onProjectFilterChange: vi.fn(),
    onQueryChange: vi.fn(),
    onSortChange: vi.fn(),
    onSprintFilterChange: vi.fn(),
    onWorkflowStatusFilterChange: vi.fn(),
    priorityFilter: "",
    projectFilterId: project.id,
    projects: [project],
    query: "",
    sort: "created_desc",
    sprintFilterId: "",
    sprints: [],
    teamMembers: [],
    today: new Date(2026, 5, 12),
    workflowStatusFilterId: "",
    workflowStatuses: [todoStatus, reviewStatus],
  };
}

function workflowStatus(
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
    created_at: "2026-06-12T00:00:00Z",
    updated_at: "2026-06-12T00:00:00Z",
    archived_at: null,
  };
}
