import type {
  Issue,
  IssueLinkIssue,
  ProjectWorkflow,
  ProjectWorkflowStatus,
  WorkflowStatus,
} from "./api-types";
import { statusLabel } from "./issue-model";

export function activeWorkflowStatuses(
  workflow: ProjectWorkflow | undefined,
): ProjectWorkflowStatus[] {
  return [...(workflow?.statuses ?? [])]
    .filter((status) => status.archived_at === null)
    .sort((left, right) => left.position - right.position);
}

export function defaultWorkflowStatus(
  workflow: ProjectWorkflow | undefined,
): ProjectWorkflowStatus | null {
  const statuses = activeWorkflowStatuses(workflow);
  return statuses.find((status) => status.key === "todo") ?? statuses[0] ?? null;
}

export function allowedTransitionStatuses(
  workflow: ProjectWorkflow | undefined,
  currentStatusId: string,
): ProjectWorkflowStatus[] {
  const allowedStatusIds = new Set(
    workflow?.transitions
      .filter((transition) => transition.from_status_id === currentStatusId)
      .map((transition) => transition.to_status_id) ?? [],
  );

  return activeWorkflowStatuses(workflow).filter(
    (status) => status.id === currentStatusId || allowedStatusIds.has(status.id),
  );
}

export function workflowStatusForIssue(
  issue: Pick<Issue, "status" | "workflow_status">,
  workflow: ProjectWorkflow | undefined,
) {
  const statuses = activeWorkflowStatuses(workflow);
  return (
    statuses.find((status) => status.id === issue.workflow_status?.id) ??
    statuses.find((status) => status.key === issue.status) ??
    null
  );
}

export function canTransitionToWorkflowStatus(
  issue: Pick<Issue, "status" | "workflow_status">,
  workflow: ProjectWorkflow | undefined,
  targetStatusId: string,
) {
  const currentStatus = workflowStatusForIssue(issue, workflow);
  if (!currentStatus) {
    return false;
  }
  if (currentStatus.id === targetStatusId) {
    return true;
  }

  return (
    workflow?.transitions.some(
      (transition) =>
        transition.from_status_id === currentStatus.id &&
        transition.to_status_id === targetStatusId,
    ) ?? false
  );
}

export function workflowStatusLabel(
  value: Pick<Issue, "status" | "workflow_status"> | IssueLinkIssue,
) {
  return value.workflow_status?.name || statusLabel(value.status);
}

export function workflowStatusStyle(
  status: WorkflowStatus | ProjectWorkflowStatus,
) {
  return {
    backgroundColor: `${status.color}1a`,
    borderColor: status.color,
  };
}
