import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import { WorkflowStatusBadge } from "../../components/workflow-status-badge";
import {
  type IssuePriority,
  type IssueType,
  type Label,
  type Project,
  type ProjectWorkflowStatus,
  type TeamMember,
} from "../../lib/api-types";
import {
  issueTypeLabels,
  priorityLabels,
  rootIssueTypeOptions,
} from "../../lib/issue-model";
import { activeTeamMembers, memberOptionLabel } from "../../lib/team-view";
import { Button, Field, Input, Select, TextArea } from "../../ui";

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
  onStoryPointsChange: (value: string) => void;
  onStatusChange: (value: string) => void;
  onTitleChange: (value: string) => void;
  onTypeChange: (value: IssueType) => void;
  priority: IssuePriority;
  projectId: string;
  projects: Project[];
  statusId: string;
  statuses: ProjectWorkflowStatus[];
  storyPoints: string;
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
  onStoryPointsChange,
  onStatusChange,
  onTitleChange,
  onTypeChange,
  priority,
  projectId,
  projects,
  statusId,
  statuses,
  storyPoints,
  teamMembers,
  title,
  type,
}: IssueCreateFormProps) {
  const selectedStatus = statuses.find((status) => status.id === statusId);

  return (
    <form className="issue-form kl-card" onSubmit={onCreateIssue}>
      <header className="kl-section-head">
        <div>
          <p className="kl-eyebrow">Issues</p>
          <h2>Create issue</h2>
        </div>
      </header>

      <Field label="Project" htmlFor="issue-project">
        <Select
          id="issue-project"
          onChange={(event) => onProjectChange(event.target.value)}
          value={projectId}
        >
          <option value="">Select project</option>
          {projects.map((project) => (
            <option key={project.id} value={project.id}>
              {project.key} · {project.name}
            </option>
          ))}
        </Select>
      </Field>

      <Field label="Title" htmlFor="issue-title">
        <Input
          id="issue-title"
          maxLength={180}
          onChange={(event) => onTitleChange(event.target.value)}
          placeholder="Create project board"
          value={title}
        />
      </Field>

      <Field label="Description" htmlFor="issue-description">
        <TextArea
          id="issue-description"
          onChange={(event) => onDescriptionChange(event.target.value)}
          placeholder="Short context for the team"
          rows={3}
          value={description}
        />
      </Field>

      <Field label="Assignee" htmlFor="issue-assignee">
        <Select
          id="issue-assignee"
          onChange={(event) => onAssigneeChange(event.target.value)}
          value={assigneeId}
        >
          <option value="">Unassigned</option>
          {activeTeamMembers(teamMembers).map((member) => (
            <option key={member.id} value={member.id}>
              {memberOptionLabel(member)}
            </option>
          ))}
        </Select>
      </Field>

      <div className="kl-field">
        <span className="kl-field__label">Labels</span>
        {labels.length > 0 ? (
          <div className="kl-label-picker">
            {labels.map((label) => (
              <label className="kl-label-check" key={label.id}>
                <input
                  checked={labelIds.includes(label.id)}
                  onChange={(event) => onLabelChange(label.id, event.target.checked)}
                  type="checkbox"
                />
                <span
                  className="kl-label-chip"
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
          <strong className="kl-muted">No labels created</strong>
        )}
      </div>

      <div className="kl-form-grid">
        <Field label="Type" htmlFor="issue-type">
          <Select
            id="issue-type"
            onChange={(event) => onTypeChange(event.target.value as IssueType)}
            value={type}
          >
            {rootIssueTypeOptions.map((value) => (
              <option key={value} value={value}>
                {issueTypeLabels[value]}
              </option>
            ))}
          </Select>
        </Field>

        <Field label="Priority" htmlFor="issue-priority">
          <Select
            id="issue-priority"
            onChange={(event) => onPriorityChange(event.target.value as IssuePriority)}
            value={priority}
          >
            {Object.entries(priorityLabels).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </Select>
        </Field>
      </div>

      <div className="kl-form-grid">
        <Field label="Status" htmlFor="issue-status">
          <Select
            id="issue-status"
            aria-label="Status"
            disabled={!projectId || statuses.length === 0}
            onChange={(event) => onStatusChange(event.target.value)}
            value={statusId}
          >
            {!projectId ? <option value="">Select project first</option> : null}
            {projectId && statuses.length === 0 ? (
              <option value="">Workflow unavailable</option>
            ) : null}
            {statuses.map((status) => (
              <option key={status.id} value={status.id}>
                {status.name} · {status.category.replaceAll("_", " ")}
              </option>
            ))}
          </Select>
          {selectedStatus ? <WorkflowStatusBadge status={selectedStatus} /> : null}
        </Field>

        <Field label="Due date" htmlFor="issue-due">
          <Input
            id="issue-due"
            onChange={(event) => onDueDateChange(event.target.value)}
            type="date"
            value={dueDate}
          />
        </Field>
      </div>

      <Field label="Story points" htmlFor="issue-points">
        <Input
          id="issue-points"
          min="0"
          max="100"
          onChange={(event) => onStoryPointsChange(event.target.value)}
          type="number"
          value={storyPoints}
        />
      </Field>

      <FormError message={formError} />

      <Button variant="primary" disabled={!canCreateIssue} type="submit">
        {isCreatingIssue ? "Creating..." : "Create issue"}
      </Button>
    </form>
  );
}
