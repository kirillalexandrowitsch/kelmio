import {
  type Issue,
  type IssueDueFilter,
  type IssuePriority,
  type IssueSort,
  type IssueStatus,
  type IssueType,
  type SavedIssueFilters,
} from "./api-types.ts";

export const columns = [
  { status: "backlog", title: "Backlog" },
  { status: "todo", title: "Todo" },
  { status: "in_progress", title: "In progress" },
  { status: "blocked", title: "Blocked" },
  { status: "done", title: "Done" },
] satisfies Array<{ status: IssueStatus; title: string }>;

export const priorityLabels: Record<IssuePriority, string> = {
  low: "Low",
  medium: "Medium",
  high: "High",
  critical: "Critical",
};

export const issueTypeLabels: Record<IssueType, string> = {
  task: "Task",
  bug: "Bug",
  story: "Story",
  epic: "Epic",
  subtask: "Subtask",
};

export const rootIssueTypeOptions = [
  "task",
  "bug",
  "story",
  "epic",
] satisfies IssueType[];

export function editableIssueTypeOptions(issue: Issue) {
  if (issue.parent_issue_id) {
    return ["task", "bug", "story", "subtask"] satisfies IssueType[];
  }

  return rootIssueTypeOptions;
}

export const issueSortLabels: Record<IssueSort, string> = {
  created_desc: "Newest first",
  created_asc: "Oldest first",
  priority_desc: "Priority high to low",
  due_date_asc: "Due date soonest",
};

export const issueDueFilterLabels: Record<IssueDueFilter, string> = {
  overdue: "Overdue",
  today: "Due today",
  due_soon: "Due soon",
  no_due: "No due date",
};

export type IssueFilterState = {
  query: string;
  sort: IssueSort;
  projectId: string;
  sprintId: string;
  status: string;
  workflowStatusId: string;
  priority: IssuePriority | "";
  assigneeId: string;
  labelId: string;
  due: IssueDueFilter | "";
};

export type DueTone = "overdue" | "due-soon" | "scheduled" | "done";

export function issueMatchesFilters(
  issue: Issue,
  projectId: string,
  sprintId: string,
  status: string,
  workflowStatusId: string,
  priority: IssuePriority | "",
  assigneeId: string,
  labelId: string,
  dueFilter: IssueDueFilter | "",
  query: string,
  today: Date,
) {
  if (projectId && issue.project_id !== projectId) {
    return false;
  }
  if (sprintId === "none" && issue.sprint_id !== null) {
    return false;
  }
  if (sprintId && sprintId !== "none" && issue.sprint_id !== sprintId) {
    return false;
  }
  if (status && issue.status !== status) {
    return false;
  }
  if (workflowStatusId && issue.workflow_status.id !== workflowStatusId) {
    return false;
  }
  if (priority && issue.priority !== priority) {
    return false;
  }
  if (assigneeId === "unassigned" && issue.assignee_id !== null) {
    return false;
  }
  if (assigneeId && assigneeId !== "unassigned" && issue.assignee_id !== assigneeId) {
    return false;
  }
  if (labelId && !issue.labels.some((label) => label.id === labelId)) {
    return false;
  }
  if (!issueMatchesDueFilter(issue, dueFilter, today)) {
    return false;
  }

  const normalizedQuery = query.trim().toLowerCase();
  if (
    normalizedQuery &&
    !issue.issue_key.toLowerCase().includes(normalizedQuery) &&
    !issue.title.toLowerCase().includes(normalizedQuery) &&
    !issue.description.toLowerCase().includes(normalizedQuery)
  ) {
    return false;
  }

  return true;
}

export function statusLabel(status: string) {
  return columns.find((column) => column.status === status)?.title ?? status;
}

export function issueLabelIds(issue: Issue) {
  return issue.labels.map((label) => label.id);
}

export function storyPointsLabel(points: number) {
  return points === 1 ? "1 point" : `${points} points`;
}

export function savedIssueFiltersFromState(
  state: IssueFilterState,
): SavedIssueFilters {
  const filters: SavedIssueFilters = {
    sort: state.sort,
  };

  const query = state.query.trim();
  if (query) {
    filters.query = query;
  }
  if (state.projectId) {
    filters.projectId = state.projectId;
  }
  if (state.sprintId) {
    filters.sprintId = state.sprintId;
  }
  if (state.status) {
    filters.status = state.status;
  }
  if (state.workflowStatusId) {
    filters.workflowStatusId = state.workflowStatusId;
  }
  if (state.priority) {
    filters.priority = state.priority;
  }
  if (state.assigneeId) {
    filters.assigneeId = state.assigneeId;
  }
  if (state.labelId) {
    filters.labelId = state.labelId;
  }
  if (state.due) {
    filters.due = state.due;
  }

  return filters;
}

export function savedIssueFilterStateFromFilters(
  filters: SavedIssueFilters,
): IssueFilterState {
  return {
    query: filters.query ?? "",
    sort: filters.sort ?? "created_desc",
    projectId: filters.projectId ?? "",
    sprintId: filters.sprintId ?? "",
    status: filters.status ?? "",
    workflowStatusId: filters.workflowStatusId ?? "",
    priority: filters.priority ?? "",
    assigneeId: filters.assigneeId ?? "",
    labelId: filters.labelId ?? "",
    due: filters.due ?? "",
  };
}

export function missingFilterOptionLabel(
  kind: "project" | "sprint" | "assignee" | "label",
) {
  return `Missing ${kind}`;
}

export function savedIssueFilterSummary(filters: SavedIssueFilters) {
  const parts: string[] = [];
  if (filters.query) {
    parts.push(`Search: ${filters.query}`);
  }
  if (filters.projectId) {
    parts.push("Project");
  }
  if (filters.sprintId) {
    parts.push(filters.sprintId === "none" ? "No sprint" : "Sprint");
  }
  if (filters.status) {
    parts.push(`Status: ${statusLabel(filters.status)}`);
  }
  if (filters.priority) {
    parts.push(`Priority: ${priorityLabels[filters.priority]}`);
  }
  if (filters.assigneeId) {
    parts.push(filters.assigneeId === "unassigned" ? "Unassigned" : "Assignee");
  }
  if (filters.labelId) {
    parts.push("Label");
  }
  if (filters.due) {
    parts.push(`Due: ${issueDueFilterLabels[filters.due]}`);
  }
  parts.push(`Sort: ${issueSortLabels[filters.sort ?? "created_desc"]}`);

  return parts;
}

export function startOfToday() {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth(), now.getDate());
}

function parseDateOnly(value: string | null) {
  if (!value) {
    return null;
  }

  const [year, month, day] = value.split("-").map(Number);
  if (!year || !month || !day) {
    return null;
  }

  return new Date(year, month - 1, day);
}

export function issueDueInfo(issue: Issue, today: Date) {
  const dueDate = parseDateOnly(issue.due_date);
  if (!dueDate) {
    return null;
  }

  const daysUntilDue = Math.round(
    (dueDate.getTime() - today.getTime()) / (24 * 60 * 60 * 1000),
  );

  if (isIssueDone(issue)) {
    return { label: `Done, due ${issue.due_date}`, tone: "done" as DueTone };
  }
  if (daysUntilDue < 0) {
    const overdueDays = Math.abs(daysUntilDue);
    return {
      label: overdueDays === 1 ? "Overdue by 1 day" : `Overdue by ${overdueDays} days`,
      tone: "overdue" as DueTone,
    };
  }
  if (daysUntilDue === 0) {
    return { label: "Due today", tone: "due-soon" as DueTone };
  }
  if (daysUntilDue === 1) {
    return { label: "Due tomorrow", tone: "due-soon" as DueTone };
  }
  if (daysUntilDue <= 7) {
    return { label: `Due in ${daysUntilDue} days`, tone: "due-soon" as DueTone };
  }

  return { label: `Due ${issue.due_date}`, tone: "scheduled" as DueTone };
}

export function issueMatchesDueFilter(
  issue: Issue,
  dueFilter: IssueDueFilter | "",
  today: Date,
) {
  if (dueFilter === "") {
    return true;
  }
  if (dueFilter === "no_due") {
    return issue.due_date === null;
  }

  if (isIssueDone(issue)) {
    return false;
  }

  const dueDate = parseDateOnly(issue.due_date);
  if (!dueDate) {
    return false;
  }

  const daysUntilDue = Math.round(
    (dueDate.getTime() - today.getTime()) / (24 * 60 * 60 * 1000),
  );

  if (dueFilter === "overdue") {
    return daysUntilDue < 0;
  }
  if (dueFilter === "today") {
    return daysUntilDue === 0;
  }

  return daysUntilDue > 0 && daysUntilDue <= 7;
}

export function isIssueDone(issue: Issue) {
  return issue.workflow_status?.category === "done" || issue.status === "done";
}
