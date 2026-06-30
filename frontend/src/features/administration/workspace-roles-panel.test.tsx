import assert from "node:assert/strict";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, test, vi } from "vitest";

const apiMocks = vi.hoisted(() => {
  class MockApiError extends Error {
    status: number;
    code: string;

    constructor(message: string, status: number, code = "") {
      super(message);
      this.name = "ApiError";
      this.status = status;
      this.code = code;
    }
  }

  return {
    ApiError: MockApiError,
    listWorkspaceRoleAssignments: vi.fn(),
    createWorkspaceRoleAssignment: vi.fn(),
    deleteWorkspaceRoleAssignment: vi.fn(),
    listDirectory: vi.fn(),
    listGroups: vi.fn(),
  };
});

vi.mock("../../lib/api", () => apiMocks);

import { WorkspaceRolesPanel } from "./workspace-roles-panel";

beforeEach(() => {
  apiMocks.listWorkspaceRoleAssignments.mockReset();
  apiMocks.createWorkspaceRoleAssignment.mockReset();
  apiMocks.deleteWorkspaceRoleAssignment.mockReset();
  apiMocks.listDirectory.mockReset();
  apiMocks.listGroups.mockReset();
});

function renderPanel() {
  return render(
    <WorkspaceRolesPanel
      workspaceId="ws-1"
      workspaceName="Home"
      onClose={vi.fn()}
    />,
  );
}

test("lists existing assignments and assigns a user role", async () => {
  const user = userEvent.setup();
  apiMocks.listWorkspaceRoleAssignments.mockResolvedValue({
    assignments: [
      {
        id: "ra-1",
        subject_type: "group",
        subject_id: "group-1",
        subject_name: "Engineers",
        role: "member",
      },
    ],
  });
  apiMocks.listDirectory.mockResolvedValue({
    users: [
      { user_id: "user-1", username: "ada", display_name: "Ada Lovelace", email: "ada@example.com", role: "org_member" },
    ],
  });
  apiMocks.listGroups.mockResolvedValue({
    groups: [{ id: "group-1", name: "Engineers", description: "", member_count: 2 }],
  });
  apiMocks.createWorkspaceRoleAssignment.mockResolvedValue({
    id: "ra-2",
    subject_type: "user",
    subject_id: "user-1",
    subject_name: "Ada Lovelace",
    role: "admin",
  });

  renderPanel();

  await screen.findByText("Engineers");

  await user.selectOptions(screen.getByLabelText("Subject"), "user");
  await user.selectOptions(screen.getByLabelText("Person"), "user-1");
  await user.selectOptions(screen.getByLabelText("Role"), "admin");
  await user.click(screen.getByRole("button", { name: "Assign" }));

  await screen.findByText("Ada Lovelace");
  assert.deepEqual(apiMocks.createWorkspaceRoleAssignment.mock.calls[0], [
    "ws-1",
    { subject_type: "user", subject_id: "user-1", role: "admin" },
  ]);
});

test("offers groups when the subject type is group", async () => {
  const user = userEvent.setup();
  apiMocks.listWorkspaceRoleAssignments.mockResolvedValue({ assignments: [] });
  apiMocks.listDirectory.mockResolvedValue({ users: [] });
  apiMocks.listGroups.mockResolvedValue({
    groups: [{ id: "group-1", name: "Engineers", description: "", member_count: 2 }],
  });

  renderPanel();

  await screen.findByText("No role assignments yet");
  await user.selectOptions(screen.getByLabelText("Subject"), "group");
  assert.ok(screen.getByRole("option", { name: "Engineers" }));
});

test("removes an assignment", async () => {
  const user = userEvent.setup();
  apiMocks.listWorkspaceRoleAssignments.mockResolvedValue({
    assignments: [
      {
        id: "ra-1",
        subject_type: "user",
        subject_id: "user-1",
        subject_name: "Ada Lovelace",
        role: "admin",
      },
    ],
  });
  apiMocks.listDirectory.mockResolvedValue({ users: [] });
  apiMocks.listGroups.mockResolvedValue({ groups: [] });
  apiMocks.deleteWorkspaceRoleAssignment.mockResolvedValue(undefined);

  renderPanel();

  const item = (await screen.findByText("Ada Lovelace")).closest("li");
  await user.click(within(item as HTMLElement).getByRole("button", { name: "Remove" }));

  await waitFor(() => {
    assert.deepEqual(apiMocks.deleteWorkspaceRoleAssignment.mock.calls[0], [
      "ws-1",
      "ra-1",
    ]);
  });
  await waitFor(() => {
    assert.equal(screen.queryByText("Ada Lovelace"), null);
  });
});
