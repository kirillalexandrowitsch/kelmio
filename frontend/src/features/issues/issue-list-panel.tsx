import { FormError } from "../../components/form-feedback";
import {
  type Issue,
  type IssueDueFilter,
  type IssuePriority,
  type IssueSort,
  type IssueStatus,
  type Label,
  type Project,
  type Sprint,
  type TeamMember,
} from "../../lib/api-types";
import {
  columns,
  issueDueFilterLabels,
  issueDueInfo,
  issueSortLabels,
  issueTypeLabels,
  priorityLabels,
  storyPointsLabel,
} from "../../lib/issue-model";
import {
  sprintDisplayName,
  sprintOptionLabel,
  sprintStatusLabels,
  sprintStatusOptions,
} from "../../lib/sprint-model";
import { memberDisplayName, memberOptionLabel } from "../../lib/team-view";
import { hasText } from "../../lib/validation";

type IssueListPanelProps = {
  archivingIssueIds: string[];
  assigneeFilterId: string;
  dueFilter: IssueDueFilter | "";
  isLoadingIssues: boolean;
  issues: Issue[];
  issuesError: string;
  labelFilterId: string;
  labels: Label[];
  onArchiveIssue: (issue: Issue) => void;
  onAssigneeFilterChange: (value: string) => void;
  onClearFilters: () => void;
  onDueFilterChange: (value: IssueDueFilter | "") => void;
  onLabelFilterChange: (value: string) => void;
  onOpenIssue: (issueId: string) => void;
  onPriorityFilterChange: (value: IssuePriority | "") => void;
  onProjectFilterChange: (value: string) => void;
  onQueryChange: (value: string) => void;
  onSortChange: (value: IssueSort) => void;
  onSprintFilterChange: (value: string) => void;
  onStatusFilterChange: (value: IssueStatus | "") => void;
  priorityFilter: IssuePriority | "";
  projectFilterId: string;
  projects: Project[];
  query: string;
  sort: IssueSort;
  sprintFilterId: string;
  sprints: Sprint[];
  statusFilter: IssueStatus | "";
  teamMembers: TeamMember[];
  today: Date;
};

export function IssueListPanel({
  archivingIssueIds,
  assigneeFilterId,
  dueFilter,
  isLoadingIssues,
  issues,
  issuesError,
  labelFilterId,
  labels,
  onArchiveIssue,
  onAssigneeFilterChange,
  onClearFilters,
  onDueFilterChange,
  onLabelFilterChange,
  onOpenIssue,
  onPriorityFilterChange,
  onProjectFilterChange,
  onQueryChange,
  onSortChange,
  onSprintFilterChange,
  onStatusFilterChange,
  priorityFilter,
  projectFilterId,
  projects,
  query,
  sort,
  sprintFilterId,
  sprints,
  statusFilter,
  teamMembers,
  today,
}: IssueListPanelProps) {
  const hasFilters =
    projectFilterId !== "" ||
    sprintFilterId !== "" ||
    statusFilter !== "" ||
    priorityFilter !== "" ||
    assigneeFilterId !== "" ||
    labelFilterId !== "" ||
    dueFilter !== "" ||
    hasText(query);
  const summary = hasFilters
    ? `${issues.length} issues match current filters`
    : sort === "created_desc"
      ? "Showing all issues across all projects"
      : `Showing issues sorted by ${issueSortLabels[sort].toLowerCase()}`;

  return (
    <div className="issues-panel">
      <header className="section-header">
        <div>
          <p className="eyebrow">Open work</p>
          <h2>Recent issues</h2>
        </div>
        {isLoadingIssues ? <span className="muted">Loading</span> : null}
      </header>

      <section className="issue-filters" aria-label="Issue filters">
        <label>
          <span>Search</span>
          <input
            onChange={(event) => onQueryChange(event.target.value)}
            placeholder="Key, title, description"
            value={query}
          />
        </label>

        <label>
          <span>Sort</span>
          <select
            onChange={(event) => onSortChange(event.target.value as IssueSort)}
            value={sort}
          >
            {Object.entries(issueSortLabels).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </select>
        </label>

        <label>
          <span>Project</span>
          <select
            onChange={(event) => onProjectFilterChange(event.target.value)}
            value={projectFilterId}
          >
            <option value="">All projects</option>
            {projects.map((project) => (
              <option key={project.id} value={project.id}>
                {project.key}
              </option>
            ))}
          </select>
        </label>

        <label>
          <span>Sprint</span>
          <select
            onChange={(event) => onSprintFilterChange(event.target.value)}
            value={sprintFilterId}
          >
            <option value="">All sprints</option>
            <option value="none">No sprint</option>
            {sprintStatusOptions.map((status) => {
              const statusSprints = sprints.filter(
                (sprint) => sprint.status === status,
              );
              if (statusSprints.length === 0) {
                return null;
              }

              return (
                <optgroup key={status} label={sprintStatusLabels[status]}>
                  {statusSprints.map((sprint) => (
                    <option key={sprint.id} value={sprint.id}>
                      {sprintOptionLabel(sprint)}
                    </option>
                  ))}
                </optgroup>
              );
            })}
          </select>
        </label>

        <label>
          <span>Status</span>
          <select
            onChange={(event) =>
              onStatusFilterChange(event.target.value as IssueStatus | "")
            }
            value={statusFilter}
          >
            <option value="">All statuses</option>
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
              onPriorityFilterChange(event.target.value as IssuePriority | "")
            }
            value={priorityFilter}
          >
            <option value="">All priorities</option>
            {Object.entries(priorityLabels).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </select>
        </label>

        <label>
          <span>Assignee</span>
          <select
            onChange={(event) => onAssigneeFilterChange(event.target.value)}
            value={assigneeFilterId}
          >
            <option value="">All assignees</option>
            <option value="unassigned">Unassigned</option>
            {teamMembers.map((member) => (
              <option key={member.id} value={member.id}>
                {memberOptionLabel(member)}
              </option>
            ))}
          </select>
        </label>

        <label>
          <span>Label</span>
          <select
            onChange={(event) => onLabelFilterChange(event.target.value)}
            value={labelFilterId}
          >
            <option value="">All labels</option>
            {labels.map((label) => (
              <option key={label.id} value={label.id}>
                {label.name}
              </option>
            ))}
          </select>
        </label>

        <label>
          <span>Due</span>
          <select
            onChange={(event) =>
              onDueFilterChange(event.target.value as IssueDueFilter | "")
            }
            value={dueFilter}
          >
            <option value="">Any due date</option>
            {Object.entries(issueDueFilterLabels).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </select>
        </label>

        <button
          className="small-button"
          disabled={!hasFilters}
          onClick={onClearFilters}
          type="button"
        >
          Clear
        </button>
      </section>

      <p className="filter-summary">{summary}</p>

      <FormError message={issuesError} />

      {issues.length > 0 ? (
        <div className="issue-list">
          {issues.map((issue) => {
            const dueInfo = issueDueInfo(issue, today);
            const sprintName = issue.sprint_id
              ? sprintDisplayName(sprints, issue.sprint_id)
              : null;

            return (
              <article className="issue-row" key={issue.id}>
                <span className="issue-key">{issue.issue_key}</span>
                <div>
                  <h3>{issue.title}</h3>
                  <p>
                    {issueTypeLabels[issue.issue_type]} ·{" "}
                    {priorityLabels[issue.priority]} ·{" "}
                    {columns.find((column) => column.status === issue.status)
                      ?.title ?? issue.status}{" "}
                    · {storyPointsLabel(issue.story_points)} ·{" "}
                    {memberDisplayName(teamMembers, issue.assignee_id)}
                    {sprintName ? ` · Sprint: ${sprintName}` : ""}
                  </p>
                  {dueInfo ? (
                    <span className={`due-badge due-badge-${dueInfo.tone}`}>
                      {dueInfo.label}
                    </span>
                  ) : null}
                  {issue.labels.length > 0 ? (
                    <div className="issue-label-row">
                      {issue.labels.map((label) => (
                        <span
                          className="label-chip label-chip-small"
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
                  ) : null}
                </div>
                <div className="issue-row-actions">
                  <button
                    className="small-button"
                    onClick={() => onOpenIssue(issue.id)}
                    type="button"
                  >
                    Open
                  </button>
                  <button
                    className="small-button danger-button"
                    disabled={archivingIssueIds.includes(issue.id)}
                    onClick={() => onArchiveIssue(issue)}
                    type="button"
                  >
                    {archivingIssueIds.includes(issue.id)
                      ? "Archiving"
                      : "Archive"}
                  </button>
                </div>
              </article>
            );
          })}
        </div>
      ) : (
        <div className="project-empty">No issues yet</div>
      )}
    </div>
  );
}
