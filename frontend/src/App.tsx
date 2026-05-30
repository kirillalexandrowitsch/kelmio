import { DragEvent, FormEvent, useEffect, useState } from "react";
import "./styles.css";
import {
  ApiError,
  API_UNAUTHORIZED_EVENT,
  CurrentUser,
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
  Sprint,
  SprintStatus,
  TeamMember,
  addIssueToSprint,
  archiveIssue,
  archiveProject,
  assignIssue,
  changePassword,
  completeSprint,
  createLabel,
  createIssue,
  createIssueComment,
  createIssueLink,
  createProject,
  createSprint,
  createSubtask,
  createTeamMember,
  deleteLabel,
  deleteIssueComment,
  deleteIssueLink,
  getIssue,
  getCurrentUser,
  getProject,
  getSprint,
  listIssueActivity,
  listIssueComments,
  listIssueChildren,
  listIssueLinks,
  listIssues,
  listLabels,
  listProjects,
  listSprints,
  listTeamMembers,
  login,
  logout,
  resetTeamMemberPassword,
  removeIssueFromSprint,
  setIssueLabels,
  startSprint,
  transitionIssue,
  updateProfile,
  updateProject,
  updateIssueComment,
  updateTeamMember,
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
  issueLabelIds,
  issueMatchesFilters,
  startOfToday,
} from "./lib/issue-model";
import { sprintMatchesFilters } from "./lib/sprint-model";
import {
  appSectionPath,
  appSections,
  currentAppSectionFromLocation,
  sprintIdFromPath,
  type AppSection,
} from "./lib/routing";
import { AppSidebar, WorkspaceTopbar } from "./components/app-shell";
import { AccountSection } from "./features/account/account-section";
import { BootingScreen, SignInScreen } from "./features/auth/auth-screens";
import { BoardSection } from "./features/board/board-section";
import { DashboardSection } from "./features/dashboard/dashboard-section";
import { IssueCreateForm } from "./features/issues/issue-create-form";
import { IssueDetailSection } from "./features/issues/issue-detail-section";
import { IssueListPanel } from "./features/issues/issue-list-panel";
import { LabelsSection } from "./features/labels/labels-section";
import { ProjectsSection } from "./features/projects/projects-section";
import { SprintsSection } from "./features/sprints/sprints-section";
import { TeamSection } from "./features/team/team-section";

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

export function App() {
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [loginValue, setLoginValue] = useState("admin");
  const [password, setPassword] = useState("admin12345");
  const [error, setError] = useState("");
  const [isBooting, setIsBooting] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const [activeSection, setActiveSection] = useState<AppSection>(
    currentAppSectionFromLocation,
  );
  const [routeSprintId, setRouteSprintId] = useState(currentSprintIdFromLocation);
  const [accountError, setAccountError] = useState("");
  const [accountSuccess, setAccountSuccess] = useState("");
  const [accountDisplayName, setAccountDisplayName] = useState("");
  const [isUpdatingProfile, setIsUpdatingProfile] = useState(false);
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmNewPassword, setConfirmNewPassword] = useState("");
  const [isChangingPassword, setIsChangingPassword] = useState(false);
  const [projects, setProjects] = useState<Project[]>([]);
  const [projectsError, setProjectsError] = useState("");
  const [projectFormError, setProjectFormError] = useState("");
  const [isLoadingProjects, setIsLoadingProjects] = useState(false);
  const [isCreatingProject, setIsCreatingProject] = useState(false);
  const [archivingProjectIds, setArchivingProjectIds] = useState<string[]>([]);
  const [editingProjectId, setEditingProjectId] = useState("");
  const [editProjectName, setEditProjectName] = useState("");
  const [editProjectDescription, setEditProjectDescription] = useState("");
  const [updatingProjectIds, setUpdatingProjectIds] = useState<string[]>([]);
  const [selectedProjectDetail, setSelectedProjectDetail] =
    useState<Project | null>(null);
  const [projectDetailError, setProjectDetailError] = useState("");
  const [isLoadingProjectDetail, setIsLoadingProjectDetail] = useState(false);
  const [teamMembers, setTeamMembers] = useState<TeamMember[]>([]);
  const [teamMembersError, setTeamMembersError] = useState("");
  const [teamMemberFormError, setTeamMemberFormError] = useState("");
  const [isLoadingTeamMembers, setIsLoadingTeamMembers] = useState(false);
  const [isCreatingTeamMember, setIsCreatingTeamMember] = useState(false);
  const [teamMemberEmail, setTeamMemberEmail] = useState("");
  const [teamMemberUsername, setTeamMemberUsername] = useState("");
  const [teamMemberDisplayName, setTeamMemberDisplayName] = useState("");
  const [teamMemberPassword, setTeamMemberPassword] = useState("");
  const [teamMemberRole, setTeamMemberRole] =
    useState<TeamMember["role"]>("member");
  const [updatingTeamMemberIds, setUpdatingTeamMemberIds] = useState<string[]>([]);
  const [passwordResetMemberId, setPasswordResetMemberId] = useState("");
  const [teamMemberResetPassword, setTeamMemberResetPassword] = useState("");
  const [resettingTeamMemberPasswordIds, setResettingTeamMemberPasswordIds] =
    useState<string[]>([]);
  const [labels, setLabels] = useState<Label[]>([]);
  const [labelsError, setLabelsError] = useState("");
  const [isLoadingLabels, setIsLoadingLabels] = useState(false);
  const [labelName, setLabelName] = useState("");
  const [labelColor, setLabelColor] = useState("#4e795d");
  const [isCreatingLabel, setIsCreatingLabel] = useState(false);
  const [deletingLabelIds, setDeletingLabelIds] = useState<string[]>([]);
  const [projectKey, setProjectKey] = useState("");
  const [projectName, setProjectName] = useState("");
  const [projectDescription, setProjectDescription] = useState("");
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
  const [issueStatus, setIssueStatus] = useState<IssueStatus>("todo");
  const [issueAssigneeId, setIssueAssigneeId] = useState("");
  const [issueDueDate, setIssueDueDate] = useState("");
  const [newIssueLabelIds, setNewIssueLabelIds] = useState<string[]>([]);
  const [issueFilterQuery, setIssueFilterQuery] = useState("");
  const [issueSort, setIssueSort] = useState<IssueSort>("created_desc");
  const [issueFilterProjectId, setIssueFilterProjectId] = useState("");
  const [issueFilterStatus, setIssueFilterStatus] = useState<IssueStatus | "">("");
  const [issueFilterPriority, setIssueFilterPriority] = useState<
    IssuePriority | ""
  >("");
  const [issueFilterAssigneeId, setIssueFilterAssigneeId] = useState("");
  const [issueFilterLabelId, setIssueFilterLabelId] = useState("");
  const [issueFilterDue, setIssueFilterDue] = useState<IssueDueFilter | "">("");
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
  const [subtaskStatus, setSubtaskStatus] = useState<IssueStatus>("todo");
  const [issueLinks, setIssueLinks] = useState<IssueLink[]>([]);
  const [linksError, setLinksError] = useState("");
  const [linkFormError, setLinkFormError] = useState("");
  const [isLoadingIssueLinks, setIsLoadingIssueLinks] = useState(false);
  const [isCreatingIssueLink, setIsCreatingIssueLink] = useState(false);
  const [deletingIssueLinkIds, setDeletingIssueLinkIds] = useState<string[]>([]);
  const [linkTargetIssueId, setLinkTargetIssueId] = useState("");
  const [linkType, setLinkType] = useState<IssueLinkType>("relates");
  const [sprints, setSprints] = useState<Sprint[]>([]);
  const [sprintsError, setSprintsError] = useState("");
  const [sprintFormError, setSprintFormError] = useState("");
  const [isLoadingSprints, setIsLoadingSprints] = useState(false);
  const [isCreatingSprint, setIsCreatingSprint] = useState(false);
  const [sprintProjectId, setSprintProjectId] = useState("");
  const [sprintName, setSprintName] = useState("");
  const [sprintGoal, setSprintGoal] = useState("");
  const [sprintStartDate, setSprintStartDate] = useState("");
  const [sprintEndDate, setSprintEndDate] = useState("");
  const [sprintFilterProjectId, setSprintFilterProjectId] = useState("");
  const [sprintFilterStatus, setSprintFilterStatus] = useState<
    SprintStatus | ""
  >("");
  const [selectedSprint, setSelectedSprint] = useState<Sprint | null>(null);
  const [selectedSprintError, setSelectedSprintError] = useState("");
  const [isLoadingSelectedSprint, setIsLoadingSelectedSprint] = useState(false);
  const [isEditingSprintDetails, setIsEditingSprintDetails] = useState(false);
  const [isUpdatingSprint, setIsUpdatingSprint] = useState(false);
  const [editSprintName, setEditSprintName] = useState("");
  const [editSprintGoal, setEditSprintGoal] = useState("");
  const [editSprintStartDate, setEditSprintStartDate] = useState("");
  const [editSprintEndDate, setEditSprintEndDate] = useState("");
  const [startingSprintIds, setStartingSprintIds] = useState<string[]>([]);
  const [completingSprintIds, setCompletingSprintIds] = useState<string[]>([]);
  const [sprintPlanningIssues, setSprintPlanningIssues] = useState<Issue[]>([]);
  const [sprintPlanningError, setSprintPlanningError] = useState("");
  const [isLoadingSprintPlanning, setIsLoadingSprintPlanning] = useState(false);
  const [addingIssueToSprintIds, setAddingIssueToSprintIds] = useState<string[]>(
    [],
  );
  const [removingIssueFromSprintIds, setRemovingIssueFromSprintIds] = useState<
    string[]
  >([]);
  const selectedIssueId = selectedIssue?.id ?? "";

  function navigateToSection(
    section: AppSection,
    mode: "push" | "replace" = "push",
  ) {
    setActiveSection(section);
    if (section !== "sprints") {
      setRouteSprintId("");
    }

    if (typeof window === "undefined") {
      return;
    }

    const nextPath = appSectionPath(section);
    if (
      window.location.pathname === nextPath &&
      window.location.search === "" &&
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

  function navigateToSprint(
    sprintId: string,
    mode: "push" | "replace" = "push",
  ) {
    setActiveSection("sprints");
    setRouteSprintId(sprintId);

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
    }

    window.addEventListener("popstate", handleRouteChange);
    return () => {
      window.removeEventListener("popstate", handleRouteChange);
    };
  }, []);

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
              response.projects.some((project) => project.id === currentProjectId)
            ) {
              return currentProjectId;
            }
            return response.projects[0]?.id ?? "";
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
      status: issueFilterStatus || undefined,
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
    issueFilterStatus,
    issueFilterPriority,
    issueFilterAssigneeId,
    issueFilterLabelId,
    issueFilterDue,
  ]);

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

    listIssues({
      projectId: selectedSprint.project_id,
      sort: "created_desc",
    })
      .then((response) => {
        if (isMounted) {
          setSprintPlanningIssues(response.issues);
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
      setSubtaskStatus("todo");
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
    setIsUpdatingProfile(false);
    setCurrentPassword("");
    setNewPassword("");
    setConfirmNewPassword("");
    setIsChangingPassword(false);
    setProjects([]);
    setTeamMembers([]);
    setLabels([]);
    setIssues([]);
    setProjectsError("");
    setProjectFormError("");
    setEditingProjectId("");
    setEditProjectName("");
    setEditProjectDescription("");
    setUpdatingProjectIds([]);
    setSelectedProjectDetail(null);
    setProjectDetailError("");
    setIsLoadingProjectDetail(false);
    setTeamMembersError("");
    setTeamMemberFormError("");
    setTeamMemberEmail("");
    setTeamMemberUsername("");
    setTeamMemberDisplayName("");
    setTeamMemberPassword("");
    setTeamMemberRole("member");
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
    setIssueFilterStatus("");
    setIssueFilterPriority("");
    setIssueFilterAssigneeId("");
    setIssueFilterLabelId("");
    setIssueFilterDue("");
    setNewIssueLabelIds([]);
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
    setSubtaskStatus("todo");
    setIssueLinks([]);
    setLinksError("");
    setLinkFormError("");
    setIsLoadingIssueLinks(false);
    setIsCreatingIssueLink(false);
    setDeletingIssueLinkIds([]);
    setLinkTargetIssueId("");
    setLinkType("relates");
    setSprints([]);
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
        currentProjectId === project.id ? nextProjects[0]?.id ?? "" : currentProjectId,
      );
      setSelectedProjectDetail((currentProject) =>
        currentProject?.id === project.id ? nextProjects[0] ?? null : currentProject,
      );
      setIssueFilterProjectId((currentProjectId) =>
        currentProjectId === project.id ? "" : currentProjectId,
      );
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

  function setIssueSprint(issueId: string, sprintId: string | null) {
    const updateIssueSprint = (issue: Issue) =>
      issue.id === issueId ? { ...issue, sprint_id: sprintId } : issue;

    setIssues((currentIssues) => currentIssues.map(updateIssueSprint));
    setSprintPlanningIssues((currentIssues) =>
      currentIssues.map(updateIssueSprint),
    );
    setIssueChildren((currentIssues) => currentIssues.map(updateIssueSprint));
    setSelectedIssue((currentIssue) =>
      currentIssue?.id === issueId ? updateIssueSprint(currentIssue) : currentIssue,
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
      setIssueSprint(issue.id, sprint.id);
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
      setIssueSprint(issue.id, null);
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
    setEditIssueDueDate(issue.due_date ?? "");
    setIsEditingIssueDetails(true);
  }

  async function handleUpdateSelectedIssue(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedIssue) {
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
        due_date: editIssueDueDate,
      });

      setIssues((currentIssues) => {
        if (
          !issueMatchesFilters(
            updatedIssue,
            issueFilterProjectId,
            issueFilterStatus,
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
      setSelectedIssue(updatedIssue);
      setIsEditingIssueDetails(false);
      await refreshIssueActivity(updatedIssue.id);
    } catch (err) {
      setSelectedIssueError(apiErrorMessage(err, "Could not update issue."));
    } finally {
      setIsUpdatingIssue(false);
    }
  }

  async function handleTransitionIssue(issueId: string, status: IssueStatus) {
    setIssuesError("");
    setTransitioningIssueIds((currentIds) =>
      currentIds.includes(issueId) ? currentIds : [...currentIds, issueId],
    );

    try {
      const updatedIssue = await transitionIssue(issueId, status);
      setIssues((currentIssues) => {
        if (
          !issueMatchesFilters(
            updatedIssue,
            issueFilterProjectId,
            issueFilterStatus,
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
      setSelectedIssue((currentIssue) =>
        currentIssue?.id === updatedIssue.id ? updatedIssue : currentIssue,
      );
      setSprintPlanningIssues((currentIssues) =>
        currentIssues.map((issue) =>
          issue.id === updatedIssue.id ? updatedIssue : issue,
        ),
      );
      if (selectedIssue?.id === updatedIssue.id) {
        await refreshIssueActivity(updatedIssue.id);
      }
    } catch (err) {
      setIssuesError(apiErrorMessage(err, "Could not update issue status."));
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
            issueFilterStatus,
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
            issueFilterStatus,
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
        setSubtaskStatus("todo");
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

    setIsCreatingIssue(true);

    try {
      const issue = await createIssue({
        project_id: selectedProjectId,
        title: issueTitle,
        description: issueDescription,
        issue_type: issueType,
        status: issueStatus,
        priority: issuePriority,
        assignee_id: issueAssigneeId,
        due_date: issueDueDate,
        label_ids: newIssueLabelIds,
      });

      if (
        issueMatchesFilters(
          issue,
          issueFilterProjectId,
          issueFilterStatus,
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
      setIssueStatus("todo");
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

    setIsCreatingSubtask(true);

    try {
      const subtask = await createSubtask(selectedIssue.id, {
        title,
        description: "",
        status: subtaskStatus,
        priority: subtaskPriority,
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
            issueFilterStatus,
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
      setSubtaskStatus("todo");
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

  function handleIssueDragOver(event: DragEvent<HTMLElement>) {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  }

  function handleIssueDrop(
    event: DragEvent<HTMLElement>,
    nextStatus: IssueStatus,
  ) {
    event.preventDefault();

    const issueId = event.dataTransfer.getData("text/plain");
    const issue = issues.find((currentIssue) => currentIssue.id === issueId);
    if (!issue || issue.status === nextStatus) {
      return;
    }

    void handleTransitionIssue(issue.id, nextStatus);
  }

  function handleSprintIssueDrop(
    event: DragEvent<HTMLElement>,
    nextStatus: IssueStatus,
  ) {
    event.preventDefault();

    const issueId = event.dataTransfer.getData("text/plain");
    const issue = sprintPlanningIssues.find(
      (currentIssue) => currentIssue.id === issueId,
    );
    if (!issue || issue.status === nextStatus) {
      return;
    }

    void handleTransitionIssue(issue.id, nextStatus);
  }

  const today = startOfToday();
  const openIssues = issues.filter((issue) => issue.status !== "done");
  const selectedProjectIssues = selectedProjectDetail
    ? issues.filter((issue) => issue.project_id === selectedProjectDetail.id)
    : [];
  const selectedProjectOpenIssues = selectedProjectIssues.filter(
    (issue) => issue.status !== "done",
  );
  const selectedSprintIssues = selectedSprint
    ? sprintPlanningIssues.filter((issue) => issue.sprint_id === selectedSprint.id)
    : [];
  const selectedSprintBacklogIssues = selectedSprint
    ? sprintPlanningIssues.filter(
        (issue) =>
          issue.project_id === selectedSprint.project_id &&
          issue.sprint_id === null &&
          issue.status !== "done",
      )
    : [];
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
  const canResetTeamMemberPassword = hasMinTrimmedLength(
    teamMemberResetPassword,
    8,
  );
  const canCreateLabel =
    hasText(labelName) &&
    isValidLabelColor(labelColor) &&
    !isCreatingLabel;
  const canCreateIssue =
    selectedProjectId !== "" && hasText(issueTitle) && !isCreatingIssue;
  const canCreateSprint =
    sprintProjectId !== "" && hasText(sprintName) && !isCreatingSprint;
  const canUpdateSprint =
    selectedSprint !== null &&
    selectedSprint.status !== "completed" &&
    hasText(editSprintName) &&
    !isUpdatingSprint;
  const canCreateComment =
    selectedIssue !== null && hasText(commentBody) && !isCreatingComment;
  const canCreateSubtask =
    selectedIssue !== null && hasText(subtaskTitle) && !isCreatingSubtask;
  const availableLinkIssues = selectedIssue
    ? issues.filter((issue) => issue.id !== selectedIssue.id)
    : [];
  const canCreateIssueLink =
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

  if (!user) {
    return (
      <SignInScreen
        canSignIn={canSignIn}
        error={error}
        isSubmitting={isSubmitting}
        loginValue={loginValue}
        onLoginChange={setLoginValue}
        onPasswordChange={setPassword}
        onSubmit={handleLogin}
        password={password}
      />
    );
  }

  return (
    <main className="app-shell">
      <AppSidebar activeSection={activeSection} onNavigate={navigateToSection} />

      <section className="workspace">
        <WorkspaceTopbar
          heading={activeSectionHeading}
          isLoggingOut={isLoggingOut}
          onLogout={handleLogout}
          role={user.workspace.role}
          subtitle={activeSectionSubtitle}
        />

        <DashboardSection
          dueSoonIssuesCount={dueSoonIssuesCount}
          isActive={activeSection === "dashboard"}
          onNavigate={navigateToSection}
          openIssuesCount={openIssuesCount}
          overdueIssuesCount={overdueIssuesCount}
          projectsCount={projects.length}
          role={user.workspace.role}
          teamMembersCount={teamMembers.length}
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
          isUpdatingProfile={isUpdatingProfile}
          newPassword={newPassword}
          onChangePassword={handleChangePassword}
          onConfirmNewPasswordChange={setConfirmNewPassword}
          onCurrentPasswordChange={setCurrentPassword}
          onDisplayNameChange={setAccountDisplayName}
          onNewPasswordChange={setNewPassword}
          onUpdateProfile={handleUpdateProfile}
          user={user}
        />

        <TeamSection
          canCreateTeamMember={canCreateTeamMember}
          canResetTeamMemberPassword={canResetTeamMemberPassword}
          currentUser={user}
          isActive={activeSection === "team"}
          isCreatingTeamMember={isCreatingTeamMember}
          isLoadingTeamMembers={isLoadingTeamMembers}
          onCancelResetPassword={cancelResetTeamMemberPassword}
          onCreateTeamMember={handleCreateTeamMember}
          onDisplayNameChange={setTeamMemberDisplayName}
          onEmailChange={setTeamMemberEmail}
          onPasswordChange={setTeamMemberPassword}
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
          resettingTeamMemberPasswordIds={resettingTeamMemberPasswordIds}
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
          archivingProjectIds={archivingProjectIds}
          canCreateProject={canCreateProject}
          editProjectDescription={editProjectDescription}
          editProjectName={editProjectName}
          editingProjectId={editingProjectId}
          isActive={activeSection === "projects"}
          isCreatingProject={isCreatingProject}
          isLoadingProjectDetail={isLoadingProjectDetail}
          isLoadingProjects={isLoadingProjects}
          onArchiveProject={(project) => {
            void handleArchiveProject(project);
          }}
          onCancelEditingProject={cancelEditingProject}
          onCreateProject={handleCreateProject}
          onEditProjectDescriptionChange={setEditProjectDescription}
          onEditProjectNameChange={setEditProjectName}
          onOpenProjectBoard={(projectId) => {
            setIssueFilterProjectId(projectId);
            navigateToSection("board");
          }}
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
          onViewProjectIssues={(projectId) => {
            setIssueFilterProjectId(projectId);
            navigateToSection("issues");
          }}
          projectDescription={projectDescription}
          projectDetailError={projectDetailError}
          projectFormError={projectFormError}
          projectKey={projectKey}
          projectName={projectName}
          projects={projects}
          projectsError={projectsError}
          role={user.workspace.role}
          selectedProjectDetail={selectedProjectDetail}
          selectedProjectIssues={selectedProjectIssues}
          selectedProjectOpenIssues={selectedProjectOpenIssues}
          updatingProjectIds={updatingProjectIds}
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
          onSprintIssueDragOver={handleIssueDragOver}
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
          onTransitionIssue={(issueId, status) => {
            void handleTransitionIssue(issueId, status);
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
            formError={issueFormError}
            isCreatingIssue={isCreatingIssue}
            labels={labels}
            labelIds={newIssueLabelIds}
            onAssigneeChange={setIssueAssigneeId}
            onCreateIssue={handleCreateIssue}
            onDescriptionChange={setIssueDescription}
            onDueDateChange={setIssueDueDate}
            onLabelChange={handleCreateIssueLabel}
            onPriorityChange={setIssuePriority}
            onProjectChange={setSelectedProjectId}
            onStatusChange={setIssueStatus}
            onTitleChange={setIssueTitle}
            onTypeChange={setIssueType}
            priority={issuePriority}
            projectId={selectedProjectId}
            projects={projects}
            status={issueStatus}
            teamMembers={teamMembers}
            title={issueTitle}
            type={issueType}
          />

          <IssueListPanel
            archivingIssueIds={archivingIssueIds}
            assigneeFilterId={issueFilterAssigneeId}
            dueFilter={issueFilterDue}
            isLoadingIssues={isLoadingIssues}
            issues={issues}
            issuesError={issuesError}
            labelFilterId={issueFilterLabelId}
            labels={labels}
            onArchiveIssue={(issue) => {
              void handleArchiveIssue(issue);
            }}
            onAssigneeFilterChange={setIssueFilterAssigneeId}
            onClearFilters={() => {
              setIssueFilterQuery("");
              setIssueFilterProjectId("");
              setIssueFilterStatus("");
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
            onProjectFilterChange={setIssueFilterProjectId}
            onQueryChange={setIssueFilterQuery}
            onSortChange={setIssueSort}
            onStatusFilterChange={setIssueFilterStatus}
            priorityFilter={issueFilterPriority}
            projectFilterId={issueFilterProjectId}
            projects={projects}
            query={issueFilterQuery}
            sort={issueSort}
            statusFilter={issueFilterStatus}
            teamMembers={teamMembers}
            today={today}
          />
        </section>

        <IssueDetailSection
          activity={issueActivity}
          activityError={activityError}
          archivingIssueIds={archivingIssueIds}
          assigningIssueIds={assigningIssueIds}
          availableLinkIssues={availableLinkIssues}
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
          issueError={selectedIssueError}
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
            setSubtaskStatus("todo");
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
          onSubtaskStatusChange={setSubtaskStatus}
          onSubtaskTitleChange={setSubtaskTitle}
          onTransitionIssue={(issueId, status) => {
            void handleTransitionIssue(issueId, status);
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
          subtaskStatus={subtaskStatus}
          subtaskTitle={subtaskTitle}
          updatingCommentIds={updatingCommentIds}
        />

        <BoardSection
          archivingIssueIds={archivingIssueIds}
          isActive={activeSection === "board"}
          issues={issues}
          onArchiveIssue={(issue) => {
            void handleArchiveIssue(issue);
          }}
          onIssueDragOver={handleIssueDragOver}
          onIssueDragStart={handleIssueDragStart}
          onIssueDrop={handleIssueDrop}
          onOpenIssue={(issueId) => {
            void handleSelectIssue(issueId);
          }}
          onTransitionIssue={(issueId, status) => {
            void handleTransitionIssue(issueId, status);
          }}
          teamMembers={teamMembers}
          today={today}
          transitioningIssueIds={transitioningIssueIds}
        />
      </section>
    </main>
  );
}
