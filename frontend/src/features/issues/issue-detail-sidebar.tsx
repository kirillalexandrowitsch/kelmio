import {
  type Issue,
  type IssueStatus,
  type Label,
  type TeamMember,
} from "../../lib/api-types";
import { columns, issueDueInfo } from "../../lib/issue-model";
import { formatDateTime } from "../../lib/formatting";
import { assignableTeamMembers, memberOptionLabel } from "../../lib/team-view";

type IssueDetailSidebarProps = {
  assigningIssueIds: string[];
  issue: Issue;
  labelingIssueIds: string[];
  labels: Label[];
  onAssignIssue: (issueId: string, assigneeId: string) => void;
  onSetIssueLabel: (
    issue: Issue,
    labelId: string,
    shouldAttach: boolean,
  ) => void;
  onTransitionIssue: (issueId: string, status: IssueStatus) => void;
  teamMembers: TeamMember[];
  today: Date;
  transitioningIssueIds: string[];
};

export function IssueDetailSidebar({
  assigningIssueIds,
  issue,
  labelingIssueIds,
  labels,
  onAssignIssue,
  onSetIssueLabel,
  onTransitionIssue,
  teamMembers,
  today,
  transitioningIssueIds,
}: IssueDetailSidebarProps) {
  const dueInfo = issueDueInfo(issue, today);

  return (
    <aside className="issue-detail-sidebar">
      <label className="issue-detail-status">
        <span>Status</span>
        <select
          disabled={transitioningIssueIds.includes(issue.id)}
          onChange={(event) =>
            onTransitionIssue(issue.id, event.target.value as IssueStatus)
          }
          value={issue.status}
        >
          {columns.map((column) => (
            <option key={column.status} value={column.status}>
              {column.title}
            </option>
          ))}
        </select>
      </label>

      <label className="issue-detail-status">
        <span>Assignee</span>
        <select
          disabled={assigningIssueIds.includes(issue.id)}
          onChange={(event) => onAssignIssue(issue.id, event.target.value)}
          value={issue.assignee_id ?? ""}
        >
          <option value="">Unassigned</option>
          {assignableTeamMembers(teamMembers, issue.assignee_id).map((member) => (
            <option
              disabled={!member.is_active}
              key={member.id}
              value={member.id}
            >
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
                  checked={issue.labels.some(
                    (issueLabel) => issueLabel.id === label.id,
                  )}
                  disabled={labelingIssueIds.includes(issue.id)}
                  onChange={(event) =>
                    onSetIssueLabel(issue, label.id, event.target.checked)
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

      <div className="metadata-grid">
        <div>
          <span>Project</span>
          <strong>{issue.project_key}</strong>
        </div>
        <div>
          <span>Due date</span>
          {dueInfo ? (
            <strong>
              <span className={`due-badge due-badge-${dueInfo.tone}`}>
                {dueInfo.label}
              </span>
            </strong>
          ) : (
            <strong>No due date</strong>
          )}
        </div>
        <div>
          <span>Created</span>
          <strong>{formatDateTime(issue.created_at)}</strong>
        </div>
        <div>
          <span>Updated</span>
          <strong>{formatDateTime(issue.updated_at)}</strong>
        </div>
      </div>
    </aside>
  );
}
