import type { WorkflowStatus } from "../lib/api-types";
import { workflowStatusStyle } from "../lib/workflow-model";

type WorkflowStatusBadgeProps = {
  fallbackLabel?: string;
  status?: WorkflowStatus;
};

export function WorkflowStatusBadge({
  fallbackLabel = "Unknown status",
  status,
}: WorkflowStatusBadgeProps) {
  if (!status) {
    return <span className="detail-chip">{fallbackLabel}</span>;
  }

  return (
    <span className="workflow-status-badge" style={workflowStatusStyle(status)}>
      <span
        aria-hidden="true"
        className="workflow-status-dot"
        style={{ backgroundColor: status.color }}
      />
      {status.name}
      <small>{status.category.replaceAll("_", " ")}</small>
    </span>
  );
}
