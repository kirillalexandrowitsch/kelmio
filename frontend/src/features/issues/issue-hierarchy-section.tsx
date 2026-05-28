import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import {
  type Issue,
  type IssuePriority,
  type IssueStatus,
} from "../../lib/api-types";
import { columns, issueTypeLabels, priorityLabels } from "../../lib/issue-model";

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
  onStatusChange: (value: IssueStatus) => void;
  onTitleChange: (value: string) => void;
  parentIssue: Issue | null;
  subtaskPriority: IssuePriority;
  subtaskStatus: IssueStatus;
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
  onStatusChange,
  onTitleChange,
  parentIssue,
  subtaskPriority,
  subtaskStatus,
  subtaskTitle,
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
                  {child.status.replaceAll("_", " ")}
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
            onChange={(event) =>
              onStatusChange(event.target.value as IssueStatus)
            }
            value={subtaskStatus}
          >
            {columns.map((column) => (
              <option key={column.status} value={column.status}>
                {column.title}
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

        <button disabled={!canCreateSubtask} type="submit">
          {isCreatingSubtask ? "Creating" : "Create subtask"}
        </button>

        {formError ? <FormError message={formError} /> : null}
      </form>
    </section>
  );
}
