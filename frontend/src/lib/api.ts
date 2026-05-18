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

type ListProjectsResponse = {
  projects: Project[];
};

type CreateProjectInput = {
  key: string;
  name: string;
  description: string;
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
