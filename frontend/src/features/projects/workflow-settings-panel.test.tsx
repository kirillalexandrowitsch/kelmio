import assert from "node:assert/strict";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { test, vi } from "vitest";

import type {
  ProjectWorkflow,
  ProjectWorkflowStatus,
} from "../../lib/api-types";
import { WorkflowSettingsPanel } from "./workflow-settings-panel";

const todo = status("todo", "Todo", 100, "todo");
const review = status("review", "Ready for review", 200, "in_progress");
const done = status("done", "Done", 300, "done");
const archived = {
  ...status("old", "Old status", 400, "todo"),
  archived_at: "2026-06-13T00:00:00Z",
};
const workflow: ProjectWorkflow = {
  project_id: "project-1",
  statuses: [archived, done, review, todo],
  transitions: [
    {
      from_status_id: todo.id,
      to_status_id: review.id,
      created_at: "2026-06-13T00:00:00Z",
    },
  ],
};

test("renders ordered active statuses, immutable keys and archived history", () => {
  render(<WorkflowSettingsPanel {...panelProps()} />);

  const cards = screen.getAllByText("Immutable key");
  assert.equal(cards.length, 3);
  assert.ok(screen.getByText("Old status · old"));
  assert.equal(screen.getByLabelText("Allow Todo to Todo").hasAttribute("disabled"), true);

  const doneCard = screen.getByLabelText("Name for Done").closest("article")!;
  assert.equal(
    within(doneCard).getByRole("button", { name: "Archive" }).hasAttribute("disabled"),
    true,
  );
  assert.ok(within(doneCard).getByText(/last active done status/));
});

test("validates and creates a normalized status", async () => {
  const user = userEvent.setup();
  const props = panelProps();
  render(<WorkflowSettingsPanel {...props} />);

  await user.click(screen.getByRole("button", { name: "Create status" }));
  assert.ok(screen.getByText(/Key must be a lowercase identifier/));

  await user.type(screen.getByLabelText("New status key"), "QA_REVIEW");
  await user.type(screen.getByLabelText("New status name"), "QA review");
  await user.click(screen.getByRole("button", { name: "Create status" }));

  assert.deepEqual(props.onCreateStatus.mock.calls[0]?.[0], {
    key: "qa_review",
    name: "QA review",
    color: "#0ea5e9",
    category: "todo",
  });
});

test("edits, reorders and archives statuses with replacement", async () => {
  const user = userEvent.setup();
  const props = panelProps();
  render(<WorkflowSettingsPanel {...props} />);

  await user.clear(screen.getByLabelText("Name for Ready for review"));
  await user.type(screen.getByLabelText("Name for Ready for review"), "Review");
  const reviewCard = screen.getByText("review").closest("article")!;
  await user.click(within(reviewCard).getByRole("button", { name: "Save" }));
  assert.equal(props.onUpdateStatus.mock.calls[0]?.[0].id, review.id);
  assert.equal(props.onUpdateStatus.mock.calls[0]?.[1].name, "Review");

  await user.click(screen.getByRole("button", { name: "Move Ready for review up" }));
  assert.deepEqual(props.onReorderStatuses.mock.calls[0]?.[0], [
    review.id,
    todo.id,
    done.id,
  ]);

  await user.click(within(reviewCard).getByRole("button", { name: "Archive" }));
  await user.selectOptions(screen.getByLabelText("Replacement status"), todo.id);
  await user.click(screen.getByRole("button", { name: "Confirm archive" }));
  assert.equal(props.onArchiveStatus.mock.calls[0]?.[0].id, review.id);
  assert.equal(props.onArchiveStatus.mock.calls[0]?.[1], todo.id);
});

test("preserves a valid archive draft while the workflow refreshes", async () => {
  const user = userEvent.setup();
  const props = panelProps();
  const { rerender } = render(<WorkflowSettingsPanel {...props} />);

  const reviewCard = screen.getByText("review").closest("article")!;
  await user.click(within(reviewCard).getByRole("button", { name: "Archive" }));
  await user.selectOptions(screen.getByLabelText("Replacement status"), todo.id);

  rerender(
    <WorkflowSettingsPanel
      {...props}
      workflow={{
        ...workflow,
        statuses: [...workflow.statuses],
        transitions: [...workflow.transitions],
      }}
    />,
  );

  assert.ok(screen.getByRole("dialog", { name: "Archive Ready for review" }));
  assert.equal(
    (screen.getByLabelText("Replacement status") as HTMLSelectElement).value,
    todo.id,
  );
  await user.click(screen.getByRole("button", { name: "Confirm archive" }));
  assert.equal(props.onArchiveStatus.mock.calls[0]?.[0].id, review.id);
  assert.equal(props.onArchiveStatus.mock.calls[0]?.[1], todo.id);
});

test("keeps the archive dialog and replacement after a failed mutation", async () => {
  const user = userEvent.setup();
  const props = panelProps();
  props.onArchiveStatus.mockResolvedValue(false);
  render(<WorkflowSettingsPanel {...props} error="Could not archive workflow status." />);

  const reviewCard = screen.getByText("review").closest("article")!;
  await user.click(within(reviewCard).getByRole("button", { name: "Archive" }));
  await user.selectOptions(screen.getByLabelText("Replacement status"), todo.id);
  await user.click(screen.getByRole("button", { name: "Confirm archive" }));

  assert.ok(screen.getByText("Could not archive workflow status."));
  assert.ok(screen.getByRole("dialog", { name: "Archive Ready for review" }));
  assert.equal(
    (screen.getByLabelText("Replacement status") as HTMLSelectElement).value,
    todo.id,
  );
});

test("saves and resets transition matrix drafts", async () => {
  const user = userEvent.setup();
  const props = panelProps();
  render(<WorkflowSettingsPanel {...props} />);

  const todoToReview = screen.getByLabelText("Allow Todo to Ready for review");
  const reviewToDone = screen.getByLabelText("Allow Ready for review to Done");
  assert.equal((todoToReview as HTMLInputElement).checked, true);
  await user.click(todoToReview);
  await user.click(reviewToDone);
  await user.click(screen.getByRole("button", { name: "Save transitions" }));
  assert.deepEqual(props.onReplaceTransitions.mock.calls[0]?.[0], [
    { from_status_id: review.id, to_status_id: done.id },
  ]);

  await user.click(screen.getByRole("button", { name: "Reset" }));
  assert.equal((todoToReview as HTMLInputElement).checked, true);
  assert.equal((reviewToDone as HTMLInputElement).checked, false);
});

test("requires confirmation before saving an empty transition graph", async () => {
  const user = userEvent.setup();
  const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
  const props = panelProps({ ...workflow, transitions: [] });
  render(<WorkflowSettingsPanel {...props} />);

  await user.click(screen.getByRole("button", { name: "Save transitions" }));
  assert.equal(confirm.mock.calls.length, 1);
  assert.equal(props.onReplaceTransitions.mock.calls.length, 0);
  confirm.mockRestore();
});

function panelProps(nextWorkflow = workflow) {
  return {
    archivingStatusIds: [],
    creatingStatus: false,
    error: "",
    isLoading: false,
    isReordering: false,
    isSavingTransitions: false,
    onArchiveStatus: vi.fn(async () => true),
    onCreateStatus: vi.fn(async () => true),
    onReorderStatuses: vi.fn(async () => true),
    onReplaceTransitions: vi.fn(async () => true),
    onUpdateStatus: vi.fn(async () => true),
    updatingStatusIds: [],
    workflow: nextWorkflow,
  };
}

function status(
  key: string,
  name: string,
  position: number,
  category: ProjectWorkflowStatus["category"],
): ProjectWorkflowStatus {
  return {
    id: `status-${key}`,
    project_id: "project-1",
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
