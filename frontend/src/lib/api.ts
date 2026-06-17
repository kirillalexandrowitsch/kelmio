import {
  type AutomationRule,
  type CreateAutomationRuleInput,
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
  type CreateWorkflowStatusInput,
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
  type ListProjectMembersResponse,
  type ListAutomationRulesResponse,
  type ListProjectsResponse,
  type ListSavedFiltersResponse,
  type ListSprintsResponse,
  type ListTeamInvitesResponse,
  type ListTeamMembersResponse,
  type PaginationParams,
  type PasswordResetPreview,
  type PasswordResetRequestResponse,
  type Project,
  type ProjectMember,
  type ProjectRole,
  type ProjectWorkflow,
  type ProjectWorkflowStatus,
  type ReplaceWorkflowTransitionsInput,
  type RuntimeVersion,
  type SavedFilter,
  type Sprint,
  type SprintFilters,
  type SprintStatus,
  type TeamInvite,
  type TeamMember,
  type TransitionIssueInput,
  type UnreadNotificationsCountResponse,
  type UpdateIssueInput,
  type UpdateProjectInput,
  type UpdateProjectMemberInput,
  type UpdateSavedFilterInput,
  type UpdateSprintInput,
  type UpdateTeamMemberInput,
  type UpdateWorkflowStatusInput,
  type UpdateAutomationRuleInput,
  type WorkflowStatus,
  type WorkflowStatusCategory,
} from "./api-types";
import {
  CSRF_HEADER_NAME,
  CSRF_TOKEN_PATH,
  isCSRFError,
  requestNeedsCSRF,
} from "./csrf";
import { appendPaginationParams, collectPaginatedItems } from "./pagination";
export type {
  AutomationAction,
  AutomationCondition,
  AutomationRule,
  AutomationTriggerType,
  AcceptInviteResponse,
  AcceptTeamInviteInput,
  AppNotification,
  CSRFTokenResponse,
  CurrentUser,
  CreateTeamInviteInput,
  CreateTeamInviteResponse,
  CreateWorkflowStatusInput,
  CreateAutomationRuleInput,
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
  PaginationParams,
  PasswordResetPreview,
  PasswordResetRequestResponse,
  Project,
  ProjectMember,
  ProjectRole,
  ProjectWorkflow,
  ProjectWorkflowStatus,
  ReplaceWorkflowTransitionsInput,
  RuntimeVersion,
  SavedFilter,
  SavedIssueFilters,
  Sprint,
  SprintFilters,
  SprintStatus,
  TeamInvite,
  TeamInviteEmailDeliveryStatus,
  TeamInviteStatus,
  TeamMember,
  TransitionIssueInput,
  CreateIssueLinkInput,
  CreateSavedFilterInput,
  CreateSprintInput,
  CreateSubtaskInput,
  UpdateIssueInput,
  UpdateProjectMemberInput,
  UpdateSavedFilterInput,
  UpdateSprintInput,
  UpdateWorkflowStatusInput,
  UpdateAutomationRuleInput,
  WorkflowStatus,
  WorkflowStatusCategory,
  WorkflowTransitionInput,
} from "./api-types";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export const API_UNAUTHORIZED_EVENT = "team-task-tracker:unauthorized";

let csrfToken: string | null = null;
let csrfTokenRequest: Promise<string> | null = null;

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(message: string, status: number, code = "") {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
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

export async function requestPasswordReset(email: string) {
  return request<PasswordResetRequestResponse>(
    "/api/v1/auth/password-reset/request",
    {
      method: "POST",
      body: JSON.stringify({ email }),
    },
  );
}

export async function getPasswordResetPreview(token: string) {
  return request<PasswordResetPreview>(
    `/api/v1/auth/password-reset/${encodeURIComponent(token)}`,
  );
}

export async function completePasswordReset(
  token: string,
  password: string,
  confirmPassword: string,
) {
  await request<void>(
    `/api/v1/auth/password-reset/${encodeURIComponent(token)}/complete`,
    {
      method: "POST",
      body: JSON.stringify({
        password,
        confirm_password: confirmPassword,
      }),
    },
  );
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

export async function getProjectWorkflow(projectId: string) {
  return request<ProjectWorkflow>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/workflow`,
  );
}

export async function createWorkflowStatus(
  projectId: string,
  input: CreateWorkflowStatusInput,
) {
  return request<ProjectWorkflowStatus>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/workflow/statuses`,
    {
      method: "POST",
      body: JSON.stringify(input),
    },
  );
}

export async function updateWorkflowStatus(
  projectId: string,
  statusId: string,
  input: UpdateWorkflowStatusInput,
) {
  return request<ProjectWorkflowStatus>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/workflow/statuses/${encodeURIComponent(
      statusId,
    )}`,
    {
      method: "PATCH",
      body: JSON.stringify(input),
    },
  );
}

export async function reorderWorkflowStatuses(
  projectId: string,
  statusIds: string[],
) {
  return request<ProjectWorkflow>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/workflow/statuses/order`,
    {
      method: "PUT",
      body: JSON.stringify({ status_ids: statusIds }),
    },
  );
}

export async function archiveWorkflowStatus(
  projectId: string,
  statusId: string,
  replacementStatusId: string,
) {
  return request<ProjectWorkflowStatus>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/workflow/statuses/${encodeURIComponent(
      statusId,
    )}/archive`,
    {
      method: "POST",
      body: JSON.stringify({ replacement_status_id: replacementStatusId }),
    },
  );
}

export async function replaceWorkflowTransitions(
  projectId: string,
  input: ReplaceWorkflowTransitionsInput,
) {
  return request<ProjectWorkflow>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/workflow/transitions`,
    {
      method: "PUT",
      body: JSON.stringify(input),
    },
  );
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

export async function listProjectMembers(projectId: string) {
  return request<ListProjectMembersResponse>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/members`,
  );
}

export async function listAutomationRules(projectId: string) {
  return request<ListAutomationRulesResponse>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/automation-rules`,
  );
}

export async function createAutomationRule(
  projectId: string,
  input: CreateAutomationRuleInput,
) {
  return request<AutomationRule>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/automation-rules`,
    { method: "POST", body: JSON.stringify(input) },
  );
}

export async function updateAutomationRule(
  projectId: string,
  ruleId: string,
  input: UpdateAutomationRuleInput,
) {
  return request<AutomationRule>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/automation-rules/${encodeURIComponent(ruleId)}`,
    { method: "PATCH", body: JSON.stringify(input) },
  );
}

export async function deleteAutomationRule(projectId: string, ruleId: string) {
  await request<void>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/automation-rules/${encodeURIComponent(ruleId)}`,
    { method: "DELETE" },
  );
}

export async function reorderAutomationRules(projectId: string, ruleIds: string[]) {
  return request<ListAutomationRulesResponse>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/automation-rules/order`,
    { method: "PUT", body: JSON.stringify({ rule_ids: ruleIds }) },
  );
}

export async function putProjectMember(
  projectId: string,
  userId: string,
  input: UpdateProjectMemberInput,
) {
  return request<ProjectMember>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/members/${encodeURIComponent(
      userId,
    )}`,
    {
      method: "PUT",
      body: JSON.stringify(input),
    },
  );
}

export async function deleteProjectMember(projectId: string, userId: string) {
  await request<void>(
    `/api/v1/projects/${encodeURIComponent(projectId)}/members/${encodeURIComponent(
      userId,
    )}`,
    {
      method: "DELETE",
    },
  );
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

export async function resendTeamInvite(inviteId: string) {
  return request<TeamInvite>(
    `/api/v1/team/invites/${encodeURIComponent(inviteId)}/resend`,
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

export async function listNotifications(pagination: PaginationParams = {}) {
  const params = new URLSearchParams();
  appendPaginationParams(params, pagination);

  const query = params.toString();
  return request<ListNotificationsResponse>(
    `/api/v1/notifications${query ? `?${query}` : ""}`,
  );
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

export async function listIssues(
  filters: IssueFilters = {},
  pagination: PaginationParams = {},
) {
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
  if (filters.workflowStatusId) {
    params.set("workflow_status_id", filters.workflowStatusId);
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
  appendPaginationParams(params, pagination);

  const query = params.toString();
  return request<ListIssuesResponse>(`/api/v1/issues${query ? `?${query}` : ""}`);
}

export function listAllIssues(filters: IssueFilters = {}) {
  return collectPaginatedItems<Issue>(async (cursor) => {
    const response = await listIssues(filters, { limit: 100, cursor });
    return {
      items: response.issues,
      nextCursor: response.next_cursor,
    };
  });
}

export async function getIssue(issueId: string) {
  return request<Issue>(`/api/v1/issues/${encodeURIComponent(issueId)}`);
}

export async function listIssueComments(issueId: string) {
  return request<ListIssueCommentsResponse>(
    `/api/v1/issues/${encodeURIComponent(issueId)}/comments`,
  );
}

export async function listIssueActivity(
  issueId: string,
  pagination: PaginationParams = {},
) {
  const params = new URLSearchParams();
  appendPaginationParams(params, pagination);

  const query = params.toString();
  return request<ListIssueActivityResponse>(
    `/api/v1/issues/${encodeURIComponent(issueId)}/activity${query ? `?${query}` : ""}`,
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

export async function transitionIssue(
  issueId: string,
  input: IssueStatus | TransitionIssueInput,
) {
  return request<Issue>(`/api/v1/issues/${encodeURIComponent(issueId)}/transition`, {
    method: "POST",
    body: JSON.stringify(typeof input === "string" ? { status: input } : input),
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
    throw new ApiError(message, response.status, errorCode ?? "");
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
          throw new ApiError(message, response.status, payload?.error?.code ?? "");
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
