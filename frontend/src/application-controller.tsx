import { DragEvent, FormEvent, useEffect } from "react";
import "./styles.css";
import "@fontsource-variable/space-grotesk";
import "@fontsource-variable/manrope";
import "./styles/index.css";
import {
  ApiError,
  AppNotification,
  AutomationRule,
  API_UNAUTHORIZED_EVENT,
  CurrentUser,
  CreateWorkflowStatusInput,
  CreateAutomationRuleInput,
  EmailDiagnostics,
  TeamInvite,
  Issue,
  IssueActivity,
  IssueComment,
  IssueDueFilter,
  IssueLink,
  IssueLinkType,
  IssuePriority,
  IssueSort,
  IssueStatus,
  IssueType,
  Label,
  Project,
  ProjectMember,
  ProjectRole,
  ProjectWorkflow,
  ProjectWorkflowStatus,
  RuntimeVersion,
  SavedFilter,
  Sprint,
  SprintStatus,
  TeamMember,
  TransitionIssueInput,
  UpdateWorkflowStatusInput,
  WorkflowTransitionInput,
  addIssueToSprint,
  archiveIssue,
  archiveProject,
  archiveWorkflowStatus,
  assignIssue,
  changePassword,
  completeSprint,
  createLabel,
  createIssue,
  createIssueComment,
  createIssueLink,
  createProject,
  createSavedFilter,
  createSprint,
  createSubtask,
  createTeamInvite,
  createTeamMember,
  createWorkflowStatus,
  createAutomationRule,
  deleteLabel,
  deleteIssueComment,
  deleteIssueLink,
  deleteProjectMember,
  deleteSavedFilter,
  deleteAutomationRule,
  getIssue,
  getCurrentUser,
  getEmailDiagnostics,
  getProject,
  getProjectWorkflow,
  getRuntimeVersion,
  getSprint,
  getUnreadNotificationsCount,
  listIssueActivity,
  listIssueComments,
  listIssueChildren,
  listIssueLinks,
  listAllIssues,
  listIssues,
  listLabels,
  listNotifications,
  listProjectMembers,
  listAutomationRules,
  listProjects,
  listSavedFilters,
  listSprints,
  listTeamInvites,
  listTeamMembers,
  login,
  logout,
  markAllNotificationsRead,
  markNotificationRead,
  putProjectMember,
  reorderWorkflowStatuses,
  reorderAutomationRules,
  replaceWorkflowTransitions,
  resetTeamMemberPassword,
  revokeTeamInvite,
  resendTeamInvite,
  removeIssueFromSprint,
  setIssueLabels,
  startSprint,
  transitionIssue,
  updateProfile,
  updateProject,
  updateSavedFilter,
  updateIssueComment,
  updateTeamMember,
  updateWorkflowStatus,
  updateAutomationRule,
  updateIssue,
  updateSprint,
} from "./lib/api";
import {
  hasMinTrimmedLength,
  hasText,
  isValidEmail,
  isValidLabelColor,
  isValidUsername,
  normalizeEmail,
  normalizeLabelColor,
  normalizeText,
  normalizeUsername,
} from "./lib/validation";
import {
  issueDueInfo,
  isIssueDone,
  issueLabelIds,
  issueMatchesFilters,
  savedIssueFilterStateFromFilters,
  savedIssueFiltersFromState,
  startOfToday,
} from "./lib/issue-model";
import { sprintMatchesFilters } from "./lib/sprint-model";
import {
  appSectionPath,
  appSections,
  boardPath,
  currentForgotPasswordRouteFromLocation,
  currentBoardProjectIdFromLocation,
  currentInviteAcceptTokenFromLocation,
  currentPasswordResetTokenFromLocation,
  currentAppSectionFromLocation,
  sprintIdFromPath,
  type AppSection,
} from "./lib/routing";
import {
  buildInviteAcceptURL,
  normalizedInviteEmail,
  validateInviteEmail,
} from "./lib/invite-view";
import { AppSidebar, WorkspaceTopbar } from "./components/app-shell";
import { AccountSection } from "./features/account/account-section";
import { InviteAcceptScreen } from "./features/auth/invite-accept-screen";
import { BootingScreen, SignInScreen } from "./features/auth/auth-screens";
import {
  ForgotPasswordScreen,
  ResetPasswordScreen,
} from "./features/auth/password-reset-screens";
import { BoardSection } from "./features/board/board-section";
import { DashboardSection } from "./features/dashboard/dashboard-section";
import { IssueCreateForm } from "./features/issues/issue-create-form";
import { IssueDetailSection } from "./features/issues/issue-detail-section";
import { IssueListPanel } from "./features/issues/issue-list-panel";
import { SavedFiltersPanel } from "./features/issues/saved-filters-panel";
import { LabelsSection } from "./features/labels/labels-section";
import { NotificationsSection } from "./features/notifications/notifications-section";
import { ProjectsSection } from "./features/projects/projects-section";
import { SprintsSection } from "./features/sprints/sprints-section";
import { TeamSection } from "./features/team/team-section";
import { useSessionAccountController } from "./controllers/use-session-account-controller";
import { useWorkspaceAdminController } from "./controllers/use-workspace-admin-controller";
import { useIssuesController } from "./controllers/use-issues-controller";
import { useNotificationsController } from "./controllers/use-notifications-controller";
import { useSprintsController } from "./controllers/use-sprints-controller";
import { useWorkflowsController } from "./controllers/use-workflows-controller";
import { useBoardController } from "./controllers/use-board-controller";
import { useAutomationsController } from "./controllers/use-automations-controller";
import {
  activeWorkflowStatuses,
  allowedTransitionStatuses,
  defaultWorkflowStatus,
} from "./lib/workflow-model";

function apiErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback;
}

function formString(formData: FormData, name: string, fallback: string) {
  const value = formData.get(name);
  return typeof value === "string" ? value : fallback;
}

function currentSprintIdFromLocation(
  location: Pick<Location, "pathname"> | undefined = undefined,
) {
  if (location) {
    return sprintIdFromPath(location.pathname);
  }

  if (typeof window === "undefined") {
    return "";
  }

  return sprintIdFromPath(window.location.pathname);
}

const issueFilterSprintStatusRank: Record<SprintStatus, number> = {
  active: 0,
  planned: 1,
  completed: 2,
};

function sortIssueFilterSprints(sprints: Sprint[]) {
  return [...sprints].sort((left, right) => {
    const statusRank =
      issueFilterSprintStatusRank[left.status] -
      issueFilterSprintStatusRank[right.status];
    if (statusRank !== 0) {
      return statusRank;
    }

    const projectRank = left.project_key.localeCompare(right.project_key);
    if (projectRank !== 0) {
      return projectRank;
    }

    return left.name.localeCompare(right.name);
  });
}

function upsertIssueFilterSprint(sprints: Sprint[], sprint: Sprint) {
  const nextSprints = sprints.some((currentSprint) => currentSprint.id === sprint.id)
    ? sprints.map((currentSprint) =>
        currentSprint.id === sprint.id ? sprint : currentSprint,
      )
    : [...sprints, sprint];

  return sortIssueFilterSprints(nextSprints);
}

function parseStoryPoints(value: string) {
  const normalizedValue = value.trim();
  if (normalizedValue === "") {
    return 0;
  }

  const points = Number(normalizedValue);
  return Number.isInteger(points) && points >= 0 && points <= 100 ? points : null;
}

function upsertSavedFilter(
  savedFilters: SavedFilter[],
  savedFilter: SavedFilter,
) {
  return [
    savedFilter,
    ...savedFilters.filter(
      (currentSavedFilter) => currentSavedFilter.id !== savedFilter.id,
    ),
  ];
}

export function ApplicationController() {
  const {
    user,
    setUser,
    loginValue,
    setLoginValue,
    password,
    setPassword,
    error,
    setError,
    isBooting,
    setIsBooting,
    isSubmitting,
    setIsSubmitting,
    isLoggingOut,
    setIsLoggingOut,
    activeSection,
    setActiveSection,
    routeSprintId,
    setRouteSprintId,
    inviteAcceptToken,
    setInviteAcceptToken,
    isForgotPasswordRoute,
    setIsForgotPasswordRoute,
    passwordResetToken,
    setPasswordResetToken,
    accountError,
    setAccountError,
    accountSuccess,
    setAccountSuccess,
    accountDisplayName,
    setAccountDisplayName,
    runtimeVersion,
    setRuntimeVersion,
    runtimeVersionError,
    setRuntimeVersionError,
    isLoadingRuntimeVersion,
    setIsLoadingRuntimeVersion,
    isUpdatingProfile,
    setIsUpdatingProfile,
    currentPassword,
    setCurrentPassword,
    newPassword,
    setNewPassword,
    confirmNewPassword,
    setConfirmNewPassword,
    isChangingPassword,
    setIsChangingPassword,
  } = useSessionAccountController({
    initialSection: currentAppSectionFromLocation(),
    initialSprintId: currentSprintIdFromLocation(),
    initialInviteAcceptToken: currentInviteAcceptTokenFromLocation(),
    initialForgotPasswordRoute: currentForgotPasswordRouteFromLocation(),
    initialPasswordResetToken: currentPasswordResetTokenFromLocation(),
  });
  const {
    projects,
    setProjects,
    projectsError,
    setProjectsError,
    projectFormError,
    setProjectFormError,
    isLoadingProjects,
    setIsLoadingProjects,
    isCreatingProject,
    setIsCreatingProject,
    archivingProjectIds,
    setArchivingProjectIds,
    editingProjectId,
    setEditingProjectId,
    editProjectName,
    setEditProjectName,
    editProjectDescription,
    setEditProjectDescription,
    updatingProjectIds,
    setUpdatingProjectIds,
    selectedProjectDetail,
    setSelectedProjectDetail,
    projectDetailError,
    setProjectDetailError,
    isLoadingProjectDetail,
    setIsLoadingProjectDetail,
    projectDetailTab,
    setProjectDetailTab,
    projectMembers,
    setProjectMembers,
    projectMembersError,
    setProjectMembersError,
    isLoadingProjectMembers,
    setIsLoadingProjectMembers,
    selectedProjectMemberUserId,
    setSelectedProjectMemberUserId,
    selectedProjectMemberRole,
    setSelectedProjectMemberRole,
    updatingProjectMemberIds,
    setUpdatingProjectMemberIds,
    removingProjectMemberIds,
    setRemovingProjectMemberIds,
    projectKey,
    setProjectKey,
    projectName,
    setProjectName,
    projectDescription,
    setProjectDescription,
    teamMembers,
    setTeamMembers,
    teamMembersError,
    setTeamMembersError,
    teamMemberFormError,
    setTeamMemberFormError,
    isLoadingTeamMembers,
    setIsLoadingTeamMembers,
    isCreatingTeamMember,
    setIsCreatingTeamMember,
    teamMemberEmail,
    setTeamMemberEmail,
    teamMemberUsername,
    setTeamMemberUsername,
    teamMemberDisplayName,
    setTeamMemberDisplayName,
    teamMemberPassword,
    setTeamMemberPassword,
    teamMemberRole,
    setTeamMemberRole,
    teamInvites,
    setTeamInvites,
    teamInvitesError,
    setTeamInvitesError,
    teamInviteFormError,
    setTeamInviteFormError,
    isLoadingTeamInvites,
    setIsLoadingTeamInvites,
    isCreatingTeamInvite,
    setIsCreatingTeamInvite,
    teamInviteEmail,
    setTeamInviteEmail,
    teamInviteRole,
    setTeamInviteRole,
    teamInviteLinksById,
    setTeamInviteLinksById,
    copiedTeamInviteId,
    setCopiedTeamInviteId,
    revokingTeamInviteIds,
    setRevokingTeamInviteIds,
    resendingTeamInviteIds,
    setResendingTeamInviteIds,
    emailDiagnostics,
    setEmailDiagnostics,
    emailDiagnosticsError,
    setEmailDiagnosticsError,
    isLoadingEmailDiagnostics,
    setIsLoadingEmailDiagnostics,
    updatingTeamMemberIds,
    setUpdatingTeamMemberIds,
    passwordResetMemberId,
    setPasswordResetMemberId,
    teamMemberResetPassword,
    setTeamMemberResetPassword,
    resettingTeamMemberPasswordIds,
    setResettingTeamMemberPasswordIds,
    labels,
    setLabels,
    labelsError,
    setLabelsError,
    isLoadingLabels,
    setIsLoadingLabels,
    labelName,
    setLabelName,
    labelColor,
    setLabelColor,
    isCreatingLabel,
    setIsCreatingLabel,
    deletingLabelIds,
    setDeletingLabelIds,
  } = useWorkspaceAdminController();
  const {
    issues,
    setIssues,
    issuesError,
    setIssuesError,
    issueFormError,
    setIssueFormError,
    isLoadingIssues,
    setIsLoadingIssues,
    isCreatingIssue,
    setIsCreatingIssue,
    selectedProjectId,
    setSelectedProjectId,
    issueTitle,
    setIssueTitle,
    issueDescription,
    setIssueDescription,
    issueType,
    setIssueType,
    issuePriority,
    setIssuePriority,
    issueStoryPoints,
    setIssueStoryPoints,
    issueWorkflowStatusId,
    setIssueWorkflowStatusId,
    issueAssigneeId,
    setIssueAssigneeId,
    issueDueDate,
    setIssueDueDate,
    newIssueLabelIds,
    setNewIssueLabelIds,
    issueFilterQuery,
    setIssueFilterQuery,
    issueSort,
    setIssueSort,
    issueFilterProjectId,
    setIssueFilterProjectId,
    issueFilterSprintId,
    setIssueFilterSprintId,
    issueFilterStatus,
    setIssueFilterStatus,
    issueFilterWorkflowStatusId,
    setIssueFilterWorkflowStatusId,
    issueFilterPriority,
    setIssueFilterPriority,
    issueFilterAssigneeId,
    setIssueFilterAssigneeId,
    issueFilterLabelId,
    setIssueFilterLabelId,
    issueFilterDue,
    setIssueFilterDue,
    savedFilters,
    setSavedFilters,
    savedFiltersError,
    setSavedFiltersError,
    savedFilterFormError,
    setSavedFilterFormError,
    savedFilterName,
    setSavedFilterName,
    isLoadingSavedFilters,
    setIsLoadingSavedFilters,
    isCreatingSavedFilter,
    setIsCreatingSavedFilter,
    updatingSavedFilterIds,
    setUpdatingSavedFilterIds,
    deletingSavedFilterIds,
    setDeletingSavedFilterIds,
    renameSavedFilterId,
    setRenameSavedFilterId,
    renameSavedFilterName,
    setRenameSavedFilterName,
    transitioningIssueIds,
    setTransitioningIssueIds,
    assigningIssueIds,
    setAssigningIssueIds,
    labelingIssueIds,
    setLabelingIssueIds,
    archivingIssueIds,
    setArchivingIssueIds,
    selectedIssue,
    setSelectedIssue,
    selectedIssueError,
    setSelectedIssueError,
    isLoadingSelectedIssue,
    setIsLoadingSelectedIssue,
    isEditingIssueDetails,
    setIsEditingIssueDetails,
    isUpdatingIssue,
    setIsUpdatingIssue,
    editIssueTitle,
    setEditIssueTitle,
    editIssueDescription,
    setEditIssueDescription,
    editIssueType,
    setEditIssueType,
    editIssuePriority,
    setEditIssuePriority,
    editIssueStoryPoints,
    setEditIssueStoryPoints,
    editIssueDueDate,
    setEditIssueDueDate,
    issueComments,
    setIssueComments,
    commentsError,
    setCommentsError,
    commentBody,
    setCommentBody,
    isLoadingComments,
    setIsLoadingComments,
    isCreatingComment,
    setIsCreatingComment,
    editingCommentId,
    setEditingCommentId,
    editCommentBody,
    setEditCommentBody,
    updatingCommentIds,
    setUpdatingCommentIds,
    deletingCommentIds,
    setDeletingCommentIds,
    issueActivity,
    setIssueActivity,
    activityError,
    setActivityError,
    isLoadingActivity,
    setIsLoadingActivity,
    issueChildren,
    setIssueChildren,
    hierarchyError,
    setHierarchyError,
    subtaskFormError,
    setSubtaskFormError,
    isLoadingIssueChildren,
    setIsLoadingIssueChildren,
    isCreatingSubtask,
    setIsCreatingSubtask,
    subtaskTitle,
    setSubtaskTitle,
    subtaskPriority,
    setSubtaskPriority,
    subtaskStoryPoints,
    setSubtaskStoryPoints,
    subtaskWorkflowStatusId,
    setSubtaskWorkflowStatusId,
    issueLinks,
    setIssueLinks,
    linksError,
    setLinksError,
    linkFormError,
    setLinkFormError,
    isLoadingIssueLinks,
    setIsLoadingIssueLinks,
    isCreatingIssueLink,
    setIsCreatingIssueLink,
    deletingIssueLinkIds,
    setDeletingIssueLinkIds,
    linkTargetIssueId,
    setLinkTargetIssueId,
    linkType,
    setLinkType,
  } = useIssuesController();
  const {
    workflowsByProjectId,
    setWorkflowsByProjectId,
    loadingWorkflowProjectIds,
    setLoadingWorkflowProjectIds,
    workflowErrorsByProjectId,
    setWorkflowErrorsByProjectId,
    workflowMutationError,
    setWorkflowMutationError,
    creatingWorkflowStatus,
    setCreatingWorkflowStatus,
    updatingWorkflowStatusIds,
    setUpdatingWorkflowStatusIds,
    archivingWorkflowStatusIds,
    setArchivingWorkflowStatusIds,
    isReorderingWorkflowStatuses,
    setIsReorderingWorkflowStatuses,
    isSavingWorkflowTransitions,
    setIsSavingWorkflowTransitions,
  } = useWorkflowsController();
  const {
    automationRules,
    setAutomationRules,
    automationRulesError,
    setAutomationRulesError,
    isLoadingAutomationRules,
    setIsLoadingAutomationRules,
    isCreatingAutomationRule,
    setIsCreatingAutomationRule,
    updatingAutomationRuleIds,
    setUpdatingAutomationRuleIds,
    deletingAutomationRuleIds,
    setDeletingAutomationRuleIds,
    isReorderingAutomationRules,
    setIsReorderingAutomationRules,
  } = useAutomationsController();
  const {
    boardProjectId,
    setBoardProjectId,
    boardIssues,
    setBoardIssues,
    boardError,
    setBoardError,
    isLoadingBoard,
    setIsLoadingBoard,
  } = useBoardController(currentBoardProjectIdFromLocation());
  const {
    sprints,
    setSprints,
    issueFilterSprints,
    setIssueFilterSprints,
    dashboardSprintIssues,
    setDashboardSprintIssues,
    dashboardSprintError,
    setDashboardSprintError,
    isLoadingDashboardSprint,
    setIsLoadingDashboardSprint,
    sprintsError,
    setSprintsError,
    sprintFormError,
    setSprintFormError,
    isLoadingSprints,
    setIsLoadingSprints,
    isCreatingSprint,
    setIsCreatingSprint,
    sprintProjectId,
    setSprintProjectId,
    sprintName,
    setSprintName,
    sprintGoal,
    setSprintGoal,
    sprintStartDate,
    setSprintStartDate,
    sprintEndDate,
    setSprintEndDate,
    sprintFilterProjectId,
    setSprintFilterProjectId,
    sprintFilterStatus,
    setSprintFilterStatus,
    selectedSprint,
    setSelectedSprint,
    selectedSprintError,
    setSelectedSprintError,
    isLoadingSelectedSprint,
    setIsLoadingSelectedSprint,
    isEditingSprintDetails,
    setIsEditingSprintDetails,
    isUpdatingSprint,
    setIsUpdatingSprint,
    editSprintName,
    setEditSprintName,
    editSprintGoal,
    setEditSprintGoal,
    editSprintStartDate,
    setEditSprintStartDate,
    editSprintEndDate,
    setEditSprintEndDate,
    startingSprintIds,
    setStartingSprintIds,
    completingSprintIds,
    setCompletingSprintIds,
    sprintPlanningIssues,
    setSprintPlanningIssues,
    sprintPlanningError,
    setSprintPlanningError,
    isLoadingSprintPlanning,
    setIsLoadingSprintPlanning,
    addingIssueToSprintIds,
    setAddingIssueToSprintIds,
    removingIssueFromSprintIds,
    setRemovingIssueFromSprintIds,
  } = useSprintsController();
  const {
    notifications,
    setNotifications,
    notificationsError,
    setNotificationsError,
    isLoadingNotifications,
    setIsLoadingNotifications,
    unreadNotificationsCount,
    setUnreadNotificationsCount,
    isNotificationsOpen,
    setIsNotificationsOpen,
  } = useNotificationsController();
  const selectedIssueId = selectedIssue?.id ?? "";

  function navigateToSection(
    section: AppSection,
    mode: "push" | "replace" = "push",
  ) {
    setActiveSection(section);
    setInviteAcceptToken(null);
    setIsForgotPasswordRoute(false);
    setPasswordResetToken(null);
    if (section !== "sprints") {
      setRouteSprintId("");
    }

    if (typeof window === "undefined") {
      return;
    }

    const nextPath =
      section === "board" ? boardPath(boardProjectId) : appSectionPath(section);
    if (
      `${window.location.pathname}${window.location.search}` === nextPath &&
      window.location.hash === ""
    ) {
      return;
    }

    if (mode === "replace") {
      window.history.replaceState({ section }, "", nextPath);
      return;
    }

    window.history.pushState({ section }, "", nextPath);
  }

  function navigateToBoard(
    projectId: string,
    mode: "push" | "replace" = "push",
  ) {
    setActiveSection("board");
    setBoardProjectId(projectId);
    setRouteSprintId("");
    setInviteAcceptToken(null);
    setIsForgotPasswordRoute(false);
    setPasswordResetToken(null);

    if (typeof window === "undefined") {
      return;
    }

    const nextPath = boardPath(projectId);
    if (
      `${window.location.pathname}${window.location.search}` === nextPath &&
      window.location.hash === ""
    ) {
      return;
    }

    if (mode === "replace") {
      window.history.replaceState({ section: "board", projectId }, "", nextPath);
      return;
    }

    window.history.pushState({ section: "board", projectId }, "", nextPath);
  }

  function navigateToSprint(
    sprintId: string,
    mode: "push" | "replace" = "push",
  ) {
    setActiveSection("sprints");
    setRouteSprintId(sprintId);
    setInviteAcceptToken(null);
    setIsForgotPasswordRoute(false);
    setPasswordResetToken(null);

    if (typeof window === "undefined") {
      return;
    }

    const nextPath = `/sprints/${encodeURIComponent(sprintId)}`;
    if (
      window.location.pathname === nextPath &&
      window.location.search === "" &&
      window.location.hash === ""
    ) {
      return;
    }

    if (mode === "replace") {
      window.history.replaceState({ section: "sprints", sprintId }, "", nextPath);
      return;
    }

    window.history.pushState({ section: "sprints", sprintId }, "", nextPath);
  }

  function navigateToForgotPassword() {
    setActiveSection("dashboard");
    setRouteSprintId("");
    setBoardProjectId("");
    setInviteAcceptToken(null);
    setIsForgotPasswordRoute(true);
    setPasswordResetToken(null);
    setError("");

    if (typeof window === "undefined") {
      return;
    }

    if (window.location.pathname === "/forgot-password" && window.location.search === "") {
      return;
    }

    window.history.pushState({ authRoute: "forgot-password" }, "", "/forgot-password");
  }

  useEffect(() => {
    let isMounted = true;

    getCurrentUser()
      .then((response) => {
        if (isMounted) {
          setUser(response.user);
        }
      })
      .catch((err: unknown) => {
        if (err instanceof ApiError && err.status === 401) {
          return;
        }

        if (isMounted) {
          setError("Backend is not ready. Run make setup-db and make dev.");
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsBooting(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, []);

  useEffect(() => {
    function handleRouteChange() {
      setActiveSection(currentAppSectionFromLocation());
      setRouteSprintId(currentSprintIdFromLocation());
      setBoardProjectId(currentBoardProjectIdFromLocation());
      setInviteAcceptToken(currentInviteAcceptTokenFromLocation());
      setIsForgotPasswordRoute(currentForgotPasswordRouteFromLocation());
      setPasswordResetToken(currentPasswordResetTokenFromLocation());
    }

    window.addEventListener("popstate", handleRouteChange);
    return () => {
      window.removeEventListener("popstate", handleRouteChange);
    };
  }, []);

  function handlePublicAuthSignIn() {
    if (user) {
      void handleLogout();
      return;
    }

    setError("");
    setLoginValue("");
    setPassword("");
    navigateToSection("dashboard", "replace");
  }

  function handlePasswordResetCompleted() {
    setUser(null);
    setError("");
    setLoginValue("");
    setPassword("");
    setIsSubmitting(false);
  }

  useEffect(() => {
    function handleUnauthorized() {
      resetLocalSession("Session expired. Sign in again.");
    }

    window.addEventListener(API_UNAUTHORIZED_EVENT, handleUnauthorized);
    return () => {
      window.removeEventListener(API_UNAUTHORIZED_EVENT, handleUnauthorized);
    };
  }, []);

  useEffect(() => {
    if (!user) {
      setAccountDisplayName("");
      return;
    }

    setAccountDisplayName(user.display_name);
  }, [user]);

  useEffect(() => {
    if (!user) {
      setRuntimeVersion(null);
      setRuntimeVersionError("");
      setIsLoadingRuntimeVersion(false);
      return;
    }

    let isMounted = true;
    setRuntimeVersionError("");
    setIsLoadingRuntimeVersion(true);

    getRuntimeVersion()
      .then((response) => {
        if (isMounted) {
          setRuntimeVersion(response);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setRuntimeVersion(null);
          setRuntimeVersionError(
            apiErrorMessage(err, "Could not load deployment metadata."),
          );
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingRuntimeVersion(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user]);

  useEffect(() => {
    if (!user) {
      setProjects([]);
      setSelectedProjectDetail(null);
      return;
    }

    let isMounted = true;
    setProjectsError("");
    setProjectFormError("");
    setEditingProjectId("");
    setEditProjectName("");
    setEditProjectDescription("");
    setUpdatingProjectIds([]);
    setArchivingProjectIds([]);
    setProjectDetailError("");
    setIsLoadingProjects(true);

    listProjects()
      .then((response) => {
        if (isMounted) {
          setProjects(response.projects);
          setSelectedProjectDetail((currentProject) => {
            if (!response.projects.length) {
              return null;
            }

            const matchingProject = response.projects.find(
              (project) => project.id === currentProject?.id,
            );
            return matchingProject ?? response.projects[0];
          });
          setSelectedProjectId((currentProjectId) => {
            if (
              currentProjectId &&
              response.projects.some(
                (project) => project.id === currentProjectId && project.can_write,
              )
            ) {
              return currentProjectId;
            }
            return response.projects.find((project) => project.can_write)?.id ?? "";
          });
          setIssueFilterProjectId((currentProjectId) =>
            currentProjectId &&
            !response.projects.some((project) => project.id === currentProjectId)
              ? ""
              : currentProjectId,
          );
          setSprintProjectId((currentProjectId) => {
            if (
              currentProjectId &&
              response.projects.some((project) => project.id === currentProjectId)
            ) {
              return currentProjectId;
            }
            return response.projects[0]?.id ?? "";
          });
          setSprintFilterProjectId((currentProjectId) =>
            currentProjectId &&
            !response.projects.some((project) => project.id === currentProjectId)
              ? ""
              : currentProjectId,
          );
        }
      })
      .catch((err) => {
        if (isMounted) {
          setProjectsError(apiErrorMessage(err, "Could not load projects."));
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingProjects(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user]);

  useEffect(() => {
    if (!user) {
      setWorkflowsByProjectId((currentWorkflows) =>
        Object.keys(currentWorkflows).length > 0 ? {} : currentWorkflows,
      );
      setLoadingWorkflowProjectIds((currentIds) =>
        currentIds.length > 0 ? [] : currentIds,
      );
      setWorkflowErrorsByProjectId((currentErrors) =>
        Object.keys(currentErrors).length > 0 ? {} : currentErrors,
      );
      return;
    }

    const projectIds = Array.from(
      new Set(
        [
          selectedProjectId,
          issueFilterProjectId,
          selectedIssue?.project_id,
          boardProjectId,
          selectedSprint?.project_id,
        ].filter((projectId): projectId is string => Boolean(projectId)),
      ),
    );

    for (const projectId of projectIds) {
      if (
        workflowsByProjectId[projectId] ||
        workflowErrorsByProjectId[projectId] ||
        loadingWorkflowProjectIds.includes(projectId)
      ) {
        continue;
      }

      setLoadingWorkflowProjectIds((currentIds) =>
        currentIds.includes(projectId) ? currentIds : [...currentIds, projectId],
      );
      setWorkflowErrorsByProjectId((currentErrors) => ({
        ...currentErrors,
        [projectId]: "",
      }));

      getProjectWorkflow(projectId)
        .then((workflow) => {
          setWorkflowsByProjectId((currentWorkflows) => ({
            ...currentWorkflows,
            [projectId]: workflow,
          }));
        })
        .catch((err) => {
          setWorkflowErrorsByProjectId((currentErrors) => ({
            ...currentErrors,
            [projectId]: apiErrorMessage(err, "Could not load project workflow."),
          }));
        })
        .finally(() => {
          setLoadingWorkflowProjectIds((currentIds) =>
            currentIds.filter((currentId) => currentId !== projectId),
          );
        });
    }
  }, [
    user,
    selectedProjectId,
    issueFilterProjectId,
    selectedIssue?.project_id,
    boardProjectId,
    selectedSprint?.project_id,
    workflowsByProjectId,
    workflowErrorsByProjectId,
    loadingWorkflowProjectIds,
    setLoadingWorkflowProjectIds,
    setWorkflowErrorsByProjectId,
    setWorkflowsByProjectId,
  ]);

  useEffect(() => {
    if (
      !user ||
      isLoadingProjects ||
      !boardProjectId ||
      projects.some((project) => project.id === boardProjectId)
    ) {
      return;
    }

    setBoardIssues([]);
    setBoardError("");
    navigateToBoard("", "replace");
  }, [user, isLoadingProjects, projects, boardProjectId]);

  useEffect(() => {
    if (!user || activeSection !== "board" || !boardProjectId) {
      setBoardIssues([]);
      setBoardError("");
      setIsLoadingBoard(false);
      return;
    }

    let isMounted = true;
    setBoardError("");
    setIsLoadingBoard(true);

    listAllIssues({ projectId: boardProjectId, sort: "created_desc" })
      .then((projectIssues) => {
        if (isMounted) {
          setBoardIssues(projectIssues);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setBoardError(apiErrorMessage(err, "Could not load project board."));
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingBoard(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user, activeSection, boardProjectId]);

  useEffect(() => {
    const workflow = workflowsByProjectId[selectedProjectId];
    const activeStatuses = activeWorkflowStatuses(workflow);
    if (
      issueWorkflowStatusId &&
      activeStatuses.some((status) => status.id === issueWorkflowStatusId)
    ) {
      return;
    }
    setIssueWorkflowStatusId(defaultWorkflowStatus(workflow)?.id ?? "");
  }, [issueWorkflowStatusId, selectedProjectId, workflowsByProjectId]);

  useEffect(() => {
    const workflow = selectedIssue
      ? workflowsByProjectId[selectedIssue.project_id]
      : undefined;
    const activeStatuses = activeWorkflowStatuses(workflow);
    if (
      subtaskWorkflowStatusId &&
      activeStatuses.some((status) => status.id === subtaskWorkflowStatusId)
    ) {
      return;
    }
    setSubtaskWorkflowStatusId(defaultWorkflowStatus(workflow)?.id ?? "");
  }, [selectedIssue?.project_id, subtaskWorkflowStatusId, workflowsByProjectId]);

  useEffect(() => {
    setProjectDetailTab("summary");
    setProjectMembers([]);
    setProjectMembersError("");
    setIsLoadingProjectMembers(false);
    setSelectedProjectMemberUserId("");
    setSelectedProjectMemberRole("contributor");
    setUpdatingProjectMemberIds([]);
    setRemovingProjectMemberIds([]);
    setWorkflowMutationError("");
    setCreatingWorkflowStatus(false);
    setUpdatingWorkflowStatusIds([]);
    setArchivingWorkflowStatusIds([]);
    setIsReorderingWorkflowStatuses(false);
    setIsSavingWorkflowTransitions(false);
    setAutomationRules([]);
    setAutomationRulesError("");
    setIsLoadingAutomationRules(false);
    setIsCreatingAutomationRule(false);
    setUpdatingAutomationRuleIds([]);
    setDeletingAutomationRuleIds([]);
    setIsReorderingAutomationRules(false);
  }, [selectedProjectDetail?.id]);

  useEffect(() => {
    if (selectedProjectDetail?.can_manage) {
      return;
    }
    setProjectDetailTab("summary");
    setProjectMembers([]);
    setProjectMembersError("");
    setIsLoadingProjectMembers(false);
    setSelectedProjectMemberUserId("");
    setSelectedProjectMemberRole("contributor");
    setUpdatingProjectMemberIds([]);
    setRemovingProjectMemberIds([]);
    setWorkflowMutationError("");
    setCreatingWorkflowStatus(false);
    setUpdatingWorkflowStatusIds([]);
    setArchivingWorkflowStatusIds([]);
    setIsReorderingWorkflowStatuses(false);
    setIsSavingWorkflowTransitions(false);
    setAutomationRules([]);
    setAutomationRulesError("");
    setIsLoadingAutomationRules(false);
    setIsCreatingAutomationRule(false);
    setUpdatingAutomationRuleIds([]);
    setDeletingAutomationRuleIds([]);
    setIsReorderingAutomationRules(false);
  }, [selectedProjectDetail?.can_manage]);

  useEffect(() => {
    if (!user) {
      setTeamMembers([]);
      return;
    }

    let isMounted = true;
    setTeamMembersError("");
    setTeamMemberFormError("");
    setIsLoadingTeamMembers(true);

    listTeamMembers()
      .then((response) => {
        if (isMounted) {
          setTeamMembers(response.members);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setTeamMembersError(apiErrorMessage(err, "Could not load team members."));
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingTeamMembers(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user]);

  useEffect(() => {
    if (!user || user.workspace.role !== "admin") {
      setTeamInvites([]);
      setTeamInvitesError("");
      setTeamInviteFormError("");
      setIsLoadingTeamInvites(false);
      setIsCreatingTeamInvite(false);
      setTeamInviteEmail("");
      setTeamInviteRole("member");
      setTeamInviteLinksById({});
      setCopiedTeamInviteId("");
      setRevokingTeamInviteIds([]);
      setResendingTeamInviteIds([]);
      setEmailDiagnostics(null);
      setEmailDiagnosticsError("");
      setIsLoadingEmailDiagnostics(false);
      return;
    }

    let isMounted = true;
    setTeamInvitesError("");
    setTeamInviteFormError("");
    setIsLoadingTeamInvites(true);
    setEmailDiagnosticsError("");
    setIsLoadingEmailDiagnostics(true);

    listTeamInvites()
      .then((response) => {
        if (isMounted) {
          setTeamInvites(response.invites);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setTeamInvitesError(apiErrorMessage(err, "Could not load team invites."));
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingTeamInvites(false);
        }
      });

    getEmailDiagnostics()
      .then((diagnostics) => {
        if (isMounted) {
          setEmailDiagnostics(diagnostics);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setEmailDiagnosticsError(
            apiErrorMessage(err, "Could not load email diagnostics."),
          );
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingEmailDiagnostics(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user]);

  useEffect(() => {
    if (!user) {
      setLabels([]);
      return;
    }

    let isMounted = true;
    setLabelsError("");
    setIsLoadingLabels(true);

    listLabels()
      .then((response) => {
        if (isMounted) {
          setLabels(response.labels);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setLabelsError(apiErrorMessage(err, "Could not load labels."));
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingLabels(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user]);

  useEffect(() => {
    if (!user) {
      setIssues([]);
      return;
    }

    let isMounted = true;
    setIssuesError("");
    setIssueFormError("");
    setIsLoadingIssues(true);

    listIssues({
      query: issueFilterQuery || undefined,
      sort: issueSort,
      projectId: issueFilterProjectId || undefined,
      sprintId: issueFilterSprintId || undefined,
      status: issueFilterStatus || undefined,
      workflowStatusId: issueFilterWorkflowStatusId || undefined,
      priority: issueFilterPriority || undefined,
      assigneeId: issueFilterAssigneeId || undefined,
      labelId: issueFilterLabelId || undefined,
      due: issueFilterDue || undefined,
    })
      .then((response) => {
        if (isMounted) {
          setIssues(response.issues);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setIssuesError(apiErrorMessage(err, "Could not load issues."));
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingIssues(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [
    user,
    issueFilterQuery,
    issueSort,
    issueFilterProjectId,
    issueFilterSprintId,
    issueFilterStatus,
    issueFilterWorkflowStatusId,
    issueFilterPriority,
    issueFilterAssigneeId,
    issueFilterLabelId,
    issueFilterDue,
  ]);

  useEffect(() => {
    if (!user) {
      setIssueFilterSprints([]);
      setIssueFilterSprintId("");
      return;
    }

    let isMounted = true;

    listSprints()
      .then((response) => {
        if (isMounted) {
          setIssueFilterSprints(sortIssueFilterSprints(response.sprints));
        }
      })
      .catch((err) => {
        if (isMounted) {
          setIssuesError(
            apiErrorMessage(err, "Could not load sprint filter options."),
          );
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user]);

  useEffect(() => {
    if (!user) {
      setSavedFilters([]);
      return;
    }

    let isMounted = true;
    setSavedFiltersError("");
    setSavedFilterFormError("");
    setIsLoadingSavedFilters(true);

    listSavedFilters()
      .then((response) => {
        if (isMounted) {
          setSavedFilters(response.saved_filters);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setSavedFiltersError(
            apiErrorMessage(err, "Could not load saved filters."),
          );
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingSavedFilters(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user]);

  useEffect(() => {
    if (!user) {
      setNotifications([]);
      setUnreadNotificationsCount(0);
      return;
    }

    let isMounted = true;
    setNotificationsError("");
    setIsLoadingNotifications(true);

    Promise.all([listNotifications(), getUnreadNotificationsCount()])
      .then(([notificationsResponse, unreadResponse]) => {
        if (isMounted) {
          setNotifications(notificationsResponse.notifications);
          setUnreadNotificationsCount(unreadResponse.unread_count);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setNotificationsError(
            apiErrorMessage(err, "Could not load notifications."),
          );
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingNotifications(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user]);

  useEffect(() => {
    if (!user) {
      return;
    }

    const intervalID = window.setInterval(() => {
      getUnreadNotificationsCount()
        .then((response) => {
          setUnreadNotificationsCount(response.unread_count);
        })
        .catch((err) => {
          setNotificationsError(
            apiErrorMessage(err, "Could not refresh notifications."),
          );
        });
    }, 30000);

    return () => {
      window.clearInterval(intervalID);
    };
  }, [user]);

  useEffect(() => {
    if (!user) {
      setSprints([]);
      setSelectedSprint(null);
      return;
    }

    let isMounted = true;
    setSprintsError("");
    setSprintFormError("");
    setIsLoadingSprints(true);

    listSprints({
      projectId: sprintFilterProjectId || undefined,
      status: sprintFilterStatus || undefined,
    })
      .then((response) => {
        if (isMounted) {
          setSprints(response.sprints);
          setSelectedSprint((currentSprint) => {
            if (!response.sprints.length) {
              return null;
            }

            const matchingSprint = response.sprints.find(
              (sprint) => sprint.id === currentSprint?.id,
            );
            return matchingSprint ?? response.sprints[0];
          });
        }
      })
      .catch((err) => {
        if (isMounted) {
          setSprintsError(apiErrorMessage(err, "Could not load sprints."));
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingSprints(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user, sprintFilterProjectId, sprintFilterStatus]);

  useEffect(() => {
    const activeSprint = issueFilterSprints.find(
      (sprint) => sprint.status === "active",
    );

    if (!user || !activeSprint) {
      setDashboardSprintIssues([]);
      setDashboardSprintError("");
      setIsLoadingDashboardSprint(false);
      return;
    }

    let isMounted = true;
    setDashboardSprintError("");
    setIsLoadingDashboardSprint(true);

    listIssues({ sprintId: activeSprint.id })
      .then((response) => {
        if (isMounted) {
          setDashboardSprintIssues(response.issues);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setDashboardSprintError(
            apiErrorMessage(err, "Could not load active sprint workload."),
          );
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingDashboardSprint(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user, issueFilterSprints]);

  useEffect(() => {
    if (!user || activeSection !== "sprints" || !routeSprintId) {
      return;
    }
    if (selectedSprint?.id === routeSprintId) {
      return;
    }

    void loadSprintDetail(routeSprintId);
  }, [user, activeSection, routeSprintId, selectedSprint?.id]);

  useEffect(() => {
    if (!user || !selectedSprint) {
      setSprintPlanningIssues([]);
      setSprintPlanningError("");
      setIsLoadingSprintPlanning(false);
      setAddingIssueToSprintIds([]);
      setRemovingIssueFromSprintIds([]);
      return;
    }

    let isMounted = true;
    setSprintPlanningError("");
    setIsLoadingSprintPlanning(true);

    listAllIssues({
      projectId: selectedSprint.project_id,
      sort: "created_desc",
    })
      .then((projectIssues) => {
        if (isMounted) {
          setSprintPlanningIssues(projectIssues);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setSprintPlanningError(
            apiErrorMessage(err, "Could not load sprint planning issues."),
          );
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingSprintPlanning(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [user, selectedSprint?.id, selectedSprint?.project_id]);

  useEffect(() => {
    if (!selectedIssueId) {
      setIssueComments([]);
      setCommentsError("");
      setCommentBody("");
      setEditingCommentId("");
      setEditCommentBody("");
      setUpdatingCommentIds([]);
      setDeletingCommentIds([]);
      return;
    }

    let isMounted = true;
    setCommentsError("");
    setEditingCommentId("");
    setEditCommentBody("");
    setUpdatingCommentIds([]);
    setDeletingCommentIds([]);
    setIsLoadingComments(true);

    listIssueComments(selectedIssueId)
      .then((response) => {
        if (isMounted) {
          setIssueComments(response.comments);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setCommentsError(apiErrorMessage(err, "Could not load comments."));
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingComments(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [selectedIssueId]);

  useEffect(() => {
    if (!selectedIssueId) {
      setIssueActivity([]);
      setActivityError("");
      return;
    }

    let isMounted = true;
    setActivityError("");
    setIsLoadingActivity(true);

    listIssueActivity(selectedIssueId)
      .then((response) => {
        if (isMounted) {
          setIssueActivity(response.activity);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setActivityError(apiErrorMessage(err, "Could not load activity."));
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingActivity(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [selectedIssueId]);

  useEffect(() => {
    if (!selectedIssueId) {
      setIssueChildren([]);
      setHierarchyError("");
      setSubtaskFormError("");
      setSubtaskTitle("");
      setSubtaskPriority("medium");
      setSubtaskStoryPoints("0");
      setSubtaskWorkflowStatusId("");
      return;
    }

    let isMounted = true;
    setHierarchyError("");
    setIsLoadingIssueChildren(true);

    listIssueChildren(selectedIssueId)
      .then((response) => {
        if (isMounted) {
          setIssueChildren(response.issues);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setHierarchyError(
            apiErrorMessage(err, "Could not load child issues."),
          );
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingIssueChildren(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [selectedIssueId]);

  useEffect(() => {
    if (!selectedIssueId) {
      setIssueLinks([]);
      setLinksError("");
      setLinkFormError("");
      setIsLoadingIssueLinks(false);
      setIsCreatingIssueLink(false);
      setDeletingIssueLinkIds([]);
      setLinkTargetIssueId("");
      setLinkType("relates");
      return;
    }

    let isMounted = true;
    setLinksError("");
    setLinkFormError("");
    setDeletingIssueLinkIds([]);
    setLinkTargetIssueId("");
    setLinkType("relates");
    setIsLoadingIssueLinks(true);

    listIssueLinks(selectedIssueId)
      .then((response) => {
        if (isMounted) {
          setIssueLinks(response.links);
        }
      })
      .catch((err) => {
        if (isMounted) {
          setLinksError(apiErrorMessage(err, "Could not load linked issues."));
        }
      })
      .finally(() => {
        if (isMounted) {
          setIsLoadingIssueLinks(false);
        }
      });

    return () => {
      isMounted = false;
    };
  }, [selectedIssueId]);

  async function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    const loginIdentifier = normalizeText(loginValue);
    const loginPassword = normalizeText(password);

    if (!hasText(loginIdentifier)) {
      setError("Username or email is required.");
      return;
    }
    if (!hasText(loginPassword)) {
      setError("Password is required.");
      return;
    }

    setIsSubmitting(true);

    try {
      const response = await login(loginIdentifier, loginPassword);
      setUser(response.user);
      setLoginValue("");
      setPassword("");
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setError("Invalid username or password.");
      } else {
        setError("Could not sign in. Check that backend is running.");
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  async function handleLogout() {
    if (isLoggingOut) {
      return;
    }

    setIsLoggingOut(true);
    try {
      await logout();
    } catch {
      // Logout is best-effort on localhost; clear local state even if the API is down.
    }
    resetLocalSession();
  }

  function resetLocalSession(loginError = "") {
    setUser(null);
    setError(loginError);
    setLoginValue("");
    setPassword("");
    setIsSubmitting(false);
    setIsLoggingOut(false);
    navigateToSection("dashboard", "replace");
    setAccountError("");
    setAccountSuccess("");
    setAccountDisplayName("");
    setRuntimeVersion(null);
    setRuntimeVersionError("");
    setIsLoadingRuntimeVersion(false);
    setIsUpdatingProfile(false);
    setCurrentPassword("");
    setNewPassword("");
    setConfirmNewPassword("");
    setIsChangingPassword(false);
    setProjects([]);
    setIsLoadingProjects(true);
    setTeamMembers([]);
    setLabels([]);
    setIssues([]);
    setBoardProjectId("");
    setBoardIssues([]);
    setBoardError("");
    setIsLoadingBoard(false);
    setProjectsError("");
    setProjectFormError("");
    setEditingProjectId("");
    setEditProjectName("");
    setEditProjectDescription("");
    setUpdatingProjectIds([]);
    setSelectedProjectDetail(null);
    setProjectDetailError("");
    setIsLoadingProjectDetail(false);
    setProjectDetailTab("summary");
    setProjectMembers([]);
    setProjectMembersError("");
    setIsLoadingProjectMembers(false);
    setSelectedProjectMemberUserId("");
    setSelectedProjectMemberRole("contributor");
    setUpdatingProjectMemberIds([]);
    setRemovingProjectMemberIds([]);
    setWorkflowMutationError("");
    setCreatingWorkflowStatus(false);
    setUpdatingWorkflowStatusIds([]);
    setArchivingWorkflowStatusIds([]);
    setIsReorderingWorkflowStatuses(false);
    setIsSavingWorkflowTransitions(false);
    setAutomationRules([]);
    setAutomationRulesError("");
    setIsLoadingAutomationRules(false);
    setIsCreatingAutomationRule(false);
    setUpdatingAutomationRuleIds([]);
    setDeletingAutomationRuleIds([]);
    setIsReorderingAutomationRules(false);
    setTeamMembersError("");
    setTeamMemberFormError("");
    setTeamMemberEmail("");
    setTeamMemberUsername("");
    setTeamMemberDisplayName("");
    setTeamMemberPassword("");
    setTeamMemberRole("member");
    setTeamInvites([]);
    setTeamInvitesError("");
    setTeamInviteFormError("");
    setIsLoadingTeamInvites(false);
    setIsCreatingTeamInvite(false);
    setTeamInviteEmail("");
    setTeamInviteRole("member");
    setTeamInviteLinksById({});
    setCopiedTeamInviteId("");
    setRevokingTeamInviteIds([]);
    setResendingTeamInviteIds([]);
    setUpdatingTeamMemberIds([]);
    setPasswordResetMemberId("");
    setTeamMemberResetPassword("");
    setResettingTeamMemberPasswordIds([]);
    setLabelsError("");
    setLabelName("");
    setLabelColor("#4e795d");
    setDeletingLabelIds([]);
    setIssuesError("");
    setIssueFormError("");
    setIssueFilterQuery("");
    setIssueSort("created_desc");
    setIssueFilterProjectId("");
    setIssueFilterSprintId("");
    setIssueFilterStatus("");
    setIssueFilterWorkflowStatusId("");
    setIssueFilterPriority("");
    setIssueFilterAssigneeId("");
    setIssueFilterLabelId("");
    setIssueFilterDue("");
    setSavedFilters([]);
    setSavedFiltersError("");
    setSavedFilterFormError("");
    setSavedFilterName("");
    setIsLoadingSavedFilters(false);
    setIsCreatingSavedFilter(false);
    setUpdatingSavedFilterIds([]);
    setDeletingSavedFilterIds([]);
    setRenameSavedFilterId("");
    setRenameSavedFilterName("");
    setNewIssueLabelIds([]);
    setIssueStoryPoints("0");
    setTransitioningIssueIds([]);
    setAssigningIssueIds([]);
    setLabelingIssueIds([]);
    setArchivingIssueIds([]);
    setSelectedIssue(null);
    setSelectedIssueError("");
    setIsEditingIssueDetails(false);
    setIsUpdatingIssue(false);
    setIssueComments([]);
    setCommentsError("");
    setCommentBody("");
    setEditingCommentId("");
    setEditCommentBody("");
    setUpdatingCommentIds([]);
    setDeletingCommentIds([]);
    setIssueActivity([]);
    setActivityError("");
    setIssueChildren([]);
    setHierarchyError("");
    setSubtaskFormError("");
    setIsLoadingIssueChildren(false);
    setIsCreatingSubtask(false);
    setSubtaskTitle("");
    setSubtaskPriority("medium");
    setSubtaskStoryPoints("0");
    setSubtaskWorkflowStatusId("");
    setIssueLinks([]);
    setLinksError("");
    setLinkFormError("");
    setIsLoadingIssueLinks(false);
    setIsCreatingIssueLink(false);
    setDeletingIssueLinkIds([]);
    setLinkTargetIssueId("");
    setLinkType("relates");
    setSprints([]);
    setIssueFilterSprints([]);
    setDashboardSprintIssues([]);
    setDashboardSprintError("");
    setIsLoadingDashboardSprint(false);
    setSprintsError("");
    setSprintFormError("");
    setIsLoadingSprints(false);
    setIsCreatingSprint(false);
    setSprintProjectId("");
    setSprintName("");
    setSprintGoal("");
    setSprintStartDate("");
    setSprintEndDate("");
    setSprintFilterProjectId("");
    setSprintFilterStatus("");
    setSelectedSprint(null);
    setSelectedSprintError("");
    setIsLoadingSelectedSprint(false);
    setIsEditingSprintDetails(false);
    setIsUpdatingSprint(false);
    setEditSprintName("");
    setEditSprintGoal("");
    setEditSprintStartDate("");
    setEditSprintEndDate("");
    setStartingSprintIds([]);
    setCompletingSprintIds([]);
    setSprintPlanningIssues([]);
    setSprintPlanningError("");
    setIsLoadingSprintPlanning(false);
    setAddingIssueToSprintIds([]);
    setRemovingIssueFromSprintIds([]);
    setNotifications([]);
    setNotificationsError("");
    setIsLoadingNotifications(false);
    setUnreadNotificationsCount(0);
    setIsNotificationsOpen(false);
  }

  async function handleUpdateProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAccountError("");
    setAccountSuccess("");
    const displayName = normalizeText(accountDisplayName);

    if (!hasText(displayName)) {
      setAccountError("Display name is required.");
      return;
    }
    if (displayName === user?.display_name) {
      setAccountError("Display name is unchanged.");
      return;
    }

    setIsUpdatingProfile(true);

    try {
      const response = await updateProfile(displayName);
      setUser(response.user);
      setTeamMembers((currentMembers) =>
        currentMembers.map((member) =>
          member.id === response.user.id
            ? { ...member, display_name: response.user.display_name }
            : member,
        ),
      );
      setIssueComments((currentComments) =>
        currentComments.map((comment) =>
          comment.author_id === response.user.id
            ? { ...comment, author_display_name: response.user.display_name }
            : comment,
        ),
      );
      setIssueActivity((currentActivity) =>
        currentActivity.map((entry) =>
          entry.actor_id === response.user.id
            ? { ...entry, actor_display_name: response.user.display_name }
            : entry,
        ),
      );
      setAccountSuccess("Profile updated.");
    } catch (err) {
      setAccountError(apiErrorMessage(err, "Could not update profile."));
    } finally {
      setIsUpdatingProfile(false);
    }
  }

  async function handleChangePassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAccountError("");
    setAccountSuccess("");
    const current = normalizeText(currentPassword);
    const next = normalizeText(newPassword);
    const confirmation = normalizeText(confirmNewPassword);

    if (!hasText(current)) {
      setAccountError("Current password is required.");
      return;
    }
    if (!hasMinTrimmedLength(next, 8)) {
      setAccountError("New password must be at least 8 characters.");
      return;
    }
    if (!hasMinTrimmedLength(confirmation, 8)) {
      setAccountError("Password confirmation must be at least 8 characters.");
      return;
    }
    if (next === current) {
      setAccountError("New password must be different from current password.");
      return;
    }
    if (next !== confirmation) {
      setAccountError("New password confirmation does not match.");
      return;
    }

    setIsChangingPassword(true);
    try {
      await changePassword(current, next);
      setCurrentPassword("");
      setNewPassword("");
      setConfirmNewPassword("");
      setAccountSuccess("Password changed. Other sessions were signed out.");
    } catch (err) {
      setAccountError(apiErrorMessage(err, "Could not change password."));
    } finally {
      setIsChangingPassword(false);
    }
  }

  async function handleCreateProject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setProjectFormError("");
    if (!hasText(projectKey)) {
      setProjectFormError("Project key is required.");
      return;
    }
    if (!hasText(projectName)) {
      setProjectFormError("Project name is required.");
      return;
    }

    setIsCreatingProject(true);

    try {
      const project = await createProject({
        key: projectKey,
        name: projectName,
        description: projectDescription,
      });
      setProjects((currentProjects) => [project, ...currentProjects]);
      setSelectedProjectId(project.id);
      setSelectedProjectDetail(project);
      setProjectKey("");
      setProjectName("");
      setProjectDescription("");
    } catch (err) {
      setProjectFormError(apiErrorMessage(err, "Could not create project."));
    } finally {
      setIsCreatingProject(false);
    }
  }

  function startEditingProject(project: Project) {
    setProjectsError("");
    setEditingProjectId(project.id);
    setEditProjectName(project.name);
    setEditProjectDescription(project.description);
  }

  function cancelEditingProject() {
    setEditingProjectId("");
    setEditProjectName("");
    setEditProjectDescription("");
  }

  async function handleSelectProjectDetail(projectId: string) {
    const projectPreview = projects.find((project) => project.id === projectId);
    if (projectPreview) {
      setSelectedProjectDetail(projectPreview);
    }

    setProjectDetailError("");
    setIsLoadingProjectDetail(true);

    try {
      const project = await getProject(projectId);
      setSelectedProjectDetail(project);
      setProjects((currentProjects) =>
        currentProjects.map((currentProject) =>
          currentProject.id === project.id ? project : currentProject,
        ),
      );
    } catch (err) {
      setProjectDetailError(apiErrorMessage(err, "Could not load project details."));
    } finally {
      setIsLoadingProjectDetail(false);
    }
  }

  function syncLoadedIssuesWithWorkflow(
    projectId: string,
    workflow: ProjectWorkflow,
  ) {
    const statusesByID = new Map(
      workflow.statuses.map((status) => [status.id, status]),
    );
    const statusesByKey = new Map(
      workflow.statuses.map((status) => [status.key, status]),
    );
    const syncIssue = (issue: Issue) => {
      if (issue.project_id !== projectId) {
        return issue;
      }
      const status =
        statusesByID.get(issue.workflow_status?.id) ?? statusesByKey.get(issue.status);
      if (!status) {
        return issue;
      }
      return {
        ...issue,
        status: status.key,
        workflow_status: {
          id: status.id,
          key: status.key,
          name: status.name,
          color: status.color,
          category: status.category,
        },
      };
    };
    setIssues((currentIssues) => currentIssues.map(syncIssue));
    setBoardIssues((currentIssues) => currentIssues.map(syncIssue));
    setDashboardSprintIssues((currentIssues) => currentIssues.map(syncIssue));
    setSprintPlanningIssues((currentIssues) => currentIssues.map(syncIssue));
    setIssueChildren((currentIssues) => currentIssues.map(syncIssue));
    setSelectedIssue((currentIssue) =>
      currentIssue ? syncIssue(currentIssue) : currentIssue,
    );
  }

  function applyWorkflowUpdate(workflow: ProjectWorkflow) {
    setWorkflowsByProjectId((currentWorkflows) => ({
      ...currentWorkflows,
      [workflow.project_id]: workflow,
    }));
    setWorkflowErrorsByProjectId((currentErrors) => ({
      ...currentErrors,
      [workflow.project_id]: "",
    }));
    syncLoadedIssuesWithWorkflow(workflow.project_id, workflow);
  }

  async function refreshProjectWorkflow(projectId: string) {
    setLoadingWorkflowProjectIds((currentIds) =>
      currentIds.includes(projectId) ? currentIds : [...currentIds, projectId],
    );
    setWorkflowErrorsByProjectId((currentErrors) => ({
      ...currentErrors,
      [projectId]: "",
    }));
    try {
      const workflow = await getProjectWorkflow(projectId);
      applyWorkflowUpdate(workflow);
      return workflow;
    } catch (err) {
      setWorkflowErrorsByProjectId((currentErrors) => ({
        ...currentErrors,
        [projectId]: apiErrorMessage(err, "Could not load project workflow."),
      }));
      return null;
    } finally {
      setLoadingWorkflowProjectIds((currentIds) =>
        currentIds.filter((currentId) => currentId !== projectId),
      );
    }
  }

  async function syncProjectIssuesAfterWorkflowArchive(projectId: string) {
    const projectIssues = await listAllIssues({ projectId, sort: "created_desc" });
    const issuesByID = new Map(projectIssues.map((issue) => [issue.id, issue]));
    const syncIssue = (issue: Issue) =>
      issue.project_id === projectId ? issuesByID.get(issue.id) ?? issue : issue;

    setIssues((currentIssues) =>
      currentIssues
        .map(syncIssue)
        .filter((issue) =>
          issueMatchesFilters(
            issue,
            issueFilterProjectId,
            issueFilterSprintId,
            issueFilterStatus,
            issueFilterWorkflowStatusId,
            issueFilterPriority,
            issueFilterAssigneeId,
            issueFilterLabelId,
            issueFilterDue,
            issueFilterQuery,
            today,
          ),
        ),
    );
    setBoardIssues((currentIssues) =>
      boardProjectId === projectId ? projectIssues : currentIssues.map(syncIssue),
    );
    setDashboardSprintIssues((currentIssues) => currentIssues.map(syncIssue));
    setSprintPlanningIssues((currentIssues) =>
      selectedSprint?.project_id === projectId
        ? projectIssues
        : currentIssues.map(syncIssue),
    );
    setIssueChildren((currentIssues) => currentIssues.map(syncIssue));
    setSelectedIssue((currentIssue) =>
      currentIssue ? syncIssue(currentIssue) : currentIssue,
    );
  }

  async function handleProjectDetailTabChange(
    tab: "summary" | "members" | "workflow" | "automation",
  ) {
    if (tab === "summary") {
      setProjectDetailTab("summary");
      return;
    }
    if (!selectedProjectDetail?.can_manage) {
      setProjectDetailTab("summary");
      return;
    }

    setProjectDetailTab(tab);
    if (tab === "workflow") {
      setWorkflowMutationError("");
      await refreshProjectWorkflow(selectedProjectDetail.id);
      return;
    }

    if (tab === "automation") {
      setAutomationRulesError("");
      setWorkflowMutationError("");
      setProjectMembersError("");
      setIsLoadingAutomationRules(true);
      setIsLoadingProjectMembers(true);
      try {
        const [rulesResponse, membersResponse] = await Promise.all([
          listAutomationRules(selectedProjectDetail.id),
          listProjectMembers(selectedProjectDetail.id),
          refreshProjectWorkflow(selectedProjectDetail.id),
        ]);
        setAutomationRules(rulesResponse.automation_rules);
        setProjectMembers(membersResponse.members);
      } catch (err) {
        setAutomationRulesError(
          apiErrorMessage(err, "Could not load automation settings."),
        );
      } finally {
        setIsLoadingAutomationRules(false);
        setIsLoadingProjectMembers(false);
      }
      return;
    }

    setProjectMembersError("");
    setIsLoadingProjectMembers(true);
    try {
      const response = await listProjectMembers(selectedProjectDetail.id);
      setProjectMembers(response.members);
    } catch (err) {
      setProjectMembersError(
        apiErrorMessage(err, "Could not load project members."),
      );
    } finally {
      setIsLoadingProjectMembers(false);
    }
  }

  async function handleCreateWorkflowStatus(input: CreateWorkflowStatusInput) {
    if (!selectedProjectDetail?.can_manage) {
      return false;
    }
    setWorkflowMutationError("");
    setCreatingWorkflowStatus(true);
    try {
      await createWorkflowStatus(selectedProjectDetail.id, input);
      return Boolean(await refreshProjectWorkflow(selectedProjectDetail.id));
    } catch (err) {
      setWorkflowMutationError(
        apiErrorMessage(err, "Could not create workflow status."),
      );
      return false;
    } finally {
      setCreatingWorkflowStatus(false);
    }
  }

  async function handleUpdateWorkflowStatus(
    status: ProjectWorkflowStatus,
    input: UpdateWorkflowStatusInput,
  ) {
    if (!selectedProjectDetail?.can_manage) {
      return false;
    }
    setWorkflowMutationError("");
    setUpdatingWorkflowStatusIds((currentIds) => [
      ...currentIds.filter((id) => id !== status.id),
      status.id,
    ]);
    try {
      await updateWorkflowStatus(selectedProjectDetail.id, status.id, input);
      return Boolean(await refreshProjectWorkflow(selectedProjectDetail.id));
    } catch (err) {
      setWorkflowMutationError(
        apiErrorMessage(err, "Could not update workflow status."),
      );
      return false;
    } finally {
      setUpdatingWorkflowStatusIds((currentIds) =>
        currentIds.filter((id) => id !== status.id),
      );
    }
  }

  async function handleReorderWorkflowStatuses(statusIds: string[]) {
    if (!selectedProjectDetail?.can_manage) {
      return false;
    }
    setWorkflowMutationError("");
    setIsReorderingWorkflowStatuses(true);
    try {
      applyWorkflowUpdate(
        await reorderWorkflowStatuses(selectedProjectDetail.id, statusIds),
      );
      return true;
    } catch (err) {
      setWorkflowMutationError(
        apiErrorMessage(err, "Could not reorder workflow statuses."),
      );
      return false;
    } finally {
      setIsReorderingWorkflowStatuses(false);
    }
  }

  async function handleReplaceWorkflowTransitions(
    transitions: WorkflowTransitionInput[],
  ) {
    if (!selectedProjectDetail?.can_manage) {
      return false;
    }
    setWorkflowMutationError("");
    setIsSavingWorkflowTransitions(true);
    try {
      applyWorkflowUpdate(
        await replaceWorkflowTransitions(selectedProjectDetail.id, { transitions }),
      );
      return true;
    } catch (err) {
      setWorkflowMutationError(
        apiErrorMessage(err, "Could not update workflow transitions."),
      );
      return false;
    } finally {
      setIsSavingWorkflowTransitions(false);
    }
  }

  async function handleArchiveWorkflowStatus(
    status: ProjectWorkflowStatus,
    replacementStatusId: string,
  ) {
    if (!selectedProjectDetail?.can_manage) {
      return false;
    }
    setWorkflowMutationError("");
    setArchivingWorkflowStatusIds((currentIds) => [
      ...currentIds.filter((id) => id !== status.id),
      status.id,
    ]);
    try {
      await archiveWorkflowStatus(
        selectedProjectDetail.id,
        status.id,
        replacementStatusId,
      );
      const workflow = await refreshProjectWorkflow(selectedProjectDetail.id);
      try {
        await syncProjectIssuesAfterWorkflowArchive(selectedProjectDetail.id);
      } catch (err) {
        setWorkflowMutationError(
          apiErrorMessage(
            err,
            "Status was archived, but project issues could not be refreshed.",
          ),
        );
      }
      return Boolean(workflow);
    } catch (err) {
      setWorkflowMutationError(
        apiErrorMessage(err, "Could not archive workflow status."),
      );
      return false;
    } finally {
      setArchivingWorkflowStatusIds((currentIds) =>
        currentIds.filter((id) => id !== status.id),
      );
    }
  }

  async function handleCreateAutomationRule(input: CreateAutomationRuleInput) {
    if (!selectedProjectDetail?.can_manage) {
      return false;
    }
    setAutomationRulesError("");
    setIsCreatingAutomationRule(true);
    try {
      const rule = await createAutomationRule(selectedProjectDetail.id, input);
      setAutomationRules((currentRules) => [...currentRules, rule]);
      return true;
    } catch (err) {
      setAutomationRulesError(
        apiErrorMessage(err, "Could not create automation rule."),
      );
      return false;
    } finally {
      setIsCreatingAutomationRule(false);
    }
  }

  async function handleUpdateAutomationRule(
    rule: AutomationRule,
    input: CreateAutomationRuleInput | { is_enabled: boolean },
  ) {
    if (!selectedProjectDetail?.can_manage) {
      return false;
    }
    setAutomationRulesError("");
    setUpdatingAutomationRuleIds((currentIds) => [
      ...currentIds.filter((id) => id !== rule.id),
      rule.id,
    ]);
    try {
      const updatedRule = await updateAutomationRule(
        selectedProjectDetail.id,
        rule.id,
        input,
      );
      setAutomationRules((currentRules) =>
        currentRules.map((currentRule) =>
          currentRule.id === updatedRule.id ? updatedRule : currentRule,
        ),
      );
      return true;
    } catch (err) {
      setAutomationRulesError(
        apiErrorMessage(err, "Could not update automation rule."),
      );
      return false;
    } finally {
      setUpdatingAutomationRuleIds((currentIds) =>
        currentIds.filter((id) => id !== rule.id),
      );
    }
  }

  async function handleDeleteAutomationRule(rule: AutomationRule) {
    if (!selectedProjectDetail?.can_manage) {
      return false;
    }
    setAutomationRulesError("");
    setDeletingAutomationRuleIds((currentIds) => [
      ...currentIds.filter((id) => id !== rule.id),
      rule.id,
    ]);
    try {
      await deleteAutomationRule(selectedProjectDetail.id, rule.id);
      setAutomationRules((currentRules) =>
        currentRules.filter((currentRule) => currentRule.id !== rule.id),
      );
      return true;
    } catch (err) {
      setAutomationRulesError(
        apiErrorMessage(err, "Could not delete automation rule."),
      );
      return false;
    } finally {
      setDeletingAutomationRuleIds((currentIds) =>
        currentIds.filter((id) => id !== rule.id),
      );
    }
  }

  async function handleReorderAutomationRules(ruleIds: string[]) {
    if (!selectedProjectDetail?.can_manage) {
      return false;
    }
    setAutomationRulesError("");
    setIsReorderingAutomationRules(true);
    try {
      const response = await reorderAutomationRules(
        selectedProjectDetail.id,
        ruleIds,
      );
      setAutomationRules(response.automation_rules);
      return true;
    } catch (err) {
      setAutomationRulesError(
        apiErrorMessage(err, "Could not reorder automation rules."),
      );
      return false;
    } finally {
      setIsReorderingAutomationRules(false);
    }
  }

  async function handleAddProjectMember(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedProjectDetail?.can_manage || !selectedProjectMemberUserId) {
      return;
    }

    const userId = selectedProjectMemberUserId;
    setProjectMembersError("");
    setUpdatingProjectMemberIds((currentIds) => [
      ...currentIds.filter((id) => id !== userId),
      userId,
    ]);
    try {
      const member = await putProjectMember(
        selectedProjectDetail.id,
        userId,
        { role: selectedProjectMemberRole },
      );
      setProjectMembers((currentMembers) => [...currentMembers, member]);
      setSelectedProjectMemberUserId("");
      setSelectedProjectMemberRole("contributor");
      if (member.user_id === user?.id) {
        await refreshProjectsAfterSelfMembershipChange(selectedProjectDetail.id);
      }
    } catch (err) {
      setProjectMembersError(apiErrorMessage(err, "Could not add project member."));
    } finally {
      setUpdatingProjectMemberIds((currentIds) =>
        currentIds.filter((id) => id !== userId),
      );
    }
  }

  async function handleProjectMemberRoleChange(
    member: ProjectMember,
    role: ProjectRole,
  ) {
    if (!selectedProjectDetail?.can_manage || member.role === role) {
      return;
    }

    setProjectMembersError("");
    setUpdatingProjectMemberIds((currentIds) => [
      ...currentIds.filter((id) => id !== member.user_id),
      member.user_id,
    ]);
    try {
      const updatedMember = await putProjectMember(
        selectedProjectDetail.id,
        member.user_id,
        { role },
      );
      setProjectMembers((currentMembers) =>
        currentMembers.map((currentMember) =>
          currentMember.user_id === updatedMember.user_id
            ? updatedMember
            : currentMember,
        ),
      );
      if (member.user_id === user?.id) {
        await refreshProjectsAfterSelfMembershipChange(selectedProjectDetail.id);
      }
    } catch (err) {
      setProjectMembersError(
        apiErrorMessage(err, "Could not update project member."),
      );
    } finally {
      setUpdatingProjectMemberIds((currentIds) =>
        currentIds.filter((id) => id !== member.user_id),
      );
    }
  }

  async function handleRemoveProjectMember(member: ProjectMember) {
    if (!selectedProjectDetail?.can_manage) {
      return;
    }
    const accessNote =
      member.workspace_role === "admin"
        ? " This workspace admin will keep full project access."
        : "";
    if (
      !window.confirm(
        `Remove ${member.display_name} from ${selectedProjectDetail.key}?${accessNote}`,
      )
    ) {
      return;
    }

    setProjectMembersError("");
    setRemovingProjectMemberIds((currentIds) => [
      ...currentIds.filter((id) => id !== member.user_id),
      member.user_id,
    ]);
    try {
      await deleteProjectMember(selectedProjectDetail.id, member.user_id);
      setProjectMembers((currentMembers) =>
        currentMembers.filter(
          (currentMember) => currentMember.user_id !== member.user_id,
        ),
      );
      if (member.user_id === user?.id) {
        await refreshProjectsAfterSelfMembershipChange(selectedProjectDetail.id);
      }
    } catch (err) {
      setProjectMembersError(
        apiErrorMessage(err, "Could not remove project member."),
      );
    } finally {
      setRemovingProjectMemberIds((currentIds) =>
        currentIds.filter((id) => id !== member.user_id),
      );
    }
  }

  async function refreshProjectsAfterSelfMembershipChange(projectId: string) {
    const response = await listProjects();
    setProjects(response.projects);
    const refreshedProject =
      response.projects.find((project) => project.id === projectId) ?? null;
    if (refreshedProject) {
      setSelectedProjectDetail(refreshedProject);
      if (!refreshedProject.can_manage) {
        setProjectDetailTab("summary");
        setProjectMembers([]);
      }
      return;
    }

    const fallbackProject = response.projects[0] ?? null;
    setSelectedProjectDetail(fallbackProject);
    setProjectDetailTab("summary");
    setProjectMembers([]);
    setSelectedProjectId((currentId) =>
      currentId === projectId
        ? response.projects.find((project) => project.can_write)?.id ?? ""
        : currentId,
    );
    setIssueFilterProjectId((currentId) =>
      currentId === projectId ? "" : currentId,
    );
    setIssueFilterStatus("");
    setIssueFilterWorkflowStatusId("");
    setSprintProjectId((currentId) =>
      currentId === projectId ? fallbackProject?.id ?? "" : currentId,
    );
    setSprintFilterProjectId((currentId) =>
      currentId === projectId ? "" : currentId,
    );
    setIssues((currentIssues) =>
      currentIssues.filter((issue) => issue.project_id !== projectId),
    );
    setSelectedIssue((currentIssue) =>
      currentIssue?.project_id === projectId ? null : currentIssue,
    );
    setSprints((currentSprints) =>
      currentSprints.filter((sprint) => sprint.project_id !== projectId),
    );
    setSelectedSprint((currentSprint) =>
      currentSprint?.project_id === projectId ? null : currentSprint,
    );
  }

  async function handleUpdateProject(
    event: FormEvent<HTMLFormElement>,
    project: Project,
  ) {
    event.preventDefault();
    setProjectsError("");
    setUpdatingProjectIds((currentIds) =>
      currentIds.includes(project.id) ? currentIds : [...currentIds, project.id],
    );

    try {
      const updatedProject = await updateProject(project.id, {
        name: editProjectName,
        description: editProjectDescription,
      });
      setProjects((currentProjects) =>
        currentProjects.map((currentProject) =>
          currentProject.id === updatedProject.id ? updatedProject : currentProject,
        ),
      );
      setSelectedProjectDetail((currentProject) =>
        currentProject?.id === updatedProject.id ? updatedProject : currentProject,
      );
      cancelEditingProject();
    } catch (err) {
      setProjectsError(apiErrorMessage(err, "Could not update project."));
    } finally {
      setUpdatingProjectIds((currentIds) =>
        currentIds.filter((currentProjectId) => currentProjectId !== project.id),
      );
    }
  }

  async function handleArchiveProject(project: Project) {
    if (!window.confirm(`Archive project ${project.key}? Its active issues will be hidden.`)) {
      return;
    }

    setProjectsError("");
    setArchivingProjectIds((currentIds) =>
      currentIds.includes(project.id) ? currentIds : [...currentIds, project.id],
    );

    try {
      await archiveProject(project.id);
      const nextProjects = projects.filter(
        (currentProject) => currentProject.id !== project.id,
      );

      setProjects((currentProjects) =>
        currentProjects.filter((currentProject) => currentProject.id !== project.id),
      );
      setSelectedProjectId((currentProjectId) =>
        currentProjectId === project.id
          ? nextProjects.find((currentProject) => currentProject.can_write)?.id ?? ""
          : currentProjectId,
      );
      setSelectedProjectDetail((currentProject) =>
        currentProject?.id === project.id ? nextProjects[0] ?? null : currentProject,
      );
      setIssueFilterProjectId((currentProjectId) =>
        currentProjectId === project.id ? "" : currentProjectId,
      );
      setIssueFilterStatus("");
      setIssueFilterWorkflowStatusId("");
      setSprintProjectId((currentProjectId) =>
        currentProjectId === project.id ? nextProjects[0]?.id ?? "" : currentProjectId,
      );
      setSprintFilterProjectId((currentProjectId) =>
        currentProjectId === project.id ? "" : currentProjectId,
      );
      setSprints((currentSprints) =>
        currentSprints.filter((sprint) => sprint.project_id !== project.id),
      );
      setSelectedSprint((currentSprint) =>
        currentSprint?.project_id === project.id ? null : currentSprint,
      );
      if (editingProjectId === project.id) {
        cancelEditingProject();
      }
      setIssues((currentIssues) =>
        currentIssues.filter((issue) => issue.project_id !== project.id),
      );
      setSelectedIssue((currentIssue) =>
        currentIssue?.project_id === project.id ? null : currentIssue,
      );
      if (selectedIssue?.project_id === project.id) {
        setIssueComments([]);
        setIssueActivity([]);
        setCommentBody("");
        setEditingCommentId("");
        setEditCommentBody("");
        setUpdatingCommentIds([]);
        setDeletingCommentIds([]);
        setIsEditingIssueDetails(false);
      }
    } catch (err) {
      setProjectsError(apiErrorMessage(err, "Could not archive project."));
    } finally {
      setArchivingProjectIds((currentIds) =>
        currentIds.filter((currentProjectId) => currentProjectId !== project.id),
      );
    }
  }

  async function handleCreateTeamMember(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setTeamMemberFormError("");
    const email = normalizeEmail(teamMemberEmail);
    const username = normalizeUsername(teamMemberUsername);
    const displayName = normalizeText(teamMemberDisplayName);
    const memberPassword = normalizeText(teamMemberPassword);

    if (!isValidEmail(email)) {
      setTeamMemberFormError("Email is invalid.");
      return;
    }
    if (!isValidUsername(username)) {
      setTeamMemberFormError(
        "Username must be 3-32 characters and contain lowercase letters, numbers, underscores, or hyphens.",
      );
      return;
    }
    if (!hasText(displayName)) {
      setTeamMemberFormError("Display name is required.");
      return;
    }
    if (!hasMinTrimmedLength(memberPassword, 8)) {
      setTeamMemberFormError("Password must be at least 8 characters.");
      return;
    }

    setIsCreatingTeamMember(true);

    try {
      const member = await createTeamMember({
        email,
        username,
        display_name: displayName,
        password: memberPassword,
        role: teamMemberRole,
      });
      setTeamMembers((currentMembers) => [...currentMembers, member]);
      setTeamMemberEmail("");
      setTeamMemberUsername("");
      setTeamMemberDisplayName("");
      setTeamMemberPassword("");
      setTeamMemberRole("member");
    } catch (err) {
      setTeamMemberFormError(apiErrorMessage(err, "Could not create team member."));
    } finally {
      setIsCreatingTeamMember(false);
    }
  }

  async function handleCreateTeamInvite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setTeamInviteFormError("");
    setTeamInvitesError("");

    const email = normalizedInviteEmail(teamInviteEmail);
    const validationError = validateInviteEmail(email);
    if (validationError) {
      setTeamInviteFormError(validationError);
      return;
    }

    setIsCreatingTeamInvite(true);

    try {
      const invite = await createTeamInvite({
        email,
        role: teamInviteRole,
      });
      const origin = typeof window === "undefined" ? "" : window.location.origin;
      const inviteURL = buildInviteAcceptURL(invite.accept_url_path, origin);

      setTeamInvites((currentInvites) => [
        invite,
        ...currentInvites.filter((currentInvite) => currentInvite.id !== invite.id),
      ]);
      setTeamInviteLinksById((currentLinks) => ({
        ...currentLinks,
        [invite.id]: inviteURL,
      }));
      setCopiedTeamInviteId("");
      setTeamInviteEmail("");
      setTeamInviteRole("member");
    } catch (err) {
      setTeamInviteFormError(apiErrorMessage(err, "Could not create invite."));
    } finally {
      setIsCreatingTeamInvite(false);
    }
  }

  async function handleRevokeTeamInvite(invite: TeamInvite) {
    if (invite.status !== "pending") {
      return;
    }

    setTeamInvitesError("");
    setRevokingTeamInviteIds((currentIds) =>
      currentIds.includes(invite.id) ? currentIds : [...currentIds, invite.id],
    );

    try {
      const revokedInvite = await revokeTeamInvite(invite.id);
      setTeamInvites((currentInvites) =>
        currentInvites.map((currentInvite) =>
          currentInvite.id === revokedInvite.id ? revokedInvite : currentInvite,
        ),
      );
      setTeamInviteLinksById((currentLinks) => {
        const nextLinks = { ...currentLinks };
        delete nextLinks[invite.id];
        return nextLinks;
      });
      setCopiedTeamInviteId((currentId) =>
        currentId === invite.id ? "" : currentId,
      );
    } catch (err) {
      setTeamInvitesError(apiErrorMessage(err, "Could not revoke invite."));
    } finally {
      setRevokingTeamInviteIds((currentIds) =>
        currentIds.filter((currentId) => currentId !== invite.id),
      );
    }
  }

  async function handleResendTeamInvite(invite: TeamInvite) {
    if (invite.status !== "pending") {
      return;
    }

    setTeamInvitesError("");
    setResendingTeamInviteIds((currentIds) =>
      currentIds.includes(invite.id) ? currentIds : [...currentIds, invite.id],
    );

    try {
      const resentInvite = await resendTeamInvite(invite.id);
      setTeamInvites((currentInvites) =>
        currentInvites.map((currentInvite) =>
          currentInvite.id === resentInvite.id ? resentInvite : currentInvite,
        ),
      );
    } catch (err) {
      setTeamInvitesError(apiErrorMessage(err, "Could not resend invite email."));
    } finally {
      setResendingTeamInviteIds((currentIds) =>
        currentIds.filter((currentId) => currentId !== invite.id),
      );
    }
  }

  async function handleRefreshEmailDiagnostics() {
    if (!user || user.workspace.role !== "admin") {
      return;
    }

    setEmailDiagnosticsError("");
    setIsLoadingEmailDiagnostics(true);

    try {
      const diagnostics: EmailDiagnostics = await getEmailDiagnostics();
      setEmailDiagnostics(diagnostics);
    } catch (err) {
      setEmailDiagnosticsError(
        apiErrorMessage(err, "Could not load email diagnostics."),
      );
    } finally {
      setIsLoadingEmailDiagnostics(false);
    }
  }

  async function handleCopyTeamInviteLink(inviteId: string) {
    setTeamInvitesError("");
    const inviteLink = teamInviteLinksById[inviteId];
    if (!inviteLink) {
      setTeamInvitesError("Invite link is only available right after creation.");
      return;
    }

    try {
      await navigator.clipboard.writeText(inviteLink);
      setCopiedTeamInviteId(inviteId);
    } catch {
      setTeamInvitesError("Could not copy invite link. Select and copy it manually.");
    }
  }

  async function handleUpdateTeamMember(
    memberId: string,
    input: { role?: TeamMember["role"]; is_active?: boolean },
  ) {
    setTeamMembersError("");
    setUpdatingTeamMemberIds((currentIds) =>
      currentIds.includes(memberId) ? currentIds : [...currentIds, memberId],
    );

    try {
      const member = await updateTeamMember(memberId, input);
      setTeamMembers((currentMembers) =>
        currentMembers.map((currentMember) =>
          currentMember.id === member.id ? member : currentMember,
        ),
      );
    } catch (err) {
      setTeamMembersError(apiErrorMessage(err, "Could not update team member."));
    } finally {
      setUpdatingTeamMemberIds((currentIds) =>
        currentIds.filter((currentId) => currentId !== memberId),
      );
    }
  }

  function startResetTeamMemberPassword(memberId: string) {
    setTeamMembersError("");
    setPasswordResetMemberId(memberId);
    setTeamMemberResetPassword("");
  }

  function cancelResetTeamMemberPassword() {
    setPasswordResetMemberId("");
    setTeamMemberResetPassword("");
  }

  async function handleResetTeamMemberPassword(
    event: FormEvent<HTMLFormElement>,
    memberId: string,
  ) {
    event.preventDefault();
    setTeamMembersError("");
    const memberPassword = normalizeText(teamMemberResetPassword);
    if (!hasMinTrimmedLength(memberPassword, 8)) {
      setTeamMembersError("Password must be at least 8 characters.");
      return;
    }

    setResettingTeamMemberPasswordIds((currentIds) =>
      currentIds.includes(memberId) ? currentIds : [...currentIds, memberId],
    );

    try {
      await resetTeamMemberPassword(memberId, memberPassword);
      cancelResetTeamMemberPassword();
    } catch (err) {
      setTeamMembersError(
        apiErrorMessage(err, "Could not reset team member password."),
      );
    } finally {
      setResettingTeamMemberPasswordIds((currentIds) =>
        currentIds.filter((currentId) => currentId !== memberId),
      );
    }
  }

  async function handleCreateLabel(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLabelsError("");
    const name = normalizeText(labelName);
    const color = normalizeLabelColor(labelColor);

    if (!hasText(name)) {
      setLabelsError("Label name is required.");
      return;
    }
    if (!isValidLabelColor(color)) {
      setLabelsError("Label color must be a hex color like #4e795d.");
      return;
    }

    setIsCreatingLabel(true);

    try {
      const label = await createLabel({
        name,
        color,
      });
      setLabels((currentLabels) =>
        [...currentLabels, label].sort((left, right) =>
          left.name.localeCompare(right.name),
        ),
      );
      setLabelName("");
      setLabelColor("#4e795d");
    } catch (err) {
      setLabelsError(apiErrorMessage(err, "Could not create label."));
    } finally {
      setIsCreatingLabel(false);
    }
  }

  async function handleDeleteLabel(label: Label) {
    if (!window.confirm(`Delete label "${label.name}"? It will be removed from existing issues.`)) {
      return;
    }

    setLabelsError("");
    setDeletingLabelIds((currentIds) =>
      currentIds.includes(label.id) ? currentIds : [...currentIds, label.id],
    );

    try {
      await deleteLabel(label.id);
      setLabels((currentLabels) =>
        currentLabels.filter((currentLabel) => currentLabel.id !== label.id),
      );
      setIssueFilterLabelId((currentLabelId) =>
        currentLabelId === label.id ? "" : currentLabelId,
      );
      setNewIssueLabelIds((currentLabelIds) =>
        currentLabelIds.filter((currentLabelId) => currentLabelId !== label.id),
      );
      setIssues((currentIssues) =>
        currentIssues.map((issue) => ({
          ...issue,
          labels: issue.labels.filter((issueLabel) => issueLabel.id !== label.id),
        })),
      );
      setSelectedIssue((currentIssue) =>
        currentIssue
          ? {
              ...currentIssue,
              labels: currentIssue.labels.filter(
                (issueLabel) => issueLabel.id !== label.id,
              ),
            }
          : currentIssue,
      );
    } catch (err) {
      setLabelsError(apiErrorMessage(err, "Could not delete label."));
    } finally {
      setDeletingLabelIds((currentIds) =>
        currentIds.filter((currentLabelId) => currentLabelId !== label.id),
      );
    }
  }

  async function refreshIssueActivity(issueId: string) {
    setActivityError("");

    try {
      const response = await listIssueActivity(issueId);
      setIssueActivity(response.activity);
    } catch (err) {
      setActivityError(apiErrorMessage(err, "Could not load activity."));
    }
  }

  async function refreshIssueChildren(issueId: string) {
    setHierarchyError("");

    try {
      const response = await listIssueChildren(issueId);
      setIssueChildren(response.issues);
    } catch (err) {
      setHierarchyError(apiErrorMessage(err, "Could not load child issues."));
    }
  }

  async function refreshIssueLinks(issueId: string) {
    setLinksError("");

    try {
      const response = await listIssueLinks(issueId);
      setIssueLinks(response.links);
    } catch (err) {
      setLinksError(apiErrorMessage(err, "Could not load linked issues."));
    }
  }

  function upsertSprintInList(sprint: Sprint) {
    setIssueFilterSprints((currentSprints) =>
      upsertIssueFilterSprint(currentSprints, sprint),
    );
    setSprints((currentSprints) => {
      if (!sprintMatchesFilters(sprint, sprintFilterProjectId, sprintFilterStatus)) {
        return currentSprints.filter((currentSprint) => currentSprint.id !== sprint.id);
      }

      if (currentSprints.some((currentSprint) => currentSprint.id === sprint.id)) {
        return currentSprints.map((currentSprint) =>
          currentSprint.id === sprint.id ? sprint : currentSprint,
        );
      }

      return [sprint, ...currentSprints];
    });
  }

  async function refreshSprints(selectedSprintOverride: Sprint | null = null) {
    setSprintsError("");

    try {
      const response = await listSprints({
        projectId: sprintFilterProjectId || undefined,
        status: sprintFilterStatus || undefined,
      });
      setSprints(response.sprints);
      setSelectedSprint((currentSprint) => {
        if (selectedSprintOverride) {
          return (
            response.sprints.find(
              (sprint) => sprint.id === selectedSprintOverride.id,
            ) ?? selectedSprintOverride
          );
        }

        if (currentSprint) {
          const matchingSprint = response.sprints.find(
            (sprint) => sprint.id === currentSprint.id,
          );
          if (matchingSprint) {
            return matchingSprint;
          }
        }

        return response.sprints[0] ?? null;
      });
    } catch (err) {
      setSprintsError(apiErrorMessage(err, "Could not load sprints."));
    }
  }

  function syncIssueSprint(issue: Issue, sprintId: string | null) {
    const updatedIssue = { ...issue, sprint_id: sprintId };
    const updateIssueSprint = (currentIssue: Issue) =>
      currentIssue.id === issue.id ? updatedIssue : currentIssue;

    setIssues((currentIssues) => {
      const matchesFilters = issueMatchesFilters(
        updatedIssue,
        issueFilterProjectId,
        issueFilterSprintId,
        issueFilterStatus,
        issueFilterWorkflowStatusId,
        issueFilterPriority,
        issueFilterAssigneeId,
        issueFilterLabelId,
        issueFilterDue,
        issueFilterQuery,
        today,
      );
      const hasIssue = currentIssues.some(
        (currentIssue) => currentIssue.id === updatedIssue.id,
      );

      if (!matchesFilters) {
        return currentIssues.filter(
          (currentIssue) => currentIssue.id !== updatedIssue.id,
        );
      }
      if (!hasIssue) {
        return [updatedIssue, ...currentIssues];
      }

      return currentIssues.map(updateIssueSprint);
    });
    setSprintPlanningIssues((currentIssues) =>
      currentIssues.map(updateIssueSprint),
    );
    setDashboardSprintIssues((currentIssues) => {
      if (sprintId === null) {
        return currentIssues.filter(
          (currentIssue) => currentIssue.id !== updatedIssue.id,
        );
      }

      return currentIssues.map(updateIssueSprint);
    });
    setIssueChildren((currentIssues) => currentIssues.map(updateIssueSprint));
    setSelectedIssue((currentIssue) =>
      currentIssue?.id === issue.id ? updateIssueSprint(currentIssue) : currentIssue,
    );
  }

  function startEditingSprint(sprint: Sprint) {
    setSelectedSprintError("");
    setEditSprintName(sprint.name);
    setEditSprintGoal(sprint.goal);
    setEditSprintStartDate(sprint.start_date ?? "");
    setEditSprintEndDate(sprint.end_date ?? "");
    setIsEditingSprintDetails(true);
  }

  function cancelEditingSprint() {
    setSelectedSprintError("");
    setIsEditingSprintDetails(false);
    setEditSprintName("");
    setEditSprintGoal("");
    setEditSprintStartDate("");
    setEditSprintEndDate("");
  }

  async function loadSprintDetail(sprintId: string) {
    const sprintPreview = sprints.find((sprint) => sprint.id === sprintId);
    if (sprintPreview) {
      setSelectedSprint(sprintPreview);
    }

    setSelectedSprintError("");
    setIsEditingSprintDetails(false);
    setIsLoadingSelectedSprint(true);

    try {
      const sprint = await getSprint(sprintId);
      setSelectedSprint(sprint);
      upsertSprintInList(sprint);
    } catch (err) {
      setSelectedSprintError(apiErrorMessage(err, "Could not load sprint details."));
    } finally {
      setIsLoadingSelectedSprint(false);
    }
  }

  async function handleSelectSprint(sprintId: string) {
    navigateToSprint(sprintId);
    await loadSprintDetail(sprintId);
  }

  async function handleCreateSprint(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSprintFormError("");

    const formData = new FormData(event.currentTarget);
    const projectId = formString(formData, "sprint-project-id", sprintProjectId);
    const name = normalizeText(formString(formData, "sprint-name", sprintName));
    const goal = normalizeText(formString(formData, "sprint-goal", sprintGoal));
    const startDate = formString(
      formData,
      "sprint-start-date",
      sprintStartDate,
    ).trim();
    const endDate = formString(formData, "sprint-end-date", sprintEndDate).trim();

    if (!projectId) {
      setSprintFormError("Choose a project.");
      return;
    }
    if (!hasText(name)) {
      setSprintFormError("Sprint name is required.");
      return;
    }
    if (startDate && endDate && startDate > endDate) {
      setSprintFormError("Start date must be before or equal to end date.");
      return;
    }

    setIsCreatingSprint(true);

    try {
      const sprint = await createSprint({
        project_id: projectId,
        name,
        goal,
        start_date: startDate,
        end_date: endDate,
      });

      setSelectedSprint(sprint);
      navigateToSprint(sprint.id);
      setSprintName("");
      setSprintGoal("");
      setSprintStartDate("");
      setSprintEndDate("");
      setSprintFilterProjectId(sprint.project_id);
      setSprintFilterStatus("");
      upsertSprintInList(sprint);
    } catch (err) {
      setSprintFormError(apiErrorMessage(err, "Could not create sprint."));
    } finally {
      setIsCreatingSprint(false);
    }
  }

  async function handleUpdateSprint(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedSprint) {
      return;
    }

    setSelectedSprintError("");

    const formData = new FormData(event.currentTarget);
    const name = normalizeText(
      formString(formData, "edit-sprint-name", editSprintName),
    );
    const goal = normalizeText(
      formString(formData, "edit-sprint-goal", editSprintGoal),
    );
    const startDate = formString(
      formData,
      "edit-sprint-start-date",
      editSprintStartDate,
    ).trim();
    const endDate = formString(
      formData,
      "edit-sprint-end-date",
      editSprintEndDate,
    ).trim();

    if (!hasText(name)) {
      setSelectedSprintError("Sprint name is required.");
      return;
    }
    if (startDate && endDate && startDate > endDate) {
      setSelectedSprintError("Start date must be before or equal to end date.");
      return;
    }

    setIsUpdatingSprint(true);

    try {
      const sprint = await updateSprint(selectedSprint.id, {
        name,
        goal,
        start_date: startDate,
        end_date: endDate,
      });
      setSelectedSprint(sprint);
      setIsEditingSprintDetails(false);
      upsertSprintInList(sprint);
    } catch (err) {
      setSelectedSprintError(apiErrorMessage(err, "Could not update sprint."));
    } finally {
      setIsUpdatingSprint(false);
    }
  }

  async function handleStartSprint(sprint: Sprint) {
    setSelectedSprintError("");
    setSprintsError("");
    setStartingSprintIds((currentIds) =>
      currentIds.includes(sprint.id) ? currentIds : [...currentIds, sprint.id],
    );

    try {
      const startedSprint = await startSprint(sprint.id);
      setSelectedSprint(startedSprint);
      await refreshSprints(startedSprint);
    } catch (err) {
      setSelectedSprintError(apiErrorMessage(err, "Could not start sprint."));
    } finally {
      setStartingSprintIds((currentIds) =>
        currentIds.filter((currentSprintId) => currentSprintId !== sprint.id),
      );
    }
  }

  async function handleCompleteSprint(sprint: Sprint) {
    setSelectedSprintError("");
    setSprintsError("");
    setCompletingSprintIds((currentIds) =>
      currentIds.includes(sprint.id) ? currentIds : [...currentIds, sprint.id],
    );

    try {
      const completedSprint = await completeSprint(sprint.id);
      setSelectedSprint(completedSprint);
      setIsEditingSprintDetails(false);
      await refreshSprints(completedSprint);
    } catch (err) {
      setSelectedSprintError(apiErrorMessage(err, "Could not complete sprint."));
    } finally {
      setCompletingSprintIds((currentIds) =>
        currentIds.filter((currentSprintId) => currentSprintId !== sprint.id),
      );
    }
  }

  async function handleAddIssueToSprint(issue: Issue) {
    if (!selectedSprint) {
      return;
    }

    setSprintPlanningError("");
    setSelectedSprintError("");
    setAddingIssueToSprintIds((currentIds) =>
      currentIds.includes(issue.id) ? currentIds : [...currentIds, issue.id],
    );

    try {
      const sprint = await addIssueToSprint(selectedSprint.id, issue.id);
      setSelectedSprint(sprint);
      upsertSprintInList(sprint);
      syncIssueSprint(issue, sprint.id);
    } catch (err) {
      setSprintPlanningError(apiErrorMessage(err, "Could not add issue to sprint."));
    } finally {
      setAddingIssueToSprintIds((currentIds) =>
        currentIds.filter((currentIssueId) => currentIssueId !== issue.id),
      );
    }
  }

  async function handleRemoveIssueFromSprint(issue: Issue) {
    if (!selectedSprint) {
      return;
    }

    setSprintPlanningError("");
    setSelectedSprintError("");
    setRemovingIssueFromSprintIds((currentIds) =>
      currentIds.includes(issue.id) ? currentIds : [...currentIds, issue.id],
    );

    try {
      await removeIssueFromSprint(selectedSprint.id, issue.id);
      const sprint = await getSprint(selectedSprint.id);
      setSelectedSprint(sprint);
      upsertSprintInList(sprint);
      syncIssueSprint(issue, null);
    } catch (err) {
      setSprintPlanningError(
        apiErrorMessage(err, "Could not remove issue from sprint."),
      );
    } finally {
      setRemovingIssueFromSprintIds((currentIds) =>
        currentIds.filter((currentIssueId) => currentIssueId !== issue.id),
      );
    }
  }

  function startEditingIssue(issue: Issue) {
    setSelectedIssueError("");
    setEditIssueTitle(issue.title);
    setEditIssueDescription(issue.description);
    setEditIssueType(issue.issue_type);
    setEditIssuePriority(issue.priority);
    setEditIssueStoryPoints(String(issue.story_points));
    setEditIssueDueDate(issue.due_date ?? "");
    setIsEditingIssueDetails(true);
  }

  async function handleUpdateSelectedIssue(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedIssue) {
      return;
    }

    const storyPoints = parseStoryPoints(editIssueStoryPoints);
    if (storyPoints === null) {
      setSelectedIssueError("Story points must be a whole number from 0 to 100.");
      return;
    }

    setSelectedIssueError("");
    setIsUpdatingIssue(true);

    try {
      const updatedIssue = await updateIssue(selectedIssue.id, {
        title: editIssueTitle,
        description: editIssueDescription,
        issue_type: editIssueType,
        priority: editIssuePriority,
        story_points: storyPoints,
        due_date: editIssueDueDate,
      });

      setIssues((currentIssues) => {
        if (
          !issueMatchesFilters(
            updatedIssue,
            issueFilterProjectId,
            issueFilterSprintId,
            issueFilterStatus,
            issueFilterWorkflowStatusId,
            issueFilterPriority,
            issueFilterAssigneeId,
            issueFilterLabelId,
            issueFilterDue,
            issueFilterQuery,
            today,
          )
        ) {
          return currentIssues.filter((issue) => issue.id !== updatedIssue.id);
        }

        return currentIssues.map((issue) =>
          issue.id === updatedIssue.id ? updatedIssue : issue,
        );
      });
      setDashboardSprintIssues((currentIssues) =>
        currentIssues.map((issue) =>
          issue.id === updatedIssue.id ? updatedIssue : issue,
        ),
      );
      setBoardIssues((currentIssues) =>
        currentIssues.map((issue) =>
          issue.id === updatedIssue.id ? updatedIssue : issue,
        ),
      );
      setSelectedIssue(updatedIssue);
      setIsEditingIssueDetails(false);
      await refreshIssueActivity(updatedIssue.id);
    } catch (err) {
      setSelectedIssueError(apiErrorMessage(err, "Could not update issue."));
    } finally {
      setIsUpdatingIssue(false);
    }
  }

  async function handleTransitionIssue(
    issueId: string,
    input: IssueStatus | TransitionIssueInput,
  ) {
    setIssuesError("");
    if (activeSection === "board") {
      setBoardError("");
    }
    setTransitioningIssueIds((currentIds) =>
      currentIds.includes(issueId) ? currentIds : [...currentIds, issueId],
    );

    try {
      const updatedIssue = await transitionIssue(issueId, input);
      setIssues((currentIssues) => {
        if (
          !issueMatchesFilters(
            updatedIssue,
            issueFilterProjectId,
            issueFilterSprintId,
            issueFilterStatus,
            issueFilterWorkflowStatusId,
            issueFilterPriority,
            issueFilterAssigneeId,
            issueFilterLabelId,
            issueFilterDue,
            issueFilterQuery,
            today,
          )
        ) {
          return currentIssues.filter((issue) => issue.id !== updatedIssue.id);
        }

        return currentIssues.map((issue) =>
          issue.id === updatedIssue.id ? updatedIssue : issue,
        );
      });
      setDashboardSprintIssues((currentIssues) =>
        currentIssues.map((issue) =>
          issue.id === updatedIssue.id ? updatedIssue : issue,
        ),
      );
      setSelectedIssue((currentIssue) =>
        currentIssue?.id === updatedIssue.id ? updatedIssue : currentIssue,
      );
      setSprintPlanningIssues((currentIssues) =>
        currentIssues.map((issue) =>
          issue.id === updatedIssue.id ? updatedIssue : issue,
        ),
      );
      setBoardIssues((currentIssues) =>
        currentIssues.map((issue) =>
          issue.id === updatedIssue.id ? updatedIssue : issue,
        ),
      );
      if (selectedIssue?.id === updatedIssue.id) {
        await refreshIssueActivity(updatedIssue.id);
      }
    } catch (err) {
      const message = apiErrorMessage(err, "Could not update issue status.");
      setIssuesError(message);
      if (activeSection === "board") {
        setBoardError(message);
      }
    } finally {
      setTransitioningIssueIds((currentIds) =>
        currentIds.filter((currentIssueId) => currentIssueId !== issueId),
      );
    }
  }

  async function handleAssignIssue(issueId: string, assigneeId: string) {
    setSelectedIssueError("");
    if (assigneeId) {
      const assignee = teamMembers.find((member) => member.id === assigneeId);
      if (!assignee?.is_active) {
        setSelectedIssueError("Choose an active assignee.");
        return;
      }
    }

    setAssigningIssueIds((currentIds) =>
      currentIds.includes(issueId) ? currentIds : [...currentIds, issueId],
    );

    try {
      const updatedIssue = await assignIssue(issueId, assigneeId);
      setIssues((currentIssues) => {
        if (
          !issueMatchesFilters(
            updatedIssue,
            issueFilterProjectId,
            issueFilterSprintId,
            issueFilterStatus,
            issueFilterWorkflowStatusId,
            issueFilterPriority,
            issueFilterAssigneeId,
            issueFilterLabelId,
            issueFilterDue,
            issueFilterQuery,
            today,
          )
        ) {
          return currentIssues.filter((issue) => issue.id !== updatedIssue.id);
        }

        return currentIssues.map((issue) =>
          issue.id === updatedIssue.id ? updatedIssue : issue,
        );
      });
      setDashboardSprintIssues((currentIssues) =>
        currentIssues.map((issue) =>
          issue.id === updatedIssue.id ? updatedIssue : issue,
        ),
      );
      setSelectedIssue((currentIssue) =>
        currentIssue?.id === updatedIssue.id ? updatedIssue : currentIssue,
      );
      if (selectedIssue?.id === updatedIssue.id) {
        await refreshIssueActivity(updatedIssue.id);
      }
    } catch (err) {
      setSelectedIssueError(apiErrorMessage(err, "Could not update assignee."));
    } finally {
      setAssigningIssueIds((currentIds) =>
        currentIds.filter((currentIssueId) => currentIssueId !== issueId),
      );
    }
  }

  async function handleSetIssueLabel(
    issue: Issue,
    labelId: string,
    shouldAttach: boolean,
  ) {
    setSelectedIssueError("");
    setLabelingIssueIds((currentIds) =>
      currentIds.includes(issue.id) ? currentIds : [...currentIds, issue.id],
    );

    const currentLabelIds = issueLabelIds(issue);
    const nextLabelIds = shouldAttach
      ? Array.from(new Set([...currentLabelIds, labelId]))
      : currentLabelIds.filter((currentLabelId) => currentLabelId !== labelId);

    try {
      const updatedIssue = await setIssueLabels(issue.id, nextLabelIds);
      setIssues((currentIssues) => {
        if (
          !issueMatchesFilters(
            updatedIssue,
            issueFilterProjectId,
            issueFilterSprintId,
            issueFilterStatus,
            issueFilterWorkflowStatusId,
            issueFilterPriority,
            issueFilterAssigneeId,
            issueFilterLabelId,
            issueFilterDue,
            issueFilterQuery,
            today,
          )
        ) {
          return currentIssues.filter((currentIssue) => currentIssue.id !== updatedIssue.id);
        }

        return currentIssues.map((currentIssue) =>
          currentIssue.id === updatedIssue.id ? updatedIssue : currentIssue,
        );
      });
      setSelectedIssue((currentIssue) =>
        currentIssue?.id === updatedIssue.id ? updatedIssue : currentIssue,
      );
      await refreshIssueActivity(updatedIssue.id);
    } catch (err) {
      setSelectedIssueError(apiErrorMessage(err, "Could not update labels."));
    } finally {
      setLabelingIssueIds((currentIds) =>
        currentIds.filter((currentIssueId) => currentIssueId !== issue.id),
      );
    }
  }

  async function handleArchiveIssue(issue: Issue) {
    if (!window.confirm(`Archive issue ${issue.issue_key}?`)) {
      return;
    }

    setIssuesError("");
    setSelectedIssueError("");
    setArchivingIssueIds((currentIds) =>
      currentIds.includes(issue.id) ? currentIds : [...currentIds, issue.id],
    );

    try {
      await archiveIssue(issue.id);
      setIssues((currentIssues) =>
        currentIssues.filter((currentIssue) => currentIssue.id !== issue.id),
      );
      setBoardIssues((currentIssues) =>
        currentIssues.filter((currentIssue) => currentIssue.id !== issue.id),
      );
      setSprintPlanningIssues((currentIssues) =>
        currentIssues.filter((currentIssue) => currentIssue.id !== issue.id),
      );
      setDashboardSprintIssues((currentIssues) =>
        currentIssues.filter((currentIssue) => currentIssue.id !== issue.id),
      );
      setSelectedIssue((currentIssue) =>
        currentIssue?.id === issue.id ? null : currentIssue,
      );
      if (selectedIssue?.id === issue.id) {
        setIssueComments([]);
        setIssueActivity([]);
        setIssueChildren([]);
        setHierarchyError("");
        setSubtaskFormError("");
        setSubtaskTitle("");
        setSubtaskPriority("medium");
        setSubtaskWorkflowStatusId("");
        setIssueLinks([]);
        setLinksError("");
        setLinkFormError("");
        setIsLoadingIssueLinks(false);
        setIsCreatingIssueLink(false);
        setDeletingIssueLinkIds([]);
        setLinkTargetIssueId("");
        setLinkType("relates");
        setCommentBody("");
        setEditingCommentId("");
        setEditCommentBody("");
        setUpdatingCommentIds([]);
        setDeletingCommentIds([]);
        setIsEditingIssueDetails(false);
      }
    } catch (err) {
      const message = apiErrorMessage(err, "Could not archive issue.");
      setIssuesError(message);
      setSelectedIssueError(message);
    } finally {
      setArchivingIssueIds((currentIds) =>
        currentIds.filter((currentIssueId) => currentIssueId !== issue.id),
      );
    }
  }

  function handleCreateIssueLabel(labelId: string, shouldAttach: boolean) {
    setNewIssueLabelIds((currentIds) =>
      shouldAttach
        ? Array.from(new Set([...currentIds, labelId]))
        : currentIds.filter((currentId) => currentId !== labelId),
    );
  }

  async function handleSelectIssue(issueId: string) {
    navigateToSection("issues");

    const issuePreview = issues.find((issue) => issue.id === issueId);
    if (issuePreview) {
      setSelectedIssue(issuePreview);
    }

    setSelectedIssueError("");
    setIsEditingIssueDetails(false);
    setEditingCommentId("");
    setEditCommentBody("");
    setUpdatingCommentIds([]);
    setDeletingCommentIds([]);
    setIsLoadingSelectedIssue(true);

    try {
      const issue = await getIssue(issueId);
      setSelectedIssue(issue);
    } catch (err) {
      setSelectedIssueError(apiErrorMessage(err, "Could not load issue details."));
    } finally {
      setIsLoadingSelectedIssue(false);
    }
  }

  async function handleCreateIssue(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIssueFormError("");
    if (selectedProjectId === "") {
      setIssueFormError("Choose a project.");
      return;
    }
    if (!selectedCreateProject?.can_write) {
      setIssueFormError("You do not have permission to create issues in this project.");
      return;
    }
    if (
      !selectedCreateWorkflowStatuses.some(
        (status) => status.id === issueWorkflowStatusId,
      )
    ) {
      setIssueFormError("Choose an active project status.");
      return;
    }
    if (!hasText(issueTitle)) {
      setIssueFormError("Issue title is required.");
      return;
    }
    if (issueAssigneeId) {
      const assignee = teamMembers.find((member) => member.id === issueAssigneeId);
      if (!assignee?.is_active) {
        setIssueFormError("Choose an active assignee.");
        return;
      }
    }
    const storyPoints = parseStoryPoints(issueStoryPoints);
    if (storyPoints === null) {
      setIssueFormError("Story points must be a whole number from 0 to 100.");
      return;
    }

    setIsCreatingIssue(true);

    try {
      const issue = await createIssue({
        project_id: selectedProjectId,
        title: issueTitle,
        description: issueDescription,
        issue_type: issueType,
        workflow_status_id: issueWorkflowStatusId,
        priority: issuePriority,
        story_points: storyPoints,
        assignee_id: issueAssigneeId,
        due_date: issueDueDate,
        label_ids: newIssueLabelIds,
      });

      if (
        issueMatchesFilters(
          issue,
          issueFilterProjectId,
          issueFilterSprintId,
          issueFilterStatus,
          issueFilterWorkflowStatusId,
          issueFilterPriority,
          issueFilterAssigneeId,
          issueFilterLabelId,
          issueFilterDue,
          issueFilterQuery,
          today,
        )
      ) {
        setIssues((currentIssues) => [issue, ...currentIssues]);
      }
      setSelectedIssue(issue);
      setIsEditingIssueDetails(false);
      setIssueTitle("");
      setIssueDescription("");
      setIssueType("task");
      setIssuePriority("medium");
      setIssueStoryPoints("0");
      setIssueWorkflowStatusId(
        defaultWorkflowStatus(workflowsByProjectId[selectedProjectId])?.id ?? "",
      );
      setIssueAssigneeId("");
      setIssueDueDate("");
      setNewIssueLabelIds([]);
    } catch (err) {
      setIssueFormError(apiErrorMessage(err, "Could not create issue."));
    } finally {
      setIsCreatingIssue(false);
    }
  }

  async function handleCreateSubtask(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedIssue) {
      return;
    }

    setSubtaskFormError("");
    setHierarchyError("");

    const title = normalizeText(subtaskTitle);
    if (!hasText(title)) {
      setSubtaskFormError("Subtask title is required.");
      return;
    }
    if (!canWriteSelectedIssue) {
      setSubtaskFormError("You do not have permission to create subtasks.");
      return;
    }
    if (
      !selectedIssueWorkflowStatuses.some(
        (status) => status.id === subtaskWorkflowStatusId,
      )
    ) {
      setSubtaskFormError("Choose an active project status.");
      return;
    }
    const storyPoints = parseStoryPoints(subtaskStoryPoints);
    if (storyPoints === null) {
      setSubtaskFormError("Story points must be a whole number from 0 to 100.");
      return;
    }

    setIsCreatingSubtask(true);

    try {
      const subtask = await createSubtask(selectedIssue.id, {
        title,
        description: "",
        workflow_status_id: subtaskWorkflowStatusId,
        priority: subtaskPriority,
        story_points: storyPoints,
        assignee_id: "",
        due_date: "",
        label_ids: [],
      });

      setIssueChildren((currentChildren) => [...currentChildren, subtask]);
      setIssues((currentIssues) => {
        if (
          !issueMatchesFilters(
            subtask,
            issueFilterProjectId,
            issueFilterSprintId,
            issueFilterStatus,
            issueFilterWorkflowStatusId,
            issueFilterPriority,
            issueFilterAssigneeId,
            issueFilterLabelId,
            issueFilterDue,
            issueFilterQuery,
            today,
          )
        ) {
          return currentIssues;
        }

        return [subtask, ...currentIssues];
      });
      setSubtaskTitle("");
      setSubtaskPriority("medium");
      setSubtaskStoryPoints("0");
      setSubtaskWorkflowStatusId(
        defaultWorkflowStatus(workflowsByProjectId[selectedIssue.project_id])
          ?.id ?? "",
      );
      await refreshIssueChildren(selectedIssue.id);
    } catch (err) {
      setSubtaskFormError(apiErrorMessage(err, "Could not create subtask."));
    } finally {
      setIsCreatingSubtask(false);
    }
  }

  async function handleCreateIssueLink(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedIssue) {
      return;
    }

    setLinkFormError("");
    setLinksError("");

    if (!linkTargetIssueId) {
      setLinkFormError("Choose a target issue.");
      return;
    }

    setIsCreatingIssueLink(true);

    try {
      const link = await createIssueLink(selectedIssue.id, {
        target_issue_id: linkTargetIssueId,
        link_type: linkType,
      });

      setIssueLinks((currentLinks) => [link, ...currentLinks]);
      setLinkTargetIssueId("");
      setLinkType("relates");
      await refreshIssueLinks(selectedIssue.id);
      await refreshIssueActivity(selectedIssue.id);
    } catch (err) {
      setLinkFormError(apiErrorMessage(err, "Could not add issue link."));
    } finally {
      setIsCreatingIssueLink(false);
    }
  }

  async function handleDeleteIssueLink(link: IssueLink) {
    if (!selectedIssue) {
      return;
    }
    if (!window.confirm("Remove this issue link?")) {
      return;
    }

    setLinksError("");
    setDeletingIssueLinkIds((currentIds) =>
      currentIds.includes(link.id) ? currentIds : [...currentIds, link.id],
    );

    try {
      await deleteIssueLink(selectedIssue.id, link.id);
      setIssueLinks((currentLinks) =>
        currentLinks.filter((currentLink) => currentLink.id !== link.id),
      );
      await refreshIssueActivity(selectedIssue.id);
    } catch (err) {
      setLinksError(apiErrorMessage(err, "Could not remove issue link."));
    } finally {
      setDeletingIssueLinkIds((currentIds) =>
        currentIds.filter((currentLinkId) => currentLinkId !== link.id),
      );
    }
  }

  async function handleCreateComment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedIssue) {
      return;
    }

    setCommentsError("");
    const body = normalizeText(commentBody);
    if (!hasText(body)) {
      setCommentsError("Comment body is required.");
      return;
    }

    setIsCreatingComment(true);

    try {
      const comment = await createIssueComment(selectedIssue.id, body);
      setIssueComments((currentComments) => [...currentComments, comment]);
      setCommentBody("");
      await refreshIssueActivity(selectedIssue.id);
    } catch (err) {
      setCommentsError(apiErrorMessage(err, "Could not create comment."));
    } finally {
      setIsCreatingComment(false);
    }
  }

  function startEditingComment(comment: IssueComment) {
    setCommentsError("");
    setEditingCommentId(comment.id);
    setEditCommentBody(comment.body);
  }

  function cancelEditingComment() {
    setEditingCommentId("");
    setEditCommentBody("");
  }

  async function handleUpdateComment(
    event: FormEvent<HTMLFormElement>,
    comment: IssueComment,
  ) {
    event.preventDefault();
    if (!selectedIssue) {
      return;
    }

    setCommentsError("");
    const body = normalizeText(editCommentBody);
    if (!hasText(body)) {
      setCommentsError("Comment body is required.");
      return;
    }

    setUpdatingCommentIds((currentIds) =>
      currentIds.includes(comment.id) ? currentIds : [...currentIds, comment.id],
    );

    try {
      const updatedComment = await updateIssueComment(
        selectedIssue.id,
        comment.id,
        body,
      );
      setIssueComments((currentComments) =>
        currentComments.map((currentComment) =>
          currentComment.id === updatedComment.id ? updatedComment : currentComment,
        ),
      );
      setEditingCommentId("");
      setEditCommentBody("");
      await refreshIssueActivity(selectedIssue.id);
    } catch (err) {
      setCommentsError(apiErrorMessage(err, "Could not update comment."));
    } finally {
      setUpdatingCommentIds((currentIds) =>
        currentIds.filter((currentCommentId) => currentCommentId !== comment.id),
      );
    }
  }

  async function handleDeleteComment(comment: IssueComment) {
    if (!selectedIssue) {
      return;
    }
    if (!window.confirm("Delete this comment? This cannot be undone.")) {
      return;
    }

    setCommentsError("");
    setDeletingCommentIds((currentIds) =>
      currentIds.includes(comment.id) ? currentIds : [...currentIds, comment.id],
    );

    try {
      await deleteIssueComment(selectedIssue.id, comment.id);
      setIssueComments((currentComments) =>
        currentComments.filter((currentComment) => currentComment.id !== comment.id),
      );
      if (editingCommentId === comment.id) {
        setEditingCommentId("");
        setEditCommentBody("");
      }
      await refreshIssueActivity(selectedIssue.id);
    } catch (err) {
      setCommentsError(apiErrorMessage(err, "Could not delete comment."));
    } finally {
      setDeletingCommentIds((currentIds) =>
        currentIds.filter((currentCommentId) => currentCommentId !== comment.id),
      );
    }
  }

  function handleIssueDragStart(
    event: DragEvent<HTMLElement>,
    issueId: string,
  ) {
    event.dataTransfer.setData("text/plain", issueId);
    event.dataTransfer.effectAllowed = "move";
  }

  function handleIssueDrop(
    event: DragEvent<HTMLElement>,
    workflowStatusId: string,
  ) {
    event.preventDefault();

    const issueId = event.dataTransfer.getData("text/plain");
    const issue = boardIssues.find((currentIssue) => currentIssue.id === issueId);
    if (!issue || issue.workflow_status?.id === workflowStatusId) {
      return;
    }

    void handleTransitionIssue(issue.id, { workflow_status_id: workflowStatusId });
  }

  function handleSprintIssueDrop(
    event: DragEvent<HTMLElement>,
    workflowStatusId: string,
  ) {
    event.preventDefault();

    const issueId = event.dataTransfer.getData("text/plain");
    const issue = sprintPlanningIssues.find(
      (currentIssue) => currentIssue.id === issueId,
    );
    if (!issue || issue.workflow_status?.id === workflowStatusId) {
      return;
    }

    void handleTransitionIssue(issue.id, { workflow_status_id: workflowStatusId });
  }

  function currentSavedIssueFilters() {
    return savedIssueFiltersFromState({
      query: issueFilterQuery,
      sort: issueSort,
      projectId: issueFilterProjectId,
      sprintId: issueFilterSprintId,
      status: issueFilterStatus,
      workflowStatusId: issueFilterWorkflowStatusId,
      priority: issueFilterPriority,
      assigneeId: issueFilterAssigneeId,
      labelId: issueFilterLabelId,
      due: issueFilterDue,
    });
  }

  function applySavedIssueFilters(savedFilter: SavedFilter) {
    const nextState = savedIssueFilterStateFromFilters(savedFilter.filters);
    setIssueFilterQuery(nextState.query);
    setIssueSort(nextState.sort);
    setIssueFilterProjectId(nextState.projectId);
    setIssueFilterSprintId(nextState.sprintId);
    setIssueFilterStatus(nextState.status);
    setIssueFilterWorkflowStatusId(nextState.workflowStatusId);
    setIssueFilterPriority(nextState.priority);
    setIssueFilterAssigneeId(nextState.assigneeId);
    setIssueFilterLabelId(nextState.labelId);
    setIssueFilterDue(nextState.due);
    setSavedFilterFormError("");
  }

  async function handleCreateSavedFilter(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const name = savedFilterName.trim();
    if (name === "") {
      setSavedFilterFormError("Saved filter name is required.");
      return;
    }
    if (name.length > 60) {
      setSavedFilterFormError("Saved filter name must be 60 characters or fewer.");
      return;
    }

    setSavedFilterFormError("");
    setIsCreatingSavedFilter(true);

    try {
      const savedFilter = await createSavedFilter({
        name,
        filters: currentSavedIssueFilters(),
      });
      setSavedFilters((currentSavedFilters) =>
        upsertSavedFilter(currentSavedFilters, savedFilter),
      );
      setSavedFilterName("");
    } catch (err) {
      setSavedFilterFormError(apiErrorMessage(err, "Could not save filter."));
    } finally {
      setIsCreatingSavedFilter(false);
    }
  }

  async function handleUpdateSavedFilter(savedFilter: SavedFilter) {
    setSavedFilterFormError("");
    setUpdatingSavedFilterIds((currentIds) => [...currentIds, savedFilter.id]);

    try {
      const updated = await updateSavedFilter(savedFilter.id, {
        name: savedFilter.name,
        filters: currentSavedIssueFilters(),
      });
      setSavedFilters((currentSavedFilters) =>
        upsertSavedFilter(currentSavedFilters, updated),
      );
    } catch (err) {
      setSavedFilterFormError(apiErrorMessage(err, "Could not update filter."));
    } finally {
      setUpdatingSavedFilterIds((currentIds) =>
        currentIds.filter((currentId) => currentId !== savedFilter.id),
      );
    }
  }

  function handleStartRenameSavedFilter(savedFilter: SavedFilter) {
    setRenameSavedFilterId(savedFilter.id);
    setRenameSavedFilterName(savedFilter.name);
    setSavedFilterFormError("");
  }

  async function handleRenameSavedFilter(savedFilter: SavedFilter) {
    const name = renameSavedFilterName.trim();
    if (name === "") {
      setSavedFilterFormError("Saved filter name is required.");
      return;
    }
    if (name.length > 60) {
      setSavedFilterFormError("Saved filter name must be 60 characters or fewer.");
      return;
    }

    setSavedFilterFormError("");
    setUpdatingSavedFilterIds((currentIds) => [...currentIds, savedFilter.id]);

    try {
      const updated = await updateSavedFilter(savedFilter.id, {
        name,
        filters: savedFilter.filters,
      });
      setSavedFilters((currentSavedFilters) =>
        upsertSavedFilter(currentSavedFilters, updated),
      );
      setRenameSavedFilterId("");
      setRenameSavedFilterName("");
    } catch (err) {
      setSavedFilterFormError(apiErrorMessage(err, "Could not rename filter."));
    } finally {
      setUpdatingSavedFilterIds((currentIds) =>
        currentIds.filter((currentId) => currentId !== savedFilter.id),
      );
    }
  }

  async function handleDeleteSavedFilter(savedFilter: SavedFilter) {
    setSavedFilterFormError("");
    setDeletingSavedFilterIds((currentIds) => [...currentIds, savedFilter.id]);

    try {
      await deleteSavedFilter(savedFilter.id);
      setSavedFilters((currentSavedFilters) =>
        currentSavedFilters.filter(
          (currentSavedFilter) => currentSavedFilter.id !== savedFilter.id,
        ),
      );
      if (renameSavedFilterId === savedFilter.id) {
        setRenameSavedFilterId("");
        setRenameSavedFilterName("");
      }
    } catch (err) {
      setSavedFilterFormError(apiErrorMessage(err, "Could not delete filter."));
    } finally {
      setDeletingSavedFilterIds((currentIds) =>
        currentIds.filter((currentId) => currentId !== savedFilter.id),
      );
    }
  }

  async function refreshNotifications() {
    setNotificationsError("");
    setIsLoadingNotifications(true);

    try {
      const [notificationsResponse, unreadResponse] = await Promise.all([
        listNotifications(),
        getUnreadNotificationsCount(),
      ]);
      setNotifications(notificationsResponse.notifications);
      setUnreadNotificationsCount(unreadResponse.unread_count);
    } catch (err) {
      setNotificationsError(apiErrorMessage(err, "Could not load notifications."));
    } finally {
      setIsLoadingNotifications(false);
    }
  }

  async function handleMarkNotificationRead(notification: AppNotification) {
    if (notification.read_at !== null) {
      return;
    }

    setNotificationsError("");

    try {
      const updatedNotification = await markNotificationRead(notification.id);
      setNotifications((currentNotifications) =>
        currentNotifications.map((currentNotification) =>
          currentNotification.id === updatedNotification.id
            ? updatedNotification
            : currentNotification,
        ),
      );
      const unreadResponse = await getUnreadNotificationsCount();
      setUnreadNotificationsCount(unreadResponse.unread_count);
    } catch (err) {
      setNotificationsError(apiErrorMessage(err, "Could not mark notification read."));
    }
  }

  async function handleMarkAllNotificationsRead() {
    setNotificationsError("");

    try {
      await markAllNotificationsRead();
      await refreshNotifications();
    } catch (err) {
      setNotificationsError(apiErrorMessage(err, "Could not mark notifications read."));
    }
  }

  async function handleOpenNotificationIssue(notification: AppNotification) {
    if (notification.read_at === null) {
      await handleMarkNotificationRead(notification);
    }
    setIsNotificationsOpen(false);
    if (notification.issue_id) {
      await handleSelectIssue(notification.issue_id);
    } else {
      navigateToSection("notifications");
    }
  }

  function handleIssueProjectFilterChange(projectId: string) {
    setIssueFilterProjectId(projectId);
    setIssueFilterStatus("");
    setIssueFilterWorkflowStatusId("");
    setIssueFilterSprintId((currentSprintId) => {
      if (!currentSprintId || currentSprintId === "none" || !projectId) {
        return currentSprintId;
      }

      const sprint = issueFilterSprints.find(
        (currentSprint) => currentSprint.id === currentSprintId,
      );
      return sprint?.project_id === projectId ? currentSprintId : "";
    });
  }

  function handleIssueSprintFilterChange(sprintId: string) {
    setIssueFilterSprintId(sprintId);
    if (!sprintId || sprintId === "none") {
      return;
    }

    const sprint = issueFilterSprints.find(
      (currentSprint) => currentSprint.id === sprintId,
    );
    if (sprint) {
      if (sprint.project_id !== issueFilterProjectId) {
        setIssueFilterStatus("");
        setIssueFilterWorkflowStatusId("");
      }
      setIssueFilterProjectId(sprint.project_id);
    }
  }

  function handleIssueWorkflowStatusFilterChange(workflowStatusId: string) {
    setIssueFilterStatus("");
    setIssueFilterWorkflowStatusId(workflowStatusId);
  }

  function handleCreateIssueProjectChange(projectId: string) {
    setSelectedProjectId(projectId);
    setIssueWorkflowStatusId("");
  }

  const today = startOfToday();
  const openIssues = issues.filter((issue) => !isIssueDone(issue));
  const selectedProjectIssues = selectedProjectDetail
    ? issues.filter((issue) => issue.project_id === selectedProjectDetail.id)
    : [];
  const selectedProjectOpenIssues = selectedProjectIssues.filter(
    (issue) => !isIssueDone(issue),
  );
  const selectedSprintIssues = selectedSprint
    ? sprintPlanningIssues.filter((issue) => issue.sprint_id === selectedSprint.id)
    : [];
  const selectedSprintBacklogIssues = selectedSprint
    ? sprintPlanningIssues.filter(
        (issue) =>
          issue.project_id === selectedSprint.project_id &&
          issue.sprint_id === null &&
          !isIssueDone(issue),
      )
    : [];
  const issueFilterVisibleSprints = issueFilterProjectId
    ? issueFilterSprints.filter((sprint) => sprint.project_id === issueFilterProjectId)
    : issueFilterSprints;
  const activeDashboardSprint =
    issueFilterSprints.find((sprint) => sprint.status === "active") ?? null;
  const overdueIssuesCount = openIssues.filter(
    (issue) => issueDueInfo(issue, today)?.tone === "overdue",
  ).length;
  const dueSoonIssuesCount = openIssues.filter(
    (issue) => issueDueInfo(issue, today)?.tone === "due-soon",
  ).length;
  const openIssuesCount = openIssues.length;
  const canSignIn =
    hasText(loginValue) && hasText(password) && !isSubmitting;
  const canUpdateProfile =
    user !== null &&
    hasText(accountDisplayName) &&
    normalizeText(accountDisplayName) !== user.display_name &&
    !isUpdatingProfile;
  const canChangePassword =
    hasText(currentPassword) &&
    hasMinTrimmedLength(newPassword, 8) &&
    hasMinTrimmedLength(confirmNewPassword, 8) &&
    !isChangingPassword;
  const canCreateProject =
    hasText(projectKey) && hasText(projectName) && !isCreatingProject;
  const canCreateTeamMember =
    isValidEmail(teamMemberEmail) &&
    isValidUsername(teamMemberUsername) &&
    hasText(teamMemberDisplayName) &&
    hasMinTrimmedLength(teamMemberPassword, 8) &&
    !isCreatingTeamMember;
  const canCreateTeamInvite =
    isValidEmail(teamInviteEmail) && !isCreatingTeamInvite;
  const canResetTeamMemberPassword = hasMinTrimmedLength(
    teamMemberResetPassword,
    8,
  );
  const canCreateLabel =
    hasText(labelName) &&
    isValidLabelColor(labelColor) &&
    !isCreatingLabel;
  const issueStoryPointsValue = parseStoryPoints(issueStoryPoints);
  const subtaskStoryPointsValue = parseStoryPoints(subtaskStoryPoints);
  const selectedCreateProject =
    projects.find((project) => project.id === selectedProjectId) ?? null;
  const selectedIssueProject = selectedIssue
    ? projects.find((project) => project.id === selectedIssue.project_id) ?? null
    : null;
  const selectedCreateWorkflow = workflowsByProjectId[selectedProjectId];
  const selectedFilterWorkflow = workflowsByProjectId[issueFilterProjectId];
  const selectedIssueWorkflow = selectedIssue
    ? workflowsByProjectId[selectedIssue.project_id]
    : undefined;
  const selectedBoardWorkflow = workflowsByProjectId[boardProjectId];
  const selectedSprintWorkflow = selectedSprint
    ? workflowsByProjectId[selectedSprint.project_id]
    : undefined;
  const selectedCreateWorkflowStatuses = activeWorkflowStatuses(
    selectedCreateWorkflow,
  );
  const selectedFilterWorkflowStatuses = activeWorkflowStatuses(
    selectedFilterWorkflow,
  );
  const selectedIssueWorkflowStatuses = activeWorkflowStatuses(
    selectedIssueWorkflow,
  );
  const selectedIssueTransitionStatuses = selectedIssue
      ? allowedTransitionStatuses(
          selectedIssueWorkflow,
          selectedIssue.workflow_status?.id ?? "",
        )
    : [];
  const workflowStatusNamesById = Object.fromEntries(
    Object.values(workflowsByProjectId).flatMap((workflow) =>
      activeWorkflowStatuses(workflow).map((status) => [status.id, status.name]),
    ),
  );
  const canWriteSelectedIssue = selectedIssueProject?.can_write ?? false;
  const canCreateIssue =
    selectedCreateProject?.can_write === true &&
    selectedCreateWorkflowStatuses.some(
      (status) => status.id === issueWorkflowStatusId,
    ) &&
    hasText(issueTitle) &&
    issueStoryPointsValue !== null &&
    !isCreatingIssue;
  const canCreateSprint =
    sprintProjectId !== "" && hasText(sprintName) && !isCreatingSprint;
  const canUpdateSprint =
    selectedSprint !== null &&
    selectedSprint.status !== "completed" &&
    hasText(editSprintName) &&
    !isUpdatingSprint;
  const canCreateComment =
    canWriteSelectedIssue && hasText(commentBody) && !isCreatingComment;
  const canCreateSubtask =
    canWriteSelectedIssue &&
    selectedIssueWorkflowStatuses.some(
      (status) => status.id === subtaskWorkflowStatusId,
    ) &&
    hasText(subtaskTitle) &&
    subtaskStoryPointsValue !== null &&
    !isCreatingSubtask;
  const availableLinkIssues = selectedIssue
    ? issues.filter((issue) => issue.id !== selectedIssue.id)
    : [];
  const canCreateIssueLink =
    canWriteSelectedIssue &&
    selectedIssue !== null &&
    linkTargetIssueId !== "" &&
    linkTargetIssueId !== selectedIssue.id &&
    !isCreatingIssueLink;
  const selectedParentIssue =
    selectedIssue?.parent_issue_id
      ? issues.find((issue) => issue.id === selectedIssue.parent_issue_id) ?? null
      : null;
  const activeSectionTitle =
    appSections.find((section) => section.id === activeSection)?.title ?? "Dashboard";
  const activeSectionSubtitle =
    activeSection === "dashboard" ? "Local workspace" : "Workspace section";
  const activeSectionHeading =
    activeSection === "dashboard"
      ? `Good to see you, ${user?.display_name ?? "there"}`
      : activeSectionTitle;

  if (isBooting) {
    return <BootingScreen />;
  }

  if (inviteAcceptToken !== null) {
    return (
      <InviteAcceptScreen
        onGoToSignIn={handlePublicAuthSignIn}
        token={inviteAcceptToken}
      />
    );
  }

  if (isForgotPasswordRoute) {
    return <ForgotPasswordScreen onGoToSignIn={handlePublicAuthSignIn} />;
  }

  if (passwordResetToken !== null) {
    return (
      <ResetPasswordScreen
        onGoToSignIn={handlePublicAuthSignIn}
        onResetCompleted={handlePasswordResetCompleted}
        token={passwordResetToken}
      />
    );
  }

  if (!user) {
    return (
      <SignInScreen
        canSignIn={canSignIn}
        error={error}
        isSubmitting={isSubmitting}
        loginValue={loginValue}
        onForgotPassword={navigateToForgotPassword}
        onLoginChange={setLoginValue}
        onPasswordChange={setPassword}
        onSubmit={handleLogin}
        password={password}
      />
    );
  }

  return (
    <div className="kl-app">
      <AppSidebar
        activeSection={activeSection}
        displayName={user.display_name}
        isLoggingOut={isLoggingOut}
        onNavigate={(section) => {
          navigateToSection(section);
          if (section === "notifications") {
            void refreshNotifications();
          }
        }}
        onSignOut={handleLogout}
        role={user.workspace.role}
        unreadNotificationsCount={unreadNotificationsCount}
      />

      <div className="kl-app__main">
        <WorkspaceTopbar
          heading={activeSectionHeading}
          subtitle={activeSectionSubtitle}
        />

        <div className="kl-app__content">

        <DashboardSection
          activeSprint={activeDashboardSprint}
          activeSprintError={dashboardSprintError}
          activeSprintIssues={dashboardSprintIssues}
          dueSoonIssuesCount={dueSoonIssuesCount}
          isActive={activeSection === "dashboard"}
          isLoadingActiveSprint={isLoadingDashboardSprint}
          onNavigate={navigateToSection}
          openIssuesCount={openIssuesCount}
          overdueIssuesCount={overdueIssuesCount}
          projectsCount={projects.length}
          role={user.workspace.role}
          teamMembers={teamMembers}
          teamMembersCount={teamMembers.length}
        />

        <NotificationsSection
          error={notificationsError}
          isActive={activeSection === "notifications"}
          isLoading={isLoadingNotifications}
          notifications={notifications}
          onMarkAllRead={() => {
            void handleMarkAllNotificationsRead();
          }}
          onMarkRead={(notification) => {
            void handleMarkNotificationRead(notification);
          }}
          onOpenIssue={(notification) => {
            void handleOpenNotificationIssue(notification);
          }}
          unreadCount={unreadNotificationsCount}
        />

        <AccountSection
          accountDisplayName={accountDisplayName}
          accountError={accountError}
          accountSuccess={accountSuccess}
          canChangePassword={canChangePassword}
          canUpdateProfile={canUpdateProfile}
          confirmNewPassword={confirmNewPassword}
          currentPassword={currentPassword}
          isActive={activeSection === "account"}
          isChangingPassword={isChangingPassword}
          isLoadingRuntimeVersion={isLoadingRuntimeVersion}
          isUpdatingProfile={isUpdatingProfile}
          newPassword={newPassword}
          onChangePassword={handleChangePassword}
          onConfirmNewPasswordChange={setConfirmNewPassword}
          onCurrentPasswordChange={setCurrentPassword}
          onDisplayNameChange={setAccountDisplayName}
          onNewPasswordChange={setNewPassword}
          onUpdateProfile={handleUpdateProfile}
          runtimeVersion={runtimeVersion}
          runtimeVersionError={runtimeVersionError}
          user={user}
        />

        <TeamSection
          canCreateTeamInvite={canCreateTeamInvite}
          canCreateTeamMember={canCreateTeamMember}
          canResetTeamMemberPassword={canResetTeamMemberPassword}
          copiedTeamInviteId={copiedTeamInviteId}
          currentUser={user}
          emailDiagnostics={emailDiagnostics}
          emailDiagnosticsError={emailDiagnosticsError}
          isCreatingTeamInvite={isCreatingTeamInvite}
          isActive={activeSection === "team"}
          isCreatingTeamMember={isCreatingTeamMember}
          isLoadingEmailDiagnostics={isLoadingEmailDiagnostics}
          isLoadingTeamInvites={isLoadingTeamInvites}
          isLoadingTeamMembers={isLoadingTeamMembers}
          onCancelResetPassword={cancelResetTeamMemberPassword}
          onCopyTeamInviteLink={(inviteId) => {
            void handleCopyTeamInviteLink(inviteId);
          }}
          onCreateTeamInvite={handleCreateTeamInvite}
          onCreateTeamMember={handleCreateTeamMember}
          onDisplayNameChange={setTeamMemberDisplayName}
          onEmailChange={setTeamMemberEmail}
          onInviteEmailChange={setTeamInviteEmail}
          onInviteRoleChange={setTeamInviteRole}
          onPasswordChange={setTeamMemberPassword}
          onRevokeTeamInvite={(invite) => {
            void handleRevokeTeamInvite(invite);
          }}
          onRefreshEmailDiagnostics={() => {
            void handleRefreshEmailDiagnostics();
          }}
          onResendTeamInvite={(invite) => {
            void handleResendTeamInvite(invite);
          }}
          onResetPassword={(event, memberId) => {
            void handleResetTeamMemberPassword(event, memberId);
          }}
          onResetPasswordChange={setTeamMemberResetPassword}
          onRoleChange={setTeamMemberRole}
          onStartResetPassword={startResetTeamMemberPassword}
          onUpdateTeamMember={(memberId, input) => {
            void handleUpdateTeamMember(memberId, input);
          }}
          onUsernameChange={setTeamMemberUsername}
          passwordResetMemberId={passwordResetMemberId}
          revokingTeamInviteIds={revokingTeamInviteIds}
          resendingTeamInviteIds={resendingTeamInviteIds}
          resettingTeamMemberPasswordIds={resettingTeamMemberPasswordIds}
          teamInviteEmail={teamInviteEmail}
          teamInviteFormError={teamInviteFormError}
          teamInviteLinksById={teamInviteLinksById}
          teamInviteRole={teamInviteRole}
          teamInvites={teamInvites}
          teamInvitesError={teamInvitesError}
          teamMemberDisplayName={teamMemberDisplayName}
          teamMemberEmail={teamMemberEmail}
          teamMemberFormError={teamMemberFormError}
          teamMemberPassword={teamMemberPassword}
          teamMemberResetPassword={teamMemberResetPassword}
          teamMemberRole={teamMemberRole}
          teamMemberUsername={teamMemberUsername}
          teamMembers={teamMembers}
          teamMembersError={teamMembersError}
          updatingTeamMemberIds={updatingTeamMemberIds}
        />

        <LabelsSection
          canCreateLabel={canCreateLabel}
          deletingLabelIds={deletingLabelIds}
          isActive={activeSection === "labels"}
          isCreatingLabel={isCreatingLabel}
          isLoadingLabels={isLoadingLabels}
          labelColor={labelColor}
          labelName={labelName}
          labels={labels}
          labelsError={labelsError}
          onColorChange={setLabelColor}
          onCreateLabel={handleCreateLabel}
          onDeleteLabel={(label) => {
            void handleDeleteLabel(label);
          }}
          onNameChange={setLabelName}
        />

        <ProjectsSection
          automationRules={automationRules}
          automationRulesError={automationRulesError}
          archivingProjectIds={archivingProjectIds}
          archivingWorkflowStatusIds={archivingWorkflowStatusIds}
          canCreateProject={canCreateProject}
          creatingWorkflowStatus={creatingWorkflowStatus}
          deletingAutomationRuleIds={deletingAutomationRuleIds}
          editProjectDescription={editProjectDescription}
          editProjectName={editProjectName}
          editingProjectId={editingProjectId}
          isActive={activeSection === "projects"}
          isCreatingProject={isCreatingProject}
          isLoadingProjectDetail={isLoadingProjectDetail}
          isLoadingProjectMembers={isLoadingProjectMembers}
          isLoadingProjectWorkflow={loadingWorkflowProjectIds.includes(
            selectedProjectDetail?.id ?? "",
          )}
          isLoadingAutomationRules={isLoadingAutomationRules}
          isCreatingAutomationRule={isCreatingAutomationRule}
          isLoadingProjects={isLoadingProjects}
          isReorderingWorkflowStatuses={isReorderingWorkflowStatuses}
          isReorderingAutomationRules={isReorderingAutomationRules}
          isSavingWorkflowTransitions={isSavingWorkflowTransitions}
          onAddProjectMember={handleAddProjectMember}
          onArchiveProject={(project) => {
            void handleArchiveProject(project);
          }}
          onArchiveWorkflowStatus={handleArchiveWorkflowStatus}
          onCreateAutomationRule={handleCreateAutomationRule}
          onCancelEditingProject={cancelEditingProject}
          onCreateProject={handleCreateProject}
          onCreateWorkflowStatus={handleCreateWorkflowStatus}
          onDeleteAutomationRule={handleDeleteAutomationRule}
          onEditProjectDescriptionChange={setEditProjectDescription}
          onEditProjectNameChange={setEditProjectName}
          onOpenProjectBoard={(projectId) => {
            navigateToBoard(projectId);
          }}
          onProjectDetailTabChange={(tab) => {
            void handleProjectDetailTabChange(tab);
          }}
          onProjectMemberRoleChange={(member, role) => {
            void handleProjectMemberRoleChange(member, role);
          }}
          onProjectMemberRoleSelectionChange={setSelectedProjectMemberRole}
          onProjectMemberUserChange={setSelectedProjectMemberUserId}
          onRemoveProjectMember={(member) => {
            void handleRemoveProjectMember(member);
          }}
          onReorderWorkflowStatuses={handleReorderWorkflowStatuses}
          onReorderAutomationRules={handleReorderAutomationRules}
          onReplaceWorkflowTransitions={handleReplaceWorkflowTransitions}
          onProjectDescriptionChange={setProjectDescription}
          onProjectKeyChange={setProjectKey}
          onProjectNameChange={setProjectName}
          onSelectIssue={(issueId) => {
            void handleSelectIssue(issueId);
          }}
          onSelectProjectDetail={(projectId) => {
            void handleSelectProjectDetail(projectId);
          }}
          onStartEditingProject={startEditingProject}
          onUpdateProject={(event, project) => {
            void handleUpdateProject(event, project);
          }}
          onUpdateWorkflowStatus={handleUpdateWorkflowStatus}
          onUpdateAutomationRule={handleUpdateAutomationRule}
          onViewProjectIssues={(projectId) => {
            setIssueFilterProjectId(projectId);
            navigateToSection("issues");
          }}
          projectDescription={projectDescription}
          projectDetailTab={projectDetailTab}
          projectDetailError={projectDetailError}
          projectFormError={projectFormError}
          projectKey={projectKey}
          projectName={projectName}
          projects={projects}
          projectsError={projectsError}
          projectMembers={projectMembers}
          projectMembersError={projectMembersError}
          projectWorkflow={
            workflowsByProjectId[selectedProjectDetail?.id ?? ""]
          }
          projectWorkflowError={
            workflowMutationError ||
            workflowErrorsByProjectId[selectedProjectDetail?.id ?? ""] ||
            ""
          }
          labels={labels}
          removingProjectMemberIds={removingProjectMemberIds}
          role={user.workspace.role}
          selectedProjectMemberRole={selectedProjectMemberRole}
          selectedProjectMemberUserId={selectedProjectMemberUserId}
          selectedProjectDetail={selectedProjectDetail}
          selectedProjectIssues={selectedProjectIssues}
          selectedProjectOpenIssues={selectedProjectOpenIssues}
          teamMembers={teamMembers}
          updatingProjectMemberIds={updatingProjectMemberIds}
          updatingProjectIds={updatingProjectIds}
          updatingWorkflowStatusIds={updatingWorkflowStatusIds}
          updatingAutomationRuleIds={updatingAutomationRuleIds}
        />

        <SprintsSection
          addingIssueToSprintIds={addingIssueToSprintIds}
          canCreateSprint={canCreateSprint}
          canUpdateSprint={canUpdateSprint}
          completingSprintIds={completingSprintIds}
          editSprintEndDate={editSprintEndDate}
          editSprintGoal={editSprintGoal}
          editSprintName={editSprintName}
          editSprintStartDate={editSprintStartDate}
          isActive={activeSection === "sprints"}
          isCreatingSprint={isCreatingSprint}
          isEditingSprint={isEditingSprintDetails}
          isLoadingSelectedSprint={isLoadingSelectedSprint}
          isLoadingSprintPlanning={isLoadingSprintPlanning}
          isLoadingSprints={isLoadingSprints}
          isUpdatingSprint={isUpdatingSprint}
          onAddIssueToSprint={(issue) => {
            void handleAddIssueToSprint(issue);
          }}
          onCancelSprintEdit={cancelEditingSprint}
          onCompleteSprint={(sprint) => {
            void handleCompleteSprint(sprint);
          }}
          onCreateSprint={handleCreateSprint}
          onEditSprintEndDateChange={setEditSprintEndDate}
          onEditSprintGoalChange={setEditSprintGoal}
          onEditSprintNameChange={setEditSprintName}
          onEditSprintStartDateChange={setEditSprintStartDate}
          onProjectFilterChange={setSprintFilterProjectId}
          onRemoveIssueFromSprint={(issue) => {
            void handleRemoveIssueFromSprint(issue);
          }}
          onSelectSprint={(sprintId) => {
            void handleSelectSprint(sprintId);
          }}
          onSprintIssueDragStart={handleIssueDragStart}
          onSprintIssueDrop={handleSprintIssueDrop}
          onSprintEndDateChange={setSprintEndDate}
          onSprintGoalChange={setSprintGoal}
          onSprintNameChange={setSprintName}
          onSprintProjectChange={setSprintProjectId}
          onSprintStartDateChange={setSprintStartDate}
          onStartEditingSprint={startEditingSprint}
          onStartSprint={(sprint) => {
            void handleStartSprint(sprint);
          }}
          onStatusFilterChange={setSprintFilterStatus}
          onTransitionIssue={(issueId, workflowStatusId) => {
            void handleTransitionIssue(issueId, {
              workflow_status_id: workflowStatusId,
            });
          }}
          onUpdateSprint={handleUpdateSprint}
          onViewSprintProjectIssues={(projectId) => {
            setIssueFilterProjectId(projectId);
            navigateToSection("issues");
          }}
          projectFilterId={sprintFilterProjectId}
          projects={projects}
          removingIssueFromSprintIds={removingIssueFromSprintIds}
          selectedSprint={selectedSprint}
          selectedSprintBacklogIssues={selectedSprintBacklogIssues}
          selectedSprintError={selectedSprintError}
          selectedSprintIssues={selectedSprintIssues}
          selectedSprintWorkflow={selectedSprintWorkflow}
          selectedSprintWorkflowError={
            workflowErrorsByProjectId[selectedSprint?.project_id ?? ""] || ""
          }
          sprintEndDate={sprintEndDate}
          sprintFormError={sprintFormError}
          sprintGoal={sprintGoal}
          sprintPlanningError={sprintPlanningError}
          sprintName={sprintName}
          sprintProjectId={sprintProjectId}
          sprintStartDate={sprintStartDate}
          sprintStatusFilter={sprintFilterStatus}
          sprints={sprints}
          sprintsError={sprintsError}
          startingSprintIds={startingSprintIds}
          teamMembers={teamMembers}
          today={today}
          transitioningIssueIds={transitioningIssueIds}
          isLoadingSelectedSprintWorkflow={loadingWorkflowProjectIds.includes(
            selectedSprint?.project_id ?? "",
          )}
        />

        <section
          className="issues-layout"
          aria-label="Issues"
          hidden={activeSection !== "issues"}
        >
          <IssueCreateForm
            assigneeId={issueAssigneeId}
            canCreateIssue={canCreateIssue}
            description={issueDescription}
            dueDate={issueDueDate}
            formError={
              issueFormError || workflowErrorsByProjectId[selectedProjectId] || ""
            }
            isCreatingIssue={isCreatingIssue}
            labels={labels}
            labelIds={newIssueLabelIds}
            onAssigneeChange={setIssueAssigneeId}
            onCreateIssue={handleCreateIssue}
            onDescriptionChange={setIssueDescription}
            onDueDateChange={setIssueDueDate}
            onLabelChange={handleCreateIssueLabel}
            onPriorityChange={setIssuePriority}
            onProjectChange={handleCreateIssueProjectChange}
            onStoryPointsChange={setIssueStoryPoints}
            onStatusChange={setIssueWorkflowStatusId}
            onTitleChange={setIssueTitle}
            onTypeChange={setIssueType}
            priority={issuePriority}
            projectId={selectedProjectId}
            projects={projects.filter((project) => project.can_write)}
            statusId={issueWorkflowStatusId}
            statuses={selectedCreateWorkflowStatuses}
            storyPoints={issueStoryPoints}
            teamMembers={teamMembers}
            title={issueTitle}
            type={issueType}
          />

          <SavedFiltersPanel
            deletingSavedFilterIds={deletingSavedFilterIds}
            isCreatingSavedFilter={isCreatingSavedFilter}
            isLoadingSavedFilters={isLoadingSavedFilters}
            onApplySavedFilter={applySavedIssueFilters}
            onCancelRenameSavedFilter={() => {
              setRenameSavedFilterId("");
              setRenameSavedFilterName("");
            }}
            onCreateSavedFilter={(event) => {
              void handleCreateSavedFilter(event);
            }}
            onDeleteSavedFilter={(savedFilter) => {
              void handleDeleteSavedFilter(savedFilter);
            }}
            onRenameSavedFilter={(savedFilter) => {
              void handleRenameSavedFilter(savedFilter);
            }}
            onRenameSavedFilterNameChange={setRenameSavedFilterName}
            onSavedFilterNameChange={setSavedFilterName}
            onStartRenameSavedFilter={handleStartRenameSavedFilter}
            onUpdateSavedFilter={(savedFilter) => {
              void handleUpdateSavedFilter(savedFilter);
            }}
            renameSavedFilterId={renameSavedFilterId}
            renameSavedFilterName={renameSavedFilterName}
            savedFilterFormError={savedFilterFormError}
            savedFilterName={savedFilterName}
            savedFilters={savedFilters}
            savedFiltersError={savedFiltersError}
            updatingSavedFilterIds={updatingSavedFilterIds}
            workflowStatusNamesById={workflowStatusNamesById}
          />

          <IssueListPanel
            archivingIssueIds={archivingIssueIds}
            assigneeFilterId={issueFilterAssigneeId}
            dueFilter={issueFilterDue}
            isLoadingIssues={isLoadingIssues}
            issues={issues}
            issuesError={
              issuesError ||
              workflowErrorsByProjectId[issueFilterProjectId] ||
              ""
            }
            labelFilterId={issueFilterLabelId}
            labels={labels}
            onArchiveIssue={(issue) => {
              void handleArchiveIssue(issue);
            }}
            onAssigneeFilterChange={setIssueFilterAssigneeId}
            onClearFilters={() => {
              setIssueFilterQuery("");
              setIssueFilterProjectId("");
              setIssueFilterSprintId("");
              setIssueFilterStatus("");
              setIssueFilterWorkflowStatusId("");
              setIssueFilterPriority("");
              setIssueFilterAssigneeId("");
              setIssueFilterLabelId("");
              setIssueFilterDue("");
            }}
            onDueFilterChange={setIssueFilterDue}
            onLabelFilterChange={setIssueFilterLabelId}
            onOpenIssue={(issueId) => {
              void handleSelectIssue(issueId);
            }}
            onPriorityFilterChange={setIssueFilterPriority}
            onProjectFilterChange={handleIssueProjectFilterChange}
            onQueryChange={setIssueFilterQuery}
            onSortChange={setIssueSort}
            onSprintFilterChange={handleIssueSprintFilterChange}
            onWorkflowStatusFilterChange={handleIssueWorkflowStatusFilterChange}
            priorityFilter={issueFilterPriority}
            projectFilterId={issueFilterProjectId}
            projects={projects}
            query={issueFilterQuery}
            sort={issueSort}
            sprintFilterId={issueFilterSprintId}
            sprints={issueFilterVisibleSprints}
            legacyStatusFilter={issueFilterStatus}
            teamMembers={teamMembers}
            today={today}
            workflowStatusFilterId={issueFilterWorkflowStatusId}
            workflowStatuses={selectedFilterWorkflowStatuses}
          />
        </section>

        <IssueDetailSection
          activity={issueActivity}
          activityError={activityError}
          archivingIssueIds={archivingIssueIds}
          assigningIssueIds={assigningIssueIds}
          availableLinkIssues={availableLinkIssues}
          canWriteIssue={canWriteSelectedIssue}
          canCreateComment={canCreateComment}
          canCreateIssueLink={canCreateIssueLink}
          canCreateSubtask={canCreateSubtask}
          childIssues={issueChildren}
          commentBody={commentBody}
          comments={issueComments}
          commentsError={commentsError}
          currentUser={user}
          deletingCommentIds={deletingCommentIds}
          deletingIssueLinkIds={deletingIssueLinkIds}
          editCommentBody={editCommentBody}
          editIssueDescription={editIssueDescription}
          editIssueDueDate={editIssueDueDate}
          editIssuePriority={editIssuePriority}
          editIssueStoryPoints={editIssueStoryPoints}
          editIssueTitle={editIssueTitle}
          editIssueType={editIssueType}
          editingCommentId={editingCommentId}
          isActive={activeSection === "issues"}
          isCreatingComment={isCreatingComment}
          isCreatingIssueLink={isCreatingIssueLink}
          isCreatingSubtask={isCreatingSubtask}
          isEditingIssueDetails={isEditingIssueDetails}
          isLoadingActivity={isLoadingActivity}
          isLoadingChildIssues={isLoadingIssueChildren}
          isLoadingComments={isLoadingComments}
          isLoadingIssueLinks={isLoadingIssueLinks}
          isLoadingIssue={isLoadingSelectedIssue}
          isUpdatingIssue={isUpdatingIssue}
          issue={selectedIssue}
          issueError={
            selectedIssueError ||
            workflowErrorsByProjectId[selectedIssue?.project_id ?? ""] ||
            ""
          }
          hierarchyError={hierarchyError}
          issueLinks={issueLinks}
          labelingIssueIds={labelingIssueIds}
          labels={labels}
          linkFormError={linkFormError}
          linksError={linksError}
          linkTargetIssueId={linkTargetIssueId}
          linkType={linkType}
          onArchiveIssue={(issue) => {
            void handleArchiveIssue(issue);
          }}
          onAssignIssue={(issueId, assigneeId) => {
            void handleAssignIssue(issueId, assigneeId);
          }}
          onCancelEditingComment={cancelEditingComment}
          onCancelIssueEdit={() => setIsEditingIssueDetails(false)}
          onCloseIssue={() => {
            setSelectedIssue(null);
            setSelectedIssueError("");
            setIsEditingIssueDetails(false);
            setIssueChildren([]);
            setHierarchyError("");
            setSubtaskFormError("");
            setSubtaskTitle("");
            setSubtaskPriority("medium");
            setSubtaskStoryPoints("0");
            setSubtaskWorkflowStatusId("");
            setIssueLinks([]);
            setLinksError("");
            setLinkFormError("");
            setIsLoadingIssueLinks(false);
            setIsCreatingIssueLink(false);
            setDeletingIssueLinkIds([]);
            setLinkTargetIssueId("");
            setLinkType("relates");
            setEditingCommentId("");
            setEditCommentBody("");
            setUpdatingCommentIds([]);
            setDeletingCommentIds([]);
          }}
          onCommentBodyChange={setCommentBody}
          onCreateComment={handleCreateComment}
          onCreateIssueLink={handleCreateIssueLink}
          onCreateSubtask={handleCreateSubtask}
          onDeleteComment={(comment) => {
            void handleDeleteComment(comment);
          }}
          onDeleteIssueLink={(link) => {
            void handleDeleteIssueLink(link);
          }}
          onEditCommentBodyChange={setEditCommentBody}
          onIssueDescriptionChange={setEditIssueDescription}
          onIssueDueDateChange={setEditIssueDueDate}
          onIssuePriorityChange={setEditIssuePriority}
          onIssueStoryPointsChange={setEditIssueStoryPoints}
          onIssueTitleChange={setEditIssueTitle}
          onIssueTypeChange={setEditIssueType}
          onIssueLinkTargetChange={setLinkTargetIssueId}
          onIssueLinkTypeChange={setLinkType}
          onOpenIssue={(issueId) => {
            void handleSelectIssue(issueId);
          }}
          onSetIssueLabel={(issue, labelId, shouldAttach) => {
            void handleSetIssueLabel(issue, labelId, shouldAttach);
          }}
          onStartEditingComment={startEditingComment}
          onStartEditingIssue={startEditingIssue}
          onSubtaskPriorityChange={setSubtaskPriority}
          onSubtaskStoryPointsChange={setSubtaskStoryPoints}
          onSubtaskStatusChange={setSubtaskWorkflowStatusId}
          onSubtaskTitleChange={setSubtaskTitle}
          onTransitionIssue={(issueId, workflowStatusId) => {
            void handleTransitionIssue(issueId, {
              workflow_status_id: workflowStatusId,
            });
          }}
          onUpdateComment={(event, comment) => {
            void handleUpdateComment(event, comment);
          }}
          onUpdateIssue={handleUpdateSelectedIssue}
          teamMembers={teamMembers}
          today={today}
          transitioningIssueIds={transitioningIssueIds}
          parentIssue={selectedParentIssue}
          subtaskFormError={subtaskFormError}
          subtaskPriority={subtaskPriority}
          subtaskStoryPoints={subtaskStoryPoints}
          subtaskStatusId={subtaskWorkflowStatusId}
          subtaskTitle={subtaskTitle}
          updatingCommentIds={updatingCommentIds}
          workflowStatuses={selectedIssueWorkflowStatuses}
          transitionStatuses={selectedIssueTransitionStatuses}
        />

        <BoardSection
          archivingIssueIds={archivingIssueIds}
          error={boardError}
          isActive={activeSection === "board"}
          isLoading={isLoadingBoard}
          issues={boardIssues}
          onArchiveIssue={(issue) => {
            void handleArchiveIssue(issue);
          }}
          onIssueDrop={handleIssueDrop}
          onOpenIssue={(issueId) => {
            void handleSelectIssue(issueId);
          }}
          onProjectChange={(projectId) => {
            navigateToBoard(projectId);
          }}
          onTransitionIssue={(issueId, workflowStatusId) => {
            void handleTransitionIssue(issueId, {
              workflow_status_id: workflowStatusId,
            });
          }}
          projectId={boardProjectId}
          projects={projects}
          teamMembers={teamMembers}
          today={today}
          transitioningIssueIds={transitioningIssueIds}
          workflow={selectedBoardWorkflow}
          workflowError={workflowErrorsByProjectId[boardProjectId] || ""}
        />
        </div>
      </div>
    </div>
  );
}
