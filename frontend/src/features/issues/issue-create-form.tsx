import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import {
  type IssuePriority,
  type IssueStatus,
  type IssueType,
  type Label,
  type Project,
  type TeamMember,
} from "../../lib/api-types";
import { columns, issueTypeLabels, priorityLabels } from "../../lib/issue-model";
import { activeTeamMembers, memberOptionLabel } from "../../lib/team-view";

type IssueCreateFormProps = {
  assigneeId: string;
  canCreateIssue: boolean;
  description: string;
  dueDate: string;
  formError: string;
  isCreatingIssue: boolean;
  labels: Label[];
  labelIds: string[];
  onAssigneeChange: (value: string) => void;
  onCreateIssue: (event: FormEvent<HTMLFormElement>) => void;
  onDescriptionChange: (value: string) => void;
  onDueDateChange: (value: string) => void;
  onLabelChange: (labelId: string, shouldAttach: boolean) => void;
  onPriorityChange: (value: IssuePriority) => void;
  onProjectChange: (value: string) => void;
  onStatusChange: (value: IssueStatus) => void;
  onTitleChange: (value: string) => void;
  onTypeChange: (value: IssueType) => void;
  priority: IssuePriority;
  projectId: string;
  projects: Project[];
  status: IssueStatus;
  teamMembers: TeamMember[];
  title: string;
  type: IssueType;
};

export function IssueCreateForm({
  assigneeId,
  canCreateIssue,
  description,
  dueDate,
  formError,
  isCreatingIssue,
  labels,
  labelIds,
  onAssigneeChange,
  onCreateIssue,
  onDescriptionChange,
  onDueDateChange,
  onLabelChange,
  onPriorityChange,
  onProjectChange,
  onStatusChange,
  onTitleChange,
  onTypeChange,
  priority,
  projectId,
  projects,
  status,
  teamMembers,
  title,
  type,
}: IssueCreateFormProps) {
  return (
    <form className="issue-form" onSubmit={onCreateIssue}>
      <header className="section-header">
        <div>
          <p className="eyebrow">Issues</p>
          <h2>Create issue</h2>
        </div>
      </header>

      <label>
        <span>Project</span>
        <select
          onChange={(event) => onProjectChange(event.target.value)}
          value={projectId}
        >
          <option value="">Select project</option>
          {projects.map((project) => (
            <option key={project.id} value={project.id}>
              {project.key} · {project.name}
            </option>
          ))}
        </select>
      </label>

      <label>
        <span>Title</span>
        <input
          maxLength={180}
          onChange={(event) => onTitleChange(event.target.value)}
          placeholder="Create project board"
          value={title}
        />
      </label>

      <label>
        <span>Description</span>
        <textarea
          onChange={(event) => onDescriptionChange(event.target.value)}
          placeholder="Short context for the team"
          rows={3}
          value={description}
        />
      </label>

      <label>
        <span>Assignee</span>
        <select
          onChange={(event) => onAssigneeChange(event.target.value)}
          value={assigneeId}
        >
          <option value="">Unassigned</option>
          {activeTeamMembers(teamMembers).map((member) => (
            <option key={member.id} value={member.id}>
              {memberOptionLabel(member)}
            </option>
          ))}
        </select>
      </label>

      <div className="issue-label-picker">
        <span>Labels</span>
        {labels.length > 0 ? (
          <div className="label-checkbox-list">
            {labels.map((label) => (
              <label className="label-checkbox" key={label.id}>
                <input
                  checked={labelIds.includes(label.id)}
                  onChange={(event) =>
                    onLabelChange(label.id, event.target.checked)
                  }
                  type="checkbox"
                />
                <span
                  className="label-chip label-chip-small"
                  style={{
                    backgroundColor: `${label.color}1a`,
                    borderColor: label.color,
                  }}
                >
                  {label.name}
                </span>
              </label>
            ))}
          </div>
        ) : (
          <strong>No labels created</strong>
        )}
      </div>

      <div className="field-grid">
        <label>
          <span>Type</span>
          <select
            onChange={(event) => onTypeChange(event.target.value as IssueType)}
            value={type}
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
            value={priority}
          >
            {Object.entries(priorityLabels).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div className="field-grid">
        <label>
          <span>Status</span>
          <select
            onChange={(event) => onStatusChange(event.target.value as IssueStatus)}
            value={status}
          >
            {columns.map((column) => (
              <option key={column.status} value={column.status}>
                {column.title}
              </option>
            ))}
          </select>
        </label>

        <label>
          <span>Due date</span>
          <input
            onChange={(event) => onDueDateChange(event.target.value)}
            type="date"
            value={dueDate}
          />
        </label>
      </div>

      <FormError message={formError} />

      <button disabled={!canCreateIssue} type="submit">
        {isCreatingIssue ? "Creating..." : "Create issue"}
      </button>
    </form>
  );
}
