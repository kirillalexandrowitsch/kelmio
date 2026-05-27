import { type DragEvent } from "react";

import { type Issue, type IssueStatus, type TeamMember } from "../../lib/api-types";
import { columns, issueDueInfo, priorityLabels } from "../../lib/issue-model";
import { memberDisplayName } from "../../lib/team-view";

type BoardSectionProps = {
  archivingIssueIds: string[];
  isActive: boolean;
  issues: Issue[];
  onArchiveIssue: (issue: Issue) => void;
  onIssueDragOver: (event: DragEvent<HTMLElement>) => void;
  onIssueDragStart: (event: DragEvent<HTMLElement>, issueId: string) => void;
  onIssueDrop: (event: DragEvent<HTMLElement>, nextStatus: IssueStatus) => void;
  onOpenIssue: (issueId: string) => void;
  onTransitionIssue: (issueId: string, status: IssueStatus) => void;
  teamMembers: TeamMember[];
  today: Date;
  transitioningIssueIds: string[];
};

export function BoardSection({
  archivingIssueIds,
  isActive,
  issues,
  onArchiveIssue,
  onIssueDragOver,
  onIssueDragStart,
  onIssueDrop,
  onOpenIssue,
  onTransitionIssue,
  teamMembers,
  today,
  transitioningIssueIds,
}: BoardSectionProps) {
  return (
    <section
      className="board"
      aria-label="Kanban board"
      hidden={!isActive}
    >
      {columns.map((column) => {
        const columnIssues = issues.filter((issue) => issue.status === column.status);

        return (
          <article
            className="board-column"
            key={column.title}
            onDragOver={onIssueDragOver}
            onDrop={(event) => onIssueDrop(event, column.status)}
          >
            <header>
              <h2>{column.title}</h2>
              <span>{columnIssues.length}</span>
            </header>
            <div className="board-card-list">
              {columnIssues.map((issue) => {
                const dueInfo = issueDueInfo(issue, today);

                return (
                  <article
                    className="issue-card"
                    draggable
                    key={issue.id}
                    onDragStart={(event) => onIssueDragStart(event, issue.id)}
                  >
                    <div className="issue-card-meta">
                      <span>{issue.issue_key}</span>
                      <span>{priorityLabels[issue.priority]}</span>
                    </div>
                    <h3>{issue.title}</h3>
                    {dueInfo ? (
                      <span className={`due-badge due-badge-${dueInfo.tone}`}>
                        {dueInfo.label}
                      </span>
                    ) : null}
                    <p>
                      Assignee: {memberDisplayName(teamMembers, issue.assignee_id)}
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
                      <label>
                        <span>Status</span>
                        <select
                          aria-label={`Status for ${issue.issue_key}`}
                          disabled={transitioningIssueIds.includes(issue.id)}
                          onChange={(event) =>
                            onTransitionIssue(
                              issue.id,
                              event.target.value as IssueStatus,
                            )
                          }
                          value={issue.status}
                        >
                          {columns.map((nextColumn) => (
                            <option
                              key={nextColumn.status}
                              value={nextColumn.status}
                            >
                              {nextColumn.title}
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
    </section>
  );
}
