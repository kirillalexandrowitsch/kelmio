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

export type Project = {
  id: string;
  key: string;
  name: string;
  description: string;
  created_by: string;
  created_at: string;
  archived_at: string | null;
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

export type Label = {
  id: string;
  name: string;
  color: string;
};

export type IssueStatus = "backlog" | "todo" | "in_progress" | "blocked" | "done";
export type IssuePriority = "low" | "medium" | "high" | "critical";
export type IssueType = "task" | "bug" | "story" | "epic" | "subtask";
export type IssueLinkType = "blocks" | "relates";
export type IssueSort = "created_desc" | "created_asc" | "priority_desc" | "due_date_asc";
export type IssueDueFilter = "overdue" | "today" | "due_soon" | "no_due";
export type SprintStatus = "planned" | "active" | "completed";

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
  status: IssueStatus;
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
  status?: IssueStatus;
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

export type ListProjectsResponse = {
  projects: Project[];
};

export type ListTeamMembersResponse = {
  members: TeamMember[];
};

export type CreateTeamMemberInput = {
  email: string;
  username: string;
  display_name: string;
  password: string;
  role: TeamMember["role"];
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
};

export type ListIssueCommentsResponse = {
  comments: IssueComment[];
};

export type ListIssueActivityResponse = {
  activity: IssueActivity[];
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

export type IssueFilters = {
  query?: string;
  sort?: IssueSort;
  projectId?: string;
  sprintId?: string;
  status?: IssueStatus;
  priority?: IssuePriority;
  assigneeId?: string;
  labelId?: string;
  due?: IssueDueFilter;
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

export type CreateIssueInput = {
  project_id: string;
  parent_issue_id?: string;
  title: string;
  description: string;
  issue_type: IssueType;
  status: IssueStatus;
  priority: IssuePriority;
  story_points: number;
  assignee_id: string;
  due_date: string;
  label_ids: string[];
};

export type CreateSubtaskInput = {
  title: string;
  description: string;
  status: IssueStatus;
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
