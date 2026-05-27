import { type FormEvent } from "react";

import {
  type Issue,
  type IssuePriority,
  type IssueType,
} from "../../lib/api-types";
import { issueTypeLabels, priorityLabels } from "../../lib/issue-model";
import { hasText } from "../../lib/validation";

type IssueDetailMainContentProps = {
  editDescription: string;
  editDueDate: string;
  editPriority: IssuePriority;
  editTitle: string;
  editType: IssueType;
  isEditing: boolean;
  isUpdating: boolean;
  issue: Issue;
  onCancelEdit: () => void;
  onDescriptionChange: (value: string) => void;
  onDueDateChange: (value: string) => void;
  onPriorityChange: (value: IssuePriority) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onTitleChange: (value: string) => void;
  onTypeChange: (value: IssueType) => void;
};

export function IssueDetailMainContent({
  editDescription,
  editDueDate,
  editPriority,
  editTitle,
  editType,
  isEditing,
  isUpdating,
  issue,
  onCancelEdit,
  onDescriptionChange,
  onDueDateChange,
  onPriorityChange,
  onSubmit,
  onTitleChange,
  onTypeChange,
}: IssueDetailMainContentProps) {
  if (isEditing) {
    return (
      <form className="issue-edit-form" onSubmit={onSubmit}>
        <label>
          <span>Title</span>
          <input
            maxLength={180}
            onChange={(event) => onTitleChange(event.target.value)}
            value={editTitle}
          />
        </label>

        <label>
          <span>Description</span>
          <textarea
            onChange={(event) => onDescriptionChange(event.target.value)}
            rows={4}
            value={editDescription}
          />
        </label>

        <div className="field-grid">
          <label>
            <span>Type</span>
            <select
              onChange={(event) => onTypeChange(event.target.value as IssueType)}
              value={editType}
            >
              {Object.entries(issueTypeLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
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
              value={editPriority}
            >
              {Object.entries(priorityLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </label>
        </div>

        <label>
          <span>Due date</span>
          <input
            onChange={(event) => onDueDateChange(event.target.value)}
            type="date"
            value={editDueDate}
          />
        </label>

        <div className="form-actions">
          <button disabled={isUpdating || !hasText(editTitle)} type="submit">
            {isUpdating ? "Saving..." : "Save changes"}
          </button>
          <button
            className="ghost-button"
            disabled={isUpdating}
            onClick={onCancelEdit}
            type="button"
          >
            Cancel
          </button>
        </div>
      </form>
    );
  }

  return (
    <>
      <div className="issue-detail-headline">
        <span className="issue-key">{issue.issue_key}</span>
        <span className="detail-chip">{issueTypeLabels[issue.issue_type]}</span>
        <span className="detail-chip">{priorityLabels[issue.priority]}</span>
      </div>

      <div>
        <p className="eyebrow">Description</p>
        <p className="issue-detail-description">
          {issue.description || "No description yet."}
        </p>
      </div>

      <div>
        <p className="eyebrow">Labels</p>
        {issue.labels.length > 0 ? (
          <div className="issue-label-row">
            {issue.labels.map((label) => (
              <span
                className="label-chip"
                key={label.id}
                style={{
                  backgroundColor: `${label.color}1a`,
                  borderColor: label.color,
                }}
              >
                {label.name}
              </span>
            ))}
          </div>
        ) : (
          <p className="issue-detail-description">No labels yet.</p>
        )}
      </div>
    </>
  );
}
