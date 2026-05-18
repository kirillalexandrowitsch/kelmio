import { FormEvent, useEffect, useState } from "react";
import "./styles.css";
import {
  ApiError,
  CurrentUser,
  Project,
  createProject,
  getCurrentUser,
  listProjects,
  login,
  logout,
} from "./lib/api";

const columns = [
  { title: "Backlog", count: 0 },
  { title: "Todo", count: 0 },
  { title: "In progress", count: 0 },
  { title: "Done", count: 0 },
];

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
  const [projectKey, setProjectKey] = useState("");
  const [projectName, setProjectName] = useState("");
  const [projectDescription, setProjectDescription] = useState("");

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
    setProjectsError("");
    setProjectFormError("");
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
            <strong>0</strong>
          </article>
          <article>
            <span>Team members</span>
            <strong>1</strong>
          </article>
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

        <section className="board" aria-label="Task board preview">
          {columns.map((column) => (
            <article className="board-column" key={column.title}>
              <header>
                <h2>{column.title}</h2>
                <span>{column.count}</span>
              </header>
              <div className="empty-state">No issues yet</div>
            </article>
          ))}
        </section>
      </section>
    </main>
  );
}
