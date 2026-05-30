import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import {
  type Issue,
  type Project,
  type Sprint,
  type SprintStatus,
} from "../../lib/api-types";
import {
  issueTypeLabels,
  priorityLabels,
  statusLabel,
} from "../../lib/issue-model";
import {
  sprintDateRange,
  sprintStatusLabels,
  sprintStatusOptions,
} from "../../lib/sprint-model";
import { hasText } from "../../lib/validation";

type SprintsSectionProps = {
  addingIssueToSprintIds: string[];
  canCreateSprint: boolean;
  canUpdateSprint: boolean;
  completingSprintIds: string[];
  editSprintEndDate: string;
  editSprintGoal: string;
  editSprintName: string;
  editSprintStartDate: string;
  isActive: boolean;
  isCreatingSprint: boolean;
  isEditingSprint: boolean;
  isLoadingSelectedSprint: boolean;
  isLoadingSprintPlanning: boolean;
  isLoadingSprints: boolean;
  isUpdatingSprint: boolean;
  onAddIssueToSprint: (issue: Issue) => void;
  onCancelSprintEdit: () => void;
  onCompleteSprint: (sprint: Sprint) => void;
  onCreateSprint: (event: FormEvent<HTMLFormElement>) => void;
  onEditSprintEndDateChange: (value: string) => void;
  onEditSprintGoalChange: (value: string) => void;
  onEditSprintNameChange: (value: string) => void;
  onEditSprintStartDateChange: (value: string) => void;
  onProjectFilterChange: (value: string) => void;
  onRemoveIssueFromSprint: (issue: Issue) => void;
  onSelectSprint: (sprintId: string) => void;
  onSprintEndDateChange: (value: string) => void;
  onSprintGoalChange: (value: string) => void;
  onSprintNameChange: (value: string) => void;
  onSprintProjectChange: (value: string) => void;
  onSprintStartDateChange: (value: string) => void;
  onStartEditingSprint: (sprint: Sprint) => void;
  onStartSprint: (sprint: Sprint) => void;
  onStatusFilterChange: (value: SprintStatus | "") => void;
  onUpdateSprint: (event: FormEvent<HTMLFormElement>) => void;
  onViewSprintProjectIssues: (projectId: string) => void;
  projectFilterId: string;
  projects: Project[];
  removingIssueFromSprintIds: string[];
  selectedSprint: Sprint | null;
  selectedSprintBacklogIssues: Issue[];
  selectedSprintError: string;
  selectedSprintIssues: Issue[];
  sprintEndDate: string;
  sprintFormError: string;
  sprintGoal: string;
  sprintName: string;
  sprintPlanningError: string;
  sprintProjectId: string;
  sprintStartDate: string;
  sprintStatusFilter: SprintStatus | "";
  sprints: Sprint[];
  sprintsError: string;
  startingSprintIds: string[];
};

export function SprintsSection({
  addingIssueToSprintIds,
  canCreateSprint,
  canUpdateSprint,
  completingSprintIds,
  editSprintEndDate,
  editSprintGoal,
  editSprintName,
  editSprintStartDate,
  isActive,
  isCreatingSprint,
  isEditingSprint,
  isLoadingSelectedSprint,
  isLoadingSprintPlanning,
  isLoadingSprints,
  isUpdatingSprint,
  onAddIssueToSprint,
  onCancelSprintEdit,
  onCompleteSprint,
  onCreateSprint,
  onEditSprintEndDateChange,
  onEditSprintGoalChange,
  onEditSprintNameChange,
  onEditSprintStartDateChange,
  onProjectFilterChange,
  onRemoveIssueFromSprint,
  onSelectSprint,
  onSprintEndDateChange,
  onSprintGoalChange,
  onSprintNameChange,
  onSprintProjectChange,
  onSprintStartDateChange,
  onStartEditingSprint,
  onStartSprint,
  onStatusFilterChange,
  onUpdateSprint,
  onViewSprintProjectIssues,
  projectFilterId,
  projects,
  removingIssueFromSprintIds,
  selectedSprint,
  selectedSprintBacklogIssues,
  selectedSprintError,
  selectedSprintIssues,
  sprintEndDate,
  sprintFormError,
  sprintGoal,
  sprintName,
  sprintPlanningError,
  sprintProjectId,
  sprintStartDate,
  sprintStatusFilter,
  sprints,
  sprintsError,
  startingSprintIds,
}: SprintsSectionProps) {
  const hasFilters = projectFilterId !== "" || sprintStatusFilter !== "";
  const summary = hasFilters
    ? `${sprints.length} sprints match current filters`
    : "Showing active, planned, and completed sprints";
  const selectedSprintIsStarting =
    selectedSprint !== null && startingSprintIds.includes(selectedSprint.id);
  const selectedSprintIsCompleting =
    selectedSprint !== null && completingSprintIds.includes(selectedSprint.id);

  return (
    <section
      className="sprints-layout"
      aria-label="Sprints"
      hidden={!isActive}
    >
      <div className="sprints-panel">
        <header className="section-header">
          <div>
            <p className="eyebrow">Iterations</p>
            <h2>Sprint list</h2>
          </div>
          {isLoadingSprints ? <span className="muted">Loading</span> : null}
        </header>

        <section className="sprint-filters" aria-label="Sprint filters">
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
            <span>Status</span>
            <select
              onChange={(event) =>
                onStatusFilterChange(event.target.value as SprintStatus | "")
              }
              value={sprintStatusFilter}
            >
              <option value="">All statuses</option>
              {sprintStatusOptions.map((status) => (
                <option key={status} value={status}>
                  {sprintStatusLabels[status]}
                </option>
              ))}
            </select>
          </label>
        </section>

        <p className="filter-summary">{summary}</p>
        <FormError message={sprintsError} />

        {sprints.length > 0 ? (
          <div className="sprint-list">
            {sprints.map((sprint) => (
              <article
                className={
                  selectedSprint?.id === sprint.id
                    ? "sprint-row sprint-row-selected"
                    : "sprint-row"
                }
                key={sprint.id}
              >
                <div>
                  <span className={`sprint-status-pill sprint-status-${sprint.status}`}>
                    {sprintStatusLabels[sprint.status]}
                  </span>
                  <h3>{sprint.name}</h3>
                  <p>
                    {sprint.project_key} · {sprintDateRange(sprint)}
                  </p>
                </div>
                <div className="sprint-row-meta">
                  <strong>{sprint.issue_count}</strong>
                  <span>{sprint.issue_count === 1 ? "issue" : "issues"}</span>
                </div>
                <button
                  className="small-button"
                  disabled={isLoadingSelectedSprint}
                  onClick={() => onSelectSprint(sprint.id)}
                  type="button"
                >
                  Details
                </button>
              </article>
            ))}
          </div>
        ) : (
          <div className="project-empty">No sprints yet</div>
        )}
      </div>

      <div className="sprints-sidebar">
        <form className="sprint-form" onSubmit={onCreateSprint}>
          <header className="section-header">
            <div>
              <p className="eyebrow">Planning</p>
              <h2>Create sprint</h2>
            </div>
          </header>

          <label>
            <span>Project</span>
            <select
              disabled={projects.length === 0}
              name="sprint-project-id"
              onChange={(event) => onSprintProjectChange(event.target.value)}
              value={sprintProjectId}
            >
              <option value="">Choose project</option>
              {projects.map((project) => (
                <option key={project.id} value={project.id}>
                  {project.key} · {project.name}
                </option>
              ))}
            </select>
          </label>

          <label>
            <span>Name</span>
            <input
              maxLength={120}
              name="sprint-name"
              onChange={(event) => onSprintNameChange(event.target.value)}
              placeholder="Sprint 1"
              value={sprintName}
            />
          </label>

          <label>
            <span>Goal</span>
            <textarea
              maxLength={1000}
              name="sprint-goal"
              onChange={(event) => onSprintGoalChange(event.target.value)}
              placeholder="What should this sprint achieve?"
              rows={3}
              value={sprintGoal}
            />
          </label>

          <div className="field-grid">
            <label>
              <span>Start date</span>
              <input
                name="sprint-start-date"
                onChange={(event) => onSprintStartDateChange(event.target.value)}
                type="date"
                value={sprintStartDate}
              />
            </label>

            <label>
              <span>End date</span>
              <input
                name="sprint-end-date"
                onChange={(event) => onSprintEndDateChange(event.target.value)}
                type="date"
                value={sprintEndDate}
              />
            </label>
          </div>

          <FormError message={sprintFormError} />

          <button disabled={!canCreateSprint} type="submit">
            {isCreatingSprint ? "Creating..." : "Create sprint"}
          </button>
        </form>

        <aside className="sprint-detail-panel" aria-label="Sprint details">
          <header className="section-header">
            <div>
              <p className="eyebrow">Sprint detail</p>
              <h2>{selectedSprint ? selectedSprint.name : "Select sprint"}</h2>
            </div>
            {isLoadingSelectedSprint ? <span className="muted">Loading</span> : null}
          </header>

          <FormError message={selectedSprintError} />

          {selectedSprint ? (
            isEditingSprint ? (
              <form className="sprint-inline-form" onSubmit={onUpdateSprint}>
                <label>
                  <span>Name</span>
                  <input
                    maxLength={120}
                    name="edit-sprint-name"
                    onChange={(event) =>
                      onEditSprintNameChange(event.target.value)
                    }
                    value={editSprintName}
                  />
                </label>

                <label>
                  <span>Goal</span>
                  <textarea
                    maxLength={1000}
                    name="edit-sprint-goal"
                    onChange={(event) =>
                      onEditSprintGoalChange(event.target.value)
                    }
                    rows={3}
                    value={editSprintGoal}
                  />
                </label>

                <div className="field-grid">
                  <label>
                    <span>Start date</span>
                    <input
                      name="edit-sprint-start-date"
                      onChange={(event) =>
                        onEditSprintStartDateChange(event.target.value)
                      }
                      type="date"
                      value={editSprintStartDate}
                    />
                  </label>

                  <label>
                    <span>End date</span>
                    <input
                      name="edit-sprint-end-date"
                      onChange={(event) =>
                        onEditSprintEndDateChange(event.target.value)
                      }
                      type="date"
                      value={editSprintEndDate}
                    />
                  </label>
                </div>

                <div className="form-actions">
                  <button
                    className="small-button"
                    disabled={!canUpdateSprint || !hasText(editSprintName)}
                    type="submit"
                  >
                    {isUpdatingSprint ? "Saving" : "Save"}
                  </button>
                  <button
                    className="ghost-button"
                    disabled={isUpdatingSprint}
                    onClick={onCancelSprintEdit}
                    type="button"
                  >
                    Cancel
                  </button>
                </div>
              </form>
            ) : (
              <>
                <p className="sprint-detail-description">
                  {selectedSprint.goal || "No sprint goal yet"}
                </p>

                <div className="sprint-detail-stats">
                  <article>
                    <span>Status</span>
                    <strong>{sprintStatusLabels[selectedSprint.status]}</strong>
                  </article>
                  <article>
                    <span>Issues</span>
                    <strong>{selectedSprint.issue_count}</strong>
                  </article>
                  <article>
                    <span>Project</span>
                    <strong>{selectedSprint.project_key}</strong>
                  </article>
                  <article>
                    <span>Dates</span>
                    <strong>{sprintDateRange(selectedSprint)}</strong>
                  </article>
                </div>

                <div className="sprint-detail-actions">
                  {selectedSprint.status !== "completed" ? (
                    <button
                      className="small-button"
                      onClick={() => onStartEditingSprint(selectedSprint)}
                      type="button"
                    >
                      Edit details
                    </button>
                  ) : null}
                  <button
                    className="small-button"
                    disabled={selectedSprint.status !== "planned" || selectedSprintIsStarting}
                    onClick={() => onStartSprint(selectedSprint)}
                    type="button"
                  >
                    {selectedSprintIsStarting ? "Starting" : "Start sprint"}
                  </button>
                  <button
                    className="small-button"
                    disabled={
                      selectedSprint.status !== "active" || selectedSprintIsCompleting
                    }
                    onClick={() => onCompleteSprint(selectedSprint)}
                    type="button"
                  >
                    {selectedSprintIsCompleting ? "Completing" : "Complete sprint"}
                  </button>
                  <button
                    className="small-button"
                    onClick={() => onViewSprintProjectIssues(selectedSprint.project_id)}
                    type="button"
                  >
                    View project issues
                  </button>
                </div>

                <SprintPlanningPanel
                  addingIssueToSprintIds={addingIssueToSprintIds}
                  backlogIssues={selectedSprintBacklogIssues}
                  isLoading={isLoadingSprintPlanning}
                  onAddIssue={onAddIssueToSprint}
                  onOpenProjectIssues={() =>
                    onViewSprintProjectIssues(selectedSprint.project_id)
                  }
                  onRemoveIssue={onRemoveIssueFromSprint}
                  planningError={sprintPlanningError}
                  removingIssueFromSprintIds={removingIssueFromSprintIds}
                  sprint={selectedSprint}
                  sprintIssues={selectedSprintIssues}
                />
              </>
            )
          ) : (
            <div className="comments-empty">No sprint selected</div>
          )}
        </aside>
      </div>
    </section>
  );
}

type SprintPlanningPanelProps = {
  addingIssueToSprintIds: string[];
  backlogIssues: Issue[];
  isLoading: boolean;
  onAddIssue: (issue: Issue) => void;
  onOpenProjectIssues: () => void;
  onRemoveIssue: (issue: Issue) => void;
  planningError: string;
  removingIssueFromSprintIds: string[];
  sprint: Sprint;
  sprintIssues: Issue[];
};

function SprintPlanningPanel({
  addingIssueToSprintIds,
  backlogIssues,
  isLoading,
  onAddIssue,
  onOpenProjectIssues,
  onRemoveIssue,
  planningError,
  removingIssueFromSprintIds,
  sprint,
  sprintIssues,
}: SprintPlanningPanelProps) {
  const canPlan = sprint.status !== "completed";

  return (
    <section className="sprint-planning-panel" aria-label="Sprint planning">
      <header className="section-header">
        <div>
          <p className="eyebrow">Backlog planning</p>
          <h3>Plan sprint issues</h3>
        </div>
        {isLoading ? <span className="muted">Loading</span> : null}
      </header>

      <FormError message={planningError} />

      {!canPlan ? (
        <p className="planning-note">
          Completed sprints are read-only. Issues stay visible for history.
        </p>
      ) : null}

      <div className="sprint-planning-grid">
        <section className="planning-column" aria-label="Sprint issues">
          <header>
            <span>In sprint</span>
            <strong>{sprintIssues.length}</strong>
          </header>

          {sprintIssues.length > 0 ? (
            <div className="planning-issue-list">
              {sprintIssues.map((issue) => (
                <PlanningIssueCard
                  actionLabel="Remove"
                  disabled={
                    !canPlan || removingIssueFromSprintIds.includes(issue.id)
                  }
                  isBusy={removingIssueFromSprintIds.includes(issue.id)}
                  issue={issue}
                  key={issue.id}
                  onAction={() => onRemoveIssue(issue)}
                />
              ))}
            </div>
          ) : (
            <div className="planning-empty">No issues in this sprint yet</div>
          )}
        </section>

        <section className="planning-column" aria-label="Project backlog">
          <header>
            <span>Project backlog</span>
            <strong>{backlogIssues.length}</strong>
          </header>

          {backlogIssues.length > 0 ? (
            <div className="planning-issue-list">
              {backlogIssues.map((issue) => (
                <PlanningIssueCard
                  actionLabel="Add"
                  disabled={!canPlan || addingIssueToSprintIds.includes(issue.id)}
                  isBusy={addingIssueToSprintIds.includes(issue.id)}
                  issue={issue}
                  key={issue.id}
                  onAction={() => onAddIssue(issue)}
                />
              ))}
            </div>
          ) : (
            <div className="planning-empty">
              <span>No unplanned open issues for this project</span>
              <button
                className="small-button"
                onClick={onOpenProjectIssues}
                type="button"
              >
                View project issues
              </button>
            </div>
          )}
        </section>
      </div>
    </section>
  );
}

type PlanningIssueCardProps = {
  actionLabel: string;
  disabled: boolean;
  isBusy: boolean;
  issue: Issue;
  onAction: () => void;
};

function PlanningIssueCard({
  actionLabel,
  disabled,
  isBusy,
  issue,
  onAction,
}: PlanningIssueCardProps) {
  return (
    <article className="planning-issue-card">
      <div>
        <span className="issue-key">{issue.issue_key}</span>
        <h4>{issue.title}</h4>
        <p>
          {issueTypeLabels[issue.issue_type]} · {priorityLabels[issue.priority]} ·{" "}
          {statusLabel(issue.status)}
        </p>
      </div>
      <button
        className="small-button"
        disabled={disabled}
        onClick={onAction}
        type="button"
      >
        {isBusy ? (actionLabel === "Add" ? "Adding" : "Removing") : actionLabel}
      </button>
    </article>
  );
}
