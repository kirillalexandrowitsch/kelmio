import { useState, type DragEvent } from "react";

import type {
  Issue,
  Project,
  ProjectWorkflow,
  TeamMember,
} from "../../lib/api-types";
import {
  issueDueInfo,
  priorityLabels,
  storyPointsLabel,
} from "../../lib/issue-model";
import { memberDisplayName } from "../../lib/team-view";
import {
  activeWorkflowStatuses,
  allowedTransitionStatuses,
  canTransitionToWorkflowStatus,
  workflowStatusForIssue,
  workflowStatusStyle,
} from "../../lib/workflow-model";

type BoardSectionProps = {
  archivingIssueIds: string[];
  error: string;
  isActive: boolean;
  isLoading: boolean;
  issues: Issue[];
  onArchiveIssue: (issue: Issue) => void;
  onIssueDrop: (event: DragEvent<HTMLElement>, workflowStatusId: string) => void;
  onOpenIssue: (issueId: string) => void;
  onProjectChange: (projectId: string) => void;
  onTransitionIssue: (issueId: string, workflowStatusId: string) => void;
  projectId: string;
  projects: Project[];
  teamMembers: TeamMember[];
  today: Date;
  transitioningIssueIds: string[];
  workflow?: ProjectWorkflow;
  workflowError: string;
};

export function BoardSection({
  archivingIssueIds,
  error,
  isActive,
  isLoading,
  issues,
  onArchiveIssue,
  onIssueDrop,
  onOpenIssue,
  onProjectChange,
  onTransitionIssue,
  projectId,
  projects,
  teamMembers,
  today,
  transitioningIssueIds,
  workflow,
  workflowError,
}: BoardSectionProps) {
  const [draggedIssueId, setDraggedIssueId] = useState("");
  const project = projects.find((currentProject) => currentProject.id === projectId);
  const statuses = activeWorkflowStatuses(workflow);
  const draggedIssue =
    issues.find((currentIssue) => currentIssue.id === draggedIssueId) ?? null;

  return (
    <section
      aria-label="Kanban board"
      className="board-section"
      hidden={!isActive}
    >
      <header className="section-header board-section-header">
        <div>
          <p className="eyebrow">Project workflow</p>
          <h2>{project ? `${project.key} · ${project.name}` : "Kanban board"}</h2>
        </div>
        <label>
          <span>Project</span>
          <select
            aria-label="Board project"
            onChange={(event) => onProjectChange(event.target.value)}
            value={projectId}
          >
            <option value="">Select a project</option>
            {projects.map((currentProject) => (
              <option key={currentProject.id} value={currentProject.id}>
                {currentProject.key} · {currentProject.name}
              </option>
            ))}
          </select>
        </label>
      </header>

      {!projectId ? (
        <div className="board-empty-state">Select a project to open its board</div>
      ) : error || workflowError ? (
        <div className="board-empty-state board-error-state">
          {error || workflowError}
        </div>
      ) : isLoading || !workflow ? (
        <div className="board-empty-state">Loading project board</div>
      ) : statuses.length === 0 ? (
        <div className="board-empty-state">This project has no active statuses</div>
      ) : (
        <>
          {!project?.can_write ? (
            <p className="board-read-only-note">
              This board is read-only for your project role.
            </p>
          ) : null}
          <div className="board">
            {statuses.map((status) => {
              const columnIssues = issues.filter(
                (issue) => workflowStatusForIssue(issue, workflow)?.id === status.id,
              );
              const canDrop =
                draggedIssue !== null &&
                project?.can_write === true &&
                canTransitionToWorkflowStatus(draggedIssue, workflow, status.id);
              const dropStateClass = draggedIssue
                ? canDrop
                  ? " board-column-drop-allowed"
                  : " board-column-drop-disabled"
                : "";

              return (
                <article
                  className={`board-column${dropStateClass}`}
                  key={status.id}
                  onDragOver={(event) => {
                    if (canDrop) {
                      event.preventDefault();
                      event.dataTransfer.dropEffect = "move";
                    }
                  }}
                  onDrop={(event) => {
                    if (canDrop) {
                      onIssueDrop(event, status.id);
                    }
                    setDraggedIssueId("");
                  }}
                  style={{ borderTopColor: status.color }}
                >
                  <header>
                    <div>
                      <h2>{status.name}</h2>
                      <small>{status.category.replaceAll("_", " ")}</small>
                    </div>
                    <span style={workflowStatusStyle(status)}>{columnIssues.length}</span>
                  </header>
                  <div className="board-card-list">
                    {columnIssues.map((issue) => {
                      const dueInfo = issueDueInfo(issue, today);
                      const isTransitioning = transitioningIssueIds.includes(issue.id);
                      const transitionStatuses = allowedTransitionStatuses(
                        workflow,
                        status.id,
                      );
                      const canWriteIssue = project?.can_write === true;

                      return (
                        <article
                          className="issue-card"
                          draggable={canWriteIssue && !isTransitioning}
                          key={issue.id}
                          onDragEnd={() => setDraggedIssueId("")}
                          onDragStart={(event) => {
                            if (!canWriteIssue) {
                              event.preventDefault();
                              return;
                            }
                            setDraggedIssueId(issue.id);
                            event.dataTransfer.setData("text/plain", issue.id);
                            event.dataTransfer.effectAllowed = "move";
                          }}
                        >
                          <div className="issue-card-meta">
                            <span>{issue.issue_key}</span>
                            <span>{priorityLabels[issue.priority]}</span>
                          </div>
                          <h3>{issue.title}</h3>
                          <span className="detail-chip">
                            {storyPointsLabel(issue.story_points)}
                          </span>
                          {dueInfo ? (
                            <span className={`due-badge due-badge-${dueInfo.tone}`}>
                              {dueInfo.label}
                            </span>
                          ) : null}
                          <p>
                            Assignee:{" "}
                            {memberDisplayName(teamMembers, issue.assignee_id)}
                          </p>
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
                          <div className="issue-card-actions">
                            <button
                              className="small-button"
                              onClick={() => onOpenIssue(issue.id)}
                              type="button"
                            >
                              Open
                            </button>
                            {canWriteIssue ? (
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
                            ) : null}
                            <label>
                              <span>Status</span>
                              <select
                                aria-label={`Status for ${issue.issue_key}`}
                                disabled={!canWriteIssue || isTransitioning}
                                onChange={(event) =>
                                  onTransitionIssue(issue.id, event.target.value)
                                }
                                value={status.id}
                              >
                                {transitionStatuses.map((nextStatus) => (
                                  <option key={nextStatus.id} value={nextStatus.id}>
                                    {nextStatus.name}
                                  </option>
                                ))}
                              </select>
                            </label>
                          </div>
                        </article>
                      );
                    })}

                    {columnIssues.length === 0 ? (
                      <div className="empty-state">No issues yet</div>
                    ) : null}
                  </div>
                </article>
              );
            })}
          </div>
        </>
      )}
    </section>
  );
}
