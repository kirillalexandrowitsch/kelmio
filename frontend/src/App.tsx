import { FormEvent, useEffect, useState } from "react";
import "./styles.css";
import {
  ApiError,
  CurrentUser,
  Issue,
  IssueActivity,
  IssueComment,
  IssuePriority,
  IssueStatus,
  IssueType,
  Project,
  TeamMember,
  assignIssue,
  createIssue,
  createIssueComment,
  createProject,
  getIssue,
  getCurrentUser,
  listIssueActivity,
  listIssueComments,
  listIssues,
  listProjects,
  listTeamMembers,
  login,
  logout,
  transitionIssue,
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

function issueMatchesFilters(
  issue: Issue,
  projectId: string,
  status: IssueStatus | "",
  priority: IssuePriority | "",
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

  return true;
}

function statusLabel(status: string) {
  return columns.find((column) => column.status === status)?.title ?? status;
}

function activityTitle(activity: IssueActivity) {
  if (activity.action === "issue_created") {
    return "Created issue";
  }
  if (activity.action === "status_changed") {
    return "Changed status";
  }
  if (activity.action === "assignee_changed") {
    return "Changed assignee";
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
  const [projects, setProjects] = useState<Project[]>([]);
  const [projectsError, setProjectsError] = useState("");
  const [projectFormError, setProjectFormError] = useState("");
  const [isLoadingProjects, setIsLoadingProjects] = useState(false);
  const [isCreatingProject, setIsCreatingProject] = useState(false);
  const [teamMembers, setTeamMembers] = useState<TeamMember[]>([]);
  const [teamMembersError, setTeamMembersError] = useState("");
  const [isLoadingTeamMembers, setIsLoadingTeamMembers] = useState(false);
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
  const [issueFilterProjectId, setIssueFilterProjectId] = useState("");
  const [issueFilterStatus, setIssueFilterStatus] = useState<IssueStatus | "">("");
  const [issueFilterPriority, setIssueFilterPriority] = useState<
    IssuePriority | ""
  >("");
  const [transitioningIssueIds, setTransitioningIssueIds] = useState<string[]>([]);
  const [assigningIssueIds, setAssigningIssueIds] = useState<string[]>([]);
  const [selectedIssue, setSelectedIssue] = useState<Issue | null>(null);
  const [selectedIssueError, setSelectedIssueError] = useState("");
  const [isLoadingSelectedIssue, setIsLoadingSelectedIssue] = useState(false);
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
    setIsLoadingProjects(true);

    listProjects()
      .then((response) => {
        if (isMounted) {
          setProjects(response.projects);
          setSelectedProjectId((currentProjectId) => {
            if (currentProjectId) {
              return currentProjectId;
            }
            return response.projects[0]?.id ?? "";
          });
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
      setIssues([]);
      return;
    }

    let isMounted = true;
    setIssuesError("");
    setIssueFormError("");
    setIsLoadingIssues(true);

    listIssues({
      projectId: issueFilterProjectId || undefined,
      status: issueFilterStatus || undefined,
      priority: issueFilterPriority || undefined,
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
  }, [user, issueFilterProjectId, issueFilterStatus, issueFilterPriority]);

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
    setProjects([]);
    setTeamMembers([]);
    setIssues([]);
    setProjectsError("");
    setProjectFormError("");
    setTeamMembersError("");
    setIssuesError("");
    setIssueFormError("");
    setIssueFilterProjectId("");
    setIssueFilterStatus("");
    setIssueFilterPriority("");
    setTransitioningIssueIds([]);
    setAssigningIssueIds([]);
    setSelectedIssue(null);
    setSelectedIssueError("");
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

  async function refreshIssueActivity(issueId: string) {
    setActivityError("");

    try {
      const response = await listIssueActivity(issueId);
      setIssueActivity(response.activity);
    } catch {
      setActivityError("Could not load activity.");
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
      setIssues((currentIssues) =>
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
    } catch {
      setSelectedIssueError("Could not update assignee.");
    } finally {
      setAssigningIssueIds((currentIds) =>
        currentIds.filter((currentIssueId) => currentIssueId !== issueId),
      );
    }
  }

  async function handleSelectIssue(issueId: string) {
    const issuePreview = issues.find((issue) => issue.id === issueId);
    if (issuePreview) {
      setSelectedIssue(issuePreview);
    }

    setSelectedIssueError("");
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
      });

      if (
        issueMatchesFilters(
          issue,
          issueFilterProjectId,
          issueFilterStatus,
          issueFilterPriority,
        )
      ) {
        setIssues((currentIssues) => [issue, ...currentIssues]);
      }
      setSelectedIssue(issue);
      setIssueTitle("");
      setIssueDescription("");
      setIssueType("task");
      setIssuePriority("medium");
      setIssueStatus("todo");
      setIssueAssigneeId("");
      setIssueDueDate("");
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
    issueFilterPriority !== "";

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
          <a aria-current="page" href="/">
            Dashboard
          </a>
          <a href="/">Projects</a>
          <a href="/">Issues</a>
          <a href="/">Team</a>
        </nav>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">Dashboard</p>
            <h1>Good to see you, {user.display_name}</h1>
          </div>
          <div className="topbar-actions">
            <div className="status-pill">{user.workspace.role}</div>
            <button className="ghost-button" onClick={handleLogout} type="button">
              Log out
            </button>
          </div>
        </header>

        <section className="summary-grid" aria-label="Project summary">
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

        <section className="team-panel" aria-label="Team members">
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
              {teamMembers.map((member) => (
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
                </article>
              ))}
            </div>
          ) : (
            <div className="project-empty">No team members yet</div>
          )}
        </section>

        <section className="projects-layout" aria-label="Projects">
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

        <section className="issues-layout" aria-label="Issues">
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

              <button
                className="small-button"
                disabled={!hasIssueFilters}
                onClick={() => {
                  setIssueFilterProjectId("");
                  setIssueFilterStatus("");
                  setIssueFilterPriority("");
                }}
                type="button"
              >
                Clear
              </button>
            </section>

            <p className="filter-summary">
              {hasIssueFilters
                ? `${issues.length} issues match current filters`
                : "Showing latest issues across all projects"}
            </p>

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
                          ?.title ?? issue.status}
                      </p>
                    </div>
                    <button
                      className="small-button"
                      onClick={() => {
                        void handleSelectIssue(issue.id);
                      }}
                      type="button"
                    >
                      Open
                    </button>
                  </article>
                ))}
              </div>
            ) : (
              <div className="project-empty">No issues yet</div>
            )}
          </div>
        </section>

        <section className="issue-detail-panel" aria-label="Issue details">
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
              <button
                className="ghost-button"
                onClick={() => {
                  setSelectedIssue(null);
                  setSelectedIssueError("");
                }}
                type="button"
              >
                Close
              </button>
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

        <section className="board" aria-label="Task board preview">
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
