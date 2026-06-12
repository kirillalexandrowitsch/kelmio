import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import {
  type Issue,
  type IssuePriority,
  type ProjectWorkflowStatus,
} from "../../lib/api-types";
import { issueTypeLabels, priorityLabels } from "../../lib/issue-model";
import { workflowStatusLabel } from "../../lib/workflow-model";

type IssueHierarchySectionProps = {
  canCreateSubtask: boolean;
  children: Issue[];
  formError: string;
  hierarchyError: string;
  isCreatingSubtask: boolean;
  isLoadingChildren: boolean;
  issue: Issue;
  onCreateSubtask: (event: FormEvent<HTMLFormElement>) => void;
  onOpenIssue: (issueId: string) => void;
  onPriorityChange: (value: IssuePriority) => void;
  onStoryPointsChange: (value: string) => void;
  onStatusChange: (value: string) => void;
  onTitleChange: (value: string) => void;
  parentIssue: Issue | null;
  subtaskPriority: IssuePriority;
  subtaskStoryPoints: string;
  subtaskStatusId: string;
  statuses: ProjectWorkflowStatus[];
  subtaskTitle: string;
};

export function IssueHierarchySection({
  canCreateSubtask,
  children,
  formError,
  hierarchyError,
  isCreatingSubtask,
  isLoadingChildren,
  issue,
  onCreateSubtask,
  onOpenIssue,
  onPriorityChange,
  onStoryPointsChange,
  onStatusChange,
  onTitleChange,
  parentIssue,
  subtaskPriority,
  subtaskStoryPoints,
  subtaskStatusId,
  subtaskTitle,
  statuses,
}: IssueHierarchySectionProps) {
  return (
    <section className="hierarchy-section">
      <div className="comments-header">
        <div>
          <p className="eyebrow">Hierarchy</p>
          <h3>Parent and subtasks</h3>
        </div>
        {isLoadingChildren ? <span className="muted">Loading</span> : null}
      </div>

      {hierarchyError ? <FormError message={hierarchyError} /> : null}

      <div className="hierarchy-parent-card">
        <span>Parent</span>
        {issue.parent_issue_id ? (
          <button
            className="hierarchy-issue-button"
            onClick={() => onOpenIssue(issue.parent_issue_id ?? "")}
            type="button"
          >
            <strong>
              {parentIssue
                ? `${parentIssue.issue_key} · ${parentIssue.title}`
                : issue.parent_issue_id}
            </strong>
            {parentIssue ? (
              <small>{issueTypeLabels[parentIssue.issue_type]}</small>
            ) : (
              <small>Open parent issue</small>
            )}
          </button>
        ) : (
          <strong>No parent issue</strong>
        )}
      </div>

      <div className="hierarchy-child-list">
        <span>Children</span>
        {children.length > 0 ? (
          <div className="hierarchy-child-grid">
            {children.map((child) => (
              <button
                className="hierarchy-issue-button"
                key={child.id}
                onClick={() => onOpenIssue(child.id)}
                type="button"
              >
                <strong>
                  {child.issue_key} · {child.title}
                </strong>
                <small>
                  {issueTypeLabels[child.issue_type]} ·{" "}
                  {workflowStatusLabel(child)} · {child.story_points} pts
                </small>
              </button>
            ))}
          </div>
        ) : (
          <strong>No child issues yet</strong>
        )}
      </div>

      <form className="subtask-form" onSubmit={onCreateSubtask}>
        <label>
          <span>New subtask</span>
          <input
            onChange={(event) => onTitleChange(event.target.value)}
            placeholder="Break this issue into a smaller task"
            value={subtaskTitle}
          />
        </label>

        <label>
          <span>Status</span>
          <select
            aria-label="Subtask status"
            disabled={statuses.length === 0}
            onChange={(event) => onStatusChange(event.target.value)}
            value={subtaskStatusId}
          >
            {statuses.length === 0 ? (
              <option value="">Workflow unavailable</option>
            ) : null}
            {statuses.map((status) => (
              <option key={status.id} value={status.id}>
                {status.name}
              </option>
            ))}
          </select>
        </label>

        <label>
          <span>Priority</span>
          <select
            onChange={(event) =>
              onPriorityChange(event.target.value as IssuePriority)
            }
            value={subtaskPriority}
          >
            {Object.entries(priorityLabels).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </select>
        </label>

        <label>
          <span>Points</span>
          <input
            min="0"
            max="100"
            onChange={(event) => onStoryPointsChange(event.target.value)}
            type="number"
            value={subtaskStoryPoints}
          />
        </label>

        <button disabled={!canCreateSubtask} type="submit">
          {isCreatingSubtask ? "Creating" : "Create subtask"}
        </button>

        {formError ? <FormError message={formError} /> : null}
      </form>
    </section>
  );
}
