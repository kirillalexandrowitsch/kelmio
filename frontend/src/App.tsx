import { FormEvent, useEffect, useState } from "react";
import "./styles.css";
import {
  ApiError,
  CurrentUser,
  Issue,
  IssueActivity,
  IssueComment,
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
  createLabel,
  createIssue,
  createIssueComment,
  createProject,
  createTeamMember,
  getIssue,
  getCurrentUser,
  listIssueActivity,
  listIssueComments,
  listIssues,
  listLabels,
  listProjects,
  listTeamMembers,
  login,
  logout,
  setIssueLabels,
  transitionIssue,
  updateTeamMember,
  updateIssue,
} from "./lib/api";

const columns = [
  { status: "backlog", title: "Backlog" },
  { status: "todo", title: "Todo" },
  { status: "in_progress", title: "In progress" },
  { status: "blocked", title: "Blocked" },
  { status: "done", title: "Done" },
] satisfies Array<{ status: IssueStatus; title: string }>;

const priorityLabels: Record<IssuePriority, string> = {
  low: "Low",
  medium: "Medium",
  high: "High",
  critical: "Critical",
};

const issueTypeLabels: Record<IssueType, string> = {
  task: "Task",
  bug: "Bug",
  story: "Story",
};

const issueSortLabels: Record<IssueSort, string> = {
  created_desc: "Newest first",
  created_asc: "Oldest first",
  priority_desc: "Priority high to low",
  due_date_asc: "Due date soonest",
};

type AppSection = "dashboard" | "projects" | "issues" | "team" | "labels";

const appSections = [
  { id: "dashboard", title: "Dashboard" },
  { id: "projects", title: "Projects" },
  { id: "issues", title: "Issues" },
  { id: "team", title: "Team" },
  { id: "labels", title: "Labels" },
] satisfies Array<{ id: AppSection; title: string }>;

function issueMatchesFilters(
  issue: Issue,
  projectId: string,
  status: IssueStatus | "",
  priority: IssuePriority | "",
  assigneeId: string,
  labelId: string,
  query: string,
) {
  if (projectId && issue.project_id !== projectId) {
    return false;
  }
  if (status && issue.status !== status) {
    return false;
  }
  if (priority && issue.priority !== priority) {
    return false;
  }
  if (assigneeId === "unassigned" && issue.assignee_id !== null) {
    return false;
  }
  if (assigneeId && assigneeId !== "unassigned" && issue.assignee_id !== assigneeId) {
    return false;
  }
  if (labelId && !issue.labels.some((label) => label.id === labelId)) {
    return false;
  }
  const normalizedQuery = query.trim().toLowerCase();
  if (
    normalizedQuery &&
    !issue.issue_key.toLowerCase().includes(normalizedQuery) &&
    !issue.title.toLowerCase().includes(normalizedQuery) &&
    !issue.description.toLowerCase().includes(normalizedQuery)
  ) {
    return false;
  }

  return true;
}

function statusLabel(status: string) {
  return columns.find((column) => column.status === status)?.title ?? status;
}

function activityTitle(activity: IssueActivity) {
  if (activity.action === "issue_created") {
    return "Created issue";
  }
  if (activity.action === "issue_updated") {
    return "Updated issue";
  }
  if (activity.action === "status_changed") {
    return "Changed status";
  }
  if (activity.action === "assignee_changed") {
    return "Changed assignee";
  }
  if (activity.action === "labels_changed") {
    return "Changed labels";
  }
  if (activity.action === "issue_archived") {
    return "Archived issue";
  }
  if (activity.action === "comment_added") {
    return "Added comment";
  }

  return activity.action.replaceAll("_", " ");
}

function activityDescription(activity: IssueActivity, members: TeamMember[]) {
  if (activity.action === "status_changed") {
    return `${statusLabel(activity.payload.from_status)} -> ${statusLabel(
      activity.payload.to_status,
    )}`;
  }
  if (activity.action === "assignee_changed") {
    return `${memberDisplayName(
      members,
      activity.payload.from_assignee_id || null,
    )} -> ${memberDisplayName(members, activity.payload.to_assignee_id || null)}`;
  }
  if (activity.action === "comment_added") {
    return activity.payload.preview ? `"${activity.payload.preview}"` : "";
  }
  if (activity.action === "issue_created") {
    return activity.payload.title ?? "";
  }
  if (activity.action === "issue_updated") {
    return activity.payload.fields
      ? `Fields: ${activity.payload.fields.replaceAll(",", ", ")}`
      : "";
  }
  if (activity.action === "labels_changed") {
    return "Labels updated";
  }

  return "";
}

function memberInitials(displayName: string) {
  const initials = displayName
    .trim()
    .split(/\s+/)
    .map((part) => part[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();

  return initials || "TM";
}

function memberDisplayName(members: TeamMember[], memberId: string | null) {
  if (!memberId) {
    return "Unassigned";
  }

  return members.find((member) => member.id === memberId)?.display_name ?? memberId;
}

function issueLabelIds(issue: Issue) {
  return issue.labels.map((label) => label.id);
}

function formatDateTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

export function App() {
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [loginValue, setLoginValue] = useState("admin");
  const [password, setPassword] = useState("admin12345");
  const [error, setError] = useState("");
  const [isBooting, setIsBooting] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [activeSection, setActiveSection] = useState<AppSection>("dashboard");
  const [projects, setProjects] = useState<Project[]>([]);
  const [projectsError, setProjectsError] = useState("");
  const [projectFormError, setProjectFormError] = useState("");
  const [isLoadingProjects, setIsLoadingProjects] = useState(false);
  const [isCreatingProject, setIsCreatingProject] = useState(false);
  const [archivingProjectIds, setArchivingProjectIds] = useState<string[]>([]);
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
  const [labels, setLabels] = useState<Label[]>([]);
  const [labelsError, setLabelsError] = useState("");
  const [isLoadingLabels, setIsLoadingLabels] = useState(false);
  const [labelName, setLabelName] = useState("");
  const [labelColor, setLabelColor] = useState("#4e795d");
  const [isCreatingLabel, setIsCreatingLabel] = useState(false);
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
  const [issueActivity, setIssueActivity] = useState<IssueActivity[]>([]);
  const [activityError, setActivityError] = useState("");
  const [isLoadingActivity, setIsLoadingActivity] = useState(false);
  const selectedIssueId = selectedIssue?.id ?? "";

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
    if (!user) {
      setProjects([]);
      return;
    }

    let isMounted = true;
    setProjectsError("");
    setProjectFormError("");
    setArchivingProjectIds([]);
    setIsLoadingProjects(true);

    listProjects()
      .then((response) => {
        if (isMounted) {
          setProjects(response.projects);
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
      .catch(() => {
        if (isMounted) {
          setProjectsError("Could not load projects.");
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
      .catch(() => {
        if (isMounted) {
          setTeamMembersError("Could not load team members.");
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
      .catch(() => {
        if (isMounted) {
          setLabelsError("Could not load labels.");
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
    })
      .then((response) => {
        if (isMounted) {
          setIssues(response.issues);
        }
      })
      .catch(() => {
        if (isMounted) {
          setIssuesError("Could not load issues.");
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
  ]);

  useEffect(() => {
    if (!selectedIssueId) {
      setIssueComments([]);
      setCommentsError("");
      setCommentBody("");
      return;
    }

    let isMounted = true;
    setCommentsError("");
    setIsLoadingComments(true);

    listIssueComments(selectedIssueId)
      .then((response) => {
        if (isMounted) {
          setIssueComments(response.comments);
        }
      })
      .catch(() => {
        if (isMounted) {
          setCommentsError("Could not load comments.");
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
      .catch(() => {
        if (isMounted) {
          setActivityError("Could not load activity.");
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
    setIsSubmitting(true);

    try {
      const response = await login(loginValue, password);
      setUser(response.user);
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
    await logout();
    setUser(null);
    setActiveSection("dashboard");
    setProjects([]);
    setTeamMembers([]);
    setLabels([]);
    setIssues([]);
    setProjectsError("");
    setProjectFormError("");
    setTeamMembersError("");
    setTeamMemberFormError("");
    setTeamMemberEmail("");
    setTeamMemberUsername("");
    setTeamMemberDisplayName("");
    setTeamMemberPassword("");
    setTeamMemberRole("member");
    setUpdatingTeamMemberIds([]);
    setLabelsError("");
    setLabelName("");
    setLabelColor("#4e795d");
    setIssuesError("");
    setIssueFormError("");
    setIssueFilterQuery("");
    setIssueSort("created_desc");
    setIssueFilterProjectId("");
    setIssueFilterStatus("");
    setIssueFilterPriority("");
    setIssueFilterAssigneeId("");
    setIssueFilterLabelId("");
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
    setIssueActivity([]);
    setActivityError("");
  }

  async function handleCreateProject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setProjectFormError("");
    setIsCreatingProject(true);

    try {
      const project = await createProject({
        key: projectKey,
        name: projectName,
        description: projectDescription,
      });
      setProjects((currentProjects) => [project, ...currentProjects]);
      setSelectedProjectId(project.id);
      setProjectKey("");
      setProjectName("");
      setProjectDescription("");
    } catch (err) {
      if (err instanceof ApiError) {
        setProjectFormError(err.message);
      } else {
        setProjectFormError("Could not create project.");
      }
    } finally {
      setIsCreatingProject(false);
    }
  }

  async function handleArchiveProject(project: Project) {
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
      setIssueFilterProjectId((currentProjectId) =>
        currentProjectId === project.id ? "" : currentProjectId,
      );
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
        setIsEditingIssueDetails(false);
      }
    } catch {
      setProjectsError("Could not archive project.");
    } finally {
      setArchivingProjectIds((currentIds) =>
        currentIds.filter((currentProjectId) => currentProjectId !== project.id),
      );
    }
  }

  async function handleCreateTeamMember(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setTeamMemberFormError("");
    setIsCreatingTeamMember(true);

    try {
      const member = await createTeamMember({
        email: teamMemberEmail,
        username: teamMemberUsername,
        display_name: teamMemberDisplayName,
        password: teamMemberPassword,
        role: teamMemberRole,
      });
      setTeamMembers((currentMembers) => [...currentMembers, member]);
      setTeamMemberEmail("");
      setTeamMemberUsername("");
      setTeamMemberDisplayName("");
      setTeamMemberPassword("");
      setTeamMemberRole("member");
    } catch (err) {
      if (err instanceof ApiError) {
        setTeamMemberFormError(err.message);
      } else {
        setTeamMemberFormError("Could not create team member.");
      }
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
      if (err instanceof ApiError) {
        setTeamMembersError(err.message);
      } else {
        setTeamMembersError("Could not update team member.");
      }
    } finally {
      setUpdatingTeamMemberIds((currentIds) =>
        currentIds.filter((currentId) => currentId !== memberId),
      );
    }
  }

  async function handleCreateLabel(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLabelsError("");
    setIsCreatingLabel(true);

    try {
      const label = await createLabel({
        name: labelName,
        color: labelColor,
      });
      setLabels((currentLabels) =>
        [...currentLabels, label].sort((left, right) =>
          left.name.localeCompare(right.name),
        ),
      );
      setLabelName("");
      setLabelColor("#4e795d");
    } catch (err) {
      if (err instanceof ApiError) {
        setLabelsError(err.message);
      } else {
        setLabelsError("Could not create label.");
      }
    } finally {
      setIsCreatingLabel(false);
    }
  }

  async function refreshIssueActivity(issueId: string) {
    setActivityError("");

    try {
      const response = await listIssueActivity(issueId);
      setIssueActivity(response.activity);
    } catch {
      setActivityError("Could not load activity.");
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
            issueFilterQuery,
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
      if (err instanceof ApiError) {
        setSelectedIssueError(err.message);
      } else {
        setSelectedIssueError("Could not update issue.");
      }
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
            issueFilterQuery,
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
    } catch {
      setIssuesError("Could not update issue status.");
    } finally {
      setTransitioningIssueIds((currentIds) =>
        currentIds.filter((currentIssueId) => currentIssueId !== issueId),
      );
    }
  }

  async function handleAssignIssue(issueId: string, assigneeId: string) {
    setSelectedIssueError("");
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
            issueFilterQuery,
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
    } catch {
      setSelectedIssueError("Could not update assignee.");
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
            issueFilterQuery,
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
    } catch {
      setSelectedIssueError("Could not update labels.");
    } finally {
      setLabelingIssueIds((currentIds) =>
        currentIds.filter((currentIssueId) => currentIssueId !== issue.id),
      );
    }
  }

  async function handleArchiveIssue(issue: Issue) {
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
        setIsEditingIssueDetails(false);
      }
    } catch {
      setIssuesError("Could not archive issue.");
      setSelectedIssueError("Could not archive issue.");
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
    setActiveSection("issues");

    const issuePreview = issues.find((issue) => issue.id === issueId);
    if (issuePreview) {
      setSelectedIssue(issuePreview);
    }

    setSelectedIssueError("");
    setIsEditingIssueDetails(false);
    setIsLoadingSelectedIssue(true);

    try {
      const issue = await getIssue(issueId);
      setSelectedIssue(issue);
    } catch {
      setSelectedIssueError("Could not load issue details.");
    } finally {
      setIsLoadingSelectedIssue(false);
    }
  }

  async function handleCreateIssue(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIssueFormError("");
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
          issueFilterQuery,
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
      if (err instanceof ApiError) {
        setIssueFormError(err.message);
      } else {
        setIssueFormError("Could not create issue.");
      }
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
    setIsCreatingComment(true);

    try {
      const comment = await createIssueComment(selectedIssue.id, commentBody);
      setIssueComments((currentComments) => [...currentComments, comment]);
      setCommentBody("");
      await refreshIssueActivity(selectedIssue.id);
    } catch (err) {
      if (err instanceof ApiError) {
        setCommentsError(err.message);
      } else {
        setCommentsError("Could not create comment.");
      }
    } finally {
      setIsCreatingComment(false);
    }
  }

  const openIssuesCount = issues.filter((issue) => issue.status !== "done").length;
  const hasIssueFilters =
    issueFilterProjectId !== "" ||
    issueFilterStatus !== "" ||
    issueFilterPriority !== "" ||
    issueFilterAssigneeId !== "" ||
    issueFilterLabelId !== "" ||
    issueFilterQuery.trim() !== "";
  const issueListSummary = hasIssueFilters
    ? `${issues.length} issues match current filters`
    : issueSort === "created_desc"
      ? "Showing latest issues across all projects"
      : `Showing issues sorted by ${issueSortLabels[issueSort].toLowerCase()}`;
  const activeSectionTitle =
    appSections.find((section) => section.id === activeSection)?.title ?? "Dashboard";
  const activeSectionSubtitle =
    activeSection === "dashboard" ? "Local workspace" : "Workspace section";
  const activeSectionHeading =
    activeSection === "dashboard"
      ? `Good to see you, ${user?.display_name ?? "there"}`
      : activeSectionTitle;

  if (isBooting) {
    return (
      <main className="auth-shell">
        <section className="auth-panel auth-panel-compact">
          <span className="brand-mark">TT</span>
          <p className="eyebrow">Checking session</p>
        </section>
      </main>
    );
  }

  if (!user) {
    return (
      <main className="auth-shell">
        <section className="auth-panel">
          <div className="brand auth-brand">
            <span className="brand-mark">TT</span>
            <div>
              <strong>Team Task Tracker</strong>
              <span>Local workspace</span>
            </div>
          </div>

          <div>
            <p className="eyebrow">Sign in</p>
            <h1>Welcome back</h1>
          </div>

          <form className="auth-form" onSubmit={handleLogin}>
            <label>
              <span>Username or email</span>
              <input
                autoComplete="username"
                autoFocus
                name="login"
                onChange={(event) => setLoginValue(event.target.value)}
                value={loginValue}
              />
            </label>

            <label>
              <span>Password</span>
              <input
                autoComplete="current-password"
                name="password"
                onChange={(event) => setPassword(event.target.value)}
                type="password"
                value={password}
              />
            </label>

            {error ? <p className="form-error">{error}</p> : null}

            <button disabled={isSubmitting} type="submit">
              {isSubmitting ? "Signing in..." : "Sign in"}
            </button>
          </form>
        </section>
      </main>
    );
  }

  return (
    <main className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-mark">TT</span>
          <div>
            <strong>Team Task Tracker</strong>
            <span>Local workspace</span>
          </div>
        </div>

        <nav className="nav-list" aria-label="Main navigation">
          {appSections.map((section) => (
            <button
              aria-current={activeSection === section.id ? "page" : undefined}
              key={section.id}
              onClick={() => setActiveSection(section.id)}
              type="button"
            >
              {section.title}
            </button>
          ))}
        </nav>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">{activeSectionSubtitle}</p>
            <h1>{activeSectionHeading}</h1>
          </div>
          <div className="topbar-actions">
            <div className="status-pill">{user.workspace.role}</div>
            <button className="ghost-button" onClick={handleLogout} type="button">
              Log out
            </button>
          </div>
        </header>

        <section
          className="summary-grid"
          aria-label="Project summary"
          hidden={activeSection !== "dashboard"}
        >
          <article>
            <span>Projects</span>
            <strong>{projects.length}</strong>
          </article>
          <article>
            <span>Open issues</span>
            <strong>{openIssuesCount}</strong>
          </article>
          <article>
            <span>Team members</span>
            <strong>{teamMembers.length}</strong>
          </article>
        </section>

        <section
          className="dashboard-actions"
          aria-label="Dashboard quick actions"
          hidden={activeSection !== "dashboard"}
        >
          <article className="dashboard-action-card">
            <div>
              <p className="eyebrow">Planning</p>
              <h2>Projects</h2>
              <p>Create or review project spaces before adding team work.</p>
            </div>
            <button
              className="small-button"
              onClick={() => setActiveSection("projects")}
              type="button"
            >
              Open projects
            </button>
          </article>

          <article className="dashboard-action-card">
            <div>
              <p className="eyebrow">Execution</p>
              <h2>Issues</h2>
              <p>Create tasks, inspect details, update status, comments, and labels.</p>
            </div>
            <button
              className="small-button"
              onClick={() => setActiveSection("issues")}
              type="button"
            >
              Open issues
            </button>
          </article>

          <article className="dashboard-action-card">
            <div>
              <p className="eyebrow">People</p>
              <h2>Team</h2>
              <p>Review workspace members and manage roles when you are an admin.</p>
            </div>
            <button
              className="small-button"
              onClick={() => setActiveSection("team")}
              type="button"
            >
              Open team
            </button>
          </article>

          <article className="dashboard-action-card">
            <div>
              <p className="eyebrow">Taxonomy</p>
              <h2>Labels</h2>
              <p>Keep issue categories clean so filtering and board scans stay useful.</p>
            </div>
            <button
              className="small-button"
              onClick={() => setActiveSection("labels")}
              type="button"
            >
              Open labels
            </button>
          </article>
        </section>

        <section
          className="team-panel"
          aria-label="Team members"
          hidden={activeSection !== "team"}
        >
          <header className="section-header">
            <div>
              <p className="eyebrow">Team</p>
              <h2>Workspace members</h2>
            </div>
            {isLoadingTeamMembers ? <span className="muted">Loading</span> : null}
          </header>

          {teamMembersError ? <p className="form-error">{teamMembersError}</p> : null}

          {teamMembers.length > 0 ? (
            <div className="team-list">
              {teamMembers.map((member) => {
                const isSelf = member.id === user.id;
                const isUpdatingMember = updatingTeamMemberIds.includes(member.id);

                return (
                  <article className="team-member-row" key={member.id}>
                    <span className="member-avatar">
                      {memberInitials(member.display_name)}
                    </span>
                    <div>
                      <h3>{member.display_name}</h3>
                      <p>
                        @{member.username} · {member.email}
                      </p>
                    </div>
                    <span className="member-role">{member.role}</span>
                    {user.workspace.role === "admin" ? (
                      <div className="member-controls">
                        <label>
                          <span>Role</span>
                          <select
                            disabled={isSelf || isUpdatingMember}
                            onChange={(event) => {
                              void handleUpdateTeamMember(member.id, {
                                role: event.target.value as TeamMember["role"],
                              });
                            }}
                            value={member.role}
                          >
                            <option value="member">Member</option>
                            <option value="admin">Admin</option>
                          </select>
                        </label>
                        <label className="member-active-toggle">
                          <input
                            checked={member.is_active}
                            disabled={isSelf || isUpdatingMember}
                            onChange={(event) => {
                              void handleUpdateTeamMember(member.id, {
                                is_active: event.target.checked,
                              });
                            }}
                            type="checkbox"
                          />
                          <span>{member.is_active ? "Active" : "Inactive"}</span>
                        </label>
                      </div>
                    ) : null}
                  </article>
                );
              })}
            </div>
          ) : (
            <div className="project-empty">No team members yet</div>
          )}

          {user.workspace.role === "admin" ? (
            <form className="team-member-form" onSubmit={handleCreateTeamMember}>
              <label>
                <span>Email</span>
                <input
                  autoComplete="off"
                  onChange={(event) => setTeamMemberEmail(event.target.value)}
                  placeholder="member@example.com"
                  type="email"
                  value={teamMemberEmail}
                />
              </label>

              <label>
                <span>Username</span>
                <input
                  autoComplete="off"
                  maxLength={32}
                  onChange={(event) =>
                    setTeamMemberUsername(event.target.value.toLowerCase())
                  }
                  placeholder="member_name"
                  value={teamMemberUsername}
                />
              </label>

              <label>
                <span>Display name</span>
                <input
                  maxLength={80}
                  onChange={(event) => setTeamMemberDisplayName(event.target.value)}
                  placeholder="Member Name"
                  value={teamMemberDisplayName}
                />
              </label>

              <label>
                <span>Role</span>
                <select
                  onChange={(event) =>
                    setTeamMemberRole(event.target.value as TeamMember["role"])
                  }
                  value={teamMemberRole}
                >
                  <option value="member">Member</option>
                  <option value="admin">Admin</option>
                </select>
              </label>

              <label>
                <span>Password</span>
                <input
                  autoComplete="new-password"
                  minLength={8}
                  onChange={(event) => setTeamMemberPassword(event.target.value)}
                  placeholder="At least 8 characters"
                  type="password"
                  value={teamMemberPassword}
                />
              </label>

              <button
                disabled={
                  isCreatingTeamMember ||
                  teamMemberEmail.trim() === "" ||
                  teamMemberUsername.trim() === "" ||
                  teamMemberDisplayName.trim() === "" ||
                  teamMemberPassword.length < 8
                }
                type="submit"
              >
                {isCreatingTeamMember ? "Creating..." : "Create member"}
              </button>

              {teamMemberFormError ? (
                <p className="form-error">{teamMemberFormError}</p>
              ) : null}
            </form>
          ) : null}
        </section>

        <section
          className="labels-panel"
          aria-label="Labels"
          hidden={activeSection !== "labels"}
        >
          <header className="section-header">
            <div>
              <p className="eyebrow">Labels</p>
              <h2>Workspace labels</h2>
            </div>
            {isLoadingLabels ? <span className="muted">Loading</span> : null}
          </header>

          {labelsError ? <p className="form-error">{labelsError}</p> : null}

          {labels.length > 0 ? (
            <div className="label-list">
              {labels.map((label) => (
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
            <div className="labels-empty">No labels yet</div>
          )}

          <form className="label-form" onSubmit={handleCreateLabel}>
            <label>
              <span>Name</span>
              <input
                maxLength={40}
                onChange={(event) => setLabelName(event.target.value)}
                placeholder="frontend"
                value={labelName}
              />
            </label>
            <label>
              <span>Color</span>
              <input
                onChange={(event) => setLabelColor(event.target.value)}
                type="color"
                value={labelColor}
              />
            </label>
            <button disabled={isCreatingLabel || labelName.trim() === ""} type="submit">
              {isCreatingLabel ? "Creating..." : "Create label"}
            </button>
          </form>
        </section>

        <section
          className="projects-layout"
          aria-label="Projects"
          hidden={activeSection !== "projects"}
        >
          <div className="projects-panel">
            <header className="section-header">
              <div>
                <p className="eyebrow">Projects</p>
                <h2>Workspace projects</h2>
              </div>
              {isLoadingProjects ? <span className="muted">Loading</span> : null}
            </header>

            {projectsError ? <p className="form-error">{projectsError}</p> : null}

            {projects.length > 0 ? (
              <div className="project-list">
                {projects.map((project) => (
                  <article className="project-row" key={project.id}>
                    <span className="project-key">{project.key}</span>
                    <div>
                      <h3>{project.name}</h3>
                      <p>{project.description || "No description"}</p>
                    </div>
                    {user.workspace.role === "admin" ? (
                      <div className="project-row-actions">
                        <button
                          className="small-button danger-button"
                          disabled={archivingProjectIds.includes(project.id)}
                          onClick={() => {
                            void handleArchiveProject(project);
                          }}
                          type="button"
                        >
                          {archivingProjectIds.includes(project.id)
                            ? "Archiving"
                            : "Archive"}
                        </button>
                      </div>
                    ) : null}
                  </article>
                ))}
              </div>
            ) : (
              <div className="project-empty">No projects yet</div>
            )}
          </div>

          {user.workspace.role === "admin" ? (
            <form className="project-form" onSubmit={handleCreateProject}>
              <header className="section-header">
                <div>
                  <p className="eyebrow">Admin</p>
                  <h2>Create project</h2>
                </div>
              </header>

              <label>
                <span>Key</span>
                <input
                  maxLength={10}
                  onChange={(event) =>
                    setProjectKey(event.target.value.toUpperCase())
                  }
                  placeholder="CORE"
                  value={projectKey}
                />
              </label>

              <label>
                <span>Name</span>
                <input
                  maxLength={120}
                  onChange={(event) => setProjectName(event.target.value)}
                  placeholder="Core Platform"
                  value={projectName}
                />
              </label>

              <label>
                <span>Description</span>
                <textarea
                  onChange={(event) => setProjectDescription(event.target.value)}
                  placeholder="Main product workspace"
                  rows={4}
                  value={projectDescription}
                />
              </label>

              {projectFormError ? (
                <p className="form-error">{projectFormError}</p>
              ) : null}

              <button disabled={isCreatingProject} type="submit">
                {isCreatingProject ? "Creating..." : "Create project"}
              </button>
            </form>
          ) : null}
        </section>

        <section
          className="issues-layout"
          aria-label="Issues"
          hidden={activeSection !== "issues"}
        >
          <form className="issue-form" onSubmit={handleCreateIssue}>
            <header className="section-header">
              <div>
                <p className="eyebrow">Issues</p>
                <h2>Create issue</h2>
              </div>
            </header>

            <label>
              <span>Project</span>
              <select
                onChange={(event) => setSelectedProjectId(event.target.value)}
                value={selectedProjectId}
              >
                <option value="">Select project</option>
                {projects.map((project) => (
                  <option key={project.id} value={project.id}>
                    {project.key} · {project.name}
                  </option>
                ))}
              </select>
            </label>

            <label>
              <span>Title</span>
              <input
                maxLength={180}
                onChange={(event) => setIssueTitle(event.target.value)}
                placeholder="Create project board"
                value={issueTitle}
              />
            </label>

            <label>
              <span>Description</span>
              <textarea
                onChange={(event) => setIssueDescription(event.target.value)}
                placeholder="Short context for the team"
                rows={3}
                value={issueDescription}
              />
            </label>

            <label>
              <span>Assignee</span>
              <select
                onChange={(event) => setIssueAssigneeId(event.target.value)}
                value={issueAssigneeId}
              >
                <option value="">Unassigned</option>
                {teamMembers.map((member) => (
                  <option key={member.id} value={member.id}>
                    {member.display_name}
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
                        checked={newIssueLabelIds.includes(label.id)}
                        onChange={(event) =>
                          handleCreateIssueLabel(label.id, event.target.checked)
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

            <div className="field-grid">
              <label>
                <span>Type</span>
                <select
                  onChange={(event) => setIssueType(event.target.value as IssueType)}
                  value={issueType}
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
                    setIssuePriority(event.target.value as IssuePriority)
                  }
                  value={issuePriority}
                >
                  {Object.entries(priorityLabels).map(([value, label]) => (
                    <option key={value} value={value}>
                      {label}
                    </option>
                  ))}
                </select>
              </label>
            </div>

            <div className="field-grid">
              <label>
                <span>Status</span>
                <select
                  onChange={(event) =>
                    setIssueStatus(event.target.value as IssueStatus)
                  }
                  value={issueStatus}
                >
                  {columns.map((column) => (
                    <option key={column.status} value={column.status}>
                      {column.title}
                    </option>
                  ))}
                </select>
              </label>

              <label>
                <span>Due date</span>
                <input
                  onChange={(event) => setIssueDueDate(event.target.value)}
                  type="date"
                  value={issueDueDate}
                />
              </label>
            </div>

            {issueFormError ? <p className="form-error">{issueFormError}</p> : null}

            <button
              disabled={isCreatingIssue || projects.length === 0}
              type="submit"
            >
              {isCreatingIssue ? "Creating..." : "Create issue"}
            </button>
          </form>

          <div className="issues-panel">
            <header className="section-header">
              <div>
                <p className="eyebrow">Open work</p>
                <h2>Recent issues</h2>
              </div>
              {isLoadingIssues ? <span className="muted">Loading</span> : null}
            </header>

            <section className="issue-filters" aria-label="Issue filters">
              <label>
                <span>Search</span>
                <input
                  onChange={(event) => setIssueFilterQuery(event.target.value)}
                  placeholder="Key, title, description"
                  value={issueFilterQuery}
                />
              </label>

              <label>
                <span>Sort</span>
                <select
                  onChange={(event) => setIssueSort(event.target.value as IssueSort)}
                  value={issueSort}
                >
                  {Object.entries(issueSortLabels).map(([value, label]) => (
                    <option key={value} value={value}>
                      {label}
                    </option>
                  ))}
                </select>
              </label>

              <label>
                <span>Project</span>
                <select
                  onChange={(event) => setIssueFilterProjectId(event.target.value)}
                  value={issueFilterProjectId}
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
                    setIssueFilterStatus(event.target.value as IssueStatus | "")
                  }
                  value={issueFilterStatus}
                >
                  <option value="">All statuses</option>
                  {columns.map((column) => (
                    <option key={column.status} value={column.status}>
                      {column.title}
                    </option>
                  ))}
                </select>
              </label>

              <label>
                <span>Priority</span>
                <select
                  onChange={(event) =>
                    setIssueFilterPriority(event.target.value as IssuePriority | "")
                  }
                  value={issueFilterPriority}
                >
                  <option value="">All priorities</option>
                  {Object.entries(priorityLabels).map(([value, label]) => (
                    <option key={value} value={value}>
                      {label}
                    </option>
                  ))}
                </select>
              </label>

              <label>
                <span>Assignee</span>
                <select
                  onChange={(event) => setIssueFilterAssigneeId(event.target.value)}
                  value={issueFilterAssigneeId}
                >
                  <option value="">All assignees</option>
                  <option value="unassigned">Unassigned</option>
                  {teamMembers.map((member) => (
                    <option key={member.id} value={member.id}>
                      {member.display_name}
                    </option>
                  ))}
                </select>
              </label>

              <label>
                <span>Label</span>
                <select
                  onChange={(event) => setIssueFilterLabelId(event.target.value)}
                  value={issueFilterLabelId}
                >
                  <option value="">All labels</option>
                  {labels.map((label) => (
                    <option key={label.id} value={label.id}>
                      {label.name}
                    </option>
                  ))}
                </select>
              </label>

              <button
                className="small-button"
                disabled={!hasIssueFilters}
                onClick={() => {
                  setIssueFilterQuery("");
                  setIssueFilterProjectId("");
                  setIssueFilterStatus("");
                  setIssueFilterPriority("");
                  setIssueFilterAssigneeId("");
                  setIssueFilterLabelId("");
                }}
                type="button"
              >
                Clear
              </button>
            </section>

            <p className="filter-summary">{issueListSummary}</p>

            {issuesError ? <p className="form-error">{issuesError}</p> : null}

            {issues.length > 0 ? (
              <div className="issue-list">
                {issues.slice(0, 6).map((issue) => (
                  <article className="issue-row" key={issue.id}>
                    <span className="issue-key">{issue.issue_key}</span>
                    <div>
                      <h3>{issue.title}</h3>
                      <p>
                        {issueTypeLabels[issue.issue_type]} ·{" "}
                        {priorityLabels[issue.priority]} ·{" "}
                        {columns.find((column) => column.status === issue.status)
                          ?.title ?? issue.status}{" "}
                        · {memberDisplayName(teamMembers, issue.assignee_id)}
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
                    </div>
                    <div className="issue-row-actions">
                      <button
                        className="small-button"
                        onClick={() => {
                          void handleSelectIssue(issue.id);
                        }}
                        type="button"
                      >
                        Open
                      </button>
                      <button
                        className="small-button danger-button"
                        disabled={archivingIssueIds.includes(issue.id)}
                        onClick={() => {
                          void handleArchiveIssue(issue);
                        }}
                        type="button"
                      >
                        {archivingIssueIds.includes(issue.id)
                          ? "Archiving"
                          : "Archive"}
                      </button>
                    </div>
                  </article>
                ))}
              </div>
            ) : (
              <div className="project-empty">No issues yet</div>
            )}
          </div>
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
            <p className="form-error">{selectedIssueError}</p>
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
                        disabled={isUpdatingIssue || editIssueTitle.trim() === ""}
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
                    <p className="form-error">{commentsError}</p>
                  ) : null}

                  {issueComments.length > 0 ? (
                    <div className="comment-list">
                      {issueComments.map((comment) => (
                        <article className="comment-card" key={comment.id}>
                          <header>
                            <strong>{comment.author_display_name}</strong>
                            <span>{formatDateTime(comment.created_at)}</span>
                          </header>
                          <p>{comment.body}</p>
                        </article>
                      ))}
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
                      disabled={isCreatingComment || commentBody.trim() === ""}
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
                    <p className="form-error">{activityError}</p>
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
                    {teamMembers.map((member) => (
                      <option key={member.id} value={member.id}>
                        {member.display_name}
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
                    <strong>{selectedIssue.due_date ?? "No due date"}</strong>
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

        <section
          className="board"
          aria-label="Task board preview"
          hidden={activeSection !== "dashboard"}
        >
          {columns.map((column) => (
            <article className="board-column" key={column.title}>
              <header>
                <h2>{column.title}</h2>
                <span>
                  {issues.filter((issue) => issue.status === column.status).length}
                </span>
              </header>
              <div className="board-card-list">
                {issues
                  .filter((issue) => issue.status === column.status)
                  .map((issue) => (
                    <article className="issue-card" key={issue.id}>
                      <div className="issue-card-meta">
                        <span>{issue.issue_key}</span>
                        <span>{priorityLabels[issue.priority]}</span>
                      </div>
                      <h3>{issue.title}</h3>
                      {issue.due_date ? <p>Due {issue.due_date}</p> : null}
                      <p>Assignee: {memberDisplayName(teamMembers, issue.assignee_id)}</p>
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
                          onClick={() => {
                            void handleSelectIssue(issue.id);
                          }}
                          type="button"
                        >
                          Open
                        </button>
                        <button
                          className="small-button danger-button"
                          disabled={archivingIssueIds.includes(issue.id)}
                          onClick={() => {
                            void handleArchiveIssue(issue);
                          }}
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
                            onChange={(event) => {
                              void handleTransitionIssue(
                                issue.id,
                                event.target.value as IssueStatus,
                              );
                            }}
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
                  ))}

                {issues.filter((issue) => issue.status === column.status).length ===
                0 ? (
                  <div className="empty-state">No issues yet</div>
                ) : null}
              </div>
            </article>
          ))}
        </section>
      </section>
    </main>
  );
}
