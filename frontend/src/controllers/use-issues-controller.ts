import { useState } from "react";
import type {
  Issue,
  IssueActivity,
  IssueComment,
  IssueDueFilter,
  IssueLink,
  IssueLinkType,
  IssuePriority,
  IssueSort,
  IssueType,
  SavedFilter,
} from "../lib/api";

export function useIssuesController() {
  const [issues, setIssues] = useState<Issue[]>([]);
  const [issuesError, setIssuesError] = useState("");
  const [issueFormError, setIssueFormError] = useState("");
  const [isLoadingIssues, setIsLoadingIssues] = useState(false);
  const [isCreatingIssue, setIsCreatingIssue] = useState(false);
  const [selectedProjectId, setSelectedProjectId] = useState("");
  const [issueTitle, setIssueTitle] = useState("");
  const [issueDescription, setIssueDescription] = useState("");
  const [issueType, setIssueType] = useState<IssueType>("task");
  const [issuePriority, setIssuePriority] = useState<IssuePriority>("medium");
  const [issueStoryPoints, setIssueStoryPoints] = useState("0");
  const [issueWorkflowStatusId, setIssueWorkflowStatusId] = useState("");
  const [issueAssigneeId, setIssueAssigneeId] = useState("");
  const [issueDueDate, setIssueDueDate] = useState("");
  const [newIssueLabelIds, setNewIssueLabelIds] = useState<string[]>([]);
  const [issueFilterQuery, setIssueFilterQuery] = useState("");
  const [issueSort, setIssueSort] = useState<IssueSort>("created_desc");
  const [issueFilterProjectId, setIssueFilterProjectId] = useState("");
  const [issueFilterSprintId, setIssueFilterSprintId] = useState("");
  const [issueFilterStatus, setIssueFilterStatus] = useState("");
  const [issueFilterWorkflowStatusId, setIssueFilterWorkflowStatusId] = useState("");
  const [issueFilterPriority, setIssueFilterPriority] = useState<
    IssuePriority | ""
  >("");
  const [issueFilterAssigneeId, setIssueFilterAssigneeId] = useState("");
  const [issueFilterLabelId, setIssueFilterLabelId] = useState("");
  const [issueFilterDue, setIssueFilterDue] = useState<IssueDueFilter | "">("");
  const [savedFilters, setSavedFilters] = useState<SavedFilter[]>([]);
  const [savedFiltersError, setSavedFiltersError] = useState("");
  const [savedFilterFormError, setSavedFilterFormError] = useState("");
  const [savedFilterName, setSavedFilterName] = useState("");
  const [isLoadingSavedFilters, setIsLoadingSavedFilters] = useState(false);
  const [isCreatingSavedFilter, setIsCreatingSavedFilter] = useState(false);
  const [updatingSavedFilterIds, setUpdatingSavedFilterIds] = useState<string[]>(
    [],
  );
  const [deletingSavedFilterIds, setDeletingSavedFilterIds] = useState<string[]>(
    [],
  );
  const [renameSavedFilterId, setRenameSavedFilterId] = useState("");
  const [renameSavedFilterName, setRenameSavedFilterName] = useState("");
  const [transitioningIssueIds, setTransitioningIssueIds] = useState<string[]>([]);
  const [assigningIssueIds, setAssigningIssueIds] = useState<string[]>([]);
  const [labelingIssueIds, setLabelingIssueIds] = useState<string[]>([]);
  const [archivingIssueIds, setArchivingIssueIds] = useState<string[]>([]);
  const [selectedIssue, setSelectedIssue] = useState<Issue | null>(null);
  const [selectedIssueError, setSelectedIssueError] = useState("");
  const [isLoadingSelectedIssue, setIsLoadingSelectedIssue] = useState(false);
  const [isEditingIssueDetails, setIsEditingIssueDetails] = useState(false);
  const [isUpdatingIssue, setIsUpdatingIssue] = useState(false);
  const [editIssueTitle, setEditIssueTitle] = useState("");
  const [editIssueDescription, setEditIssueDescription] = useState("");
  const [editIssueType, setEditIssueType] = useState<IssueType>("task");
  const [editIssuePriority, setEditIssuePriority] =
    useState<IssuePriority>("medium");
  const [editIssueStoryPoints, setEditIssueStoryPoints] = useState("0");
  const [editIssueDueDate, setEditIssueDueDate] = useState("");
  const [issueComments, setIssueComments] = useState<IssueComment[]>([]);
  const [commentsError, setCommentsError] = useState("");
  const [commentBody, setCommentBody] = useState("");
  const [isLoadingComments, setIsLoadingComments] = useState(false);
  const [isCreatingComment, setIsCreatingComment] = useState(false);
  const [editingCommentId, setEditingCommentId] = useState("");
  const [editCommentBody, setEditCommentBody] = useState("");
  const [updatingCommentIds, setUpdatingCommentIds] = useState<string[]>([]);
  const [deletingCommentIds, setDeletingCommentIds] = useState<string[]>([]);
  const [issueActivity, setIssueActivity] = useState<IssueActivity[]>([]);
  const [activityError, setActivityError] = useState("");
  const [isLoadingActivity, setIsLoadingActivity] = useState(false);
  const [issueChildren, setIssueChildren] = useState<Issue[]>([]);
  const [hierarchyError, setHierarchyError] = useState("");
  const [subtaskFormError, setSubtaskFormError] = useState("");
  const [isLoadingIssueChildren, setIsLoadingIssueChildren] = useState(false);
  const [isCreatingSubtask, setIsCreatingSubtask] = useState(false);
  const [subtaskTitle, setSubtaskTitle] = useState("");
  const [subtaskPriority, setSubtaskPriority] =
    useState<IssuePriority>("medium");
  const [subtaskStoryPoints, setSubtaskStoryPoints] = useState("0");
  const [subtaskWorkflowStatusId, setSubtaskWorkflowStatusId] = useState("");
  const [issueLinks, setIssueLinks] = useState<IssueLink[]>([]);
  const [linksError, setLinksError] = useState("");
  const [linkFormError, setLinkFormError] = useState("");
  const [isLoadingIssueLinks, setIsLoadingIssueLinks] = useState(false);
  const [isCreatingIssueLink, setIsCreatingIssueLink] = useState(false);
  const [deletingIssueLinkIds, setDeletingIssueLinkIds] = useState<string[]>([]);
  const [linkTargetIssueId, setLinkTargetIssueId] = useState("");
  const [linkType, setLinkType] = useState<IssueLinkType>("relates");

  return {
    issues, setIssues, issuesError, setIssuesError, issueFormError,
    setIssueFormError, isLoadingIssues, setIsLoadingIssues, isCreatingIssue,
    setIsCreatingIssue, selectedProjectId, setSelectedProjectId, issueTitle,
    setIssueTitle, issueDescription, setIssueDescription, issueType, setIssueType,
    issuePriority, setIssuePriority, issueStoryPoints, setIssueStoryPoints,
    issueWorkflowStatusId, setIssueWorkflowStatusId, issueAssigneeId,
    setIssueAssigneeId,
    issueDueDate, setIssueDueDate, newIssueLabelIds, setNewIssueLabelIds,
    issueFilterQuery, setIssueFilterQuery, issueSort, setIssueSort,
    issueFilterProjectId, setIssueFilterProjectId, issueFilterSprintId,
    setIssueFilterSprintId, issueFilterStatus, setIssueFilterStatus,
    issueFilterWorkflowStatusId, setIssueFilterWorkflowStatusId,
    issueFilterPriority, setIssueFilterPriority, issueFilterAssigneeId,
    setIssueFilterAssigneeId, issueFilterLabelId, setIssueFilterLabelId,
    issueFilterDue, setIssueFilterDue, savedFilters, setSavedFilters,
    savedFiltersError, setSavedFiltersError, savedFilterFormError,
    setSavedFilterFormError, savedFilterName, setSavedFilterName,
    isLoadingSavedFilters, setIsLoadingSavedFilters, isCreatingSavedFilter,
    setIsCreatingSavedFilter, updatingSavedFilterIds, setUpdatingSavedFilterIds,
    deletingSavedFilterIds, setDeletingSavedFilterIds, renameSavedFilterId,
    setRenameSavedFilterId, renameSavedFilterName, setRenameSavedFilterName,
    transitioningIssueIds, setTransitioningIssueIds, assigningIssueIds,
    setAssigningIssueIds, labelingIssueIds, setLabelingIssueIds,
    archivingIssueIds, setArchivingIssueIds, selectedIssue, setSelectedIssue,
    selectedIssueError, setSelectedIssueError, isLoadingSelectedIssue,
    setIsLoadingSelectedIssue, isEditingIssueDetails, setIsEditingIssueDetails,
    isUpdatingIssue, setIsUpdatingIssue, editIssueTitle, setEditIssueTitle,
    editIssueDescription, setEditIssueDescription, editIssueType,
    setEditIssueType, editIssuePriority, setEditIssuePriority,
    editIssueStoryPoints, setEditIssueStoryPoints, editIssueDueDate,
    setEditIssueDueDate, issueComments, setIssueComments, commentsError,
    setCommentsError, commentBody, setCommentBody, isLoadingComments,
    setIsLoadingComments, isCreatingComment, setIsCreatingComment,
    editingCommentId, setEditingCommentId, editCommentBody, setEditCommentBody,
    updatingCommentIds, setUpdatingCommentIds, deletingCommentIds,
    setDeletingCommentIds, issueActivity, setIssueActivity, activityError,
    setActivityError, isLoadingActivity, setIsLoadingActivity, issueChildren,
    setIssueChildren, hierarchyError, setHierarchyError, subtaskFormError,
    setSubtaskFormError, isLoadingIssueChildren, setIsLoadingIssueChildren,
    isCreatingSubtask, setIsCreatingSubtask, subtaskTitle, setSubtaskTitle,
    subtaskPriority, setSubtaskPriority, subtaskStoryPoints,
    setSubtaskStoryPoints, subtaskWorkflowStatusId, setSubtaskWorkflowStatusId,
    issueLinks,
    setIssueLinks, linksError, setLinksError, linkFormError, setLinkFormError,
    isLoadingIssueLinks, setIsLoadingIssueLinks, isCreatingIssueLink,
    setIsCreatingIssueLink, deletingIssueLinkIds, setDeletingIssueLinkIds,
    linkTargetIssueId, setLinkTargetIssueId, linkType, setLinkType,
  };
}
