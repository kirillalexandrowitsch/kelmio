import {
  type AcceptInviteResponse,
  type AcceptTeamInviteInput,
  type AppNotification,
  type AuthResponse,
  type CSRFTokenResponse,
  type CreateIssueInput,
  type CreateIssueLinkInput,
  type CreateLabelInput,
  type CreateProjectInput,
  type CreateSavedFilterInput,
  type CreateSprintInput,
  type CreateSubtaskInput,
  type CreateTeamInviteInput,
  type CreateTeamInviteResponse,
  type CreateTeamMemberInput,
  type InvitePreview,
  type Issue,
  type IssueComment,
  type IssueFilters,
  type IssueLink,
  type IssueStatus,
  type Label,
  type ListIssueActivityResponse,
  type ListIssueCommentsResponse,
  type ListIssueLinksResponse,
  type ListIssuesResponse,
  type ListLabelsResponse,
  type ListNotificationsResponse,
  type ListProjectsResponse,
  type ListSavedFiltersResponse,
  type ListSprintsResponse,
  type ListTeamInvitesResponse,
  type ListTeamMembersResponse,
  type Project,
  type RuntimeVersion,
  type SavedFilter,
  type Sprint,
  type SprintFilters,
  type SprintStatus,
  type TeamInvite,
  type TeamMember,
  type UnreadNotificationsCountResponse,
  type UpdateIssueInput,
  type UpdateProjectInput,
  type UpdateSavedFilterInput,
  type UpdateSprintInput,
  type UpdateTeamMemberInput,
} from "./api-types";
import {
  CSRF_HEADER_NAME,
  CSRF_TOKEN_PATH,
  isCSRFError,
  requestNeedsCSRF,
} from "./csrf";
export type {
  AcceptInviteResponse,
  AcceptTeamInviteInput,
  AppNotification,
  CSRFTokenResponse,
  CurrentUser,
  CreateTeamInviteInput,
  CreateTeamInviteResponse,
  InvitePreview,
  Issue,
  IssueActivity,
  IssueComment,
  IssueDueFilter,
  IssueFilters,
  IssueLink,
  IssueLinkType,
  IssuePriority,
  IssueSort,
  IssueStatus,
  IssueType,
  Label,
  NotificationType,
  Project,
  RuntimeVersion,
  SavedFilter,
  SavedIssueFilters,
  Sprint,
  SprintFilters,
  SprintStatus,
  TeamInvite,
  TeamInviteStatus,
  TeamMember,
  CreateIssueLinkInput,
  CreateSavedFilterInput,
  CreateSprintInput,
  CreateSubtaskInput,
  UpdateIssueInput,
  UpdateSavedFilterInput,
  UpdateSprintInput,
} from "./api-types";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export const API_UNAUTHORIZED_EVENT = "team-task-tracker:unauthorized";

let csrfToken: string | null = null;
let csrfTokenRequest: Promise<string> | null = null;

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

export async function getRuntimeVersion() {
  return request<RuntimeVersion>("/api/v1/version");
}

export async function logout() {
  await request<void>("/api/v1/auth/logout", {
    method: "POST",
  });
}

export async function changePassword(currentPassword: string, newPassword: string) {
  await request<void>("/api/v1/auth/password", {
    method: "PATCH",
    body: JSON.stringify({
      current_password: currentPassword,
      new_password: newPassword,
    }),
  });
}

export async function updateProfile(displayName: string) {
  return request<AuthResponse>("/api/v1/auth/profile", {
    method: "PATCH",
    body: JSON.stringify({
      display_name: displayName,
    }),
  });
}

export async function listProjects() {
  return request<ListProjectsResponse>("/api/v1/projects");
}

export async function getProject(projectId: string) {
  return request<Project>(`/api/v1/projects/${encodeURIComponent(projectId)}`);
}

export async function createProject(input: CreateProjectInput) {
  return request<Project>("/api/v1/projects", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateProject(projectId: string, input: UpdateProjectInput) {
  return request<Project>(`/api/v1/projects/${encodeURIComponent(projectId)}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function archiveProject(projectId: string) {
  await request<void>(`/api/v1/projects/${encodeURIComponent(projectId)}/archive`, {
    method: "POST",
  });
}

export async function listTeamMembers() {
  return request<ListTeamMembersResponse>("/api/v1/team/members");
}

export async function listTeamInvites() {
  return request<ListTeamInvitesResponse>("/api/v1/team/invites");
}

export async function createTeamInvite(input: CreateTeamInviteInput) {
  return request<CreateTeamInviteResponse>("/api/v1/team/invites", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function revokeTeamInvite(inviteId: string) {
  return request<TeamInvite>(
    `/api/v1/team/invites/${encodeURIComponent(inviteId)}/revoke`,
    {
      method: "POST",
    },
  );
}

export async function getTeamInvitePreview(token: string) {
  return request<InvitePreview>(
    `/api/v1/auth/invites/${encodeURIComponent(token)}`,
  );
}

export async function acceptTeamInvite(
  token: string,
  input: AcceptTeamInviteInput,
) {
  return request<AcceptInviteResponse>(
    `/api/v1/auth/invites/${encodeURIComponent(token)}/accept`,
    {
      method: "POST",
      body: JSON.stringify(input),
    },
  );
}

export async function createTeamMember(input: CreateTeamMemberInput) {
  return request<TeamMember>("/api/v1/team/members", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateTeamMember(memberId: string, input: UpdateTeamMemberInput) {
  return request<TeamMember>(
    `/api/v1/team/members/${encodeURIComponent(memberId)}`,
    {
      method: "PATCH",
      body: JSON.stringify(input),
    },
  );
}

export async function resetTeamMemberPassword(memberId: string, password: string) {
  await request<void>(
    `/api/v1/team/members/${encodeURIComponent(memberId)}/password`,
    {
      method: "PATCH",
      body: JSON.stringify({ password }),
    },
  );
}

export async function listLabels() {
  return request<ListLabelsResponse>("/api/v1/labels");
}

export async function createLabel(input: CreateLabelInput) {
  return request<Label>("/api/v1/labels", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function deleteLabel(labelId: string) {
  await request<void>(`/api/v1/labels/${encodeURIComponent(labelId)}`, {
    method: "DELETE",
  });
}

export async function listSavedFilters() {
  return request<ListSavedFiltersResponse>("/api/v1/saved-filters");
}

export async function createSavedFilter(input: CreateSavedFilterInput) {
  return request<SavedFilter>("/api/v1/saved-filters", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateSavedFilter(
  savedFilterId: string,
  input: UpdateSavedFilterInput,
) {
  return request<SavedFilter>(
    `/api/v1/saved-filters/${encodeURIComponent(savedFilterId)}`,
    {
      method: "PATCH",
      body: JSON.stringify(input),
    },
  );
}

export async function deleteSavedFilter(savedFilterId: string) {
  await request<void>(
    `/api/v1/saved-filters/${encodeURIComponent(savedFilterId)}`,
    {
      method: "DELETE",
    },
  );
}

export async function listNotifications() {
  return request<ListNotificationsResponse>("/api/v1/notifications");
}

export async function getUnreadNotificationsCount() {
  return request<UnreadNotificationsCountResponse>(
    "/api/v1/notifications/unread-count",
  );
}

export async function markNotificationRead(notificationId: string) {
  return request<AppNotification>(
    `/api/v1/notifications/${encodeURIComponent(notificationId)}/read`,
    {
      method: "POST",
    },
  );
}

export async function markAllNotificationsRead() {
  await request<void>("/api/v1/notifications/read-all", {
    method: "POST",
  });
}

export async function listIssues(filters: IssueFilters = {}) {
  const params = new URLSearchParams();
  if (filters.query) {
    params.set("q", filters.query);
  }
  if (filters.sort) {
    params.set("sort", filters.sort);
  }
  if (filters.projectId) {
    params.set("project_id", filters.projectId);
  }
  if (filters.sprintId) {
    params.set("sprint_id", filters.sprintId);
  }
  if (filters.status) {
    params.set("status", filters.status);
  }
  if (filters.priority) {
    params.set("priority", filters.priority);
  }
  if (filters.assigneeId) {
    params.set("assignee_id", filters.assigneeId);
  }
  if (filters.labelId) {
    params.set("label_id", filters.labelId);
  }
  if (filters.due) {
    params.set("due", filters.due);
  }

  const query = params.toString();
  return request<ListIssuesResponse>(`/api/v1/issues${query ? `?${query}` : ""}`);
}

export async function getIssue(issueId: string) {
  return request<Issue>(`/api/v1/issues/${encodeURIComponent(issueId)}`);
}

export async function listIssueComments(issueId: string) {
  return request<ListIssueCommentsResponse>(
    `/api/v1/issues/${encodeURIComponent(issueId)}/comments`,
  );
}

export async function listIssueActivity(issueId: string) {
  return request<ListIssueActivityResponse>(
    `/api/v1/issues/${encodeURIComponent(issueId)}/activity`,
  );
}

export async function listIssueChildren(issueId: string) {
  return request<ListIssuesResponse>(
    `/api/v1/issues/${encodeURIComponent(issueId)}/children`,
  );
}

export async function listIssueLinks(issueId: string) {
  return request<ListIssueLinksResponse>(
    `/api/v1/issues/${encodeURIComponent(issueId)}/links`,
  );
}

export async function createIssue(input: CreateIssueInput) {
  return request<Issue>("/api/v1/issues", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function createSubtask(issueId: string, input: CreateSubtaskInput) {
  return request<Issue>(`/api/v1/issues/${encodeURIComponent(issueId)}/subtasks`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function createIssueLink(
  issueId: string,
  input: CreateIssueLinkInput,
) {
  return request<IssueLink>(`/api/v1/issues/${encodeURIComponent(issueId)}/links`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateIssue(issueId: string, input: UpdateIssueInput) {
  return request<Issue>(`/api/v1/issues/${encodeURIComponent(issueId)}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function setIssueParent(issueId: string, parentIssueId: string | null) {
  return request<Issue>(`/api/v1/issues/${encodeURIComponent(issueId)}/parent`, {
    method: "PATCH",
    body: JSON.stringify({ parent_issue_id: parentIssueId }),
  });
}

export async function transitionIssue(issueId: string, status: IssueStatus) {
  return request<Issue>(`/api/v1/issues/${encodeURIComponent(issueId)}/transition`, {
    method: "POST",
    body: JSON.stringify({ status }),
  });
}

export async function assignIssue(issueId: string, assigneeId: string) {
  return request<Issue>(`/api/v1/issues/${encodeURIComponent(issueId)}/assign`, {
    method: "POST",
    body: JSON.stringify({ assignee_id: assigneeId }),
  });
}

export async function setIssueLabels(issueId: string, labelIds: string[]) {
  return request<Issue>(`/api/v1/issues/${encodeURIComponent(issueId)}/labels`, {
    method: "PUT",
    body: JSON.stringify({ label_ids: labelIds }),
  });
}

export async function archiveIssue(issueId: string) {
  await request<void>(`/api/v1/issues/${encodeURIComponent(issueId)}/archive`, {
    method: "POST",
  });
}

export async function deleteIssueLink(issueId: string, linkId: string) {
  await request<void>(
    `/api/v1/issues/${encodeURIComponent(issueId)}/links/${encodeURIComponent(
      linkId,
    )}`,
    {
      method: "DELETE",
    },
  );
}

export async function listSprints(filters: SprintFilters = {}) {
  const params = new URLSearchParams();
  if (filters.projectId) {
    params.set("project_id", filters.projectId);
  }
  if (filters.status) {
    params.set("status", filters.status);
  }

  const query = params.toString();
  return request<ListSprintsResponse>(
    `/api/v1/sprints${query ? `?${query}` : ""}`,
  );
}

export async function getSprint(sprintId: string) {
  return request<Sprint>(`/api/v1/sprints/${encodeURIComponent(sprintId)}`);
}

export async function createSprint(input: CreateSprintInput) {
  return request<Sprint>("/api/v1/sprints", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateSprint(sprintId: string, input: UpdateSprintInput) {
  return request<Sprint>(`/api/v1/sprints/${encodeURIComponent(sprintId)}`, {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}

export async function startSprint(sprintId: string) {
  return request<Sprint>(`/api/v1/sprints/${encodeURIComponent(sprintId)}/start`, {
    method: "POST",
  });
}

export async function addIssueToSprint(sprintId: string, issueId: string) {
  return request<Sprint>(
    `/api/v1/sprints/${encodeURIComponent(sprintId)}/issues`,
    {
      method: "POST",
      body: JSON.stringify({ issue_id: issueId }),
    },
  );
}

export async function removeIssueFromSprint(sprintId: string, issueId: string) {
  await request<void>(
    `/api/v1/sprints/${encodeURIComponent(sprintId)}/issues/${encodeURIComponent(
      issueId,
    )}`,
    {
      method: "DELETE",
    },
  );
}

export async function completeSprint(sprintId: string) {
  return request<Sprint>(
    `/api/v1/sprints/${encodeURIComponent(sprintId)}/complete`,
    {
      method: "POST",
    },
  );
}

export async function createIssueComment(issueId: string, body: string) {
  return request<IssueComment>(
    `/api/v1/issues/${encodeURIComponent(issueId)}/comments`,
    {
      method: "POST",
      body: JSON.stringify({ body }),
    },
  );
}

export async function updateIssueComment(
  issueId: string,
  commentId: string,
  body: string,
) {
  return request<IssueComment>(
    `/api/v1/issues/${encodeURIComponent(issueId)}/comments/${encodeURIComponent(
      commentId,
    )}`,
    {
      method: "PATCH",
      body: JSON.stringify({ body }),
    },
  );
}

export async function deleteIssueComment(issueId: string, commentId: string) {
  await request<void>(
    `/api/v1/issues/${encodeURIComponent(issueId)}/comments/${encodeURIComponent(
      commentId,
    )}`,
    {
      method: "DELETE",
    },
  );
}

async function request<T>(
  path: string,
  init: RequestInit = {},
  allowCSRFRetry = true,
): Promise<T> {
  const method = (init.method ?? "GET").toUpperCase();
  const headers = new Headers(init.headers);
  if (!headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const needsCSRF = requestNeedsCSRF(path, method);
  if (needsCSRF) {
    headers.set(CSRF_HEADER_NAME, await getCSRFToken());
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    credentials: "include",
    headers,
  });

  if (response.status === 204) {
    if (path === "/api/v1/auth/logout") {
      resetCSRFToken();
    }
    return undefined as T;
  }

  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    const errorCode = payload?.error?.code;
    if (allowCSRFRetry && needsCSRF && isCSRFError(response.status, errorCode)) {
      resetCSRFToken();
      return request<T>(path, init, false);
    }

    if (
      response.status === 401 &&
      path !== "/api/v1/auth/login" &&
      path !== "/api/v1/auth/me"
    ) {
      resetCSRFToken();
      dispatchUnauthorizedEvent();
    }

    const message =
      payload?.error?.message ?? `Request failed with status ${response.status}`;
    throw new ApiError(message, response.status);
  }

  if (path === "/api/v1/auth/login" || path === "/api/v1/auth/logout") {
    resetCSRFToken();
  }

  return payload as T;
}

async function getCSRFToken() {
  if (csrfToken) {
    return csrfToken;
  }

  if (!csrfTokenRequest) {
    csrfTokenRequest = fetch(`${API_BASE_URL}${CSRF_TOKEN_PATH}`, {
      credentials: "include",
    })
      .then(async (response) => {
        const payload = await response.json().catch(() => null);
        if (!response.ok) {
          if (response.status === 401) {
            resetCSRFToken();
            dispatchUnauthorizedEvent();
          }

          const message =
            payload?.error?.message ??
            `Request failed with status ${response.status}`;
          throw new ApiError(message, response.status);
        }

        const token = (payload as CSRFTokenResponse | null)?.csrf_token;
        if (!token) {
          throw new ApiError("CSRF token response is invalid", response.status);
        }

        csrfToken = token;
        return token;
      })
      .finally(() => {
        csrfTokenRequest = null;
      });
  }

  return csrfTokenRequest;
}

function resetCSRFToken() {
  csrfToken = null;
  csrfTokenRequest = null;
}

function dispatchUnauthorizedEvent() {
  if (typeof document === "undefined" || typeof window === "undefined") {
    return;
  }

  const event = document.createEvent("Event");
  event.initEvent(API_UNAUTHORIZED_EVENT, false, false);
  window.dispatchEvent(event);
}
