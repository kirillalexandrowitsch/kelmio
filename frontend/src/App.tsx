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
  IssuePriority,
  IssueSort,
  IssueStatus,
  IssueType,
  Label,
  Project,
  TeamMember,
  archiveIssue,
  archiveProject,
  assignIssue,
  changePassword,
  createLabel,
  createIssue,
  createIssueComment,
  createProject,
  createTeamMember,
  deleteLabel,
  deleteIssueComment,
  getIssue,
  getCurrentUser,
  getProject,
  listIssueActivity,
  listIssueComments,
  listIssues,
  listLabels,
  listProjects,
  listTeamMembers,
  login,
  logout,
  resetTeamMemberPassword,
  setIssueLabels,
  transitionIssue,
  updateProfile,
  updateProject,
  updateIssueComment,
  updateTeamMember,
  updateIssue,
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
  columns,
  issueDueInfo,
  issueLabelIds,
  issueMatchesFilters,
  issueTypeLabels,
  priorityLabels,
  startOfToday,
} from "./lib/issue-model";
import {
  assignableTeamMembers,
  memberOptionLabel,
} from "./lib/team-view";
import { activityDescription, activityTitle } from "./lib/activity-view";
import { formatDateTime } from "./lib/formatting";
import {
  appSectionPath,
  appSections,
  currentAppSectionFromLocation,
  type AppSection,
} from "./lib/routing";
import { FormError } from "./components/form-feedback";
import { AppSidebar, WorkspaceTopbar } from "./components/app-shell";
import { AccountSection } from "./features/account/account-section";
import { BootingScreen, SignInScreen } from "./features/auth/auth-screens";
import { BoardSection } from "./features/board/board-section";
import { DashboardSection } from "./features/dashboard/dashboard-section";
import { IssueCreateForm } from "./features/issues/issue-create-form";
import { IssueListPanel } from "./features/issues/issue-list-panel";
import { LabelsSection } from "./features/labels/labels-section";
import { ProjectsSection } from "./features/projects/projects-section";
import { TeamSection } from "./features/team/team-section";

function apiErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback;
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
  const selectedIssueId = selectedIssue?.id ?? "";

  function navigateToSection(
    section: AppSection,
    mode: "push" | "replace" = "push",
  ) {
    setActiveSection(section);

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

  const today = startOfToday();
  const openIssues = issues.filter((issue) => issue.status !== "done");
  const selectedProjectIssues = selectedProjectDetail
    ? issues.filter((issue) => issue.project_id === selectedProjectDetail.id)
    : [];
  const selectedProjectOpenIssues = selectedProjectIssues.filter(
    (issue) => issue.status !== "done",
  );
  const selectedIssueDueInfo = selectedIssue ? issueDueInfo(selectedIssue, today) : null;
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
  const canCreateComment =
    selectedIssue !== null && hasText(commentBody) && !isCreatingComment;
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

        <section
          className="issue-detail-panel"
          aria-label="Issue details"
          hidden={activeSection !== "issues"}
        >
          <header className="section-header">
            <div>
              <p className="eyebrow">Issue details</p>
              <h2>
                {selectedIssue
                  ? `${selectedIssue.issue_key} · ${selectedIssue.title}`
                  : "Select an issue"}
              </h2>
            </div>
            {selectedIssue ? (
              <div className="detail-actions">
                <button
                  className="ghost-button"
                  onClick={() => {
                    if (isEditingIssueDetails) {
                      setIsEditingIssueDetails(false);
                    } else {
                      startEditingIssue(selectedIssue);
                    }
                  }}
                  type="button"
                >
                  {isEditingIssueDetails ? "Cancel edit" : "Edit"}
                </button>
                <button
                  className="ghost-button"
                  onClick={() => {
                    setSelectedIssue(null);
                    setSelectedIssueError("");
                    setIsEditingIssueDetails(false);
                    setEditingCommentId("");
                    setEditCommentBody("");
                    setUpdatingCommentIds([]);
                    setDeletingCommentIds([]);
                  }}
                  type="button"
                >
                  Close
                </button>
                <button
                  className="ghost-button danger-button"
                  disabled={archivingIssueIds.includes(selectedIssue.id)}
                  onClick={() => {
                    void handleArchiveIssue(selectedIssue);
                  }}
                  type="button"
                >
                  {archivingIssueIds.includes(selectedIssue.id)
                    ? "Archiving"
                    : "Archive"}
                </button>
              </div>
            ) : null}
          </header>

          {selectedIssueError ? (
            <FormError message={selectedIssueError} />
          ) : null}

          {isLoadingSelectedIssue ? (
            <span className="muted">Loading details</span>
          ) : null}

          {selectedIssue ? (
            <div className="issue-detail-body">
              <div className="issue-detail-main">
                {isEditingIssueDetails ? (
                  <form
                    className="issue-edit-form"
                    onSubmit={handleUpdateSelectedIssue}
                  >
                    <label>
                      <span>Title</span>
                      <input
                        maxLength={180}
                        onChange={(event) => setEditIssueTitle(event.target.value)}
                        value={editIssueTitle}
                      />
                    </label>

                    <label>
                      <span>Description</span>
                      <textarea
                        onChange={(event) =>
                          setEditIssueDescription(event.target.value)
                        }
                        rows={4}
                        value={editIssueDescription}
                      />
                    </label>

                    <div className="field-grid">
                      <label>
                        <span>Type</span>
                        <select
                          onChange={(event) =>
                            setEditIssueType(event.target.value as IssueType)
                          }
                          value={editIssueType}
                        >
                          {Object.entries(issueTypeLabels).map(([value, label]) => (
                            <option key={value} value={value}>
                              {label}
                            </option>
                          ))}
                        </select>
                      </label>

                      <label>
                        <span>Priority</span>
                        <select
                          onChange={(event) =>
                            setEditIssuePriority(
                              event.target.value as IssuePriority,
                            )
                          }
                          value={editIssuePriority}
                        >
                          {Object.entries(priorityLabels).map(([value, label]) => (
                            <option key={value} value={value}>
                              {label}
                            </option>
                          ))}
                        </select>
                      </label>
                    </div>

                    <label>
                      <span>Due date</span>
                      <input
                        onChange={(event) => setEditIssueDueDate(event.target.value)}
                        type="date"
                        value={editIssueDueDate}
                      />
                    </label>

                    <div className="form-actions">
                      <button
                        disabled={isUpdatingIssue || !hasText(editIssueTitle)}
                        type="submit"
                      >
                        {isUpdatingIssue ? "Saving..." : "Save changes"}
                      </button>
                      <button
                        className="ghost-button"
                        disabled={isUpdatingIssue}
                        onClick={() => setIsEditingIssueDetails(false)}
                        type="button"
                      >
                        Cancel
                      </button>
                    </div>
                  </form>
                ) : (
                  <>
                    <div className="issue-detail-headline">
                      <span className="issue-key">{selectedIssue.issue_key}</span>
                      <span className="detail-chip">
                        {issueTypeLabels[selectedIssue.issue_type]}
                      </span>
                      <span className="detail-chip">
                        {priorityLabels[selectedIssue.priority]}
                      </span>
                    </div>

                    <div>
                      <p className="eyebrow">Description</p>
                      <p className="issue-detail-description">
                        {selectedIssue.description || "No description yet."}
                      </p>
                    </div>

                    <div>
                      <p className="eyebrow">Labels</p>
                      {selectedIssue.labels.length > 0 ? (
                        <div className="issue-label-row">
                          {selectedIssue.labels.map((label) => (
                            <span
                              className="label-chip"
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
                      ) : (
                        <p className="issue-detail-description">No labels yet.</p>
                      )}
                    </div>
                  </>
                )}

                <section className="comments-section" aria-label="Issue comments">
                  <header className="comments-header">
                    <div>
                      <p className="eyebrow">Comments</p>
                      <h3>{issueComments.length}</h3>
                    </div>
                    {isLoadingComments ? (
                      <span className="muted">Loading comments</span>
                    ) : null}
                  </header>

                  {commentsError ? (
                    <FormError message={commentsError} />
                  ) : null}

                  {issueComments.length > 0 ? (
                    <div className="comment-list">
                      {issueComments.map((comment) => {
                        const isEditingComment = editingCommentId === comment.id;
                        const isUpdatingComment = updatingCommentIds.includes(
                          comment.id,
                        );
                        const isDeletingComment = deletingCommentIds.includes(
                          comment.id,
                        );
                        const canEditComment =
                          comment.author_id === user.id ||
                          user.workspace.role === "admin";
                        const wasEdited = comment.updated_at !== comment.created_at;

                        return (
                          <article className="comment-card" key={comment.id}>
                            <header>
                              <div className="comment-author">
                                <strong>{comment.author_display_name}</strong>
                                <span>
                                  {formatDateTime(comment.created_at)}
                                  {wasEdited
                                    ? ` · edited ${formatDateTime(
                                        comment.updated_at,
                                      )}`
                                    : ""}
                                </span>
                              </div>
                              {canEditComment ? (
                                <div className="comment-actions">
                                  <button
                                    className="small-button"
                                    disabled={isUpdatingComment || isDeletingComment}
                                    onClick={() => {
                                      if (isEditingComment) {
                                        cancelEditingComment();
                                      } else {
                                        startEditingComment(comment);
                                      }
                                    }}
                                    type="button"
                                  >
                                    {isEditingComment ? "Cancel" : "Edit"}
                                  </button>
                                  <button
                                    className="small-button danger-button"
                                    disabled={isUpdatingComment || isDeletingComment}
                                    onClick={() => {
                                      void handleDeleteComment(comment);
                                    }}
                                    type="button"
                                  >
                                    {isDeletingComment ? "Deleting..." : "Delete"}
                                  </button>
                                </div>
                              ) : null}
                            </header>

                            {isEditingComment ? (
                              <form
                                className="comment-edit-form"
                                onSubmit={(event) => {
                                  void handleUpdateComment(event, comment);
                                }}
                              >
                                <textarea
                                  maxLength={4000}
                                  onChange={(event) =>
                                    setEditCommentBody(event.target.value)
                                  }
                                  rows={3}
                                  value={editCommentBody}
                                />
                                <div className="form-actions">
                                  <button
                                    disabled={
                                      isUpdatingComment || !hasText(editCommentBody)
                                    }
                                    type="submit"
                                  >
                                    {isUpdatingComment ? "Saving..." : "Save"}
                                  </button>
                                  <button
                                    className="ghost-button"
                                    disabled={isUpdatingComment}
                                    onClick={cancelEditingComment}
                                    type="button"
                                  >
                                    Cancel
                                  </button>
                                </div>
                              </form>
                            ) : (
                              <p>{comment.body}</p>
                            )}
                          </article>
                        );
                      })}
                    </div>
                  ) : (
                    <div className="comments-empty">No comments yet</div>
                  )}

                  <form className="comment-form" onSubmit={handleCreateComment}>
                    <label>
                      <span>Add comment</span>
                      <textarea
                        maxLength={4000}
                        onChange={(event) => setCommentBody(event.target.value)}
                        placeholder="Share context, decisions, or next steps"
                        rows={3}
                        value={commentBody}
                      />
                    </label>
                    <button
                      disabled={!canCreateComment}
                      type="submit"
                    >
                      {isCreatingComment ? "Posting..." : "Post comment"}
                    </button>
                  </form>
                </section>

                <section className="activity-section" aria-label="Issue activity">
                  <header className="comments-header">
                    <div>
                      <p className="eyebrow">Activity</p>
                      <h3>{issueActivity.length}</h3>
                    </div>
                    {isLoadingActivity ? (
                      <span className="muted">Loading activity</span>
                    ) : null}
                  </header>

                  {activityError ? (
                    <FormError message={activityError} />
                  ) : null}

                  {issueActivity.length > 0 ? (
                    <div className="activity-list">
                      {issueActivity.map((activity) => (
                        <article className="activity-card" key={activity.id}>
                          <span className="activity-dot" aria-hidden="true" />
                          <div>
                            <header>
                              <strong>{activityTitle(activity)}</strong>
                              <span>{formatDateTime(activity.created_at)}</span>
                            </header>
                            <p>
                              {activity.actor_display_name ?? "System"}
                              {activityDescription(activity, teamMembers)
                                ? ` · ${activityDescription(activity, teamMembers)}`
                                : ""}
                            </p>
                          </div>
                        </article>
                      ))}
                    </div>
                  ) : (
                    <div className="comments-empty">No activity yet</div>
                  )}
                </section>
              </div>

              <aside className="issue-detail-sidebar">
                <label className="issue-detail-status">
                  <span>Status</span>
                  <select
                    disabled={transitioningIssueIds.includes(selectedIssue.id)}
                    onChange={(event) => {
                      void handleTransitionIssue(
                        selectedIssue.id,
                        event.target.value as IssueStatus,
                      );
                    }}
                    value={selectedIssue.status}
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
                    disabled={assigningIssueIds.includes(selectedIssue.id)}
                    onChange={(event) => {
                      void handleAssignIssue(selectedIssue.id, event.target.value);
                    }}
                    value={selectedIssue.assignee_id ?? ""}
                  >
                    <option value="">Unassigned</option>
                    {assignableTeamMembers(
                      teamMembers,
                      selectedIssue.assignee_id,
                    ).map((member) => (
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
                            checked={selectedIssue.labels.some(
                              (issueLabel) => issueLabel.id === label.id,
                            )}
                            disabled={labelingIssueIds.includes(selectedIssue.id)}
                            onChange={(event) => {
                              void handleSetIssueLabel(
                                selectedIssue,
                                label.id,
                                event.target.checked,
                              );
                            }}
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
                    <strong>{selectedIssue.project_key}</strong>
                  </div>
                  <div>
                    <span>Due date</span>
                    {selectedIssueDueInfo ? (
                      <strong>
                        <span
                          className={`due-badge due-badge-${selectedIssueDueInfo.tone}`}
                        >
                          {selectedIssueDueInfo.label}
                        </span>
                      </strong>
                    ) : (
                      <strong>No due date</strong>
                    )}
                  </div>
                  <div>
                    <span>Created</span>
                    <strong>{formatDateTime(selectedIssue.created_at)}</strong>
                  </div>
                  <div>
                    <span>Updated</span>
                    <strong>{formatDateTime(selectedIssue.updated_at)}</strong>
                  </div>
                </div>
              </aside>
            </div>
          ) : (
            <div className="issue-detail-empty">
              Open a card from Recent issues or the board to inspect its details.
            </div>
          )}
        </section>

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
