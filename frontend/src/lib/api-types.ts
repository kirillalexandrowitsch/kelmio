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
export type IssueType = "task" | "bug" | "story";
export type IssueSort = "created_desc" | "created_asc" | "priority_desc" | "due_date_asc";
export type IssueDueFilter = "overdue" | "today" | "due_soon" | "no_due";

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

export type IssueFilters = {
  query?: string;
  sort?: IssueSort;
  projectId?: string;
  status?: IssueStatus;
  priority?: IssuePriority;
  assigneeId?: string;
  labelId?: string;
  due?: IssueDueFilter;
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
  title: string;
  description: string;
  issue_type: IssueType;
  status: IssueStatus;
  priority: IssuePriority;
  assignee_id: string;
  due_date: string;
  label_ids: string[];
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
  due_date: string;
};
