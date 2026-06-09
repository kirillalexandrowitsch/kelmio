import assert from "node:assert/strict";
import { type ComponentProps, type FormEvent } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { test, vi } from "vitest";

import { SignInScreen } from "./auth/auth-screens";
import { IssueCreateForm } from "./issues/issue-create-form";
import { SavedFiltersPanel } from "./issues/saved-filters-panel";
import { NotificationsSection } from "./notifications/notifications-section";
import { SprintsSection } from "./sprints/sprints-section";
import { TeamSection } from "./team/team-section";
import {
  type AppNotification,
  type CurrentUser,
  type Project,
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
  workspace: { id: "workspace-1", role: "admin" },
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
  const onSubmit = vi.fn(preventSubmit);

  render(
    <SignInScreen
      canSignIn
      error=""
      isSubmitting={false}
      loginValue=""
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
      status="todo"
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
    onSprintIssueDragOver: vi.fn(),
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
  };

  render(<SprintsSection {...props} />);

  await user.click(screen.getByRole("button", { name: "Start sprint" }));

  assert.equal(onStartSprint.mock.calls[0]?.[0], sprint);
  assert.equal(
    screen.getByRole("button", { name: "Complete sprint" }).hasAttribute("disabled"),
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

function teamProps(currentUser: CurrentUser): ComponentProps<typeof TeamSection> {
  return {
    canCreateTeamInvite: true,
    canCreateTeamMember: true,
    canResetTeamMemberPassword: false,
    copiedTeamInviteId: "",
    currentUser,
    isCreatingTeamInvite: false,
    isActive: true,
    isCreatingTeamMember: false,
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
    onResetPassword: vi.fn(),
    onResetPasswordChange: vi.fn(),
    onRoleChange: vi.fn(),
    onStartResetPassword: vi.fn(),
    onUpdateTeamMember: vi.fn(),
    onUsernameChange: vi.fn(),
    passwordResetMemberId: "",
    revokingTeamInviteIds: [],
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
});
