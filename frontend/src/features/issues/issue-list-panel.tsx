import { FormError } from "../../components/form-feedback";
import {
  type Issue,
  type IssueDueFilter,
  type IssuePriority,
  type IssueSort,
  type Label,
  type Project,
  type ProjectWorkflowStatus,
  type Sprint,
  type TeamMember,
} from "../../lib/api-types";
import {
  issueDueFilterLabels,
  issueDueInfo,
  issueSortLabels,
  issueTypeLabels,
  missingFilterOptionLabel,
  priorityLabels,
  storyPointsLabel,
} from "../../lib/issue-model";
import { WorkflowStatusBadge } from "../../components/workflow-status-badge";
import {
  sprintDisplayName,
  sprintOptionLabel,
  sprintStatusLabels,
  sprintStatusOptions,
} from "../../lib/sprint-model";
import { memberDisplayName, memberOptionLabel } from "../../lib/team-view";
import { hasText } from "../../lib/validation";
import { Button, Field, Input, Select } from "../../ui";

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
  onWorkflowStatusFilterChange: (value: string) => void;
  priorityFilter: IssuePriority | "";
  projectFilterId: string;
  projects: Project[];
  query: string;
  sort: IssueSort;
  sprintFilterId: string;
  sprints: Sprint[];
  legacyStatusFilter: string;
  workflowStatusFilterId: string;
  workflowStatuses: ProjectWorkflowStatus[];
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
  onWorkflowStatusFilterChange,
  priorityFilter,
  projectFilterId,
  projects,
  query,
  sort,
  sprintFilterId,
  sprints,
  legacyStatusFilter,
  teamMembers,
  today,
  workflowStatusFilterId,
  workflowStatuses,
}: IssueListPanelProps) {
  const hasFilters =
    projectFilterId !== "" ||
    sprintFilterId !== "" ||
    legacyStatusFilter !== "" ||
    workflowStatusFilterId !== "" ||
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
  const hasMissingProjectFilter =
    projectFilterId !== "" &&
    !projects.some((project) => project.id === projectFilterId);
  const hasMissingSprintFilter =
    sprintFilterId !== "" &&
    sprintFilterId !== "none" &&
    !sprints.some((sprint) => sprint.id === sprintFilterId);
  const hasMissingAssigneeFilter =
    assigneeFilterId !== "" &&
    assigneeFilterId !== "unassigned" &&
    !teamMembers.some((member) => member.id === assigneeFilterId);
  const hasMissingLabelFilter =
    labelFilterId !== "" && !labels.some((label) => label.id === labelFilterId);
  const hasMissingWorkflowStatusFilter =
    workflowStatusFilterId !== "" &&
    !workflowStatuses.some((status) => status.id === workflowStatusFilterId);
  const statusFilterValue =
    workflowStatusFilterId || (legacyStatusFilter ? `legacy:${legacyStatusFilter}` : "");

  return (
    <div className="kl-issues-list">
      <header className="kl-section-head">
        <div>
          <p className="kl-eyebrow">Open work</p>
          <h2>Recent issues</h2>
        </div>
        {isLoadingIssues ? <span className="kl-muted">Loading</span> : null}
      </header>

      <section className="issue-filters kl-issue-filters" aria-label="Issue filters">
        <Field label="Search" htmlFor="filter-search">
          <Input
            id="filter-search"
            onChange={(event) => onQueryChange(event.target.value)}
            placeholder="Key, title, description"
            value={query}
          />
        </Field>

        <Field label="Sort" htmlFor="filter-sort">
          <Select
            id="filter-sort"
            onChange={(event) => onSortChange(event.target.value as IssueSort)}
            value={sort}
          >
            {Object.entries(issueSortLabels).map(([value, label]) => (
              <option key={value} value={value}>
                {label}
              </option>
            ))}
          </Select>
        </Field>

        <Field label="Project" htmlFor="filter-project">
          <Select
            id="filter-project"
            onChange={(event) => onProjectFilterChange(event.target.value)}
            value={projectFilterId}
          >
            <option value="">All projects</option>
            {hasMissingProjectFilter ? (
              <option disabled value={projectFilterId}>
                {missingFilterOptionLabel("project")}
              </option>
            ) : null}
            {projects.map((project) => (
              <option key={project.id} value={project.id}>
                {project.key}
              </option>
            ))}
          </Select>
        </Field>

        <Field label="Sprint" htmlFor="filter-sprint">
          <Select
            id="filter-sprint"
            onChange={(event) => onSprintFilterChange(event.target.value)}
            value={sprintFilterId}
          >
            <option value="">All sprints</option>
            <option value="none">No sprint</option>
            {hasMissingSprintFilter ? (
              <option disabled value={sprintFilterId}>
                {missingFilterOptionLabel("sprint")}
              </option>
            ) : null}
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
          </Select>
        </Field>

        <Field label="Status" htmlFor="filter-status">
          <Select
            id="filter-status"
            aria-label="Status"
            disabled={!projectFilterId && !statusFilterValue}
            onChange={(event) => onWorkflowStatusFilterChange(event.target.value)}
            value={statusFilterValue}
          >
            <option value="">
              {projectFilterId ? "All statuses" : "Select a project first"}
            </option>
            {hasMissingWorkflowStatusFilter ? (
              <option disabled value={workflowStatusFilterId}>
                {missingFilterOptionLabel("status")}
              </option>
            ) : null}
            {legacyStatusFilter ? (
              <option disabled value={`legacy:${legacyStatusFilter}`}>
                Legacy status: {legacyStatusFilter.replaceAll("_", " ")}
              </option>
            ) : null}
            {workflowStatuses.map((status) => (
              <option key={status.id} value={status.id}>
                {status.name}
              </option>
            ))}
          </Select>
        </Field>

        <Field label="Priority" htmlFor="filter-priority">
          <Select
            id="filter-priority"
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
          </Select>
        </Field>

        <Field label="Assignee" htmlFor="filter-assignee">
          <Select
            id="filter-assignee"
            onChange={(event) => onAssigneeFilterChange(event.target.value)}
            value={assigneeFilterId}
          >
            <option value="">All assignees</option>
            <option value="unassigned">Unassigned</option>
            {hasMissingAssigneeFilter ? (
              <option disabled value={assigneeFilterId}>
                {missingFilterOptionLabel("assignee")}
              </option>
            ) : null}
            {teamMembers.map((member) => (
              <option key={member.id} value={member.id}>
                {memberOptionLabel(member)}
              </option>
            ))}
          </Select>
        </Field>

        <Field label="Label" htmlFor="filter-label">
          <Select
            id="filter-label"
            onChange={(event) => onLabelFilterChange(event.target.value)}
            value={labelFilterId}
          >
            <option value="">All labels</option>
            {hasMissingLabelFilter ? (
              <option disabled value={labelFilterId}>
                {missingFilterOptionLabel("label")}
              </option>
            ) : null}
            {labels.map((label) => (
              <option key={label.id} value={label.id}>
                {label.name}
              </option>
            ))}
          </Select>
        </Field>

        <Field label="Due" htmlFor="filter-due">
          <Select
            id="filter-due"
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
          </Select>
        </Field>

        <Button
          variant="ghost"
          size="sm"
          disabled={!hasFilters}
          onClick={onClearFilters}
        >
          Clear
        </Button>
      </section>

      <p className="kl-muted kl-issues-list__summary">{summary}</p>

      <FormError message={issuesError} />

      {issues.length > 0 ? (
        <div className="kl-issues-list__rows">
          {issues.map((issue) => {
            const dueInfo = issueDueInfo(issue, today);
            const sprintName = issue.sprint_id
              ? sprintDisplayName(sprints, issue.sprint_id)
              : null;
            const canArchive = projects.find(
              (project) => project.id === issue.project_id,
            )?.can_write;

            return (
              <article className="issue-row kl-issue-row" key={issue.id}>
                <span className="kl-issue-row__key">{issue.issue_key}</span>
                <div className="kl-issue-row__body">
                  <h3>{issue.title}</h3>
                  <p className="kl-issue-row__meta">
                    {issueTypeLabels[issue.issue_type]} ·{" "}
                    {priorityLabels[issue.priority]} ·{" "}
                    {storyPointsLabel(issue.story_points)} ·{" "}
                    {memberDisplayName(teamMembers, issue.assignee_id)}
                    {sprintName ? ` · Sprint: ${sprintName}` : ""}
                  </p>
                  <div className="kl-issue-row__tags">
                    <WorkflowStatusBadge
                      fallbackLabel={issue.status.replaceAll("_", " ")}
                      status={issue.workflow_status}
                    />
                    {dueInfo ? (
                      <span className={`kl-due kl-due--${dueInfo.tone}`}>
                        {dueInfo.label}
                      </span>
                    ) : null}
                    {issue.labels.map((label) => (
                      <span
                        className="kl-label-chip"
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
                </div>
                <div className="kl-issue-row__actions">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => onOpenIssue(issue.id)}
                  >
                    Open
                  </Button>
                  <Button
                    variant="danger"
                    size="sm"
                    disabled={!canArchive || archivingIssueIds.includes(issue.id)}
                    onClick={() => onArchiveIssue(issue)}
                  >
                    {archivingIssueIds.includes(issue.id) ? "Archiving" : "Archive"}
                  </Button>
                </div>
              </article>
            );
          })}
        </div>
      ) : (
        <div className="kl-empty-block">No issues yet</div>
      )}
    </div>
  );
}
