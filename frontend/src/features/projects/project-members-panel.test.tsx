import assert from "node:assert/strict";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { test, vi } from "vitest";

import {
  type ProjectMember,
  type TeamMember,
} from "../../lib/api-types";
import { ProjectMembersPanel } from "./project-members-panel";

const workspaceAdmin: TeamMember = {
  id: "admin-1",
  email: "admin@example.com",
  username: "admin",
  display_name: "Admin",
  role: "admin",
  is_active: true,
  joined_at: "2026-06-07T00:00:00Z",
};

const workspaceMember: TeamMember = {
  id: "member-1",
  email: "member@example.com",
  username: "member",
  display_name: "Member",
  role: "member",
  is_active: true,
  joined_at: "2026-06-07T00:00:00Z",
};

function projectMember(
  member: TeamMember,
  role: ProjectMember["role"],
  isActive = member.is_active,
): ProjectMember {
  return {
    project_id: "project-1",
    user_id: member.id,
    email: member.email,
    username: member.username,
    display_name: member.display_name,
    role,
    workspace_role: member.role,
    is_active: isActive,
    created_at: "2026-06-07T00:00:00Z",
    updated_at: "2026-06-07T00:00:00Z",
  };
}

function panelProps(members: ProjectMember[], teamMembers: TeamMember[]) {
  return {
    error: "",
    isLoading: false,
    members,
    onAddMember: vi.fn((event) => event.preventDefault()),
    onMemberRoleChange: vi.fn(),
    onRemoveMember: vi.fn(),
    onRoleChange: vi.fn(),
    onUserChange: vi.fn(),
    removingMemberIds: [],
    role: "contributor" as const,
    selectedUserId: "",
    teamMembers,
    updatingMemberIds: [],
  };
}

test("adds an available workspace member and changes project role", async () => {
  const user = userEvent.setup();
  const props = panelProps(
    [projectMember(workspaceAdmin, "lead")],
    [workspaceAdmin, workspaceMember],
  );
  const { rerender } = render(<ProjectMembersPanel {...props} />);

  await user.selectOptions(screen.getByLabelText("Workspace member"), "member-1");
  assert.equal(props.onUserChange.mock.calls[0]?.[0], "member-1");

  const selectedProps = { ...props, selectedUserId: "member-1" };
  rerender(<ProjectMembersPanel {...selectedProps} />);
  await user.selectOptions(screen.getByLabelText("New project member role"), "viewer");
  await user.click(screen.getByRole("button", { name: "Add member" }));

  assert.equal(props.onRoleChange.mock.calls[0]?.[0], "viewer");
  assert.equal(props.onAddMember.mock.calls.length, 1);
});

test("forwards role changes and removal actions", async () => {
  const user = userEvent.setup();
  const member = projectMember(workspaceMember, "contributor");
  const props = panelProps(
    [projectMember(workspaceAdmin, "lead"), member],
    [workspaceAdmin, workspaceMember],
  );
  render(<ProjectMembersPanel {...props} />);

  await user.selectOptions(
    screen.getByLabelText("Project role for Member"),
    "viewer",
  );
  await user.click(screen.getAllByRole("button", { name: "Remove" })[1]);

  assert.equal(props.onMemberRoleChange.mock.calls[0]?.[0], member);
  assert.equal(props.onMemberRoleChange.mock.calls[0]?.[1], "viewer");
  assert.equal(props.onRemoveMember.mock.calls[0]?.[0], member);
});

test("keeps inactive membership removable but role read-only", () => {
  const inactive = projectMember({ ...workspaceMember, is_active: false }, "viewer");
  render(
    <ProjectMembersPanel
      {...panelProps([inactive], [{ ...workspaceMember, is_active: false }])}
    />,
  );

  assert.equal(
    screen.getByLabelText("Project role for Member").hasAttribute("disabled"),
    true,
  );
  assert.equal(
    screen.getByRole("button", { name: "Remove" }).hasAttribute("disabled"),
    false,
  );
  assert.ok(screen.getByText(/remain visible for audit/));
});

test("protects the last active lead without a workspace admin", () => {
  const lead = projectMember(workspaceMember, "lead");
  render(<ProjectMembersPanel {...panelProps([lead], [workspaceMember])} />);

  assert.equal(
    screen.getByLabelText("Project role for Member").hasAttribute("disabled"),
    true,
  );
  assert.equal(
    screen.getByRole("button", { name: "Remove" }).hasAttribute("disabled"),
    true,
  );
  assert.ok(screen.getByText(/Add another active lead/));
});

test("explains workspace admin effective access", () => {
  render(
    <ProjectMembersPanel
      {...panelProps(
        [projectMember(workspaceAdmin, "contributor")],
        [workspaceAdmin],
      )}
    />,
  );

  assert.ok(screen.getByText(/Workspace admins keep full project access/));
  assert.equal(
    screen.getByRole("button", { name: "Remove" }).hasAttribute("disabled"),
    false,
  );
});
