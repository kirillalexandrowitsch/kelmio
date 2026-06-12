import { type FormEvent } from "react";

import { FormError } from "../../components/form-feedback";
import {
  type CurrentUser,
  type Issue,
  type IssueActivity,
  type IssueComment,
  type IssueLink,
  type IssueLinkType,
  type IssuePriority,
  type IssueType,
  type Label,
  type ProjectWorkflowStatus,
  type WorkflowStatus,
  type TeamMember,
} from "../../lib/api-types";
import { IssueActivitySection } from "./issue-activity-section";
import { IssueCommentsSection } from "./issue-comments-section";
import { IssueDetailMainContent } from "./issue-detail-main-content";
import { IssueDetailSidebar } from "./issue-detail-sidebar";
import { IssueHierarchySection } from "./issue-hierarchy-section";
import { IssueLinksSection } from "./issue-links-section";

type IssueDetailSectionProps = {
  activity: IssueActivity[];
  activityError: string;
  archivingIssueIds: string[];
  assigningIssueIds: string[];
  availableLinkIssues: Issue[];
  canWriteIssue: boolean;
  canCreateComment: boolean;
  canCreateIssueLink: boolean;
  canCreateSubtask: boolean;
  childIssues: Issue[];
  commentBody: string;
  comments: IssueComment[];
  commentsError: string;
  currentUser: CurrentUser;
  deletingCommentIds: string[];
  deletingIssueLinkIds: string[];
  editCommentBody: string;
  editIssueDescription: string;
  editIssueDueDate: string;
  editIssuePriority: IssuePriority;
  editIssueStoryPoints: string;
  editIssueTitle: string;
  editIssueType: IssueType;
  editingCommentId: string;
  isActive: boolean;
  isCreatingComment: boolean;
  isCreatingIssueLink: boolean;
  isCreatingSubtask: boolean;
  isEditingIssueDetails: boolean;
  isLoadingActivity: boolean;
  isLoadingChildIssues: boolean;
  isLoadingComments: boolean;
  isLoadingIssueLinks: boolean;
  isLoadingIssue: boolean;
  isUpdatingIssue: boolean;
  issue: Issue | null;
  issueError: string;
  hierarchyError: string;
  issueLinks: IssueLink[];
  labelingIssueIds: string[];
  labels: Label[];
  linkFormError: string;
  linksError: string;
  linkTargetIssueId: string;
  linkType: IssueLinkType;
  onArchiveIssue: (issue: Issue) => void;
  onAssignIssue: (issueId: string, assigneeId: string) => void;
  onCancelEditingComment: () => void;
  onCancelIssueEdit: () => void;
  onCloseIssue: () => void;
  onCommentBodyChange: (value: string) => void;
  onCreateComment: (event: FormEvent<HTMLFormElement>) => void;
  onCreateIssueLink: (event: FormEvent<HTMLFormElement>) => void;
  onCreateSubtask: (event: FormEvent<HTMLFormElement>) => void;
  onDeleteComment: (comment: IssueComment) => void;
  onDeleteIssueLink: (link: IssueLink) => void;
  onEditCommentBodyChange: (value: string) => void;
  onIssueDescriptionChange: (value: string) => void;
  onIssueDueDateChange: (value: string) => void;
  onIssuePriorityChange: (value: IssuePriority) => void;
  onIssueStoryPointsChange: (value: string) => void;
  onIssueTitleChange: (value: string) => void;
  onIssueTypeChange: (value: IssueType) => void;
  onIssueLinkTargetChange: (issueId: string) => void;
  onIssueLinkTypeChange: (linkType: IssueLinkType) => void;
  onOpenIssue: (issueId: string) => void;
  onSubtaskPriorityChange: (value: IssuePriority) => void;
  onSubtaskStoryPointsChange: (value: string) => void;
  onSubtaskStatusChange: (value: string) => void;
  onSubtaskTitleChange: (value: string) => void;
  onSetIssueLabel: (
    issue: Issue,
    labelId: string,
    shouldAttach: boolean,
  ) => void;
  onStartEditingComment: (comment: IssueComment) => void;
  onStartEditingIssue: (issue: Issue) => void;
  onTransitionIssue: (issueId: string, workflowStatusId: string) => void;
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
  subtaskStoryPoints: string;
  subtaskStatusId: string;
  workflowStatuses: ProjectWorkflowStatus[];
  transitionStatuses: WorkflowStatus[];
  subtaskTitle: string;
  updatingCommentIds: string[];
};

export function IssueDetailSection({
  activity,
  activityError,
  archivingIssueIds,
  assigningIssueIds,
  availableLinkIssues,
  canWriteIssue,
  canCreateComment,
  canCreateIssueLink,
  canCreateSubtask,
  childIssues,
  commentBody,
  comments,
  commentsError,
  currentUser,
  deletingCommentIds,
  deletingIssueLinkIds,
  editCommentBody,
  editIssueDescription,
  editIssueDueDate,
  editIssuePriority,
  editIssueStoryPoints,
  editIssueTitle,
  editIssueType,
  editingCommentId,
  isActive,
  isCreatingComment,
  isCreatingIssueLink,
  isCreatingSubtask,
  isEditingIssueDetails,
  isLoadingActivity,
  isLoadingChildIssues,
  isLoadingComments,
  isLoadingIssueLinks,
  isLoadingIssue,
  isUpdatingIssue,
  issue,
  issueError,
  hierarchyError,
  issueLinks,
  labelingIssueIds,
  labels,
  linkFormError,
  linksError,
  linkTargetIssueId,
  linkType,
  onArchiveIssue,
  onAssignIssue,
  onCancelEditingComment,
  onCancelIssueEdit,
  onCloseIssue,
  onCommentBodyChange,
  onCreateComment,
  onCreateIssueLink,
  onCreateSubtask,
  onDeleteComment,
  onDeleteIssueLink,
  onEditCommentBodyChange,
  onIssueDescriptionChange,
  onIssueDueDateChange,
  onIssuePriorityChange,
  onIssueStoryPointsChange,
  onIssueTitleChange,
  onIssueTypeChange,
  onIssueLinkTargetChange,
  onIssueLinkTypeChange,
  onOpenIssue,
  onSetIssueLabel,
  onStartEditingComment,
  onStartEditingIssue,
  onSubtaskPriorityChange,
  onSubtaskStoryPointsChange,
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
  subtaskStoryPoints,
  subtaskStatusId,
  subtaskTitle,
  transitionStatuses,
  updatingCommentIds,
  workflowStatuses,
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
              disabled={!canWriteIssue}
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
              disabled={!canWriteIssue || archivingIssueIds.includes(issue.id)}
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
              editStoryPoints={editIssueStoryPoints}
              editTitle={editIssueTitle}
              editType={editIssueType}
              isEditing={isEditingIssueDetails}
              isUpdating={isUpdatingIssue}
              issue={issue}
              onCancelEdit={onCancelIssueEdit}
              onDescriptionChange={onIssueDescriptionChange}
              onDueDateChange={onIssueDueDateChange}
              onPriorityChange={onIssuePriorityChange}
              onStoryPointsChange={onIssueStoryPointsChange}
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
              onStoryPointsChange={onSubtaskStoryPointsChange}
              onStatusChange={onSubtaskStatusChange}
              onTitleChange={onSubtaskTitleChange}
              parentIssue={parentIssue}
              subtaskPriority={subtaskPriority}
              subtaskStoryPoints={subtaskStoryPoints}
              subtaskStatusId={subtaskStatusId}
              subtaskTitle={subtaskTitle}
              statuses={workflowStatuses}
            />

            <IssueLinksSection
              availableIssues={availableLinkIssues}
              canCreateLink={canCreateIssueLink}
              deletingLinkIds={deletingIssueLinkIds}
              formError={linkFormError}
              isCreatingLink={isCreatingIssueLink}
              isLoadingLinks={isLoadingIssueLinks}
              issue={issue}
              linkTargetIssueId={linkTargetIssueId}
              linkType={linkType}
              links={issueLinks}
              linksError={linksError}
              onCreateLink={onCreateIssueLink}
              onDeleteLink={onDeleteIssueLink}
              onOpenIssue={onOpenIssue}
              onTargetIssueChange={onIssueLinkTargetChange}
              onTypeChange={onIssueLinkTypeChange}
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
            canWriteIssue={canWriteIssue}
            issue={issue}
            labelingIssueIds={labelingIssueIds}
            labels={labels}
            onAssignIssue={onAssignIssue}
            onSetIssueLabel={onSetIssueLabel}
            onTransitionIssue={onTransitionIssue}
            teamMembers={teamMembers}
            today={today}
            transitionStatuses={transitionStatuses}
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
