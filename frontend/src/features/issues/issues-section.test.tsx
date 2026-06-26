import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { IssueCreateForm } from "./issue-create-form";
import { IssueListPanel } from "./issue-list-panel";
import { SavedFiltersPanel } from "./saved-filters-panel";

describe("issues section contracts", () => {
  it("IssueCreateForm keeps the .issue-form scope, labels and Create issue action", () => {
    const props = {
      assigneeId: "",
      canCreateIssue: false,
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
      projects: [],
      statusId: "",
      statuses: [],
      storyPoints: "",
      teamMembers: [],
      title: "",
      type: "task",
    };
    const { container } = render(<IssueCreateForm {...(props as never)} />);
    const form = container.querySelector(".issue-form");
    expect(form).not.toBeNull();
    const scope = within(form as HTMLElement);

    for (const label of ["Project", "Title", "Description", "Priority", "Status"]) {
      expect(scope.getByLabelText(label)).toBeTruthy();
    }
    expect(scope.getByRole("button", { name: "Create issue" })).toBeTruthy();
  });

  it("IssueListPanel keeps the Issue filters region, labels and Clear", () => {
    const props = {
      archivingIssueIds: [],
      assigneeFilterId: "",
      dueFilter: "",
      isLoadingIssues: false,
      issues: [],
      issuesError: "",
      labelFilterId: "",
      labels: [],
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
      projectFilterId: "",
      projects: [],
      query: "",
      sort: "created_desc",
      sprintFilterId: "",
      sprints: [],
      legacyStatusFilter: "",
      workflowStatusFilterId: "",
      workflowStatuses: [],
      teamMembers: [],
      today: new Date(),
    };
    render(<IssueListPanel {...(props as never)} />);
    const filters = screen.getByRole("region", { name: "Issue filters" });
    const scope = within(filters);
    for (const label of ["Search", "Sort", "Project", "Status"]) {
      expect(scope.getByLabelText(label)).toBeTruthy();
    }
    expect(scope.getByRole("button", { name: "Clear" })).toBeTruthy();
  });

  it("SavedFiltersPanel keeps the Saved issue filters region and view actions", () => {
    const props = {
      deletingSavedFilterIds: [],
      isCreatingSavedFilter: false,
      isLoadingSavedFilters: false,
      onApplySavedFilter: vi.fn(),
      onCancelRenameSavedFilter: vi.fn(),
      onCreateSavedFilter: vi.fn(),
      onDeleteSavedFilter: vi.fn(),
      onRenameSavedFilter: vi.fn(),
      onRenameSavedFilterNameChange: vi.fn(),
      onSavedFilterNameChange: vi.fn(),
      onStartRenameSavedFilter: vi.fn(),
      onUpdateSavedFilter: vi.fn(),
      renameSavedFilterId: "",
      renameSavedFilterName: "",
      savedFilterFormError: "",
      savedFilterName: "",
      savedFilters: [],
      savedFiltersError: "",
      updatingSavedFilterIds: [],
      workflowStatusNamesById: {},
    };
    render(<SavedFiltersPanel {...(props as never)} />);
    const region = screen.getByRole("region", { name: "Saved issue filters" });
    expect(within(region).getByLabelText("View name")).toBeTruthy();
    expect(
      within(region).getByRole("button", { name: "Save current view" }),
    ).toBeTruthy();
  });
});
