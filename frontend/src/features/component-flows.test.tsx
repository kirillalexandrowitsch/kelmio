import assert from "node:assert/strict";
import { type ComponentProps, type FormEvent } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { test, vi } from "vitest";

import { SignInScreen } from "./auth/auth-screens";
import { IssueCreateForm } from "./issues/issue-create-form";
import { SavedFiltersPanel } from "./issues/saved-filters-panel";
import { NotificationsSection } from "./notifications/notifications-section";
import { ProjectsSection } from "./projects/projects-section";
import { SprintsSection } from "./sprints/sprints-section";
import { TeamSection } from "./team/team-section";
import {
  type AppNotification,
  type CurrentUser,
  type EmailDiagnostics,
  type Project,
  type ProjectMember,
  type SavedFilter,
  type Sprint,
  type TeamMember,
} from "../lib/api-types";

const project: Project = {
  id: "project-1",
  key: "DEMO",
  name: "Demo Project",
  description: "",
  created_by: "admin-1",
  created_at: "2026-06-07T00:00:00Z",
  archived_at: null,
  project_role: "lead",
  can_write: true,
  can_manage: true,
};

const admin: CurrentUser = {
  id: "admin-1",
  email: "admin@example.com",
  username: "admin",
  display_name: "Admin",
  is_site_admin: true,
  workspace: { id: "workspace-1", role: "admin" },
  organization: { id: "organization-1", role: "org_admin" },
};

const memberUser: CurrentUser = {
  ...admin,
  id: "member-1",
  email: "member@example.com",
  username: "member",
  display_name: "Member",
  workspace: { id: "workspace-1", role: "member" },
};

const teamMember: TeamMember = {
  id: "member-1",
  email: "member@example.com",
  username: "member",
  display_name: "Member",
  role: "member",
  is_active: true,
  joined_at: "2026-06-07T00:00:00Z",
};

const projectMember: ProjectMember = {
  project_id: project.id,
  user_id: teamMember.id,
  email: teamMember.email,
  username: teamMember.username,
  display_name: teamMember.display_name,
  role: "contributor",
  workspace_role: teamMember.role,
  is_active: true,
  created_at: "2026-06-07T00:00:00Z",
  updated_at: "2026-06-07T00:00:00Z",
};

const sprint: Sprint = {
  id: "sprint-1",
  workspace_id: "workspace-1",
  project_id: project.id,
  project_key: project.key,
  project_name: project.name,
  name: "Sprint 1",
  goal: "Ship safely",
  status: "planned",
  start_date: "2026-06-07",
  end_date: "2026-06-14",
  created_by: admin.id,
  created_at: "2026-06-07T00:00:00Z",
  completed_at: null,
  issue_count: 0,
  done_count: 0,
  points_total: 0,
  points_done: 0,
  points_open: 0,
};

const savedFilter: SavedFilter = {
  id: "filter-1",
  workspace_id: "workspace-1",
  user_id: admin.id,
  name: "Blocked work",
  filters: { sort: "priority_desc", status: "blocked" },
  created_at: "2026-06-07T00:00:00Z",
  updated_at: "2026-06-07T00:00:00Z",
};

const notification: AppNotification = {
  id: "notification-1",
  workspace_id: "workspace-1",
  user_id: admin.id,
  actor_id: teamMember.id,
  actor_display_name: teamMember.display_name,
  issue_id: "issue-1",
  issue_key: "DEMO-1",
  issue_title: "Verify component tests",
  notification_type: "issue_commented",
  payload: { comment_preview: "Please review." },
  read_at: null,
  created_at: "2026-06-07T00:00:00Z",
};

function preventSubmit(event: FormEvent<HTMLFormElement>) {
  event.preventDefault();
}

test("sign-in form forwards credentials and submit action", async () => {
  const user = userEvent.setup();
  const onLoginChange = vi.fn();
  const onPasswordChange = vi.fn();
  const onForgotPassword = vi.fn();
  const onSubmit = vi.fn(preventSubmit);

  render(
    <SignInScreen
      canSignIn
      error=""
      isSubmitting={false}
      loginValue=""
      onForgotPassword={onForgotPassword}
      onLoginChange={onLoginChange}
      onPasswordChange={onPasswordChange}
      onSubmit={onSubmit}
      password=""
    />,
  );

  await user.type(screen.getByLabelText("Username or email"), "admin");
  await user.type(screen.getByLabelText("Password"), "secret123");
  await user.click(screen.getByRole("button", { name: "Sign in" }));

  assert.equal(onLoginChange.mock.calls.at(-1)?.[0], "n");
  assert.equal(onPasswordChange.mock.calls.at(-1)?.[0], "3");
  assert.equal(onSubmit.mock.calls.length, 1);

  await user.click(screen.getByRole("button", { name: "Forgot password?" }));
  assert.equal(onForgotPassword.mock.calls.length, 1);
});

test("issue create form exposes project, label, and disabled submit behavior", async () => {
  const user = userEvent.setup();
  const onProjectChange = vi.fn();
  const onLabelChange = vi.fn();

  render(
    <IssueCreateForm
      assigneeId=""
      canCreateIssue={false}
      description=""
      dueDate=""
      formError=""
      isCreatingIssue={false}
      labels={[{ id: "label-1", name: "bug", color: "#b24c42" }]}
      labelIds={[]}
      onAssigneeChange={vi.fn()}
      onCreateIssue={vi.fn(preventSubmit)}
      onDescriptionChange={vi.fn()}
      onDueDateChange={vi.fn()}
      onLabelChange={onLabelChange}
      onPriorityChange={vi.fn()}
      onProjectChange={onProjectChange}
      onStoryPointsChange={vi.fn()}
      onStatusChange={vi.fn()}
      onTitleChange={vi.fn()}
      onTypeChange={vi.fn()}
      priority="medium"
      projectId=""
      projects={[project]}
      statusId=""
      statuses={[]}
      storyPoints="0"
      teamMembers={[teamMember]}
      title=""
      type="task"
    />,
  );

  await user.selectOptions(screen.getByLabelText("Project"), project.id);
  await user.click(screen.getByRole("checkbox", { name: "bug" }));

  assert.equal(onProjectChange.mock.calls[0]?.[0], project.id);
  assert.deepEqual(onLabelChange.mock.calls[0], ["label-1", true]);
  assert.equal(
    screen.getByRole("button", { name: "Create issue" }).hasAttribute("disabled"),
    true,
  );
});

test("saved filters expose apply, rename, and delete actions", async () => {
  const user = userEvent.setup();
  const onApplySavedFilter = vi.fn();
  const onStartRenameSavedFilter = vi.fn();
  const onDeleteSavedFilter = vi.fn();

  render(
    <SavedFiltersPanel
      deletingSavedFilterIds={[]}
      isCreatingSavedFilter={false}
      isLoadingSavedFilters={false}
      onApplySavedFilter={onApplySavedFilter}
      onCancelRenameSavedFilter={vi.fn()}
      onCreateSavedFilter={vi.fn(preventSubmit)}
      onDeleteSavedFilter={onDeleteSavedFilter}
      onRenameSavedFilter={vi.fn()}
      onRenameSavedFilterNameChange={vi.fn()}
      onSavedFilterNameChange={vi.fn()}
      onStartRenameSavedFilter={onStartRenameSavedFilter}
      onUpdateSavedFilter={vi.fn()}
      renameSavedFilterId=""
      renameSavedFilterName=""
      savedFilterFormError=""
      savedFilterName=""
      savedFilters={[savedFilter]}
      savedFiltersError=""
      updatingSavedFilterIds={[]}
      workflowStatusNamesById={{}}
    />,
  );

  await user.click(screen.getByRole("button", { name: "Apply" }));
  await user.click(screen.getByRole("button", { name: "Rename" }));
  await user.click(screen.getByRole("button", { name: "Delete" }));

  assert.equal(onApplySavedFilter.mock.calls[0]?.[0], savedFilter);
  assert.equal(onStartRenameSavedFilter.mock.calls[0]?.[0], savedFilter);
  assert.equal(onDeleteSavedFilter.mock.calls[0]?.[0], savedFilter);
});

test("planned sprint exposes the start action and locks complete", async () => {
  const user = userEvent.setup();
  const onStartSprint = vi.fn();
  const props: ComponentProps<typeof SprintsSection> = {
    addingIssueToSprintIds: [],
    canCreateSprint: false,
    canUpdateSprint: true,
    completingSprintIds: [],
    editSprintEndDate: "",
    editSprintGoal: "",
    editSprintName: "",
    editSprintStartDate: "",
    isActive: true,
    isCreatingSprint: false,
    isEditingSprint: false,
    isLoadingSelectedSprint: false,
    isLoadingSprintPlanning: false,
    isLoadingSprints: false,
    isUpdatingSprint: false,
    onAddIssueToSprint: vi.fn(),
    onCancelSprintEdit: vi.fn(),
    onCompleteSprint: vi.fn(),
    onCreateSprint: vi.fn(preventSubmit),
    onEditSprintEndDateChange: vi.fn(),
    onEditSprintGoalChange: vi.fn(),
    onEditSprintNameChange: vi.fn(),
    onEditSprintStartDateChange: vi.fn(),
    onProjectFilterChange: vi.fn(),
    onRemoveIssueFromSprint: vi.fn(),
    onSelectSprint: vi.fn(),
    onSprintIssueDragStart: vi.fn(),
    onSprintIssueDrop: vi.fn(),
    onSprintEndDateChange: vi.fn(),
    onSprintGoalChange: vi.fn(),
    onSprintNameChange: vi.fn(),
    onSprintProjectChange: vi.fn(),
    onSprintStartDateChange: vi.fn(),
    onStartEditingSprint: vi.fn(),
    onStartSprint,
    onStatusFilterChange: vi.fn(),
    onTransitionIssue: vi.fn(),
    onUpdateSprint: vi.fn(preventSubmit),
    onViewSprintProjectIssues: vi.fn(),
    projectFilterId: "",
    projects: [project],
    removingIssueFromSprintIds: [],
    selectedSprint: sprint,
    selectedSprintBacklogIssues: [],
    selectedSprintError: "",
    selectedSprintIssues: [],
    selectedSprintWorkflow: undefined,
    selectedSprintWorkflowError: "",
    sprintEndDate: "",
    sprintFormError: "",
    sprintGoal: "",
    sprintName: "",
    sprintPlanningError: "",
    sprintProjectId: "",
    sprintStartDate: "",
    sprintStatusFilter: "",
    sprints: [sprint],
    sprintsError: "",
    startingSprintIds: [],
    teamMembers: [teamMember],
    today: new Date("2026-06-07T00:00:00Z"),
    transitioningIssueIds: [],
    isLoadingSelectedSprintWorkflow: false,
  };

  const { rerender } = render(<SprintsSection {...props} />);

  await user.click(screen.getByRole("button", { name: "Start sprint" }));

  assert.equal(onStartSprint.mock.calls[0]?.[0], sprint);
  assert.equal(
    screen.getByRole("button", { name: "Complete sprint" }).hasAttribute("disabled"),
    true,
  );

  const reviewStatus = {
    id: "status-review",
    project_id: project.id,
    key: "review",
    name: "Ready for review",
    color: "#0ea5e9",
    category: "in_progress" as const,
    position: 100,
    created_at: "2026-06-07T00:00:00Z",
    updated_at: "2026-06-07T00:00:00Z",
    archived_at: null,
  };
  const doneStatus = {
    ...reviewStatus,
    id: "status-done",
    key: "done",
    name: "Done",
    category: "done" as const,
    position: 200,
  };
  const sprintIssue = {
    id: "issue-1",
    project_id: project.id,
    project_key: project.key,
    number: 1,
    issue_key: "DEMO-1",
    title: "Dynamic sprint workflow",
    description: "",
    issue_type: "task" as const,
    status: reviewStatus.key,
    workflow_status: reviewStatus,
    priority: "medium" as const,
    story_points: 3,
    reporter_id: admin.id,
    assignee_id: null,
    parent_issue_id: null,
    sprint_id: sprint.id,
    due_date: null,
    labels: [],
    created_at: "2026-06-07T00:00:00Z",
    updated_at: "2026-06-07T00:00:00Z",
  };

  rerender(
    <SprintsSection
      {...props}
      selectedSprint={{ ...sprint, status: "active" }}
      selectedSprintIssues={[sprintIssue]}
      selectedSprintWorkflow={{
        project_id: project.id,
        statuses: [doneStatus, reviewStatus],
        transitions: [
          {
            from_status_id: reviewStatus.id,
            to_status_id: doneStatus.id,
            created_at: "2026-06-07T00:00:00Z",
          },
        ],
      }}
    />,
  );

  assert.ok(screen.getAllByText("Ready for review").length >= 1);
  assert.ok(screen.getAllByText("Dynamic sprint workflow").length >= 1);
  assert.equal(
    screen
      .getByLabelText("Status for DEMO-1")
      .querySelectorAll("option").length,
    2,
  );

  rerender(
    <SprintsSection
      {...props}
      canUpdateSprint={false}
      projects={[{ ...project, project_role: "viewer", can_write: false }]}
      selectedSprint={{ ...sprint, status: "active" }}
      selectedSprintIssues={[sprintIssue]}
      selectedSprintWorkflow={{
        project_id: project.id,
        statuses: [doneStatus, reviewStatus],
        transitions: [
          {
            from_status_id: reviewStatus.id,
            to_status_id: doneStatus.id,
            created_at: "2026-06-07T00:00:00Z",
          },
        ],
      }}
    />,
  );

  assert.ok(
    screen.getByText("This sprint board is read-only for your project role."),
  );
  assert.equal(
    screen.getByLabelText("Status for DEMO-1").hasAttribute("disabled"),
    true,
  );
});

test("notifications expose per-item and mark-all actions", async () => {
  const user = userEvent.setup();
  const onMarkAllRead = vi.fn();
  const onMarkRead = vi.fn();
  const onOpenIssue = vi.fn();

  render(
    <NotificationsSection
      error=""
      isActive
      isLoading={false}
      notifications={[notification]}
      onMarkAllRead={onMarkAllRead}
      onMarkRead={onMarkRead}
      onOpenIssue={onOpenIssue}
      unreadCount={1}
    />,
  );

  await user.click(screen.getByRole("button", { name: "Open issue" }));
  await user.click(screen.getByRole("button", { name: "Mark read" }));
  await user.click(screen.getByRole("button", { name: "Mark all read" }));

  assert.equal(onOpenIssue.mock.calls[0]?.[0], notification);
  assert.equal(onMarkRead.mock.calls[0]?.[0], notification);
  assert.equal(onMarkAllRead.mock.calls.length, 1);
});

test("notifications present automation actor and final change preview", () => {
  render(
    <NotificationsSection
      error=""
      isActive
      isLoading={false}
      notifications={[
        {
          ...notification,
          actor_id: null,
          actor_display_name: null,
          notification_type: "issue_automation_status_changed",
          payload: {
            from_status: "todo",
            to_status: "review",
          },
        },
      ]}
      onMarkAllRead={vi.fn()}
      onMarkRead={vi.fn()}
      onOpenIssue={vi.fn()}
      unreadCount={1}
    />,
  );

  assert.ok(screen.getByRole("heading", { name: "Automation changed issue status" }));
  assert.ok(screen.getByText("Todo -> Review"));
});

function projectsProps(selectedProjectDetail: Project) {
  return {
    automationRules: [],
    automationRulesError: "",
    archivingProjectIds: [],
    archivingWorkflowStatusIds: [],
    canCreateProject: true,
    creatingWorkflowStatus: false,
    deletingAutomationRuleIds: [],
    editProjectDescription: "",
    editProjectName: "",
    editingProjectId: "",
    isActive: true,
    isCreatingProject: false,
    isLoadingProjectDetail: false,
    isLoadingProjectMembers: false,
    isLoadingProjectWorkflow: false,
    isLoadingAutomationRules: false,
    isCreatingAutomationRule: false,
    isLoadingProjects: false,
    isReorderingWorkflowStatuses: false,
    isReorderingAutomationRules: false,
    isSavingWorkflowTransitions: false,
    onAddProjectMember: vi.fn(preventSubmit),
    onArchiveProject: vi.fn(),
    onArchiveWorkflowStatus: vi.fn(async () => true),
    onCreateAutomationRule: vi.fn(async () => true),
    onCancelEditingProject: vi.fn(),
    onCreateProject: vi.fn(preventSubmit),
    onCreateWorkflowStatus: vi.fn(async () => true),
    onDeleteAutomationRule: vi.fn(async () => true),
    onEditProjectDescriptionChange: vi.fn(),
    onEditProjectNameChange: vi.fn(),
    onOpenProjectBoard: vi.fn(),
    onProjectDescriptionChange: vi.fn(),
    onProjectDetailTabChange: vi.fn(),
    onProjectKeyChange: vi.fn(),
    onProjectMemberRoleChange: vi.fn(),
    onProjectMemberRoleSelectionChange: vi.fn(),
    onProjectMemberUserChange: vi.fn(),
    onProjectNameChange: vi.fn(),
    onReorderWorkflowStatuses: vi.fn(async () => true),
    onReorderAutomationRules: vi.fn(async () => true),
    onRemoveProjectMember: vi.fn(),
    onReplaceWorkflowTransitions: vi.fn(async () => true),
    onSelectIssue: vi.fn(),
    onSelectProjectDetail: vi.fn(),
    onStartEditingProject: vi.fn(),
    onUpdateProject: vi.fn(preventSubmit),
    onUpdateWorkflowStatus: vi.fn(async () => true),
    onUpdateAutomationRule: vi.fn(async () => true),
    onViewProjectIssues: vi.fn(),
    projectDescription: "",
    projectDetailError: "",
    projectDetailTab: "summary" as const,
    projectFormError: "",
    projectKey: "",
    projectMembers: [projectMember],
    projectMembersError: "",
    projectWorkflow: undefined,
    projectWorkflowError: "",
    labels: [],
    projectName: "",
    projects: [selectedProjectDetail],
    projectsError: "",
    removingProjectMemberIds: [],
    role: "admin" as const,
    selectedProjectDetail,
    selectedProjectIssues: [],
    selectedProjectMemberRole: "contributor" as const,
    selectedProjectMemberUserId: "",
    selectedProjectOpenIssues: [],
    teamMembers: [teamMember],
    updatingProjectIds: [],
    updatingProjectMemberIds: [],
    updatingWorkflowStatusIds: [],
    updatingAutomationRuleIds: [],
  };
}

test("project details expose members tab only to project managers", async () => {
  const user = userEvent.setup();
  const managerProps = projectsProps(project);
  const { rerender } = render(<ProjectsSection {...managerProps} />);

  await user.click(screen.getByRole("tab", { name: "Members" }));
  assert.equal(managerProps.onProjectDetailTabChange.mock.calls[0]?.[0], "members");
  await user.click(screen.getByRole("tab", { name: "Workflow" }));
  assert.equal(managerProps.onProjectDetailTabChange.mock.calls[1]?.[0], "workflow");
  await user.click(screen.getByRole("tab", { name: "Automation" }));
  assert.equal(managerProps.onProjectDetailTabChange.mock.calls[2]?.[0], "automation");

  const viewerProject: Project = {
    ...project,
    project_role: "viewer",
    can_write: false,
    can_manage: false,
  };
  rerender(
    <ProjectsSection
      {...projectsProps(viewerProject)}
      role="member"
      selectedProjectDetail={viewerProject}
    />,
  );

  assert.equal(screen.queryByRole("tab", { name: "Members" }), null);
  assert.equal(screen.queryByRole("tab", { name: "Workflow" }), null);
  assert.equal(screen.queryByRole("tab", { name: "Automation" }), null);
  assert.ok(screen.getByText("Viewer access"));
  assert.ok(screen.getByText(/This project is read-only/));
});

function teamProps(currentUser: CurrentUser): ComponentProps<typeof TeamSection> {
  return {
    canCreateTeamInvite: true,
    canCreateTeamMember: true,
    canResetTeamMemberPassword: false,
    copiedTeamInviteId: "",
    currentUser,
    emailDiagnostics: null,
    emailDiagnosticsError: "",
    isCreatingTeamInvite: false,
    isActive: true,
    isCreatingTeamMember: false,
    isLoadingEmailDiagnostics: false,
    isLoadingTeamInvites: false,
    isLoadingTeamMembers: false,
    onCancelResetPassword: vi.fn(),
    onCopyTeamInviteLink: vi.fn(),
    onCreateTeamInvite: vi.fn(preventSubmit),
    onCreateTeamMember: vi.fn(preventSubmit),
    onDisplayNameChange: vi.fn(),
    onEmailChange: vi.fn(),
    onInviteEmailChange: vi.fn(),
    onInviteRoleChange: vi.fn(),
    onPasswordChange: vi.fn(),
    onRevokeTeamInvite: vi.fn(),
    onRefreshEmailDiagnostics: vi.fn(),
    onResendTeamInvite: vi.fn(),
    onResetPassword: vi.fn(),
    onResetPasswordChange: vi.fn(),
    onRoleChange: vi.fn(),
    onStartResetPassword: vi.fn(),
    onUpdateTeamMember: vi.fn(),
    onUsernameChange: vi.fn(),
    passwordResetMemberId: "",
    revokingTeamInviteIds: [],
    resendingTeamInviteIds: [],
    resettingTeamMemberPasswordIds: [],
    teamInviteEmail: "",
    teamInviteFormError: "",
    teamInviteLinksById: {},
    teamInviteRole: "member",
    teamInvites: [],
    teamInvitesError: "",
    teamMemberDisplayName: "",
    teamMemberEmail: "",
    teamMemberFormError: "",
    teamMemberPassword: "",
    teamMemberResetPassword: "",
    teamMemberRole: "member",
    teamMemberUsername: "",
    teamMembers: [teamMember],
    teamMembersError: "",
    updatingTeamMemberIds: [],
  };
}

test("team view exposes admin controls and member read-only state", () => {
  const { rerender } = render(<TeamSection {...teamProps(admin)} />);

  assert.ok(screen.getByRole("button", { name: "Create invite" }));
  assert.ok(screen.getByRole("button", { name: "Create member" }));

  rerender(<TeamSection {...teamProps(memberUser)} />);

  assert.ok(screen.getByRole("heading", { name: "Team management" }));
  assert.equal(screen.queryByRole("button", { name: "Create invite" }), null);
  assert.equal(screen.queryByRole("button", { name: "Create member" }), null);
  assert.equal(
    screen.queryByRole("region", { name: "Email delivery diagnostics" }),
    null,
  );
});

test("team invites show delivery state and resend controls", async () => {
  const user = userEvent.setup();
  const pendingInvite = {
    id: "invite-1",
    workspace_id: "workspace-1",
    email: "new-member@example.com",
    role: "member" as const,
    status: "pending" as const,
    created_by: admin.id,
    created_at: "2026-06-17T10:00:00Z",
    expires_at: "2026-06-24T10:00:00Z",
    accepted_at: null,
    revoked_at: null,
    email_delivery_status: "pending" as const,
    email_queued_at: "2026-06-17T10:00:00Z",
    email_sent_at: null,
  };
  const props = {
    ...teamProps(admin),
    copiedTeamInviteId: "invite-1",
    teamInvites: [pendingInvite],
    teamInviteLinksById: {
      "invite-1": "http://localhost:5173/accept-invite?token=invite-token",
    },
  };
  render(<TeamSection {...props} />);

  assert.ok(screen.getByText("Email: Pending"));
  await user.click(screen.getByRole("button", { name: "Resend email" }));
  assert.equal(props.onResendTeamInvite.mock.calls[0]?.[0], pendingInvite);
  await user.click(screen.getByRole("button", { name: "Copied" }));
  assert.equal(props.onCopyTeamInviteLink.mock.calls[0]?.[0], pendingInvite.id);
});

test("team invites expose delivery states and disable resend while pending", () => {
  const baseInvite = {
    id: "invite-base",
    workspace_id: "workspace-1",
    email: "delivery@example.com",
    role: "member" as const,
    status: "pending" as const,
    created_by: admin.id,
    created_at: "2026-06-17T10:00:00Z",
    expires_at: "2026-06-24T10:00:00Z",
    accepted_at: null,
    revoked_at: null,
    email_queued_at: "2026-06-17T10:00:00Z",
    email_sent_at: null,
  };
  const teamInvites = (["not_sent", "pending", "processing", "sent", "failed"] as const).map(
    (deliveryStatus, index) => ({
      ...baseInvite,
      id: `invite-${deliveryStatus}`,
      email: `delivery-${index}@example.com`,
      email_delivery_status: deliveryStatus,
    }),
  );

  render(
    <TeamSection
      {...teamProps(admin)}
      resendingTeamInviteIds={["invite-pending"]}
      teamInvites={teamInvites}
    />,
  );

  for (const label of ["Not sent", "Pending", "Processing", "Sent", "Failed"]) {
    assert.ok(screen.getByText(`Email: ${label}`));
  }
  assert.ok(screen.getByRole("button", { name: "Resending..." }).hasAttribute("disabled"));
});

test("team invite resend failure remains visible without hiding the invite", () => {
  const props = teamProps(admin);
  render(
    <TeamSection
      {...props}
      teamInvitesError="Could not resend invite email."
      teamInvites={[
        {
          id: "invite-failed",
          workspace_id: "workspace-1",
          email: "failed@example.com",
          role: "member",
          status: "pending",
          created_by: admin.id,
          created_at: "2026-06-17T10:00:00Z",
          expires_at: "2026-06-24T10:00:00Z",
          accepted_at: null,
          revoked_at: null,
          email_delivery_status: "failed",
          email_queued_at: "2026-06-17T10:00:00Z",
          email_sent_at: null,
        },
      ]}
    />,
  );

  assert.ok(screen.getByText("Could not resend invite email."));
  assert.ok(screen.getByText("failed@example.com"));
  assert.ok(screen.getByRole("button", { name: "Resend email" }));
});

test("team email diagnostics show counts, failures and refresh action", async () => {
  const user = userEvent.setup();
  const diagnostics: EmailDiagnostics = {
    total: 12,
    counts: {
      pending: 2,
      processing: 1,
      sent: 8,
      failed: 1,
    },
    oldest_pending_at: "2026-06-18T10:00:00Z",
    oldest_processing_started_at: null,
    recent_terminal_failures: [
      {
        id: "email-1",
        email_type: "password_reset",
        recipient_email: "a***@example.com",
        attempt_count: 5,
        last_error: "smtp password=[redacted] token=[redacted]",
        created_at: "2026-06-18T09:00:00Z",
        updated_at: "2026-06-18T11:00:00Z",
        next_attempt_at: "2026-06-18T11:00:00Z",
        sent_at: null,
      },
    ],
  };
  const props = {
    ...teamProps(admin),
    emailDiagnostics: diagnostics,
  };

  render(<TeamSection {...props} />);

  assert.ok(screen.getByRole("heading", { name: "Delivery health" }));
  assert.ok(screen.getByText("12"));
  assert.ok(screen.getByText("password_reset"));
  assert.ok(screen.getByText("a***@example.com"));
  assert.ok(screen.getByText("smtp password=[redacted] token=[redacted]"));

  await user.click(screen.getByRole("button", { name: "Refresh" }));
  assert.equal(props.onRefreshEmailDiagnostics.mock.calls.length, 1);
});

test("team email diagnostics expose loading, error and empty states", () => {
  const { rerender } = render(
    <TeamSection
      {...teamProps(admin)}
      isLoadingEmailDiagnostics={true}
    />,
  );

  assert.ok(screen.getByText("Loading email diagnostics"));
  assert.ok(screen.getByRole("button", { name: "Refreshing..." }));

  rerender(
    <TeamSection
      {...teamProps(admin)}
      emailDiagnosticsError="Could not load email diagnostics."
    />,
  );

  assert.ok(screen.getByText("Could not load email diagnostics."));
  assert.ok(screen.getByText("No email diagnostics loaded"));
});
