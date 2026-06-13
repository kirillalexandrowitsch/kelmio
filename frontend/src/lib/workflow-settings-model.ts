import type {
  CreateWorkflowStatusInput,
  ProjectWorkflow,
  ProjectWorkflowStatus,
  WorkflowStatusCategory,
  WorkflowTransitionInput,
} from "./api-types";

const workflowKeyPattern = /^[a-z][a-z0-9_]{0,31}$/;
const workflowColorPattern = /^#[0-9a-f]{6}$/;

export const workflowStatusCategories: WorkflowStatusCategory[] = [
  "backlog",
  "todo",
  "in_progress",
  "done",
];

export function normalizeWorkflowStatusInput(
  input: CreateWorkflowStatusInput,
): CreateWorkflowStatusInput {
  return {
    key: input.key.trim().toLowerCase(),
    name: input.name.trim(),
    color: input.color.trim().toLowerCase(),
    category: input.category,
  };
}

export function validateWorkflowStatusInput(input: CreateWorkflowStatusInput) {
  const normalized = normalizeWorkflowStatusInput(input);
  if (!workflowKeyPattern.test(normalized.key)) {
    return "Key must be a lowercase identifier with 1-32 letters, numbers, or underscores.";
  }
  if (!normalized.name) {
    return "Name is required.";
  }
  if (normalized.name.length > 60) {
    return "Name must be 60 characters or fewer.";
  }
  if (!workflowColorPattern.test(normalized.color)) {
    return "Color must be a valid #RRGGBB value.";
  }
  return "";
}

export function transitionKey(fromStatusId: string, toStatusId: string) {
  return `${fromStatusId}:${toStatusId}`;
}

export function transitionDraftFromWorkflow(workflow: ProjectWorkflow | undefined) {
  return new Set(
    workflow?.transitions.map((transition) =>
      transitionKey(transition.from_status_id, transition.to_status_id),
    ) ?? [],
  );
}

export function transitionsFromDraft(
  statuses: ProjectWorkflowStatus[],
  draft: Set<string>,
): WorkflowTransitionInput[] {
  const statusIDs = new Set(statuses.map((status) => status.id));
  return Array.from(draft)
    .map((key) => key.split(":"))
    .filter(
      ([fromStatusId, toStatusId]) =>
        fromStatusId !== toStatusId &&
        statusIDs.has(fromStatusId) &&
        statusIDs.has(toStatusId),
    )
    .map(([fromStatusId, toStatusId]) => ({
      from_status_id: fromStatusId,
      to_status_id: toStatusId,
    }));
}

export function moveWorkflowStatus(
  statuses: ProjectWorkflowStatus[],
  statusId: string,
  direction: -1 | 1,
) {
  const ids = statuses.map((status) => status.id);
  const index = ids.indexOf(statusId);
  const nextIndex = index + direction;
  if (index < 0 || nextIndex < 0 || nextIndex >= ids.length) {
    return ids;
  }
  [ids[index], ids[nextIndex]] = [ids[nextIndex], ids[index]];
  return ids;
}
