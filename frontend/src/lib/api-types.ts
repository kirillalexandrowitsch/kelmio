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

export type AuthResponse = {
  user: CurrentUser;
};

export type CSRFTokenResponse = {
  csrf_token: string;
};

export type PasswordResetRequestResponse = {
  message: string;
};

export type PasswordResetPreview = {
  email: string;
  expires_at: string;
};

export type RuntimeVersion = {
  version: string;
  commit: string;
  environment: string;
  build_time: string | null;
};

export type Project = {
  id: string;
  key: string;
  name: string;
  description: string;
  created_by: string;
  created_at: string;
  archived_at: string | null;
  project_role: "lead" | "contributor" | "viewer" | "";
  can_write: boolean;
  can_manage: boolean;
};

export type ProjectRole = "lead" | "contributor" | "viewer";

export type ProjectMember = {
  project_id: string;
  user_id: string;
  email: string;
  username: string;
  display_name: string;
  role: ProjectRole;
  workspace_role: TeamMember["role"];
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type TeamMember = {
  id: string;
  email: string;
  username: string;
  display_name: string;
  role: "admin" | "member";
  is_active: boolean;
  joined_at: string;
};

export type TeamInviteStatus = "pending" | "accepted" | "revoked" | "expired";
export type TeamInviteEmailDeliveryStatus =
  | "not_sent"
  | "pending"
  | "processing"
  | "sent"
  | "failed";

export type TeamInvite = {
  id: string;
  workspace_id: string;
  email: string;
  role: TeamMember["role"];
  status: TeamInviteStatus;
  created_by: string;
  created_at: string;
  expires_at: string;
  accepted_at: string | null;
  revoked_at: string | null;
  email_delivery_status: TeamInviteEmailDeliveryStatus;
  email_queued_at: string | null;
  email_sent_at: string | null;
};

export type EmailDiagnosticsCounts = {
  pending: number;
  processing: number;
  sent: number;
  failed: number;
};

export type EmailDiagnosticsFailure = {
  id: string;
  email_type: string;
  recipient_email: string;
  attempt_count: number;
  last_error: string;
  created_at: string;
  updated_at: string;
  next_attempt_at: string;
  sent_at: string | null;
};

export type EmailDiagnostics = {
  total: number;
  counts: EmailDiagnosticsCounts;
  oldest_pending_at: string | null;
  oldest_processing_started_at: string | null;
  recent_terminal_failures: EmailDiagnosticsFailure[];
};

export type CreateTeamInviteResponse = TeamInvite & {
  accept_token: string;
  accept_url_path: string;
};

export type InvitePreview = {
  workspace_id: string;
  workspace_name: string;
  email: string;
  role: TeamMember["role"];
  expires_at: string;
};

export type AcceptInviteResponse = {
  accepted: boolean;
  workspace_id: string;
  email: string;
  username: string;
  role: TeamMember["role"];
};

export type Label = {
  id: string;
  name: string;
  color: string;
};

export type IssueStatus = "backlog" | "todo" | "in_progress" | "blocked" | "done";
export type WorkflowStatusCategory = "backlog" | "todo" | "in_progress" | "done";
export type WorkflowStatus = {
  id: string;
  key: string;
  name: string;
  color: string;
  category: WorkflowStatusCategory;
};
export type ProjectWorkflowStatus = WorkflowStatus & {
  project_id: string;
  position: number;
  created_at: string;
  updated_at: string;
  archived_at: string | null;
};
export type ProjectWorkflowTransition = {
  from_status_id: string;
  to_status_id: string;
  created_at: string;
};
export type ProjectWorkflow = {
  project_id: string;
  statuses: ProjectWorkflowStatus[];
  transitions: ProjectWorkflowTransition[];
};
export type CreateWorkflowStatusInput = {
  key: string;
  name: string;
  color: string;
  category: WorkflowStatusCategory;
};
export type UpdateWorkflowStatusInput = {
  name?: string;
  color?: string;
  category?: WorkflowStatusCategory;
};
export type WorkflowTransitionInput = {
  from_status_id: string;
  to_status_id: string;
};
export type ReplaceWorkflowTransitionsInput = {
  transitions: WorkflowTransitionInput[];
};
export type AutomationTriggerType =
  | "issue_created"
  | "status_changed"
  | "assignee_changed"
  | "priority_changed";
export type AutomationCondition =
  | { type: "issue_type"; value: IssueType }
  | { type: "workflow_status"; workflow_status_id: string }
  | { type: "priority"; value: IssuePriority }
  | { type: "assignee"; user_id: string | null }
  | { type: "reporter"; user_id: string }
  | { type: "label"; label_id: string };
export type AutomationAction =
  | { type: "change_workflow_status"; workflow_status_id: string }
  | { type: "change_assignee"; user_id: string | null }
  | { type: "change_priority"; value: IssuePriority }
  | { type: "add_label"; label_id: string }
  | { type: "remove_label"; label_id: string };
export type AutomationRule = {
  id: string;
  project_id: string;
  name: string;
  trigger_type: AutomationTriggerType;
  conditions: AutomationCondition[];
  actions: AutomationAction[];
  position: number;
  is_enabled: boolean;
  disabled_reason: string | null;
  created_by: string;
  created_at: string;
  updated_at: string;
};
export type CreateAutomationRuleInput = {
  name: string;
  trigger_type: AutomationTriggerType;
  conditions: AutomationCondition[];
  actions: AutomationAction[];
  is_enabled?: boolean;
};
export type UpdateAutomationRuleInput = Partial<CreateAutomationRuleInput>;
export type IssuePriority = "low" | "medium" | "high" | "critical";
export type IssueType = "task" | "bug" | "story" | "epic" | "subtask";
export type IssueLinkType = "blocks" | "relates";
export type IssueSort = "created_desc" | "created_asc" | "priority_desc" | "due_date_asc";
export type IssueDueFilter = "overdue" | "today" | "due_soon" | "no_due";
export type SprintStatus = "planned" | "active" | "completed";
export type NotificationType =
  | "issue_assigned"
  | "issue_mentioned"
  | "issue_commented"
  | "issue_automation_assigned"
  | "issue_automation_status_changed"
  | "sprint_started"
  | "sprint_completed";

export type Issue = {
  id: string;
  project_id: string;
  project_key: string;
  number: number;
  issue_key: string;
  title: string;
  description: string;
  issue_type: IssueType;
  status: string;
  workflow_status: WorkflowStatus;
  priority: IssuePriority;
  story_points: number;
  reporter_id: string;
  assignee_id: string | null;
  parent_issue_id: string | null;
  sprint_id: string | null;
  due_date: string | null;
  labels: Label[];
  created_at: string;
  updated_at: string;
};

export type IssueComment = {
  id: string;
  issue_id: string;
  author_id: string;
  author_display_name: string;
  body: string;
  created_at: string;
  updated_at: string;
};

export type IssueActivity = {
  id: string;
  issue_id: string;
  action:
    | "issue_created"
    | "issue_updated"
    | "status_changed"
    | "assignee_changed"
    | "labels_changed"
    | "issue_parent_changed"
    | "issue_link_created"
    | "issue_link_deleted"
    | "issue_archived"
    | "comment_added"
    | "comment_updated"
    | "comment_deleted"
    | string;
  actor_id: string | null;
  actor_display_name: string | null;
  payload: Record<string, string>;
  created_at: string;
};

export type IssueLinkIssue = {
  id: string;
  issue_key: string;
  title: string;
  issue_type: IssueType;
  status: string;
  workflow_status: WorkflowStatus;
  priority: IssuePriority;
};

export type IssueLink = {
  id: string;
  source_issue_id: string;
  target_issue_id: string;
  link_type: IssueLinkType;
  created_by: string;
  created_at: string;
  source_issue: IssueLinkIssue;
  target_issue: IssueLinkIssue;
};

export type Sprint = {
  id: string;
  workspace_id: string;
  project_id: string;
  project_key: string;
  project_name: string;
  name: string;
  goal: string;
  status: SprintStatus;
  start_date: string | null;
  end_date: string | null;
  created_by: string;
  created_at: string;
  completed_at: string | null;
  issue_count: number;
  done_count: number;
  points_total: number;
  points_done: number;
  points_open: number;
};

export type SavedIssueFilters = {
  query?: string;
  sort: IssueSort;
  projectId?: string;
  sprintId?: string;
  status?: string;
  workflowStatusId?: string;
  priority?: IssuePriority;
  assigneeId?: string;
  labelId?: string;
  due?: IssueDueFilter;
};

export type SavedFilter = {
  id: string;
  workspace_id: string;
  user_id: string;
  name: string;
  filters: SavedIssueFilters;
  created_at: string;
  updated_at: string;
};

export type AppNotification = {
  id: string;
  workspace_id: string;
  user_id: string;
  actor_id: string | null;
  actor_display_name: string | null;
  issue_id: string | null;
  issue_key: string | null;
  issue_title: string | null;
  notification_type: NotificationType;
  payload: Record<string, string>;
  read_at: string | null;
  created_at: string;
};

export type ListProjectsResponse = {
  projects: Project[];
};

export type ListProjectMembersResponse = {
  members: ProjectMember[];
};

export type ListAutomationRulesResponse = {
  automation_rules: AutomationRule[];
};

export type ListTeamMembersResponse = {
  members: TeamMember[];
};

export type ListTeamInvitesResponse = {
  invites: TeamInvite[];
};

export type CreateTeamMemberInput = {
  email: string;
  username: string;
  display_name: string;
  password: string;
  role: TeamMember["role"];
};

export type CreateTeamInviteInput = {
  email: string;
  role: TeamMember["role"];
};

export type AcceptTeamInviteInput = {
  username: string;
  display_name: string;
  password: string;
};

export type UpdateTeamMemberInput = {
  role?: TeamMember["role"];
  is_active?: boolean;
};

export type ListLabelsResponse = {
  labels: Label[];
};

export type ListIssuesResponse = {
  issues: Issue[];
  next_cursor?: string | null;
};

export type ListIssueCommentsResponse = {
  comments: IssueComment[];
};

export type ListIssueActivityResponse = {
  activity: IssueActivity[];
  next_cursor: string | null;
};

export type ListIssueLinksResponse = {
  links: IssueLink[];
};

export type ListSprintsResponse = {
  sprints: Sprint[];
};

export type ListSavedFiltersResponse = {
  saved_filters: SavedFilter[];
};

export type ListNotificationsResponse = {
  notifications: AppNotification[];
  next_cursor: string | null;
};

export type UnreadNotificationsCountResponse = {
  unread_count: number;
};

export type IssueFilters = {
  query?: string;
  sort?: IssueSort;
  projectId?: string;
  sprintId?: string;
  status?: string;
  workflowStatusId?: string;
  priority?: IssuePriority;
  assigneeId?: string;
  labelId?: string;
  due?: IssueDueFilter;
};

export type PaginationParams = {
  limit?: number;
  cursor?: string;
};

export type SprintFilters = {
  projectId?: string;
  status?: SprintStatus;
};

export type CreateSavedFilterInput = {
  name: string;
  filters: SavedIssueFilters;
};

export type UpdateSavedFilterInput = {
  name?: string;
  filters?: SavedIssueFilters;
};

export type CreateProjectInput = {
  key: string;
  name: string;
  description: string;
};

export type UpdateProjectInput = {
  name: string;
  description: string;
};

export type UpdateProjectMemberInput = {
  role: ProjectRole;
};

export type CreateIssueInput = {
  project_id: string;
  parent_issue_id?: string;
  title: string;
  description: string;
  issue_type: IssueType;
  status?: string;
  workflow_status_id?: string;
  priority: IssuePriority;
  story_points: number;
  assignee_id: string;
  due_date: string;
  label_ids: string[];
};

export type CreateSubtaskInput = {
  title: string;
  description: string;
  status?: string;
  workflow_status_id?: string;
  priority: IssuePriority;
  story_points: number;
  assignee_id: string;
  due_date: string;
  label_ids: string[];
};

export type CreateIssueLinkInput = {
  target_issue_id: string;
  link_type: IssueLinkType;
};

export type TransitionIssueInput = {
  status?: string;
  workflow_status_id?: string;
};

export type CreateSprintInput = {
  project_id: string;
  name: string;
  goal: string;
  start_date: string;
  end_date: string;
};

export type CreateLabelInput = {
  name: string;
  color: string;
};

export type UpdateIssueInput = {
  title: string;
  description: string;
  issue_type: IssueType;
  priority: IssuePriority;
  story_points: number;
  due_date: string;
};

export type UpdateSprintInput = {
  name: string;
  goal: string;
  start_date: string;
  end_date: string;
};
