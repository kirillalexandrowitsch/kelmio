const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export type CurrentUser = {
  id: string;
  email: string;
  username: string;
  display_name: string;
  workspace: {
    id: string;
    role: "admin" | "member";
  };
};

type AuthResponse = {
  user: CurrentUser;
};

export type Project = {
  id: string;
  key: string;
  name: string;
  description: string;
  created_by: string;
  created_at: string;
  archived_at: string | null;
};

export type IssueStatus = "backlog" | "todo" | "in_progress" | "blocked" | "done";
export type IssuePriority = "low" | "medium" | "high" | "critical";
export type IssueType = "task" | "bug" | "story";

export type Issue = {
  id: string;
  project_id: string;
  project_key: string;
  number: number;
  issue_key: string;
  title: string;
  description: string;
  issue_type: IssueType;
  status: IssueStatus;
  priority: IssuePriority;
  reporter_id: string;
  assignee_id: string | null;
  due_date: string | null;
  created_at: string;
  updated_at: string;
};

type ListProjectsResponse = {
  projects: Project[];
};

type ListIssuesResponse = {
  issues: Issue[];
};

type CreateProjectInput = {
  key: string;
  name: string;
  description: string;
};

type CreateIssueInput = {
  project_id: string;
  title: string;
  description: string;
  issue_type: IssueType;
  status: IssueStatus;
  priority: IssuePriority;
  due_date: string;
};

export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export async function login(loginValue: string, password: string) {
  return request<AuthResponse>("/api/v1/auth/login", {
    method: "POST",
    body: JSON.stringify({
      login: loginValue,
      password,
    }),
  });
}

export async function getCurrentUser() {
  return request<AuthResponse>("/api/v1/auth/me");
}

export async function logout() {
  await request<void>("/api/v1/auth/logout", {
    method: "POST",
  });
}

export async function listProjects() {
  return request<ListProjectsResponse>("/api/v1/projects");
}

export async function createProject(input: CreateProjectInput) {
  return request<Project>("/api/v1/projects", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function listIssues(projectId?: string) {
  const params = new URLSearchParams();
  if (projectId) {
    params.set("project_id", projectId);
  }

  const query = params.toString();
  return request<ListIssuesResponse>(`/api/v1/issues${query ? `?${query}` : ""}`);
}

export async function getIssue(issueId: string) {
  return request<Issue>(`/api/v1/issues/${encodeURIComponent(issueId)}`);
}

export async function createIssue(input: CreateIssueInput) {
  return request<Issue>("/api/v1/issues", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function transitionIssue(issueId: string, status: IssueStatus) {
  return request<Issue>(`/api/v1/issues/${encodeURIComponent(issueId)}/transition`, {
    method: "POST",
    body: JSON.stringify({ status }),
  });
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...init.headers,
    },
  });

  if (response.status === 204) {
    return undefined as T;
  }

  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    const message =
      payload?.error?.message ?? `Request failed with status ${response.status}`;
    throw new ApiError(message, response.status);
  }

  return payload as T;
}
