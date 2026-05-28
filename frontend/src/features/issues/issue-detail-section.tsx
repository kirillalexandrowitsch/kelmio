import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import {
  type CurrentUser,
  type Issue,
  type IssueActivity,
  type IssueComment,
  type IssuePriority,
  type IssueStatus,
  type IssueType,
  type Label,
  type TeamMember,
} from "../../lib/api-types";
import { IssueActivitySection } from "./issue-activity-section";
import { IssueCommentsSection } from "./issue-comments-section";
import { IssueDetailMainContent } from "./issue-detail-main-content";
import { IssueDetailSidebar } from "./issue-detail-sidebar";
import { IssueHierarchySection } from "./issue-hierarchy-section";

type IssueDetailSectionProps = {
  activity: IssueActivity[];
  activityError: string;
  archivingIssueIds: string[];
  assigningIssueIds: string[];
  canCreateComment: boolean;
  canCreateSubtask: boolean;
  childIssues: Issue[];
  commentBody: string;
  comments: IssueComment[];
  commentsError: string;
  currentUser: CurrentUser;
  deletingCommentIds: string[];
  editCommentBody: string;
  editIssueDescription: string;
  editIssueDueDate: string;
  editIssuePriority: IssuePriority;
  editIssueTitle: string;
  editIssueType: IssueType;
  editingCommentId: string;
  isActive: boolean;
  isCreatingComment: boolean;
  isCreatingSubtask: boolean;
  isEditingIssueDetails: boolean;
  isLoadingActivity: boolean;
  isLoadingChildIssues: boolean;
  isLoadingComments: boolean;
  isLoadingIssue: boolean;
  isUpdatingIssue: boolean;
  issue: Issue | null;
  issueError: string;
  hierarchyError: string;
  labelingIssueIds: string[];
  labels: Label[];
  onArchiveIssue: (issue: Issue) => void;
  onAssignIssue: (issueId: string, assigneeId: string) => void;
  onCancelEditingComment: () => void;
  onCancelIssueEdit: () => void;
  onCloseIssue: () => void;
  onCommentBodyChange: (value: string) => void;
  onCreateComment: (event: FormEvent<HTMLFormElement>) => void;
  onCreateSubtask: (event: FormEvent<HTMLFormElement>) => void;
  onDeleteComment: (comment: IssueComment) => void;
  onEditCommentBodyChange: (value: string) => void;
  onIssueDescriptionChange: (value: string) => void;
  onIssueDueDateChange: (value: string) => void;
  onIssuePriorityChange: (value: IssuePriority) => void;
  onIssueTitleChange: (value: string) => void;
  onIssueTypeChange: (value: IssueType) => void;
  onOpenIssue: (issueId: string) => void;
  onSubtaskPriorityChange: (value: IssuePriority) => void;
  onSubtaskStatusChange: (value: IssueStatus) => void;
  onSubtaskTitleChange: (value: string) => void;
  onSetIssueLabel: (
    issue: Issue,
    labelId: string,
    shouldAttach: boolean,
  ) => void;
  onStartEditingComment: (comment: IssueComment) => void;
  onStartEditingIssue: (issue: Issue) => void;
  onTransitionIssue: (issueId: string, status: IssueStatus) => void;
  onUpdateComment: (
    event: FormEvent<HTMLFormElement>,
    comment: IssueComment,
  ) => void;
  onUpdateIssue: (event: FormEvent<HTMLFormElement>) => void;
  teamMembers: TeamMember[];
  today: Date;
  transitioningIssueIds: string[];
  parentIssue: Issue | null;
  subtaskFormError: string;
  subtaskPriority: IssuePriority;
  subtaskStatus: IssueStatus;
  subtaskTitle: string;
  updatingCommentIds: string[];
};

export function IssueDetailSection({
  activity,
  activityError,
  archivingIssueIds,
  assigningIssueIds,
  canCreateComment,
  canCreateSubtask,
  childIssues,
  commentBody,
  comments,
  commentsError,
  currentUser,
  deletingCommentIds,
  editCommentBody,
  editIssueDescription,
  editIssueDueDate,
  editIssuePriority,
  editIssueTitle,
  editIssueType,
  editingCommentId,
  isActive,
  isCreatingComment,
  isCreatingSubtask,
  isEditingIssueDetails,
  isLoadingActivity,
  isLoadingChildIssues,
  isLoadingComments,
  isLoadingIssue,
  isUpdatingIssue,
  issue,
  issueError,
  hierarchyError,
  labelingIssueIds,
  labels,
  onArchiveIssue,
  onAssignIssue,
  onCancelEditingComment,
  onCancelIssueEdit,
  onCloseIssue,
  onCommentBodyChange,
  onCreateComment,
  onCreateSubtask,
  onDeleteComment,
  onEditCommentBodyChange,
  onIssueDescriptionChange,
  onIssueDueDateChange,
  onIssuePriorityChange,
  onIssueTitleChange,
  onIssueTypeChange,
  onOpenIssue,
  onSetIssueLabel,
  onStartEditingComment,
  onStartEditingIssue,
  onSubtaskPriorityChange,
  onSubtaskStatusChange,
  onSubtaskTitleChange,
  onTransitionIssue,
  onUpdateComment,
  onUpdateIssue,
  teamMembers,
  today,
  transitioningIssueIds,
  parentIssue,
  subtaskFormError,
  subtaskPriority,
  subtaskStatus,
  subtaskTitle,
  updatingCommentIds,
}: IssueDetailSectionProps) {
  return (
    <section
      className="issue-detail-panel"
      aria-label="Issue details"
      hidden={!isActive}
    >
      <header className="section-header">
        <div>
          <p className="eyebrow">Issue details</p>
          <h2>
            {issue ? `${issue.issue_key} · ${issue.title}` : "Select an issue"}
          </h2>
        </div>
        {issue ? (
          <div className="detail-actions">
            <button
              className="ghost-button"
              onClick={() => {
                if (isEditingIssueDetails) {
                  onCancelIssueEdit();
                } else {
                  onStartEditingIssue(issue);
                }
              }}
              type="button"
            >
              {isEditingIssueDetails ? "Cancel edit" : "Edit"}
            </button>
            <button
              className="ghost-button"
              onClick={onCloseIssue}
              type="button"
            >
              Close
            </button>
            <button
              className="ghost-button danger-button"
              disabled={archivingIssueIds.includes(issue.id)}
              onClick={() => onArchiveIssue(issue)}
              type="button"
            >
              {archivingIssueIds.includes(issue.id) ? "Archiving" : "Archive"}
            </button>
          </div>
        ) : null}
      </header>

      {issueError ? <FormError message={issueError} /> : null}

      {isLoadingIssue ? <span className="muted">Loading details</span> : null}

      {issue ? (
        <div className="issue-detail-body">
          <div className="issue-detail-main">
            <IssueDetailMainContent
              editDescription={editIssueDescription}
              editDueDate={editIssueDueDate}
              editPriority={editIssuePriority}
              editTitle={editIssueTitle}
              editType={editIssueType}
              isEditing={isEditingIssueDetails}
              isUpdating={isUpdatingIssue}
              issue={issue}
              onCancelEdit={onCancelIssueEdit}
              onDescriptionChange={onIssueDescriptionChange}
              onDueDateChange={onIssueDueDateChange}
              onPriorityChange={onIssuePriorityChange}
              onSubmit={onUpdateIssue}
              onTitleChange={onIssueTitleChange}
              onTypeChange={onIssueTypeChange}
            />

            <IssueHierarchySection
              canCreateSubtask={canCreateSubtask}
              children={childIssues}
              formError={subtaskFormError}
              hierarchyError={hierarchyError}
              isCreatingSubtask={isCreatingSubtask}
              isLoadingChildren={isLoadingChildIssues}
              issue={issue}
              onCreateSubtask={onCreateSubtask}
              onOpenIssue={onOpenIssue}
              onPriorityChange={onSubtaskPriorityChange}
              onStatusChange={onSubtaskStatusChange}
              onTitleChange={onSubtaskTitleChange}
              parentIssue={parentIssue}
              subtaskPriority={subtaskPriority}
              subtaskStatus={subtaskStatus}
              subtaskTitle={subtaskTitle}
            />

            <IssueCommentsSection
              canCreateComment={canCreateComment}
              commentBody={commentBody}
              comments={comments}
              commentsError={commentsError}
              currentUser={currentUser}
              deletingCommentIds={deletingCommentIds}
              editCommentBody={editCommentBody}
              editingCommentId={editingCommentId}
              isCreatingComment={isCreatingComment}
              isLoadingComments={isLoadingComments}
              onCancelEditingComment={onCancelEditingComment}
              onCommentBodyChange={onCommentBodyChange}
              onCreateComment={onCreateComment}
              onDeleteComment={onDeleteComment}
              onEditCommentBodyChange={onEditCommentBodyChange}
              onStartEditingComment={onStartEditingComment}
              onUpdateComment={onUpdateComment}
              updatingCommentIds={updatingCommentIds}
            />

            <IssueActivitySection
              activity={activity}
              activityError={activityError}
              isLoadingActivity={isLoadingActivity}
              teamMembers={teamMembers}
            />
          </div>

          <IssueDetailSidebar
            assigningIssueIds={assigningIssueIds}
            issue={issue}
            labelingIssueIds={labelingIssueIds}
            labels={labels}
            onAssignIssue={onAssignIssue}
            onSetIssueLabel={onSetIssueLabel}
            onTransitionIssue={onTransitionIssue}
            teamMembers={teamMembers}
            today={today}
            transitioningIssueIds={transitioningIssueIds}
          />
        </div>
      ) : (
        <div className="issue-detail-empty">
          Open a card from Recent issues or the board to inspect its details.
        </div>
      )}
    </section>
  );
}
